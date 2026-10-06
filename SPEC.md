# Sprawl — Design Spec

Working name **Sprawl**. The name lives in one constant (`internal/meta.Name`)
and in derived paths (`~/.config/sprawl`, `~/.local/share/sprawl`); renaming
means changing that constant plus the module path.

A small, keyboard-first, pixel-art city builder for Linux, tuned for Omarchy
(Arch + Hyprland). Scope is SimCity Classic minus traffic, disasters and
agents: every value is an aggregate per tile.

---

## 1. Goals and non-goals

**Goals**
- Crisp pixel art at integer scale, no blur, no external assets.
- Keyboard first (vim grammar), mouse optional.
- Near-zero CPU when idle: no redraw unless something changed.
- Follows the Omarchy theme, including live theme switches.
- Single static-ish binary; config in TOML; saves in XDG data dir.

**Non-goals (v1)**: traffic simulation, disasters, citizen agents, terrain
editing, multi-tile zone buildings, sound, mods, Windows/macOS builds.

---

## 2. Stack and dependencies

| Dependency | Why |
|---|---|
| Go 1.27 (stdlib) | Language. |
| `github.com/hajimehoshi/ebiten/v2` | Window, GPU drawing, input. The only engine allowed. |
| `github.com/BurntSushi/toml` | Parse `config.toml`, `colors.toml`, `alacritty.toml`. A hand-written TOML parser is not worth the bugs. |

Nothing else. Specifically **not** used:
- No font library: the pixel font is a 5×7 bitmap defined in Go code (our own glyphs, no licence questions).
- No `fsnotify`: theme watching polls two `stat` calls every 2 s (cost is negligible, avoids a dependency and inotify edge cases with Omarchy's swap-the-directory theme switch).
- No image files: all sprites are generated from the palette at startup.

---

## 3. Package layout

```
cmd/sprawl/        main: flags, config, wiring, ebiten.RunGame
internal/meta/     Name, Version, XDG paths
sim/               pure simulation (no ebiten import), fully unit tested
  grid.go          Tile, Map, terrain generation
  build.go         placement/bulldoze rules + costs
  network.go       power and water flood fill + capacity allocation
  demand.go        RCI demand
  growth.go        zone growth/decline
  landvalue.go     coverage, pollution, land value
  budget.go        taxes, upkeep, loans, bankruptcy
  clock.go         Tick(), calendar, speeds
  events.go        event log entries
  save.go          versioned save format
render/            camera, sprite atlas, tile drawing, HUD, overlays, effects
  sprites/         procedural sprite generators
  font/            5×7 bitmap font
input/             actions, modes, keymap (configurable), mouse
theme/             palette type, Omarchy loader, fallback, watcher
config/            config.toml schema and defaults
```

Rule: `sim` imports only the stdlib. `render` and `input` depend on `sim`
read-only views plus an `sim.Command` interface for mutations, so all
state changes go through one place (and are testable).

---

## 4. Data model (`sim`)

```go
type Terrain uint8 // Land, Water, Trees

type Kind uint8    // Empty, Road, ZoneR, ZoneC, ZoneI,
                   // PowerPlant, WaterPump, Police, Fire, School

type Tile struct {
    Terrain   Terrain
    Kind      Kind
    Level     uint8   // 0 = empty lot, 1 low, 2 medium, 3 high (zones only)
    Variant   uint8   // random sprite variant, rolled when Level changes
    Line      bool    // overhead power line (may share a tile with Road)
    Pipe      bool    // underground pipe (may share a tile with anything)
    Anchor    int32   // for multi-tile buildings: index of top-left tile, else -1
    // derived each sim day, not saved:
    Powered, Watered bool
    LandValue        float32 // 0..1
    Pollution        float32 // 0..1
    Cover            [3]float32 // police, fire, school, 0..1
    UnpoweredDays    uint16
}

type City struct {
    W, H     int
    Tiles    []Tile
    Seed     int64
    Day      int     // days since founding; month = 30 days, year = 12 months
    Funds    float64
    Tax      [3]int  // R, C, I, percent 0..20
    Loan     *Loan
    Debt     DebtState
    Demand   [3]float64 // smoothed, -1..1
    Stats    Stats      // population, jobs, power/water use & capacity
    Log      []Event
    rng      *rand.Rand // seeded, saved as seed+draw count for determinism
}
```

- Map default **128×128**, seeded generation: value noise gives a river or
  lake (~10 % water) and tree clusters (~15 %). Building on trees clears them
  for free.
- Zones are 1×1 lots. Service buildings are 2×2, power plant 3×3,
  water pump 2×2 (must touch water terrain on at least one edge tile).
- Power lines may cross roads (same tile). Pipes are underground and can go
  under anything except water.

---

## 5. Time

- `Tick()` advances the sim by one tick. **4 ticks = 1 day.**
- Default 4 ticks/s (speed 1). Speed 2 = 8/s, speed 3 = 16/s. Space pauses.
- Per tick: process ¼ of the map's zone tiles for growth (striped by
  index), so work is spread evenly and frame time stays flat.
