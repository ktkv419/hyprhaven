package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSanitizePurity(t *testing.T) {
	tests := []struct {
		name   string
		purity string
		apiKey string
		want   string
	}{
		{"safe stays safe", "100", "", "100"},
		{"nsfw disabled without key", "101", "", "100"},
		{"all disabled without key", "111", "", "110"},
		{"nsfw only disabled without key", "001", "", "000"},
		{"nsfw kept with key", "101", "secret", "101"},
		{"empty", "", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := sanitizePurity(tt.purity, tt.apiKey); got != tt.want {
				t.Fatalf("sanitizePurity(%q, %q) = %q, want %q", tt.purity, tt.apiKey, got, tt.want)
			}
		})
	}
}

func TestNormalizeBits(t *testing.T) {
	tests := []struct {
		name  string
		raw   string
		names []string
		want  string
	}{
		{"mask", "111", purityNames, "111"},
		{"mask with spaces", "1 0 1", purityNames, "101"},
		{"names", "sfw,nsfw", purityNames, "101"},
		{"single name", "sketchy", purityNames, "010"},
		{"categories", "general,people", categoryNames, "101"},
		{"empty", "", purityNames, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeBits(tt.raw, tt.names)
			if err != nil {
				t.Fatalf("normalizeBits(%q) error: %v", tt.raw, err)
			}
			if got != tt.want {
				t.Fatalf("normalizeBits(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}

	for _, raw := range []string{"12", "sfw,bogus", "abcd"} {
		if _, err := normalizeBits(raw, purityNames); err == nil {
			t.Errorf("normalizeBits(%q) expected an error", raw)
		}
	}
}

func TestNormalizeSorting(t *testing.T) {
	tests := map[string]string{
		"date_added": "date_added",
		"Date Added": "date_added",
		"date-added": "date_added",
		"RANDOM":     "random",
		"toplist":    "toplist",
	}
	for raw, want := range tests {
		got, err := normalizeSorting(raw)
		if err != nil {
			t.Fatalf("normalizeSorting(%q) error: %v", raw, err)
		}
		if got != want {
			t.Errorf("normalizeSorting(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := normalizeSorting("popular"); err == nil {
		t.Error("normalizeSorting(popular) expected an error")
	}
}

func TestNormalizeOrderAndTopRange(t *testing.T) {
	if got, err := normalizeOrder("ASC"); err != nil || got != "asc" {
		t.Errorf("normalizeOrder(ASC) = %q, %v", got, err)
	}
	if _, err := normalizeOrder("sideways"); err == nil {
		t.Error("normalizeOrder(sideways) expected an error")
	}

	for raw, want := range map[string]string{"1d": "1d", "1m": "1M", "3M": "3M", "6m": "6M", "1Y": "1y"} {
		got, err := normalizeTopRange(raw)
		if err != nil {
			t.Fatalf("normalizeTopRange(%q) error: %v", raw, err)
		}
		if got != want {
			t.Errorf("normalizeTopRange(%q) = %q, want %q", raw, got, want)
		}
	}
	if _, err := normalizeTopRange("2w"); err == nil {
		t.Error("normalizeTopRange(2w) expected an error")
	}
}

func TestNormalizeDimensionsAndLists(t *testing.T) {
	for raw, want := range map[string]string{
		"1920x1080": "1920x1080",
		"1920X1080": "1920x1080",
		"1920×1080": "1920x1080",
		"1920:1080": "1920x1080",
		" 16 x 9 ":  "16x9",
		"3840x2160": "3840x2160",
	} {
		got, err := normalizeDimensions(raw)
		if err != nil {
			t.Fatalf("normalizeDimensions(%q) error: %v", raw, err)
		}
		if got != want {
			t.Errorf("normalizeDimensions(%q) = %q, want %q", raw, got, want)
		}
	}
	for _, raw := range []string{"1920", "x1080", "1920x", "abcxdef", "1920x1080x1"} {
		if _, err := normalizeDimensions(raw); err == nil {
			t.Errorf("normalizeDimensions(%q) expected an error", raw)
		}
	}

	got, err := normalizeResolutions("1920x1080, 2560:1440")
	if err != nil {
		t.Fatalf("normalizeResolutions error: %v", err)
	}
	if want := "1920x1080,2560x1440"; got != want {
		t.Errorf("normalizeResolutions = %q, want %q", got, want)
	}

	for raw, want := range map[string]string{
		"landscape": "landscape",
		"PORTRAIT":  "portrait",
		"all":       "all",
		"16x9,9x16": "16x9,9x16",
		"21:9":      "21x9",
	} {
		got, err := normalizeRatios(raw)
		if err != nil {
			t.Fatalf("normalizeRatios(%q) error: %v", raw, err)
		}
		if got != want {
			t.Errorf("normalizeRatios(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNormalizeColors(t *testing.T) {
	got, err := normalizeColors("#990000, 000000")
	if err != nil {
		t.Fatalf("normalizeColors error: %v", err)
	}
	if want := "990000,000000"; got != want {
		t.Errorf("normalizeColors = %q, want %q", got, want)
	}

	if got, err := normalizeColors(""); err != nil || got != "" {
		t.Errorf("normalizeColors(empty) = %q, %v", got, err)
	}
	for _, raw := range []string{"123456", "#abc", "990000,zzzzzz"} {
		if _, err := normalizeColors(raw); err == nil {
			t.Errorf("normalizeColors(%q) expected an error", raw)
		}
	}
}

func TestNormalizeSeed(t *testing.T) {
	if got, err := normalizeSeed("abc123"); err != nil || got != "abc123" {
		t.Errorf("normalizeSeed(abc123) = %q, %v", got, err)
	}
	for _, raw := range []string{"abc12", "abc1234", "abc-12"} {
		if _, err := normalizeSeed(raw); err == nil {
			t.Errorf("normalizeSeed(%q) expected an error", raw)
		}
	}
}

func TestParseFlags(t *testing.T) {
	opts, err := parseFlags([]string{
		"-t", "5", "-pq", "3", "-page", "7",
		"-s", "toplist", "-o", "asc", "-tr", "3m",
		"-c", "general,anime", "-p", "sfw,sketchy",
		"-sz", "2560:1440", "-res", "1920x1080, 2560x1440",
		"-r", "16:9", "-color", "#990000,000000", "-q", "cats",
	})
	if err != nil {
		t.Fatalf("parseFlags error: %v", err)
	}
	want := options{
		interval: 5 * time.Minute, pageLimit: 3, page: 7,
		sorting: "toplist", order: "asc", topRange: "3M",
		categories: "110", query: "cats", purity: "110",
		atLeast: "2560x1440", resolutions: "1920x1080,2560x1440",
		ratios: "16x9", colors: "990000,000000",
	}
	if opts != want {
		t.Errorf("parseFlags = %+v, want %+v", opts, want)
	}

	random, err := parseFlags([]string{"-s", "random", "-seed", "abc123"})
	if err != nil {
		t.Fatalf("parseFlags(random) error: %v", err)
	}
	if random.seed != "abc123" {
		t.Errorf("seed = %q, want abc123", random.seed)
	}

	nsfw, err := parseFlags([]string{"-p", "111", "-apikey", "secret"})
	if err != nil {
		t.Fatalf("parseFlags(nsfw) error: %v", err)
	}
	if nsfw.purity != "111" {
		t.Errorf("purity with key = %q, want 111", nsfw.purity)
	}

	for _, args := range [][]string{
		{"-s", "bogus"},
		{"-o", "sideways"},
		{"-tr", "2w"},
		{"-c", "12"},
		{"-p", "nsfw"},
		{"-sz", "1080"},
		{"-res", "1920x"},
		{"-r", "16x9x1"},
		{"-color", "123456"},
		{"-seed", "abc123"},
		{"-seed", "abc"},
		{"-page", "-1"},
		{"-t", "-1"},
		{"positional"},
	} {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("parseFlags(%v) expected an error", args)
		}
	}
}

func TestParseIDs(t *testing.T) {
	got := parseIDs(" 45p3m5, 1p6d79 ,, ")
	want := []string{"45p3m5", "1p6d79"}
	if len(got) != len(want) {
		t.Fatalf("parseIDs returned %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseIDs returned %v, want %v", got, want)
		}
	}

	if ids := parseIDs(" , "); len(ids) != 0 {
		t.Fatalf("parseIDs returned %v, want no IDs", ids)
	}
}

func TestMaxPage(t *testing.T) {
	tests := []struct {
		lastPage  int
		pageLimit int
		want      int
	}{
		{0, 5, 1},
		{1, 5, 1},
		{3, 5, 3},
		{10, 5, 5},
		{10, 0, 10},
		{10, -1, 10},
	}
	for _, tt := range tests {
		if got := maxPage(tt.lastPage, tt.pageLimit); got != tt.want {
			t.Errorf("maxPage(%d, %d) = %d, want %d", tt.lastPage, tt.pageLimit, got, tt.want)
		}
	}
}

func TestSearchParamsValues(t *testing.T) {
	p := searchParams{
		Sorting:     "toplist",
		Order:       "desc",
		TopRange:    "1M",
		Categories:  "111",
		Query:       "tagname",
		Purity:      "100",
		AtLeast:     "1920x1080",
		Resolutions: "1920x1080,2560x1440",
		Ratios:      "landscape",
		Colors:      "990000",
	}
	v := p.values()

	for key, want := range map[string]string{
		"sorting":     "toplist",
		"order":       "desc",
		"topRange":    "1M",
		"categories":  "111",
		"q":           "tagname",
		"purity":      "100",
		"resolutions": "1920x1080,2560x1440",
		"ratios":      "landscape",
		"colors":      "990000",
	} {
		if got := v.Get(key); got != want {
			t.Errorf("values()[%q] = %q, want %q", key, got, want)
		}
	}
	if _, ok := v["atleast"]; ok {
		t.Error("atleast should be omitted when exact resolutions are set")
	}
	if _, ok := v["seed"]; ok {
		t.Error("seed should only be sent for random sorting")
	}

	minOnly := searchParams{AtLeast: "1920x1080"}.values()
	if got := minOnly.Get("atleast"); got != "1920x1080" {
		t.Errorf("values()[atleast] = %q, want 1920x1080", got)
	}
	if _, ok := minOnly["resolutions"]; ok {
		t.Error("resolutions should be omitted when only atleast is set")
	}

	// topRange and seed are only meaningful for their respective sortings.
	random := searchParams{Sorting: "random", Seed: "abc123"}.values()
	if got := random.Get("seed"); got != "abc123" {
		t.Errorf("random values()[seed] = %q, want abc123", got)
	}
	if _, ok := random["topRange"]; ok {
		t.Error("topRange should only be sent for toplist sorting")
	}

	empty := searchParams{}.values()
	for _, key := range []string{"q", "colors", "seed", "topRange", "resolutions"} {
		if _, ok := empty[key]; ok {
			t.Errorf("empty params should omit %q", key)
		}
	}
}

func TestWallpaperExtension(t *testing.T) {
	tests := []struct {
		rawURL string
		want   string
	}{
		{"https://w.wallhaven.cc/full/9m/wallhaven-9mjoy1.png", ".png"},
		{"https://w.wallhaven.cc/full/9m/wallhaven-9mjoy1.JPG", ".jpg"},
		{"https://example.com/image", ".jpg"},
		{"://bad", ".jpg"},
	}
	for _, tt := range tests {
		if got := wallpaperExtension(tt.rawURL); got != tt.want {
			t.Errorf("wallpaperExtension(%q) = %q, want %q", tt.rawURL, got, tt.want)
		}
	}
}

func TestFlexInt(t *testing.T) {
	for _, input := range []string{`{"n": 24}`, `{"n": "24"}`, `{"n": null}`} {
		var v struct {
			N flexInt `json:"n"`
		}
		if err := json.Unmarshal([]byte(input), &v); err != nil {
			t.Fatalf("unmarshal %s: %v", input, err)
		}
		want := flexInt(24)
		if input == `{"n": null}` {
			want = 0
		}
		if v.N != want {
			t.Errorf("unmarshal %s: got %d, want %d", input, v.N, want)
		}
	}
}
