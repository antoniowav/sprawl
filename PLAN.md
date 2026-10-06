# Sprawl — Build Plan

Each milestone ends runnable. Gate for every milestone:
`gofmt -l .` empty, `go vet ./...` clean, `go build ./...` and
`go test ./...` pass, then a short summary, how to try it, and a screenshot.

Look first: M1–M3 are about feel (camera, crisp tiles, HUD, theme) before
the simulation gets deep.

---

## M1 — Window, map, camera ✅ (2026-10-06)
- `go mod init`, `cmd/sprawl`, `internal/meta` (name constant, XDG paths).
- `config` package: load/create `config.toml` with defaults.
- `sim`: `Tile`, `City`, seeded terrain generation (grass, water, trees).
  Tests: same seed → same map; water/tree ratios within bounds.
- `render`: procedural grass/tree/water sprites from a hard-coded palette,
  integer zoom 1–4×, nearest filtering, snapped eased camera, chunk cache.
- `input` (minimal): hjkl/HJKL/arrows, `+`/`-`/scroll, mouse pan, `Esc`.
- Idle-CPU work: dirty flag, no redraw when idle, unfocused throttle.
  Measure CPU idle vs scrolling and report numbers.

**Try:** `go run ./cmd/sprawl` → walk around a generated map, zoom.

## M2 — Theme + HUD frame ✅ (2026-10-06)
- `theme`: palette type, built-in "Sprawl Dusk", `colors.toml` parser,
  `alacritty.toml` fallback, search order (§8.1), role mapping, light-mode
  handling, contrast fallback. Tests with fixture theme dirs.
- Live theme polling + atlas regeneration.
- `render/font`: 5×7 bitmap font (ASCII + box/arrow glyphs).
- HUD frame: top bar, side panel (placeholder values), status line,
  log pane, `?` overlay generated from the keymap.
- `:` prompt with `:quit`, `:help`, `:theme reload`.

**Try:** switch Omarchy theme while the game runs → recolours live.

Done notes: keymap layer (bindings, config overrides, help table) landed
here already; counts, `gg`, `z` prefix, visual and paint modes stay in M3.
Placeholder HUD values (demand, power, water, net) go live in M4–M7.

## M3 — Keymap + tools + placement ✅ (2026-10-06)
- Full `input` layer: actions, modes, configurable keymap from TOML,
  counts, `gg`/`G`, `z` prefix, visual mode, paint mode, mouse click/drag.
  Tests: keymap parsing, mode transitions, count parsing.
- `sim/build.go`: place road/line/pipe/zones, bulldoze, cost checks,
  trees cleared. Tests for every rule.
- Autotile sprites (16 masks) for road, line, pipe; empty zone lots;
  visual-mode preview with cost in the status line; underground view `u`.

**Try:** draw roads with paint mode, zone blocks with `v`, see funds drop.

## M4 — Clock, demand, growth ✅ (2026-10-06)
- Fixed-step accumulator, `Tick()`, speeds `1/2/3`, pause.
- Aggregates, demand formulas (§6.2), growth/decline (§6.3) with power and
  water temporarily treated as "always on" (flag), so growth is visible now.
  Tests: demand values for hand-built cities, determinism, growth respects
  road access and LV caps.
- Building sprites: R/C/I × 3 levels × 4 variants; grow animation.
- Demand bars, population and date in the HUD go live.

**Try:** zone next to roads, unpause, watch the town grow.

## M5 — Power and water ✅ (2026-10-06)
- Build menu `b`; multi-tile placement for plant and pump (pump must touch
  water).
- Network BFS + nearest-first capacity allocation (§6.4, §6.5). Tests:
  connectivity, brownout order, zones conducting, pumps needing power.
- Turn off the "always on" flag. Unpowered/unwatered icons.
- Overlays `o`: power, water. HUD power/water meters.

**Try:** a city with one plant browning out as it grows; add a second.