- Per day (every 4th tick): networks, demand, coverage/land value (land
  value is recomputed every 5 days, it changes slowly), events.
- Per month (every 30 days): budget.
- Sim runs on a fixed-step accumulator in `Update()`, independent of frame
  rate. When the window is unfocused it pauses (configurable:
  `pause_unfocused = true`).

---

## 6. Simulation rules

All constants live in `sim/tuning.go` so balance passes touch one file.

### 6.1 Aggregates
Computed each day from tiles:

```
Residents  P = Σ resCap[level]      over R tiles     resCap = {0, 8, 30, 90}
CommJobs   C = Σ comCap[level]      over C tiles     comCap = {0, 4, 15, 45}
IndJobs    I = Σ indCap[level]      over I tiles     indCap = {0, 6, 20, 50}
Workforce  L = 0.5 · P
Jobs       J = C + I
```

A tile counts toward these only if it is developed (level ≥ 1). Unpowered
tiles count at half capacity.

### 6.2 Demand
Raw demand per zone, each clamped to [-1, 1]:

```
rawR = (J + 20 − L) / max(J + 20, 1)            // jobs open → people move in
rawC = (0.2·P − C)  / max(0.2·P, 10)            // shops follow residents
rawI = (0.8·L + 15 − I) / max(0.8·L + 15, 1)    // factories follow workers
```

The `+20` and `+15` are bootstrap terms so an empty city has positive R and I
demand.

Tax pressure, with 9 % as neutral:

```
taxMod(t) = 0.04 · (t − 9)        // t = 20 → −0.44, t = 0 → +0.36
target_z  = clamp(raw_z − taxMod(tax_z), −1, 1)
```

Smoothing (once a day), so meters move visibly but don't jitter:

```
Demand_z ← Demand_z + 0.15 · (target_z − Demand_z)
```

### 6.3 Growth and decline
A zone tile is evaluated once per day (striped across ticks). Preconditions:

- **Road access**: a road tile 4-adjacent (see Open decisions §12).
- Level 0 → 1 needs **power**.
- Level 2 and 3 need **power and water**.
- Max level allowed by land value: `LV < 0.35 → 1`, `LV < 0.60 → 2`, else 3.

Score:

```
S = Demand_z + 0.5 · (LV − 0.5) + 0.2 · (avgCover − 0.5)
avgCover = (police + fire + school) / 3
```

Rolls (seeded RNG):

```
if preconditions met and level < maxLevel and S > 0:
    grow one level with probability  pGrow  = min(0.10, 0.15 · S)
    (for an empty lot, S is replaced by Demand_z alone: demand fills lots,
     land value and services decide upgrades)
if S < −0.2 or level > maxLevel:
    drop one level with probability  pDecay = min(0.06, 0.05 · |S|)
if unpowered ≥ 30 days or lost road access:
    drop one level with probability  0.05
```

On any level change the tile rerolls `Variant` (0..3) so blocks don't look
copy-pasted, and the renderer plays the grow animation.

### 6.4 Power
- Plant: capacity **200** units. Consumption per tile:
  R `1·level`, C `2·level`, I `3·level`, each service building `4`, pump `3`.
