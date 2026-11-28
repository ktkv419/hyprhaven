package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// usedWallpapers remembers the most recent wallpaper IDs so the daemon can
// avoid showing the same wallpaper again within a configurable window.
type usedWallpapers struct {
	path string
	max  int
	ids  []string
}

type usedWallpapersFile struct {
	IDs []string `json:"ids"`
}

// loadUsedWallpapers reads the history from path. A missing file is not an
// error. A corrupt file returns an empty tracker together with the parse error
// so the caller can warn and continue.
func loadUsedWallpapers(path string, max int) (*usedWallpapers, error) {
	u := &usedWallpapers{path: path, max: max}
	if max <= 0 || path == "" {
		return u, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return u, nil
	}
	if err != nil {
		return u, fmt.Errorf("read wallpaper history: %w", err)
	}

	var file usedWallpapersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return u, fmt.Errorf("parse wallpaper history: %w", err)
	}
	u.ids = trimIDs(file.IDs, max)
	return u, nil
}

func (u *usedWallpapers) enabled() bool {
	return u != nil && u.max > 0
}

func (u *usedWallpapers) contains(id string) bool {
	if !u.enabled() || id == "" {
		return false
	}
	for _, seen := range u.ids {
		if seen == id {
			return true
		}
	}
	return false
}

// add records a successfully applied wallpaper and persists the history. When
// the window is full the oldest entry is dropped.
func (u *usedWallpapers) add(id string) error {
	if !u.enabled() || id == "" {
		return nil
	}
	if u.contains(id) {
		// A repeated ID (the API pool was smaller than the window) should move
		// to the front of the window rather than occupy two slots.
		u.ids = removeID(u.ids, id)
	}
	u.ids = append(u.ids, id)
	if len(u.ids) > u.max {
		u.ids = append([]string(nil), u.ids[len(u.ids)-u.max:]...)
	}
	return u.save()
}

func (u *usedWallpapers) save() error {
	if !u.enabled() || u.path == "" {
		return nil
	}

	data, err := json.Marshal(usedWallpapersFile{IDs: u.ids})
	if err != nil {
		return fmt.Errorf("encode wallpaper history: %w", err)
	}

	dir := filepath.Dir(u.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create data directory: %w", err)
	}

	tmp, err := os.CreateTemp(dir, ".history-*")
	if err != nil {
		return fmt.Errorf("create wallpaper history: %w", err)
	}
	tmpName := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("write wallpaper history: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write wallpaper history: %w", err)
	}
	if err := os.Chmod(tmpName, 0o644); err != nil {
		return fmt.Errorf("set wallpaper history permissions: %w", err)
	}
	if err := os.Rename(tmpName, u.path); err != nil {
		return fmt.Errorf("store wallpaper history: %w", err)
	}
	return nil
}

func removeID(ids []string, id string) []string {
	for i, existing := range ids {
		if existing == id {
			return append(ids[:i], ids[i+1:]...)
		}
	}
	return ids
}

// trimIDs keeps at most max entries, dropping duplicates while preserving the
// most recent occurrence of each ID.
func trimIDs(ids []string, max int) []string {
	if max <= 0 || len(ids) == 0 {
		return nil
	}
	start := len(ids) - max
	if start < 0 {
		start = 0
	}

	out := make([]string, 0, len(ids)-start)
	for _, id := range ids[start:] {
		if id == "" {
			continue
		}
		out = removeID(out, id)
		out = append(out, id)
	}
	return out
}
