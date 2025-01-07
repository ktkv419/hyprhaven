<h1 align="center">Hyprhaven</h1>

<p align="center">Tired of your own wallpaper collection? Hyprhaven has you covered</p>

## About

This script fetches wallpapers from [wallhaven.cc](https://wallhaven.cc/) and sets it as desktop image

## Install

### Windows

TBD

## Build

1. Run makefile

    `make`

## Configuration

These are all optional parameters

The script can be customized using the following command-line flags:

| Flag  | Type     | Default Value | Description                                                          |
| ----- | -------- | ------------- | -------------------------------------------------------------------- |
| `-t`  | `int`    | `15`          | The time interval (in minutes) to wait before setting the wallpaper. |
| `-pq` | `int`    | `5`           | The page number to query the wallpaper from the API.                 |
| `-s`  | `string` | `favorites`   | The sorting criterion for the wallpapers.                            |
| `-c`  | `string` | `010`         | The category IDs to filter wallpapers.                               |
| `-q`  | `string` | `""`          | The search query for wallpaper search.                               |
| `-p`  | `string` | `100`         | The purity level of the wallpapers.                                  |
| `-sz` | `string` | `1920x1080`   | The minimum resolution for the wallpaper.                            |

All possible values can be referenced from [wallhaven API](https://wallhaven.cc/help/api)

## API key

API key is not required, if you don't plan on using your [wallhaven.cc](https://wallhaven.cc/) presets or NSFW search

TBD
