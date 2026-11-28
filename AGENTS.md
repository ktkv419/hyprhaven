# AGENTS.md

Guidance for AI coding agents working on Hyprhaven.

## Project

Hyprhaven is a small Go daemon that fetches a random wallpaper from [wallhaven.cc](https://wallhaven.cc/), downloads it, and applies it to the desktop. It rotates the wallpaper on an interval and runs on Windows, macOS, and Linux.

- Module: `ktkv419/hyprhaven`
- Go: 1.23+
- External dependency: `golang.org/x/sys` only (Windows registry)
- Wallpaper backends: first-party `internal/wallpaper` (no external wallpaper module)

## Where to work

The Go module root is this directory (the one containing `go.mod`). Run all Go commands here. The parent workspace may also contain a separate `wallpaper/` clone; it is legacy and is not referenced by this module.

## Layout

| Path                                    | Responsibility                                              |
| --------------------------------------- | ----------------------------------------------------------- |
| `main.go`                               | Flags, config, signal handling, run loop, `cycle`           |
| `wallhaven.go`                          | Wallhaven API client, search/ID lookup, pagination, parsing |
| `wallpaper.go`                          | Download, persistent storage, apply, cleanup, data dirs     |
| `history.go`                            | Persistent no-repeat wallpaper ID history                   |
| `instance.go`                           | Cross-platform PID-file single-instance lock                |
| `wallhaven_test.go`                     | Table-driven unit tests for pure logic                      |
| `internal/wallpaper/`                   | Platform wallpaper setters; public API is `SetFromFile`     |
| `internal/wallpaper/darwin.go`          | macOS: `NSWorkspace` via AppleScriptObjC                    |
| `internal/wallpaper/windows.go`         | Windows: `SystemParametersInfoW` + lock-screen registry     |
| `internal/wallpaper/linux.go` + helpers | Linux: GNOME/KDE/XFCE/LXDE/Cinnamon/MATE/Deepin/swaybg/feh  |
| `Makefile`                              | Build/test/format/cross-compile targets                     |
| `README.md`                             | User-facing install and flag documentation                  |
| `.github/workflows/ci.yml`              | CI on push/PR; tagged releases via GoReleaser               |
| `.goreleaser.yaml`                      | Release targets, archives, checksums, changelog             |

## Commands

```bash
# Build for the host
make build                 # or: go build -o hyprhaven .

# Test / vet / format
make test                  # go test ./...
go vet ./...
make fmt                   # go fmt ./...

# Run one cycle (safe for testing)
./hyprhaven -once

# Run the daemon
./hyprhaven -t 15

# Cross-compile
GOOS=windows GOARCH=amd64 go build -o /tmp/hyprhaven.exe .
GOOS=linux   GOARCH=amd64 go build -o /tmp/hyprhaven .
GOOS=darwin  GOARCH=arm64 go build -o /tmp/hyprhaven .
```

Makefile targets: `build`, `build-darwin` (host arch), `build-darwin-arm64`, `build-darwin-amd64`, `build-linux`, `build-windows` (tray, `-H windowsgui`), `build-windows-debug` (console), `test`, `fmt`, `clean`, `install`, `doc`. `build-windows-dev` is WSL-specific.

## Architecture

1. `main.go` parses flags, creates a `signal.NotifyContext` for `SIGINT`/`SIGTERM`, acquires the PID lock, and loops over `cycle`.
2. `main.go` loads the no-repeat history, then `cycle` asks the client for one `wallpaperInfo` (skipping recent IDs), downloads it with the shared HTTP client, calls `applyWallpaper` (which wraps `wallpaper.SetFromFile`), removes older downloaded files, and records the ID only after the apply succeeds.
3. `wallhaven.go`:
    - Search: fetch page 1 for metadata; use `-page` when set, otherwise choose a random page in `[1, min(meta.last_page, -pq)]`; pick a random item from that page that is not in the no-repeat window.
    - IDs: pick a random ID from `-id` that is not in the no-repeat window and fetch `/w/{id}`.
    - All search API parameters are exposed as flags and validated by `normalize*` helpers (`q`, `categories`, `purity`, `sorting`, `order`, `topRange`, `atleast`, `resolutions`, `ratios`, `colors`, `page`, `seed`, `apikey`).
4. `wallpaper.go` owns the HTTP download (2-minute timeout, 100 MiB cap), atomic storage, persistent data directory, and cleanup.
5. `history.go` keeps the last N applied wallpaper IDs in `<dataDir>/history.json`, loaded at startup and saved atomically after each successful apply.
6. `instance.go` writes `<dataDir>/hyprhaven.pid`; on startup it kills the previous PID if it is still alive.

## Rules

- Prefer the standard library. Do not add dependencies unless necessary; the only external module is `golang.org/x/sys`.
- Never reintroduce `github.com/ktkv419/wallpaper`, a `replace` directive, or a separate wallpaper module. The source of truth is `internal/wallpaper`.
- Keep `internal/wallpaper` minimal: it intentionally exposes only `SetFromFile`. Add `Get`/`SetMode`/`SetFromURL` only if a feature actually needs them.
- Return errors from fetch/download/apply paths and wrap them with `%w`. Do not call `log.Fatalf` there; the loop is designed to log and retry.
- Use `context.Context` for all HTTP requests and the shared `*http.Client` from `newHTTPClient`.
- Use `math/rand/v2` (`rand.IntN`), not the legacy global `math/rand`.
- Use `any` instead of `interface{}` in new code.
- Keep the run loop interruptible with `select` on `ctx.Done()` and `time.After`; do not use `time.Sleep`.
- Log with the standard `log` package (the `hyprhaven:` prefix is set in `main`). Never log `-apikey`.
- Add table-driven tests for pure logic in `wallhaven_test.go`.

## Platform notes

### macOS

- `internal/wallpaper/darwin.go` uses `NSWorkspace` through `osascript`/AppleScriptObjC. This is deliberate: the old `System Events`/`Finder` `set picture` AppleScript is broken on macOS 15 (it rewrites the wallpaper store to a relative path and falls back to the system wallpaper). Do not revert it.
- The `NSWorkspace` path does not require Automation/TCC permission.
- Verify changes with `./hyprhaven -once` and confirm the desktop actually changed. `NSWorkspace`'s current URL should point at the downloaded file; the old Finder/System Events query may return `missing value`.

### Windows

- `internal/wallpaper/windows.go` sets the desktop via `SystemParametersInfoW` and mirrors it to the lock screen via registry keys.
- `setLockscreen` may relaunch the process elevated and exit. Do not assume `SetFromFile` always returns in the same process.
- `make build-windows` uses `-H windowsgui` for a tray/no-console build; use `make build-windows-debug` for a console build.

### Linux

- `internal/wallpaper/linux.go` dispatches on `XDG_CURRENT_DESKTOP` and shells out to `gsettings`, `qdbus`, `dconf`, `xfconf-query`, `pcmanfm`, `swaybg`, or `feh`.
- The fallback path is `swaybg`, then `feh`.

## CLI flags

Defaults and help text live in `parseFlags` (`main.go`) and the README. Keep them in sync.

| Flag      | Default     | Meaning                                   |
| --------- | ----------- | ----------------------------------------- |
| `-t`      | `15`        | Minutes between changes                   |
| `-pq`     | `5`         | Maximum random search page                |
| `-page`   | `0`         | Exact search page, `0` = random           |
| `-norepeat` | `5`       | Recent wallpaper IDs to skip, `0` = off   |
| `-s`      | `favorites` | Sorting                                   |
| `-o`      | `desc`      | Sorting order                             |
| `-tr`     | `1M`        | Toplist range (with `-s toplist`)         |
| `-c`      | `111`       | Categories (mask or names)                |
| `-q`      | `""`        | Search query                              |
| `-p`      | `100`       | Purity (mask or names)                    |
| `-apikey` | `""`        | Wallhaven API key                         |
| `-sz`     | `1920x1080` | Minimum resolution                        |
| `-res`    | `""`        | Exact resolutions, comma-separated        |
| `-r`      | `landscape` | Ratio(s)                                  |
| `-color`  | `""`        | Palette colors, comma-separated hex       |
| `-seed`   | `""`        | Random seed (requires `-s random`)        |
| `-id`     | `""`        | Comma-separated wallpaper IDs             |
| `-once`   | `false`     | Set one wallpaper and exit                |

## Data and state

- Wallpapers: `<dataDir>/wallpapers/wallhaven-<id>-<unixnano>.<ext>`
- History: `<dataDir>/history.json` (`{"ids":["..."]}`, newest last)
- PID file: `<dataDir>/hyprhaven.pid`
- `<dataDir>`:
    - macOS: `~/Library/Application Support/hyprhaven`
    - Windows: `%LocalAppData%\hyprhaven`
    - Linux: `$XDG_DATA_HOME/hyprhaven` or `~/.local/share/hyprhaven`
- Override with `HYPRHAVEN_DATA_DIR`.
- After a successful apply, only the current wallpaper file is kept; older `wallhaven-*` and `.download-*` files are removed.

## Pitfalls

- Use `meta.last_page` for pagination. Do not compute pages as `total/per_page + 1`.
- Check `len(data) > 0` before indexing; the last page can be short and empty searches exist.
- `-pq` is an upper bound on the random page, not the page to fetch. `-page` overrides it with an exact page.
- Without `-apikey`, `sanitizePurity` clears the NSFW bit (last digit) of the purity mask. The API key query parameter is `apikey`, not `wh_api`.
- `searchParams.values` only sends `topRange` for `-s toplist` and `seed` for `-s random`. Random sorting returns `meta.seed` on page 1, which `pickRandom` reuses on the chosen page so the shuffle is stable.
- Validate search filters through the `normalize*` helpers before requesting. The API silently ignores bad `purity`/`sorting`/`order` values, returns a 500 for malformed `categories`, and only accepts the documented color palette.
- The API ignores `resolutions` when `atleast` is also sent, so `searchParams.values` sends one or the other and `-res` takes precedence over `-sz`.
- Record a wallpaper ID in the history only after `SetFromFile` succeeds; a failed download or apply must not consume a window slot.
- History is a rolling window: `trimIDs` keeps the newest N unique IDs, and `-norepeat 0` disables both reading and writing. A corrupt `history.json` returns an error with an empty tracker so the daemon warns and continues.
- `pickRandom` checks up to `maxPageAttempts` pages for an unused ID. If the whole pool is inside the window (e.g. an `-id` list smaller than `-norepeat`), it returns `errAllUsed` rather than repeating.
- Downloads use unique filenames so a failed rename cannot delete the currently displayed wallpaper.
- Run `cleanupWallpapers` only after `SetFromFile` succeeds.
- The PID lock uses `os.Process.Kill`; do not replace it with a process-name scan.
- The sibling `../wallpaper/` directory is a legacy separate clone; do not import from it.

## Before finishing

- `gofmt -w` changed files (or `make fmt`)
- `go vet ./...`
- `go test ./...`
- `go build ./...`
- If you touched `internal/wallpaper`, cross-compile Windows, Linux, and macOS.
- If you touched the macOS backend, run `./hyprhaven -once` and confirm the wallpaper actually changes.
- Update `README.md` when flags, defaults, data paths, or platform behavior change.

## Git conventions

History uses scoped Conventional Commit-style messages: `feat(scope): ...`, `fix(scope): ...`, `chore(scope): ...`, `update(scope): ...`. Keep messages imperative and scoped.

CI runs on every push to `main`/`dev` and on pull requests: gofmt, `go vet` (host + Windows + macOS), `go test -race`, and a cross-build of all release targets. Pushing a `v*` tag runs the same checks and then publishes archives and `checksums.txt` to GitHub Releases through GoReleaser. Do not tag to test CI; use `workflow_dispatch` or a branch push.

## License

`internal/wallpaper/LICENSE` is the public-domain notice for the merged wallpaper backend. Keep it when modifying that package.
