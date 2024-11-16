<h1 align="center">Hyprhaven</h1>

<p align="center">Tired of your own wallpaper collection? Hyprhaven has you covered</p>

## About

This script fetches wallpapers from [wallhaven.cc](https://wallhaven.cc/) and passes it to [hyprpaper](https://github.com/hyprwm/hyprpaper). Search is customizable through variables inside the file

## Install

1. Move script file to a location, that is added to your path, for example `~/.local/bin/`

2. Install [jq](https://github.com/jqlang/jq)

3. Add hyprhaven to [hyprland](https://github.com/hyprwm/Hyprland) config

_~/.config/hypr/hyprland.conf_

```
exec-once = ~/.local/bin/hyprhaven
```

3. If you want to hot-reload your wallpaper, create keyboard shortcut

_~/.config/hypr/hyprland.conf_

```
bind = $mainMod Control_L, w, exec, ~/.local/bin/hyprhaven
```

_Hot-reloading is done through new hyprhaven instance killing older instanced_

## Configuration

Currently only in-file configuration is set up. In order to change these settings you have to manually change `hyprhaven` file.

## API key

API key is not required, if you don't plan on using your [wallhaven.cc](https://wallhaven.cc/) presets or NSFW search

`WALLHAVEN_API` is fetched from your environment, therefore has to be set before running the script

> [!WARNING]  
> Setting variable through `.bashrc` or `.zshrc` is not viable due to hyprland running script in a non-interactive shell. So, in order to succesfully set variable in user environment `~/.profile` file has to be used
