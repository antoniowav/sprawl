# Changelog

## 0.3.0 — 2026-10-06

- Cars are real trips now: one car for about five commuters leaves a home
  in the morning rush, drives the routed path through the junctions to
  work, parks, and drives home in the evening; a few errands at midday,
  quiet at night. Jams slow them down.
- 2×2 buildings: four high-density lots of one zone with land value 0.75+,
  power, water and demand merge into a residential tower, an office tower
  or mall, or an industrial complex, with 25% more capacity. They split
  again if one tile declines; bulldozing removes the whole block.
- Terrain: every map has hills (heights 0–10), with shading and terrace
  edges. Water lies in the valleys. Zones and buildings need gentle ground,
  roads can climb two-level steps at triple cost, cliffs take nothing.
  Hilltops with a view raise land value. Terraforming: t then r/l/f
  (raise, lower, level) or the mountain button, $25 per level per tile;
  lowering ground beside water floods it, raising water makes land.
  Height overlay. Saves from 0.2.0 load flat.

## 0.2.0 — 2026-10-06

- Traffic: homes route their commuters to the nearest jobs over the roads;
  busy roads get cars, congestion lowers land value, and homes with no road
  to any job grow worse. Traffic overlay. Bus stops (with a bus depot) take
  30% of nearby commuters off the road.
- New buildings: hospital (health coverage), bus stop, bus depot, solar
  farm, university, nuclear plant, monument.
- Map types: river valley, coast, lakes, islands, highlands (rock you can't
  build on). Roads cross water as bridges. Maps up to 256×256.
- Scenarios: Boomtown and Island, with goals, a deadline, events partway
  through, and win and lose screens.
- The title screen shows a town building itself, through day and night.
- Minimap (m), zoom toward the mouse, day/night mode (n: cycle, always
  day, always night, paused).
- Building inspector (click, or i): what a tile holds and why it isn't
  growing. Advisor tips. Achievements (F5). Photo mode (F12) saves a PNG to
  ~/Pictures/Sprawl. Key bindings editable in Settings.
- Follows Cuore's theme location as well as Omarchy's.

## 0.1.0 — 2026-10-06

First release.

- Roads, power lines and water pipes; residential, commercial and industrial
  zones that grow, densify and decline with demand.
- Power plant, wind turbine, water pump, water tower, police, fire station,
  school, park; city hall and stadium unlock with population.
- Budget with taxes, upkeep, loans, debt and bankruptcy; milestone grants
  from hamlet to metropolis; statistics charts.
- Overlays for power, water, service coverage, land value and pollution.
- Title screen, new-city setup with map preview, pause menu, settings,
  getting-started guide, toolbar, undo/redo.
- Ctrl+S/O/N/Q/Z shortcuts alongside vim-style keys; every key rebindable.
- Pixel art, font and sound effects all generated in code; day and night,
  smoke, water shimmer.
- Follows the Cuore or Omarchy desktop theme live.
- Saves and autosave in `~/.local/share/sprawl`.
