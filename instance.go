package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// instanceLock makes sure only one hyprhaven process runs at a time. When a
// new instance starts, it terminates the previous one recorded in the PID
// file. This replaces the old Windows-only process-name scan and works on
// macOS, Linux, and Windows.
type instanceLock struct {
	path string
	pid  int
}

func acquireInstanceLock() (*instanceLock, error) {
	dir, err := dataDir()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}

	path := filepath.Join(dir, "hyprhaven.pid")
	if data, err := os.ReadFile(path); err == nil {
		if oldPID, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil && oldPID > 0 && oldPID != os.Getpid() {
			if old, err := os.FindProcess(oldPID); err == nil {
				if err := old.Kill(); err == nil {
					// Give the old process a moment to release its resources.
					time.Sleep(250 * time.Millisecond)
				}
			}
		}
	}

	if err := os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())+"\n"), 0o644); err != nil {
		return nil, fmt.Errorf("write PID file: %w", err)
	}
	return &instanceLock{path: path, pid: os.Getpid()}, nil
}

func (l *instanceLock) Release() {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return
	}
	if strings.TrimSpace(string(data)) == strconv.Itoa(l.pid) {
		_ = os.Remove(l.path)
	}
}
