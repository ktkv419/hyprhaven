package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const wallhavenAPI = "https://wallhaven.cc/api/v1"

var (
	errNoWallpapers = errors.New("no wallpapers matched the search")
	errAllUsed      = errors.New("all matching wallpapers are in the no-repeat history")
)

// maxPageAttempts bounds how many result pages are fetched while looking for a
// wallpaper that is not in the recent-history window.
const maxPageAttempts = 5

type searchParams struct {
	Sorting     string
	Order       string
	TopRange    string
	Categories  string
	Query       string
	Purity      string
	AtLeast     string
	Resolutions string
	Ratios      string
	Colors      string
	Seed        string
}

func (p searchParams) values() url.Values {
	v := url.Values{}
	set := func(key, value string) {
		if value != "" {
			v.Set(key, value)
		}
	}
	set("sorting", p.Sorting)
	set("order", p.Order)
	if p.Sorting == "toplist" {
		set("topRange", p.TopRange)
	}
	set("categories", p.Categories)
	set("q", p.Query)
	set("purity", p.Purity)
	// wallhaven ignores resolutions when atleast is also present, so an exact
	// resolution list takes precedence over the minimum resolution.
	if p.Resolutions != "" {
		set("resolutions", p.Resolutions)
	} else {
		set("atleast", p.AtLeast)
	}
	set("ratios", p.Ratios)
	set("colors", p.Colors)
	if p.Sorting == "random" {
		set("seed", p.Seed)
	}
	return v
}

type wallpaperInfo struct {
	ID         string `json:"id"`
	Path       string `json:"path"`
	Resolution string `json:"resolution"`
	FileType   string `json:"file_type"`
}

type searchMeta struct {
	CurrentPage flexInt `json:"current_page"`
	LastPage    flexInt `json:"last_page"`
	PerPage     flexInt `json:"per_page"`
	Total       flexInt `json:"total"`
	Seed        string  `json:"seed"`
}

type searchResponse struct {
	Data []wallpaperInfo `json:"data"`
	Meta searchMeta      `json:"meta"`
}

type singleResponse struct {
	Data wallpaperInfo `json:"data"`
}

type wallhavenClient struct {
	http   *http.Client
	apiKey string
}

func newWallhavenClient(client *http.Client, apiKey string) *wallhavenClient {
	return &wallhavenClient{http: client, apiKey: apiKey}
}

// pickWallpaper returns one wallpaper, either from an explicit ID list, an
// explicit search page, or a randomized page of a search. skip, when non-nil,
// excludes wallpaper IDs that should not be reused.
func (c *wallhavenClient) pickWallpaper(ctx context.Context, p searchParams, pageLimit, page int, ids string, skip func(string) bool) (wallpaperInfo, error) {
	if strings.TrimSpace(ids) != "" {
		return c.pickByIDs(ctx, ids, skip)
	}
	if page > 0 {
		return c.pickPage(ctx, p, page, skip)
	}
	return c.pickRandom(ctx, p, pageLimit, skip)
}

func (c *wallhavenClient) pickPage(ctx context.Context, p searchParams, page int, skip func(string) bool) (wallpaperInfo, error) {
	resp, err := c.search(ctx, p, page)
	if err != nil {
		return wallpaperInfo{}, err
	}
	items := unusedWallpapers(resp.Data, skip)
	if len(items) == 0 {
		if len(resp.Data) == 0 {
			return wallpaperInfo{}, fmt.Errorf("%w (page %d)", errNoWallpapers, page)
		}
		return wallpaperInfo{}, fmt.Errorf("%w (page %d)", errAllUsed, page)
	}
	return items[rand.IntN(len(items))], nil
}

func (c *wallhavenClient) pickRandom(ctx context.Context, p searchParams, pageLimit int, skip func(string) bool) (wallpaperInfo, error) {
	first, err := c.search(ctx, p, 1)
	if err != nil {
		return wallpaperInfo{}, err
	}
	if len(first.Data) == 0 {
		return wallpaperInfo{}, errNoWallpapers
	}

	// Random sorting returns a seed that must be reused on later pages so the
	// shuffle stays stable; without it each page is a different ordering.
	if p.Sorting == "random" && p.Seed == "" {
		p.Seed = first.Meta.Seed
	}

	last := maxPage(int(first.Meta.LastPage), pageLimit)
	tried := make(map[int]struct{}, maxPageAttempts)
	seen := false
	for attempt := 0; attempt < maxPageAttempts; attempt++ {
		page := unusedPage(last, tried)
		if page == 0 {
			break
		}
		tried[page] = struct{}{}

		items := first.Data
		if page > 1 {
			resp, err := c.search(ctx, p, page)
			if err != nil {
				return wallpaperInfo{}, err
			}
			items = resp.Data
		}
		if len(items) > 0 {
			seen = true
		}

		if items = unusedWallpapers(items, skip); len(items) > 0 {
			return items[rand.IntN(len(items))], nil
		}
	}
	if !seen {
		return wallpaperInfo{}, errNoWallpapers
	}
	return wallpaperInfo{}, fmt.Errorf("%w (checked %d page(s))", errAllUsed, len(tried))
}

