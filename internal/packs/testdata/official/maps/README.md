# Neural Junkie — Maps pack

Maps tools geocode places and build walking/driving routes with OpenStreetMap (Nominatim + OSRM), then publish interactive maps to Neural Canvas (`nj.map`). Device location is an optional sensitive capability (`maps_locate` + composer share).

## Install

Install from Neural Junkie → Settings → Domain packs (catalog entry `maps`), or download the release zip.

## Develop

```bash
make verify
make pack-smoke
make pack-zip
```

See [assets/WORKSPACE.md](assets/WORKSPACE.md).
