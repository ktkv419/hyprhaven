package main

import (
	"encoding/json"
	"testing"
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
		Sorting:    "favorites",
		Categories: "111",
		Purity:     "100",
		AtLeast:    "1920x1080",
		Ratios:     "landscape",
	}
	v := p.values()

	for key, want := range map[string]string{
		"sorting":    "favorites",
		"categories": "111",
		"purity":     "100",
		"atleast":    "1920x1080",
		"ratios":     "landscape",
	} {
		if got := v.Get(key); got != want {
			t.Errorf("values()[%q] = %q, want %q", key, got, want)
		}
	}
	if _, ok := v["q"]; ok {
		t.Error("empty query should be omitted")
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
