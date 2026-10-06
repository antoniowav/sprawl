# Sprawl

A small, keyboard-first, pixel-art city builder for Linux, made to feel at
home on Omarchy (Arch + Hyprland). Roads, zones, power, water, services and
a budget — no traffic, disasters or individual citizens. It follows your
Omarchy theme, live.

![screenshot](docs/screenshot.png)

## Install

Needs Go 1.27+ and the usual X11/GL and ALSA libraries (present on any
Omarchy desktop):

```sh
sudo pacman -S --needed go mesa libxrandr libxcursor libxinerama libxi
git clone https://github.com/antoniowav/sprawl && cd sprawl
dist/install.sh            # builds, then adds Sprawl to your app launcher
```

`dist/install.sh` puts the binary in `~/.local/bin`, the icon in
`~/.local/share/icons` and a launcher entry in
`~/.local/share/applications`. `dist/install.sh --uninstall` removes them
(saves and config stay). To just run it from the source tree:
`go build -o sprawl ./cmd/sprawl && ./sprawl`.

Distributions: `make && make install PREFIX=/usr DESTDIR=...` installs the
binary, icons (32–512 px), desktop entry, AppStream metadata, man page and
third-party licence notices. An AUR package stub lives in `dist/PKGBUILD`, with a desktop entry in
`dist/sprawl.desktop` (window class `sprawl`, for Hyprland rules).

## Playing

Lay roads, zone next to them, connect power, and the city grows. Lots only
develop when they touch a road. Medium and high density need water, and high
density needs good land value: services, clean air, or a waterfront. Power
travels along roads, power lines, zones and buildings, so a block touching a
road that leads to a plant has power; use lines to reach the rest.

### Keys

| Key | Action |
|---|---|
| `h j k l` / arrows | move cursor |
| `H J K L` / Shift+arrows | move 8 tiles |
| `c` | centre camera on cursor |
| `+` `-` / scroll | zoom (1×–4×, always crisp; the wheel zooms toward the mouse) |
| right or middle drag | pan |
| `r` `p` `w` `d` | road, power line, water pipe, bulldoze |
| `z` then `r` `c` `i` | residential, commercial, industrial zone |
| `b` | buildings menu |
| `Enter` | apply the tool at the cursor |
| `v` … `Enter` | visual mode: rectangle (zones, bulldoze) or L-path (roads, lines, pipes); `o` swaps the corner |
| `Shift+Enter` | paint mode: every move applies the tool |
| left click / drag | apply the tool / select like visual mode |
| `o` | cycle overlays: power, water, traffic, police, fire, school, health, land value, pollution |
| `u` | underground view (pipes); in it, bulldoze removes pipes |
| `Space`, `1` `2` `3` | pause, speed |
| `Tab` | side panel |
| `e` / `F2` | event log |
| `Ctrl+S`, `Ctrl+Shift+S` | save, save as |
| `Ctrl+O`, `Ctrl+N`, `Ctrl+Q` | open a save, new city, quit (asks if unsaved) |
| `Ctrl+Z`, `Ctrl+Shift+Z` / `Ctrl+Y` | undo, redo (money included) |
| `Ctrl+B`, `Ctrl+G` | budget, statistics charts |
| `F1` | getting-started guide |
| `Esc` with nothing selected | menu: save, open, settings, quit to title |
| click / `i` | inspect a tile: what it holds and why it isn't growing |
| `m`, `n` | minimap, day/night mode |
| `F5`, `F12` | achievements, photo (PNG in ~/Pictures/Sprawl) |
| `:` | command palette (Tab completes, ↑↓ history) |
| `?` | key overlay (generated from your keymap) |
| `Esc` | back out: visual → paint → tool |

### Commands

| Command | |
|---|---|
| `:budget` | income and upkeep breakdown |
| `:tax 12`, `:tax r 7` | set all taxes, or one zone's (0–20, 9 is neutral) |
| `:loan`, `:repay` | borrow $10,000 (repaid $450/month for 24 months), or pay off early |
| `:w [name]`, `:e [name]` | save, load (`:e` alone lists saves) |
| `:new [seed]`, `:name <city>` | new map, rename |
| `:theme reload` | re-read the Omarchy theme |
| `:q`, `:wq` | quit, save and quit |

Twelve months in debt and the city goes bankrupt.

### Maps, traffic and scenarios

New cities pick a map: river valley, coast, lakes, islands or highlands,
from 96×96 up to 256×256. Roads cross water as bridges ($60 a tile).
Homes send their commuters along the roads to the nearest jobs: busy roads
fill with cars, jammed roads lower land value, and homes with no road to
any job stop growing. Bus stops near homes (with a bus depot somewhere)
take 30% of commuters off the road. Two scenarios, Boomtown and Island,
give you a goal and a deadline, and something happens partway through.

### Buildings and goals

Power plant (smoky, 200 power), wind turbine (clean, 25), solar farm
(60, from 500 people), nuclear plant (1,000, from 5,000), water pump (by
water, 150), water tower (anywhere, 40), police, fire station, school,
hospital (from 250), university (from 2,500), park, bus stop and depot,
city hall (from 1,000; +5 % taxes), stadium (from 2,500) and a monument
(from 10,000). Population milestones pay a grant and promote the
settlement from hamlet to metropolis. The next goal is always in the top
bar, and achievements (F5) are kept across all your cities.

## Files

- Config: `~/.config/sprawl/config.toml`, written with comments on first
  run. Every key is rebindable under `[keys]`, e.g. `road = ["R"]`.
- Saves: `~/.local/share/sprawl/*.city` (gzip'd JSON), autosave every 6
  game months.
- Theme: read from Cuore (`~/.local/state/cuore/current/theme`) or Omarchy
  (`~/.config/omarchy/current/theme`, `~/.local/state/omarchy/current/theme`):
  `colors.toml`, falling back to `alacritty.toml`. `SPRAWL_THEME_DIR`
  overrides. Without either it uses a built-in palette.

## Flags

`-seed N` map seed · `-config path` · `-version` ·
`-screenshot out.png` (render a 1280×800 frame and exit) with
`-actions "r,v,L,apply,:tax 10"` and `-ticks N` to script it.

## Dependencies

Ebitengine (graphics, input), oto (sound; Ebitengine's audio library) and
BurntSushi/toml (config and theme files). Everything else, including the
font, sprites and sound effects, is generated by Sprawl's own code. Licence
texts of everything linked into the binary are in
`dist/THIRD_PARTY_LICENSES.txt` (`make licenses` regenerates it).

## Development

`make check` runs tests, gofmt and vet. `SPRAWL_CPUPROFILE=cpu.prof` writes
a 15-second CPU profile. Design and formulas are in
`SPEC.md`; milestones in `PLAN.md`. `BALANCE=1 go test ./sim -run Balance -v`
prints a growth curve.

## Licence

MIT, see `LICENSE`. Third-party notices: `dist/THIRD_PARTY_LICENSES.txt`.