- **Conductors**: roads, power line tiles, developed or empty zone tiles,
  and building tiles. A block touching a road that leads to a plant has
  power; lines reach places roads don't. (Changed 2026-10-06 at the user's
  request: roads used to block power, which left road-ringed blocks dark.)
- Daily: BFS from every plant through 4-connected conductors. Each connected
  component gets `supply = Σ plant capacity`. Consumers are served in BFS
  order (nearest first) until supply runs out; the rest are `Powered=false`
  (brownout at the network edge). One event is logged per brownout start.

### 6.5 Water
- Pump: capacity **150** units, must touch water terrain, needs power.
  Consumption: R `2·level`, C `1·level`, I `2·level`, services `1`.
- **Conductors**: pipe tiles only. BFS from pumps through 4-connected pipes.
- A tile is watered if it, or a 4-adjacent tile, holds a pipe in a
  component with spare capacity (served nearest first, like power).

### 6.6 Services, pollution, land value
Coverage from a service building at distance `d` (Euclidean, tile centres):

```
cover(d) = clamp(1.25 · (1 − d / r), 0, 1)   r: police 10, fire 10, school 12
Cover_s(tile) = min(1, Σ cover over buildings of type s)
```

A service that is unpowered gives half coverage.

Pollution sources: industry `0.08·level` radius 6, power plant `0.5` radius 8.

```
Pollution(tile) = min(1, Σ src · max(0, 1 − d / r))
```

Land value:

```
LV = clamp( 0.30
          + 0.15 · nearWater            // 1 if water terrain within 3 tiles
          + 0.10 · Cover_police
          + 0.10 · Cover_fire
          + 0.15 · Cover_school
          + 0.10 · neighbourhood        // avg level/3 of R and C zones within 2 tiles
          − 0.35 · Pollution , 0, 1)
```

Industry ignores pollution in its own score (it only feels the LV cap).

### 6.7 Budget
Money is `float64` dollars, shown rounded.

**Construction costs**

| Item | Cost |
|---|---|
| Road | $10 / tile |
| Power line | $5 / tile |
| Pipe | $5 / tile |
| Zone (any) | $5 / tile |
| Bulldoze | $1 / tile (+$20 for a building tile) |
| Power plant 3×3 | $3,000 |
| Water pump 2×2 | $1,500 |
| Police / Fire 2×2 | $500 |
| School 2×2 | $800 |

**Monthly** (every 30 days):

```
income  = 0.1 · (P · tR + C · tC + I · tI)        // t in percent
upkeep  = 0.5·roads + 0.25·lines + 0.25·pipes
        + 100·plants + 50·pumps + 60·police + 60·fire + 80·schools
        + loanPayment
Funds  += income − upkeep
```

Example: 1,000 residents, 300 jobs, all taxes 9 % → income ≈ $1,170/month.

**Loan**: `:loan` borrows $10,000 (one at a time), repaid at $450/month for
24 months ($10,800 total). `:repay` pays off the remainder early.

**Debt states**
- `Funds ≥ 0`: normal.
- `Funds < 0`: **in debt**. Building is blocked (bulldoze still allowed),
  status line turns red, event logged.
- In debt for **12 consecutive months**: **bankrupt**. A modal shows the
  final stats; options are `:load` or `:new`. (See Open decisions §12.)

Starting funds: **$20,000**. Default taxes 9 / 9 / 9, range 0..20.

### 6.8 Events
Ring buffer of 200 entries, `[day N] message`, with level `info|warn|err`.
Emitted on: demand above 0.7 for 10 days (once per episode), brownout start /
end, water shortage, entering debt, bankruptcy warning (month 9), bankruptcy,
population milestones (100, 500, 1k, 5k, 10k, 50k), a yearly budget line
plus a warning when the monthly net turns negative, save/load, theme change.

---

## 7. Rendering (`render`)

### 7.1 Pixels
- Base tile 16×16. World zoom is an integer scale **1×, 2×, 3×, 4×**
  (default 3×), nearest-neighbour filtering everywhere.
- The camera position is stored in world pixels and always snapped to whole
  screen pixels, so nothing shimmers while scrolling. Camera follows the
  cursor with a short ease (snapped each frame), and stops easing when
  within 1 px.
- HUD uses its own integer scale (`ui_scale`, default 2), independent of
  world zoom.