// unusedPage returns a random page in [1, last] that is not in tried. It falls
// back to a linear scan when random picks collide, and returns 0 when every
// page has been tried.
func unusedPage(last int, tried map[int]struct{}) int {
	if last < 1 {
		return 0
	}
	for i := 0; i < 8; i++ {
		page := rand.IntN(last) + 1
		if _, ok := tried[page]; !ok {
			return page
		}
	}
	for page := 1; page <= last; page++ {
		if _, ok := tried[page]; !ok {
			return page
		}
	}
	return 0
}

func (c *wallhavenClient) pickByIDs(ctx context.Context, raw string, skip func(string) bool) (wallpaperInfo, error) {
	ids := parseIDs(raw)
	if len(ids) == 0 {
		return wallpaperInfo{}, errors.New("no wallpaper IDs provided")
	}
	ids = unusedIDs(ids, skip)
	if len(ids) == 0 {
		return wallpaperInfo{}, fmt.Errorf("%w (every ID from -id)", errAllUsed)
	}
	return c.wallpaper(ctx, ids[rand.IntN(len(ids))])
}

func unusedWallpapers(items []wallpaperInfo, skip func(string) bool) []wallpaperInfo {
	if skip == nil {
		return items
	}
	out := make([]wallpaperInfo, 0, len(items))
	for _, item := range items {
		if !skip(item.ID) {
			out = append(out, item)
		}
	}
	return out
}

func unusedIDs(ids []string, skip func(string) bool) []string {
	if skip == nil {
		return ids
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if !skip(id) {
			out = append(out, id)
		}
	}
	return out
}

func (c *wallhavenClient) search(ctx context.Context, p searchParams, page int) (searchResponse, error) {
	u, err := url.Parse(wallhavenAPI + "/search")
	if err != nil {
		return searchResponse{}, err
	}

	q := p.values()
	if page > 1 {
		q.Set("page", strconv.Itoa(page))
	}
	if c.apiKey != "" {
		q.Set("apikey", c.apiKey)
	}
	u.RawQuery = q.Encode()

	var resp searchResponse
	if err := c.getJSON(ctx, u.String(), &resp); err != nil {
		return searchResponse{}, err
	}
	return resp, nil
}

func (c *wallhavenClient) wallpaper(ctx context.Context, id string) (wallpaperInfo, error) {
	u, err := url.Parse(wallhavenAPI + "/w/" + url.PathEscape(id))
	if err != nil {
		return wallpaperInfo{}, err
	}
	if c.apiKey != "" {
		q := u.Query()
		q.Set("apikey", c.apiKey)
		u.RawQuery = q.Encode()
	}

	var resp singleResponse
	if err := c.getJSON(ctx, u.String(), &resp); err != nil {
		return wallpaperInfo{}, err
	}
	if resp.Data.Path == "" {
		return wallpaperInfo{}, fmt.Errorf("wallpaper %q has no download path", id)
	}
	return resp.Data, nil
}

func (c *wallhavenClient) getJSON(ctx context.Context, rawURL string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("request wallhaven: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("wallhaven returned %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 8<<20)).Decode(dst); err != nil {
		return fmt.Errorf("decode wallhaven response: %w", err)
	}
	return nil
}

// maxPage clamps the random page range to the requested limit and the number of
// pages the search actually has.
func maxPage(lastPage, pageLimit int) int {
	if lastPage < 1 {
		return 1
	}
	if pageLimit > 0 && pageLimit < lastPage {
		return pageLimit
	}
	return lastPage
}

