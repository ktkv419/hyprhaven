package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"ktkv419/hyprhaven/internal/wallpaper"
)

const maxWallpaperBytes = 100 << 20 // 100 MiB

func newHTTPClient() *http.Client {
	return &http.Client{Timeout: 2 * time.Minute}
}

// downloadWallpaper downloads the image to the persistent data directory and
// returns its local path.
func downloadWallpaper(ctx context.Context, client *http.Client, info wallpaperInfo) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, info.Path, nil)
	if err != nil {
		return "", fmt.Errorf("create download request: %w", err)
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download wallpaper: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download wallpaper: server returned %s", resp.Status)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, "image/") {
		return "", fmt.Errorf("download wallpaper: unexpected content type %q", ct)
	}

	dir, err := wallpaperDir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("create wallpaper directory: %w", err)
	}

	id := safeName(info.ID)
	if id == "" {
		id = "unknown"
	}
	// A unique name keeps the currently displayed file intact if a later
	// download or rename fails, and avoids Windows' rename-over-existing
	// restriction.
	name := fmt.Sprintf("wallhaven-%s-%d%s", id, time.Now().UnixNano(), wallpaperExtension(info.Path))
	dest := filepath.Join(dir, name)

	tmp, err := os.CreateTemp(dir, ".download-*")
	if err != nil {
		return "", fmt.Errorf("create temporary wallpaper: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	n, err := io.Copy(tmp, io.LimitReader(resp.Body, maxWallpaperBytes+1))
	if err != nil {
		return "", fmt.Errorf("save wallpaper: %w", err)
	}
	if n > maxWallpaperBytes {
		return "", fmt.Errorf("wallpaper exceeds %d MiB limit", maxWallpaperBytes>>20)
	}
	if err := tmp.Close(); err != nil {
		return "", fmt.Errorf("save wallpaper: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return "", fmt.Errorf("set wallpaper permissions: %w", err)
	}
	if err := os.Rename(tmpName, dest); err != nil {
		return "", fmt.Errorf("store wallpaper: %w", err)
	}
	return dest, nil
}

func applyWallpaper(file string) error {
	if err := wallpaper.SetFromFile(file); err != nil {
		return fmt.Errorf("set wallpaper: %w", err)
	}
	return nil
}

// cleanupWallpapers removes previously downloaded wallpapers, keeping only the
// file that is currently set as the desktop background.
func cleanupWallpapers(dir, keep string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	keepName := filepath.Base(keep)
	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == keepName {
			continue
		}
		if strings.HasPrefix(entry.Name(), "wallhaven-") || strings.HasPrefix(entry.Name(), ".download-") {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

func wallpaperExtension(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ".jpg"
	}
	switch ext := strings.ToLower(path.Ext(u.Path)); ext {
	case ".jpg", ".jpeg", ".png", ".webp", ".gif":
		return ext
	default:
		return ".jpg"
	}
}

func safeName(value string) string {
	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-',
			r == '_':
			b.WriteRune(r)
		}
	}
	return b.String()
}

func wallpaperDir() (string, error) {
	base, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "wallpapers"), nil
}

// dataDir returns a persistent per-user directory. Wallpapers must survive
// reboots and cache purges, so a cache directory is not suitable here.
func dataDir() (string, error) {
	if dir := os.Getenv("HYPRHAVEN_DATA_DIR"); dir != "" {
		return dir, nil
	}

	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", "hyprhaven"), nil
	case "windows":
		dir, err := os.UserCacheDir() // %LocalAppData%
		if err != nil {
			return "", err
		}
		return filepath.Join(dir, "hyprhaven"), nil
	default:
		if dir := os.Getenv("XDG_DATA_HOME"); dir != "" {
			return filepath.Join(dir, "hyprhaven"), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "share", "hyprhaven"), nil
	}
}
