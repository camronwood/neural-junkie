# Maps pack

Geocode places and build walking or driving routes with the public OpenStreetMap stack (Nominatim + OSRM), then publish interactive maps to Neural Canvas (`nj.map`). Tools attach to **Assistant** when this pack is enabled (Composition Model can also grant them to custom experts). Device location is a separate **sensitive** capability (`maps-location`).

## Install

1. Install and enable the **Maps** pack in Settings → Domain packs.
2. No API keys are required for the default public endpoints.
3. Start or restart the hub so the maps sidecar comes up (`/api/maps/status`).
4. Ask **Assistant** for directions or a map — there is no MapsExpert agent.

## Tools

| Tool | Purpose |
|------|---------|
| `maps_geocode` | Resolve a place name or address to lat/lon (optional `near` / `viewbox` bias) |
| `maps_route` | Compute walking or driving route between waypoints |
| `maps_create` | Publish a Neural Canvas map artifact (markers + routes) |
| `maps_update` | Update an existing map artifact |
| `maps_locate` | Read this device's GPS (sensitive grant + user consent). Prefer session-shared location when present. |

## Settings

| Key | Default | Notes |
|-----|---------|--------|
| `nominatim_base_url` | `https://nominatim.openstreetmap.org` | Geocoding |
| `osrm_base_url` | `https://router.project-osrm.org` | Routing (public demo — light use) |
| `osm_tile_url_template` | OSM raster tiles | Used in canvas basemap metadata |
| `maps_user_agent` | `NeuralJunkieMaps/1.0 (...)` | Required by OSM usage policies |
| `maps_rate_limit_rps` | `1` | Client-side throttle (Nominatim policy) |

For heavier routing, point `osrm_base_url` at a self-hosted OSRM or Valhalla-compatible gateway.

## Device location

1. Grant **Device location** in Settings → Capabilities (sensitive; not inherited).
2. Tap **Share location** in the composer to inject the current place into this session (used for “near me” and `web_search` rewrite).
3. `maps_locate` asks for a fresh reading when session share is off or the cache is older than ~2 minutes.
4. Precise coordinates are not persisted in chat history (at most a coarse place label).

Never invent coordinates. Do not store location in map artifacts unless the user asked for a map of “here”.

## OSM usage

- Identify this app via `maps_user_agent`.
- Keep geocode traffic modest (default 1 req/s; results are session-cached in the sidecar).
- Always show OpenStreetMap attribution on maps (canvas renderer does this).
- Public OSRM is for demos and light personal use — not production bulk routing.

## Artifact format

Media type: `application/vnd.neural-junkie.map+json`  
Files: `*.nj-map.json`

Payload includes `center`, `zoom`, `markers[]`, `routes[]` with GeoJSON `LineString` geometry (`[lon, lat]` coordinates).

## Smoke

```bash
make verify
make pack-smoke
```