func parseIDs(raw string) []string {
	parts := strings.Split(raw, ",")
	ids := make([]string, 0, len(parts))
	for _, part := range parts {
		if id := strings.TrimSpace(part); id != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// sanitizePurity disables NSFW results when no API key is supplied, mirroring
// the wallhaven API rule. The purity string is a three-bit mask where the last
// bit is NSFW.
func sanitizePurity(purity, apiKey string) string {
	if apiKey == "" && len(purity) == 3 && purity[2] == '1' {
		return purity[:2] + "0"
	}
	return purity
}

// The values below mirror the wallhaven search API. Anything not listed here is
// silently ignored by the server (or, for categories, returns a 500), so the
// client validates up front instead of producing mysterious empty results.
var (
	sortingNames  = []string{"date_added", "relevance", "random", "views", "favorites", "toplist"}
	orderNames    = []string{"desc", "asc"}
	topRangeNames = []string{"1d", "3d", "1w", "1M", "3M", "6M", "1y"}
	purityNames   = []string{"sfw", "sketchy", "nsfw"}
	categoryNames = []string{"general", "anime", "people"}

	// colorPalette is the only set of colors the wallhaven search accepts.
	colorPalette = []string{
		"660000", "990000", "cc0000", "cc3333", "ea4c88", "993399",
		"663399", "333399", "0066cc", "0099cc", "66cccc", "77cc33",
		"669900", "336600", "666600", "999900", "cccc33", "ffff00",
		"ffcc33", "ff9900", "ff6600", "cc6633", "996633", "663300",
		"000000", "999999", "cccccc", "ffffff", "424153",
	}
)

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

// normalizeBits turns a three-digit mask such as "111" or a comma-separated
// list of names such as "sfw,sketchy" into the three-digit mask.
func normalizeBits(raw string, names []string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return "", nil
	}

	compact := strings.NewReplacer(" ", "", "\t", "", ",", "", "_", "", "-", "").Replace(s)
	if len(compact) == 3 && strings.Trim(compact, "01") == "" {
		return compact, nil
	}

	var on [3]bool
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := -1
		for i, name := range names {
			if part == name {
				idx = i
				break
			}
		}
		if idx < 0 {
			return "", fmt.Errorf("invalid value %q: use a three-digit mask or %s", raw, strings.Join(names, ", "))
		}
		on[idx] = true
	}

	var b strings.Builder
	for _, enabled := range on {
		if enabled {
			b.WriteByte('1')
		} else {
			b.WriteByte('0')
		}
	}
	return b.String(), nil
}

func normalizeSorting(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.NewReplacer(" ", "_", "-", "_").Replace(s)
	if !contains(sortingNames, s) {
		return "", fmt.Errorf("invalid sorting %q: use %s", raw, strings.Join(sortingNames, ", "))
	}
	return s, nil
}

func normalizeOrder(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	if !contains(orderNames, s) {
		return "", fmt.Errorf("invalid order %q: use %s", raw, strings.Join(orderNames, ", "))
	}
	return s, nil
}

func normalizeTopRange(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	for _, name := range topRangeNames {
		if strings.ToLower(name) == s {
			return name, nil
		}
	}
	return "", fmt.Errorf("invalid top range %q: use one of %s", raw, strings.Join(topRangeNames, ", "))
}

// normalizeDimensions validates a WxH value and normalizes common separators.
func normalizeDimensions(raw string) (string, error) {
	s := strings.ToLower(strings.TrimSpace(raw))
	s = strings.NewReplacer("×", "x", ":", "x", " ", "").Replace(s)
	parts := strings.Split(s, "x")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid dimensions %q: use WIDTHxHEIGHT (e.g. 1920x1080)", raw)
	}
	for _, part := range parts {
		if part == "" || strings.Trim(part, "0123456789") != "" {
			return "", fmt.Errorf("invalid dimensions %q: use WIDTHxHEIGHT (e.g. 1920x1080)", raw)
		}
	}
	return parts[0] + "x" + parts[1], nil
}

func normalizeResolutions(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if strings.TrimSpace(part) == "" {
			continue
		}
		value, err := normalizeDimensions(part)
		if err != nil {
			return "", err
		}
		out = append(out, value)
	}
	if len(out) == 0 {
		return "", nil
	}
	return strings.Join(out, ","), nil
}

func normalizeRatios(raw string) (string, error) {
	s := strings.TrimSpace(strings.ToLower(raw))
	if s == "" {
		return "", nil
	}
	if contains([]string{"landscape", "portrait", "all"}, s) {
		return s, nil
	}
	return normalizeResolutions(s)
}

func normalizeColors(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		color := strings.TrimPrefix(strings.TrimSpace(strings.ToLower(part)), "#")
		if color == "" {
			continue
		}
		if !contains(colorPalette, color) {
			return "", fmt.Errorf("unsupported color %q: use a hex value from %s", part, strings.Join(colorPalette, ", "))
		}
		out = append(out, color)
	}
	if len(out) == 0 {
		return "", nil
	}
	return strings.Join(out, ","), nil
}

func normalizeSeed(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", nil
	}
	if len(s) != 6 || strings.Trim(s, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") != "" {
		return "", fmt.Errorf("invalid seed %q: use six letters or digits", raw)
	}
	return s, nil
}

// flexInt accepts both JSON numbers and numeric strings. Some wallhaven
// responses have been observed to encode these fields as strings.
type flexInt int

func (f *flexInt) UnmarshalJSON(data []byte) error {
	data = bytes.TrimSpace(data)
	if len(data) == 0 || bytes.Equal(data, []byte("null")) {
		*f = 0
		return nil
	}
	if data[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return fmt.Errorf("invalid integer %q: %w", s, err)
		}
		*f = flexInt(n)
		return nil
	}

	var n int
	if err := json.Unmarshal(data, &n); err != nil {
		return err
	}
	*f = flexInt(n)
	return nil
}