## M6 — Services, pollution, land value ✅ (2026-10-06)
- Police, fire, school buildings; coverage, pollution, land value (§6.6).
  Tests: coverage falloff, pollution sum, LV clamp, LV → max level.
- Overlays: police, fire, school, land value, pollution with legends.
- Inspect info in the status line (LV, coverage, pollution of the tile).

**Try:** place a school, watch the LV overlay and density rise.

## M7 — Budget and commands ✅ (2026-10-06)
- Monthly budget (§6.7), taxes, loan, debt and bankruptcy states.
  Tests: month rollover math, loan schedule, debt blocks building,
  bankruptcy after N months.
- Full command palette (§9.4) with history and tab completion;
  `:budget` panel with breakdown.

**Try:** raise taxes to 20 and watch demand fall; go broke on purpose.

## M8 — Events, saves, polish ✅ (2026-10-06)
- Event log rules (§6.8), log pane styling.
- Save/load/autosave, versioned format. Tests: round trip + 1,000-tick
  determinism after load.
- Day/night with lit windows, smoke particles, water shimmer.
- Performance pass: re-measure idle/scroll CPU at 128×128 fully built.

**Try:** build, `:w town`, quit, `:e town`.

## M9 — Packaging and tuning (packaging ✅ 2026-10-06; tuning waits for your play session)
- README (install, controls, screenshot placeholder), `PKGBUILD` stub,
  `.desktop` file, `-version` flag.
- Balance pass with you (a short play session, then adjust `sim/tuning.go`).

---

Rough size: M1–M3 are the bulk of the render/input work; M4–M7 are mostly
`sim` with tests; M8–M9 are polish. Your review points: after M2 (look),
after M4 (growth feel), after M9 (balance).

---

# Phase 2 — from prototype to full game (started 2026-10-06)

Goal: something you can ship in a distro. Standard desktop shortcuts
(Ctrl+S/O/N/Q/Z) next to the vim keys; menus for everything a newcomer
would look for; enough content and goals to keep playing.

## M10 — File shortcuts ✅ (2026-10-06)
Ctrl+S save (asks for a name the first time), Ctrl+Shift+S save as,
Ctrl+O open (list, Del deletes), Ctrl+N new, Ctrl+Q quit; unsaved-changes
prompt on quit, new, open and the window close button.

## M11 — Title screen and pause menu ✅ (2026-10-06)
Title: Continue (latest save), New city, Open, Settings, Quit, over a live
animated map. New city: name, map size (96/128/192), seed with a terrain
preview. Esc with no tool opens a pause menu (Resume, Save, Open,
Settings, Quit to title, Quit). Mouse works everywhere.

## M12 — Undo and toolbar ✅ (2026-10-06)
Ctrl+Z / Ctrl+Shift+Z (and Ctrl+Y) undo/redo any build action with its
money. A clickable toolbar (tool icons, hover names with keys) and a hover
tooltip for the tile under the mouse.

## M13 — First-time guide ✅ (2026-10-06)
A short, skippable step list in a corner panel that reacts to what you do
(road → zone → power → wait for growth → water → services → taxes).
Shown on the first city only; re-open from the help.

## M14 — Content and goals ✅ (2026-10-06)
Parks (land value, small upkeep), a larger power plant, water tower,
city hall, stadium. Milestones unlock buildings and grant rewards;
a goal line in the top bar ("Reach 1,000 people"). A city stats screen
with history charts (population, funds, demand).

## M15 — Settings and sound ✅ (2026-10-06)
In-game settings screen (UI scale, zoom, animations, day/night,
autosave, volume, key list). Procedural sound effects and ambience with
Ebitengine's audio package (no new dependency), off by default if no
audio device.

## M16 — Performance ✅ (2026-10-06)
Chunk cache so a running city redraws only changed tiles; target < 3 %
of a core at speed 1.