### 7.2 Procedural sprites
Generated into one atlas at startup and regenerated on theme change
(<20 ms target):

- Terrain: 4 grass variants (noise speckle), trees (3 variants), water with
  16 shoreline masks and 4 shimmer frames.
- Road, power line, pipe: 16 autotile masks each (4-bit neighbour mask).
- Zones: empty lot per zone type (coloured dotted border, so zoning reads
  at a glance). Buildings per zone × level × 4 variants:
  - **R**: low = small house with pitched roof and yard, medium = row
    houses, high = apartment block with window grid.
  - **C**: low = shop with awning, medium = 2-storey office,
    high = tower with glass bands.
  - **I**: low = shed, medium = warehouse with sawtooth roof,
    high = factory with chimneys (smoke emitters).
  - Variants change roof colour shade, window layout, footprint inset,
    door position, chimney count. Seeded per variant, so stable.
- Buildings: power plant (cooling tower), pump, police, fire, school —
  each with a readable symbol.
- Icons: lightning bolt (no power), droplet (no water), both blink at 1 Hz.

### 7.3 Living touches (all cheap)
- **Water shimmer**: 4-frame cycle at 4 fps on water tiles.
- **Smoke**: pooled particles (max 64 on screen) from high industry and
  plants; 3-pixel puffs drifting up and fading.
- **Grow animation**: on level change, the building rises from its base over
  300 ms in whole-pixel steps (clipped sprite).
- **Day/night**: a cycle of 3 real minutes at speed 1 (scaled by speed,
  frozen when paused). At night a dark tint covers the world and a separate
  window-light layer (precomputed per sprite) draws lit windows; ~70 % of
  windows lit, chosen per building by hash so they don't flicker.

### 7.4 Frame budget and idle CPU
- `ebiten.SetScreenClearedEveryFrame(false)`; `Draw` returns immediately when
  nothing is dirty (the last frame stays on screen).
- Dirty sources: input, camera motion, sim tick that changed a visible tile,
  animation step, theme change, HUD value change.
- Animations step at most 8 times per second, and only when an animated
  thing is on screen. `animations = false` in config turns them off.
- World tiles are drawn into a cached offscreen chunk layer (32×32 tile
  chunks); a chunk is redrawn only when one of its tiles changes. A frame
  blits visible chunks + dynamic layers (cursor, icons, smoke, HUD).
- Unfocused: TPS drops to 10 and the sim pauses (default).
- Target: < 2 % of one core idle, < 15 % while scrolling, on an Intel iGPU.
  (Exact Ebitengine API for minimal redraw is checked against the current
  release during M1.)

### 7.5 HUD (TUI dashboard in pixels)
Drawn with the 5×7 font, 1-px borders in theme colours, box-drawing style
corners.

```
┌ SPRAWL ─ Newburg ───────── Mar 14, Y3 ─ ▶▶ ─────── $18,240 ─ pop 3,410 ┐
│                                                         ┌ demand ─────┐│
│                                                         │ R ████▌   + ││
│               (world view)                              │ C ██      + ││
│                                                         │ I ▌       − ││
│                                                         ├ power ──────┤│
│                                                         │ 340/400 ███ ││
│                                                         ├ water ──────┤│
│                                                         │ 120/150 ██▌ ││
│                                                         ├ budget ─────┤│
│                                                         │ tax 9/9/9   ││
│                                                         │ +$1,170/mo  ││
│                                                         └─────────────┘│
├ log ───────────────────────────────────────────────────────────────────┤
│ [day 142] residential demand high                                      │
├────────────────────────────────────────────────────────────────────────┤
│ NORMAL  road  (64,41)  R lvl2  pwr ✓ wat ✓  LV .62        speed 2  ▮▮ │
└────────────────────────────────────────────────────────────────────────┘
```

- Top bar: name, city name, date, speed glyph, funds (red when negative),
  population.
- Right panel (toggle `Tab`): demand bars (signed, centred), power and
  water use/capacity, budget summary.
- Log pane (toggle `e`): last N events, coloured by level, like `journalctl`.
- Status line: vim-style mode tag, current tool, cursor coords, tile
  summary, transient message (e.g. "not enough funds").
