package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// version is injected at release time with -X main.version=<tag>.
var version = "dev"

var userAgent = "hyprhaven/" + version + " (+https://github.com/ktkv419/hyprhaven)"

type options struct {
	interval    time.Duration
	pageLimit   int
	page        int
	sorting     string
	order       string
	topRange    string
	categories  string
	query       string
	purity      string
	apiKey      string
	atLeast     string
	resolutions string
	ratios      string
	colors      string
	seed        string
	ids         string
	once        bool
}

func main() {
	log.SetPrefix("hyprhaven: ")
	if err := run(os.Args[1:]); err != nil {
		log.Print(err)
		os.Exit(1)
	}
}

func run(args []string) error {
	opts, err := parseFlags(args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if lock, err := acquireInstanceLock(); err != nil {
		log.Printf("warning: single-instance lock unavailable: %v", err)
	} else {
		defer lock.Release()
	}

	client := newWallhavenClient(newHTTPClient(), opts.apiKey)
	params := searchParams{
		Sorting:     opts.sorting,
		Order:       opts.order,
		TopRange:    opts.topRange,
		Categories:  opts.categories,
		Query:       opts.query,
		Purity:      opts.purity,
		AtLeast:     opts.atLeast,
		Resolutions: opts.resolutions,
		Ratios:      opts.ratios,
		Colors:      opts.colors,
		Seed:        opts.seed,
	}

	for {
		err := cycle(ctx, client, params, opts)
		if ctx.Err() != nil {
			log.Print("shutting down")
			return nil
		}
		if err != nil {
			if opts.once {
				return err
			}
			log.Printf("error: %v", err)
		}
		if opts.once {
			return nil
		}

		log.Printf("next wallpaper in %s", opts.interval)
		select {
		case <-ctx.Done():
			log.Print("shutting down")
			return nil
		case <-time.After(opts.interval):
		}
	}
}

func cycle(ctx context.Context, client *wallhavenClient, params searchParams, opts options) error {
	info, err := client.pickWallpaper(ctx, params, opts.pageLimit, opts.page, opts.ids)
	if err != nil {
		return fmt.Errorf("fetch wallpaper: %w", err)
	}
	if info.Resolution != "" {
		log.Printf("selected %s (%s)", info.Path, info.Resolution)
	} else {
		log.Printf("selected %s", info.Path)
	}

	file, err := downloadWallpaper(ctx, client.http, info)
	if err != nil {
		return err
	}
	if err := applyWallpaper(file); err != nil {
		return err
	}
	cleanupWallpapers(filepath.Dir(file), file)
	log.Print("wallpaper set")
	return nil
}

func parseFlags(args []string) (options, error) {
	var opts options

	fs := flag.NewFlagSet("hyprhaven", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	fs.Usage = func() {
		fmt.Fprintf(fs.Output(), "Usage: %s [options]\n\nOptions:\n", fs.Name())
		fs.PrintDefaults()
	}

	interval := fs.Int("t", 15, "minutes between wallpaper changes")
	pageLimit := fs.Int("pq", 5, "maximum search page to pick a random wallpaper from")
	page := fs.Int("page", 0, "exact search page to use (0 picks a random page up to -pq)")
	sorting := fs.String("s", "favorites", "sorting: date_added, relevance, random, views, favorites, toplist")
	order := fs.String("o", "desc", "sorting order: desc, asc")
	topRange := fs.String("tr", "1M", "toplist time range when -s toplist: 1d, 3d, 1w, 1M, 3M, 6M, 1y")
	categories := fs.String("c", "111", "categories: general, anime, people (e.g. 111 or general,anime)")
	query := fs.String("q", "", "search query")
	purity := fs.String("p", "100", "purity: sfw, sketchy, nsfw (e.g. 100, 110, 111 or sfw,sketchy)")
	apiKey := fs.String("apikey", "", "wallhaven API key (required for NSFW and some sorting modes)")
	atLeast := fs.String("sz", "1920x1080", "minimum resolution (e.g. 1920x1080)")
	resolutions := fs.String("res", "", "exact resolutions to allow, comma-separated; overrides -sz (e.g. 1920x1080,2560x1440)")
	ratios := fs.String("r", "landscape", "ratios: landscape, portrait, all, or WxH (e.g. 16x9)")
	colors := fs.String("color", "", "color filter from the wallhaven palette, comma-separated hex (e.g. 990000,000000)")
	seed := fs.String("seed", "", "random seed for -s random (six letters or digits)")
	ids := fs.String("id", "", "comma-separated wallpaper IDs (e.g. 45p3m5,1p6d79)")
	once := fs.Bool("once", false, "set one wallpaper and exit")

	if err := fs.Parse(args); err != nil {
		return options{}, err
	}
	if fs.NArg() > 0 {
		return options{}, fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}
	if *interval < 0 {
		return options{}, errors.New("-t must not be negative")
	}
	if *page < 0 {
		return options{}, errors.New("-page must not be negative")
	}

	sortingValue, err := normalizeSorting(*sorting)
	if err != nil {
		return options{}, err
	}
	orderValue, err := normalizeOrder(*order)
	if err != nil {
		return options{}, err
	}
	topRangeValue, err := normalizeTopRange(*topRange)
	if err != nil {
		return options{}, err
	}
	categoriesValue, err := normalizeBits(*categories, categoryNames)
	if err != nil {
		return options{}, err
	}
	purityValue, err := normalizeBits(*purity, purityNames)
	if err != nil {
		return options{}, err
	}
	purityValue = sanitizePurity(purityValue, *apiKey)
	if purityValue == "000" {
		if *apiKey == "" {
			return options{}, errors.New("NSFW is the only purity selected, which requires -apikey")
		}
		return options{}, errors.New("no purity selected: enable at least one of sfw, sketchy, nsfw")
	}

	atLeastValue := ""
	if strings.TrimSpace(*atLeast) != "" {
		if atLeastValue, err = normalizeDimensions(*atLeast); err != nil {
			return options{}, err
		}
	}
	resolutionsValue, err := normalizeResolutions(*resolutions)
	if err != nil {
		return options{}, err
	}
	ratiosValue, err := normalizeRatios(*ratios)
	if err != nil {
		return options{}, err
	}
	colorsValue, err := normalizeColors(*colors)
	if err != nil {
		return options{}, err
	}
	seedValue, err := normalizeSeed(*seed)
	if err != nil {
		return options{}, err
	}
	if seedValue != "" && sortingValue != "random" {
		return options{}, errors.New("-seed requires -s random")
	}

	opts = options{
		interval:    time.Duration(*interval) * time.Minute,
		pageLimit:   *pageLimit,
		page:        *page,
		sorting:     sortingValue,
		order:       orderValue,
		topRange:    topRangeValue,
		categories:  categoriesValue,
		query:       *query,
		purity:      purityValue,
		apiKey:      *apiKey,
		atLeast:     atLeastValue,
		resolutions: resolutionsValue,
		ratios:      ratiosValue,
		colors:      colorsValue,
		seed:        seedValue,
		ids:         *ids,
		once:        *once,
	}
	if opts.pageLimit < 1 {
		opts.pageLimit = 1
	}
	return opts, nil
}