## M17 — Distro packaging ✅ (2026-10-06)
Licence (your choice) with third-party notices (Ebitengine: Apache-2.0,
BurntSushi/toml: MIT), AppStream metainfo, icons at 32–512 px, man page,
`make install PREFIX=/usr`, finished PKGBUILD.

Phase 2 notes:
- Settings has no key-binding editor; the screen points to `[keys]` in the
  config file.
- M17 left for you: licence, final app id (reverse-DNS) and homepage in
  `dist/sprawl.metainfo.xml` and `dist/PKGBUILD`, real screenshots.

---

# Phase 3 — more game (started 2026-10-06)

Chosen by the user: traffic with congestion and cars, Boomtown and Island
scenarios, timelapse title screen, photo mode, building inspector, advisor
tips, achievements; plus map types, more buildings, manual day/night,
key editor, minimap, bigger maps, zoom toward the cursor.

## M18 — Comfort ✅ (2026-10-06)
Wheel zoom keeps the tile under the mouse in place. Day/night mode:
cycle, always day, always night, frozen (setting + `n` key). Map sizes up
to 256×256. Minimap (`m`), click or drag to jump.

## M19 — Map types and bridges ✅ (2026-10-06)
River valley, coast, lakes, islands, highlands (rock you can't build on).
Roads may cross water as bridges ($60/tile). Chosen in the new-city form
with the preview.

## M20 — Traffic ✅ (2026-10-06)
Monthly commute routing on the road graph: every developed home sends its
workers along the shortest road path to the nearest jobs; load per road
tile vs capacity gives congestion. Congestion lowers land value along the
road; homes that can't reach any job by road grow worse. Traffic overlay.
Pixel cars on busy roads. Bus stops (need a bus depot) take a share of
trips off the road.

## M21 — More buildings ✅ (2026-10-06)
Hospital (health coverage), bus stop, bus depot, solar farm (unlocks at
500), university (2,500), nuclear plant (5,000, no smog), monument
(10,000).

## M22 — Scenarios ✅ (2026-10-06)
Game mode in the new-city form: Sandbox or a scenario. Boomtown: a factory
announces 2,000 jobs; reach 2,000 workers and a positive budget by Y5.
Island: islands map, tight money, tourism boom in Y3; reach 3,000 people
by Y10. Goal panel with days left; win and lose screens; saved with the
city.

## M23 — Timelapse title ✅ (2026-10-06)
Behind the title, a city builds itself on a fresh map: roads, zones,
plant, pipes, services, growth with rising buildings, sped-up day and
night, slow camera drift.

## M24 — Extras ✅ (2026-10-06)
Key-binding editor in Settings; photo mode (F12 → ~/Pictures/Sprawl);
building inspector (click a building); advisor tips; achievements.

---

# Phase 4 — living traffic, big buildings, terrain (started 2026-10-06)

## M25 — Real car trips ✅ (2026-10-06)
Cars are sampled real trips (one car ≈ 20 commuters): leave a home in the
morning rush, drive the routed path through junctions to work, park, and
drive home in the evening; a few shopping trips at midday; quiet at night.
Speed drops on congested tiles. Trips follow a traffic clock that always
cycles, even when the sky is held at day or night.

## M26 — 2×2 buildings ✅ (2026-10-06)
Four high-density lots of the same zone in a square, with land value
≥ 0.75, power, water and demand, merge into one big building (residential
tower, office tower or mall, factory complex) with 25% more capacity. If
any of the four would decline, or is bulldozed, the block splits again.

## M27 — Terrain ✅ (2026-10-06)
Every tile has a height (0–10). Generated from noise per map type; rivers
run downhill along valleys into lakes or off the map. Hill shading and
terrace edges show height. Rules: zones and buildings need gentle ground
(neighbours within 1 level; a building's footprint within 1 level), roads
climb anything up to 2 levels at triple cost, nothing on cliffs. Hilltops
add land value (the view). Terraforming tools (t then r/l/f: raise, lower,
level) at $25 per level per tile; lowering next to water floods, raising
water makes land. Saved with the city; older saves load flat.