- `?` overlay: keybinding table generated from the live keymap, so it never
  goes stale.
- `:` command palette: one-line prompt in the status line, history with
  up/down, tab completion of command names.

### 7.6 Overlays
`o` cycles: none → power → water → police → fire → school → land value →
pollution → none. Overlays tint tiles with a two-colour ramp from the
palette and show a legend in the panel. `u` toggles underground view (pipes
drawn, surface dimmed); selecting the pipe tool turns it on automatically.

---

## 8. Theme (`theme`)

### 8.1 Where the theme lives
Search order, first hit wins:
1. `$SPRAWL_THEME_DIR`
2. `~/.config/omarchy/current/theme` (classic Omarchy)
3. `~/.local/state/omarchy/current/theme` (newer layout; this is where it is
   on this machine, with `../theme.name` next to it)
4. Built-in palette "Sprawl Dusk".

### 8.2 Files parsed
1. `colors.toml` (preferred): keys `mode, accent, selection, muted,
   background, dark_background, darker_background, lighter_background,
   foreground, dark_foreground, red, yellow, orange, green, cyan, blue,
   magenta, brown` and `bright_*`.
2. Fallback `alacritty.toml`: `[colors.primary]`, `[colors.normal]`,
   `[colors.bright]` mapped to the same names.
Missing keys are filled from the built-in palette.

### 8.3 Mapping to game palette

| Game role | Source |
|---|---|
| HUD background / panel / border | `background` / `dark_background` / `muted` |
| HUD text / dim text | `foreground` / `dark_foreground` |
| Cursor, selection, focus | `accent`, `selection` |
| Grass (2 shades) | `green` mixed 55 % / 45 % toward `background` |
| Trees | `green` mixed 30 % toward `darker_background` |
| Water (deep/shallow/shimmer) | `blue` mixed toward `darker_background`; `bright_blue` |
| Road / road marking | `lighter_background` / `yellow` |
| Zone R / C / I | `green` / `blue` / `yellow` |
| Roofs and walls | `brown`, `orange`, `red`, `magenta` shades mixed with `muted` |
| Power | `yellow`; Water pipe `cyan` |
| Warn / error | `orange` / `red` |
| Lit windows | `bright_yellow` |

For `mode = "light"` themes, terrain mixes toward `foreground` instead of
`background`, so the map doesn't go washed out. Contrast check: if text vs
panel contrast < 4.5:1, text falls back to the higher-contrast of
`foreground` / `bright_foreground` / pure black-white.

### 8.4 Live recolour
Every 2 s, `stat` the theme dir and `colors.toml`; on mtime/inode change,
reload the palette, regenerate the atlas, mark everything dirty, log
"theme → gruvbox". Can be disabled (`watch_theme = false`).

---

## 9. Input (`input`)

### 9.1 Modes
`NORMAL` (move cursor, pick tools), `ZONE` (pending after `z`),
`VISUAL` (rectangle from anchor to cursor), `COMMAND` (`:` prompt),
`MENU` (build menu), `HELP` (`?` overlay). `Esc` always returns to
`NORMAL`; in NORMAL it clears the tool (inspect mode).

### 9.2 Applying tools
- `Enter` applies the current tool at the cursor.
- **Paint**: `Enter` while moving is tedious, so holding `Shift+Enter`
  toggles *paint mode*: every cursor move applies the tool (status line shows
  `PAINT`). This is the fast way to draw roads with hjkl.
