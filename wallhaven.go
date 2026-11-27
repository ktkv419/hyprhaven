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

var errNoWallpapers = errors.New("no wallpapers matched the search")

type searchParams struct {
	Sorting    string
	Categories string
	Query      string
	Purity     string
	AtLeast    string
	Ratios     string
}

func (p searchParams) values() url.Values {
	v := url.Values{}
	set := func(key, value string) {
		if value != "" {
			v.Set(key, value)
		}
	}
	set("sorting", p.Sorting)
	set("categories", p.Categories)
	set("q", p.Query)
	set("purity", p.Purity)
	set("atleast", p.AtLeast)
	set("ratios", p.Ratios)
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

// pickWallpaper returns one wallpaper, either from an explicit ID list or from
// a randomized page of a search.
func (c *wallhavenClient) pickWallpaper(ctx context.Context, p searchParams, pageLimit int, ids string) (wallpaperInfo, error) {
	if strings.TrimSpace(ids) != "" {
		return c.pickByIDs(ctx, ids)
	}
	return c.pickRandom(ctx, p, pageLimit)
}

func (c *wallhavenClient) pickRandom(ctx context.Context, p searchParams, pageLimit int) (wallpaperInfo, error) {
	first, err := c.search(ctx, p, 1)
	if err != nil {
		return wallpaperInfo{}, err
	}
	if len(first.Data) == 0 {
		return wallpaperInfo{}, errNoWallpapers
	}

	page := 1
	if last := maxPage(int(first.Meta.LastPage), pageLimit); last > 1 {
		page = rand.IntN(last) + 1
	}

	items := first.Data
	if page > 1 {
		resp, err := c.search(ctx, p, page)
		if err != nil {
			return wallpaperInfo{}, err
		}
		items = resp.Data
	}
	if len(items) == 0 {
		return wallpaperInfo{}, errNoWallpapers
	}
	return items[rand.IntN(len(items))], nil
}

func (c *wallhavenClient) pickByIDs(ctx context.Context, raw string) (wallpaperInfo, error) {
	ids := parseIDs(raw)
	if len(ids) == 0 {
		return wallpaperInfo{}, errors.New("no wallpaper IDs provided")
	}
	return c.wallpaper(ctx, ids[rand.IntN(len(ids))])
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
// the wallhaven API rule.
func sanitizePurity(purity, apiKey string) string {
	if apiKey == "" && strings.HasSuffix(purity, "1") {
		return purity[:len(purity)-1] + "0"
	}
	return purity
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
