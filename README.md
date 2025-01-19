<h1 align="center">Hyprhaven</h1>

<p align="center">Tired of your own wallpaper collection? Hyprhaven has you covered</p>

## About

This script fetches wallpapers from [wallhaven.cc](https://wallhaven.cc/) and sets it as desktop image

## Install

### Windows

1. Either build or download latest version from release

2. Press <kbd>Ctrl</kbd> + <kbd>R</kbd> and type `shell:startup` to open folder with apps that will launch on user login

3. Copy `hyprhaven.exe` to the startup folder

4. Run executable file / Reboot

## Build

1. Run makefile

    `make`

_if building using WSL you can run `make build-windows` for the program to run in tray or `make build-windows-debug` for the program to run in CMD_

## Configuration

These are all optional parameters

The script can be customized using the following command-line flags:

| Flag   | Type     | Default Value | Description                                                                                                          |
| ------ | -------- | ------------- | -------------------------------------------------------------------------------------------------------------------- |
| `-t`   | `int`    | `15`          | The time interval (in minutes) to wait before setting the wallpaper.                                                 |
| `-pq`  | `int`    | `5`           | The page number to query the wallpaper from the API.                                                                 |
| `-s`   | `string` | `favorites`   | The sorting criterion for the wallpapers.                                                                            |
| `-c`   | `string` | `010`         | The category IDs to filter wallpapers.                                                                               |
| `-q`   | `string` | `""`          | The search query for wallpaper search.                                                                               |
| `-p`   | `string` | `100`         | The purity level of the wallpapers.                                                                                  |
| `-key` | `string` | `""`          | Apikey to make user specific/NSFW requests.                                                                          |
| `-sz`  | `string` | `1920x1080`   | The minimum resolution for the wallpaper.                                                                            |
| `-r`   | `string` | `landscape`   | The ratio for the wallpaper (accepts values like `21x9` as well as `portrait` or `landscape` for tall/wide pictures. |

All possible values can be referenced from [wallhaven API](https://wallhaven.cc/help/api)