- **Visual** (`v`, then move, then `Enter`): zones and bulldoze fill the
  rectangle; road, line and pipe draw an L-path from anchor to cursor
  (horizontal leg first; `o` in visual swaps the corner, like vim's `o`).
  The cost preview shows in the status line before applying.

### 9.3 Keybindings (defaults, all rebindable)

| Key | Action | Mode |
|---|---|---|
| `h j k l` / arrows | move cursor 1 | NORMAL, VISUAL |
| `H J K L` / Shift+arrows | move cursor 8 | NORMAL, VISUAL |
| `c` | centre camera on cursor | NORMAL |
| `r` | road tool | NORMAL |
| `p` | power line tool | NORMAL |
| `w` | water pipe tool (enables underground view) | NORMAL |
| `d` | bulldoze tool | NORMAL |
| `z r` / `z c` / `z i` | residential / commercial / industrial zone | NORMAL |
| `b` | build menu (plant, pump, police, fire, school) | NORMAL |
| `Enter` | apply tool | NORMAL, VISUAL |
| `Shift+Enter` | toggle paint mode | NORMAL |
| `v` | visual (rectangle) mode | NORMAL |
| `o` | swap L-path corner | VISUAL |
| `o` | cycle overlay | NORMAL |
| `u` | toggle underground view | NORMAL |
| `+` / `-` / scroll | zoom in / out | any |
| `Space` | pause / resume | NORMAL |
| `1` `2` `3` | sim speed | NORMAL |
| `Tab` | toggle side panel | NORMAL |
| `e` / `F2` | toggle event log | NORMAL |
| `:` | command palette | NORMAL |
| `?` | keybinding overlay | NORMAL |
| `Esc` | back to NORMAL / clear tool | any |
| Mouse left click / drag | move cursor / apply (drag = visual) | — |
| Mouse right or middle drag | pan | — |

Note: the brief uses `L` for both "move right fast" and "toggle log". The
table keeps `HJKL` for fast movement and puts the log on `e` (§12). `o` meaning "cycle overlay" in NORMAL and
"swap corner" in VISUAL is intentional: it's mode-dependent, like vim.

### 9.4 Commands

| Command | Effect |
|---|---|
| `:budget` | open budget panel (income/upkeep breakdown) |
| `:tax <n>` | set all taxes to n (0..20) |
| `:tax r\|c\|i <n>` | set one tax |
| `:loan` / `:repay` | take / repay loan |
| `:save [name]` / `:w [name]` | save (default: city name) |
| `:load [name]` / `:e [name]` | load; no name lists saves |
| `:new [seed]` | new map |
| `:name <city>` | rename city |
| `:theme reload` | force theme reload |
| `:quit` / `:q` / `:wq` | quit / save and quit |
| `:help` | same as `?` |

### 9.5 Keymap config
```toml
[keys]
road = ["r"]
move_left = ["h", "Left"]
move_left_fast = ["H", "Shift+Left"]
zone_prefix = ["z"]
```
Unknown actions or keys are reported once in the event log at startup, the
rest of the map still loads.

---

## 10. Config

`~/.config/sprawl/config.toml`, created with commented defaults on first run.

```toml
ui_scale = 2            # HUD pixel scale
zoom = 3                # starting world zoom (1..4)
ticks_per_second = 4    # speed 1
pause_unfocused = true
animations = true
day_night = true
watch_theme = true
low_power = true        # render only on change (see §13)
autosave_months = 6     # 0 = off
map_size = 128
[keys]                  # overrides only
```

---

## 11. Saves

- `~/.local/share/sprawl/<name>.city` = gzip'd JSON, `{"version":1, ...}`.
  Derived per-tile values are not saved; they're recomputed on load.
- `autosave.city` every `autosave_months`.
- Loading an unknown newer version fails with a clear message; older
  versions go through `migrate()` (empty for v1).
- Round-trip test: save → load → identical `City` and identical next 1,000
  ticks (determinism check, seeded RNG).

---

### 6.9 Traffic (added in 0.2.0)
Every land-value update (5 days) a breadth-first search from all roads
touching developed C/I tiles gives each road tile its distance to work.
Each developed home starts at its adjacent road nearest to work and walks
downhill, adding `0.5 · residents` commuters (× 0.7 within 6 tiles of a bus
stop, if a powered bus depot exists) to each road tile on the way.
`congestion = load / 150`. Tiles next to a road with congestion above 0.5
lose up to 0.15 land value. Homes with no road path to jobs get −0.3 on
their growth score (only once the city has jobs); commutes over 40 tiles
get −0.15. Cars (0.3.0) are sampled real trips over the same routes: one per ~5
commuters, morning to work, evening home, a few midday errands; slower on
congested tiles. They're drawn only, not saved.

### 6.10 Big buildings (0.3.0)
A 2×2 square of level-3 lots of one zone, all powered and watered, with
mean land value ≥ 0.75 and zone demand > 0.2, merges with chance 0.03 a
day. Each tile then holds 1.25× its capacity. A merged block counts as
reachable if any tile touches a road. Any decline splits it first.

### 6.11 Terrain (0.3.0)
Heights 0–10; water at 0. Land height = 1 + scaled noise (range per map
type: river/lakes 4, coast/islands 6, highlands 10), capped at
`1 + (distance to water − 1) / 2` so banks rise gently, rock +2, then
lowered until no neighbour step exceeds 2. Slope = largest step to a
neighbour. Zones need slope ≤ 1; buildings need a footprint height range
≤ 1; roads need slope ≤ 2 and cost 3× at 2. Land value +0.05 per level a
tile stands above the mean of its surroundings (radius 4, up to 3).
Terraforming: ±1 level or level to the first tile, $25 per level per tile,
open ground only.

## 12. Decisions

Settled 2026-10-06 (the user asked me to pick; they judge by playing):

1. **Log toggle**: `e` (events) and `F2`. Backtick is a dead key on the
   Swedish layout this machine uses.
2. **Road access**: 4-adjacent, as in the brief.
3. **Bankruptcy**: 12 months in debt ends the game.
4. **Day/night**: 3 real minutes per cycle at speed 1.
5. **Map size**: 128×128.
6. **Default zoom**: 3×.

Balance numbers in §6 are first-pass; expect a tuning milestone.

## 13. Deviations found while building

- **No chunk cache (M1).** Drawing the visible tiles straight from the
  sprite images is cheap because Ebitengine batches them, and frames are
  only drawn when something changed. Revisit if M8 profiling disagrees.
- **Low-power frame pacing.** Skipping draws alone left ~3 % of a core in
  Ebitengine's 60 Hz loop. The game uses Ebitengine's minimal-FPS mode
  (deprecated in name, still implemented on desktop in 2.10) plus a small
  pacer: 60 Hz while keys/mouse are held or the camera eases, 8 Hz while
  animations are visible, 2 Hz otherwise. `low_power = false` turns it off.
  Measured: 0.7 % of one core idle, 1.2 % with water shimmer.
- **No vim counts or `gg`/`G` (M3).** `1 2 3` are the speed keys from the
  brief, so `3l` would be ambiguous; HJKL already moves 8.
- **Mouse drag with a tool** works like visual mode and applies on release.
- **Paint mode** skips tiles it can't change without an error, so painting
  across an existing road doesn't nag.
- **Balance (M4).** With 0.5 factory jobs per worker, jobs per worker stay
  below 1 and the city stalls near 700 people; raised to 0.8. Empty lots
  now grow on demand alone (see §6.3) and growth rates went up. A skipped
  test prints a growth curve: `BALANCE=1 go test ./sim -run Balance -v`.
- **Overlays don't rely on hue (M5).** Unserved tiles are hatched, not just
  red: in some themes (hackerman) red and green are nearly identical.
- **High density at LV ≥ 0.60** (was 0.65) and coverage falls off from a
  plateau (§6.6): with the old numbers a block next to all three services
  and its own plant still couldn't reach high density.
- **Performance (M8).** Paused: 0.3 % of a core. Running a small town:
  5 % at speed 1, 8 % at speed 3, because every tick redraws the visible
  world. A cached chunk layer (§7.4) would cut that; not done yet.
- **Sound through oto, not ebiten/audio (M15).** An open audio stream
  costs ~1.5 % of a core even when silent and Ebitengine's audio package
  can't pause it; oto (its underlying library) can, so the stream is
  suspended 3 s after the last effect. Effects are synthesised in code.
- **Performance, phase 2 (M16).** Static layers are cached in 32×32-tile
  chunks, the HUD is cached by a content key, a running sim only requests
  a frame when the date or a tile changes, and rising buildings animate at
  24 fps. Speed 1: 2.8 % of a core while growing, 1.5 % steady; paused
  0.2–0.3 %.
- **Buildings are data (M14).** Size, cost, upkeep, power/water supply and
  use, coverage, land-value bonus, smog and unlock live in one table
  (`sim.Buildings`).
- **Window class** is `sprawl` (XWayland), for Hyprland window rules.
