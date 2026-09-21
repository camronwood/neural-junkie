"""Maps sidecar routes: Nominatim geocode + OSRM walking/driving routes."""
from __future__ import annotations

import json
import threading
import time
import urllib.error
import urllib.parse
import urllib.request
from typing import Any

_DEFAULT_NOMINATIM = "https://nominatim.openstreetmap.org"
_DEFAULT_OSRM = "https://router.project-osrm.org"
_DEFAULT_UA = "NeuralJunkieMaps/1.0 (https://github.com/camronwood/neural-junkie)"
_DEFAULT_ATTRIBUTION = "© OpenStreetMap contributors"

_rate_lock = threading.Lock()
_last_request_at = 0.0
_geocode_cache: dict[str, dict[str, Any]] = {}
_geocode_cache_lock = threading.Lock()


def _settings_str(settings: dict, key: str, default: str) -> str:
    val = settings.get(key)
    if val is None:
        return default
    text = str(val).strip()
    return text or default


def _rate_limit_rps(settings: dict) -> float:
    raw = settings.get("maps_rate_limit_rps", "1")
    try:
        rps = float(raw)
    except (TypeError, ValueError):
        rps = 1.0
    return max(0.1, rps)


def _throttle(settings: dict) -> None:
    global _last_request_at
    min_interval = 1.0 / _rate_limit_rps(settings)
    with _rate_lock:
        now = time.monotonic()
        wait = min_interval - (now - _last_request_at)
        if wait > 0:
            time.sleep(wait)
        _last_request_at = time.monotonic()


def _http_get_json(url: str, user_agent: str, timeout: float = 30.0) -> tuple[int, Any]:
    req = urllib.request.Request(
        url,
        headers={
            "User-Agent": user_agent,
            "Accept": "application/json",
        },
        method="GET",
    )
    try:
        with urllib.request.urlopen(req, timeout=timeout) as resp:
            raw = resp.read().decode("utf-8")
            return resp.status, json.loads(raw) if raw else {}
    except urllib.error.HTTPError as exc:
        body = ""
        try:
            body = exc.read().decode("utf-8", errors="replace")
        except Exception:
            pass
        try:
            payload = json.loads(body) if body else {"error": str(exc)}
        except json.JSONDecodeError:
            payload = {"error": body or str(exc)}
        return exc.code, payload
    except Exception as exc:  # noqa: BLE001
        return 502, {"error": str(exc)}


def _json_response(handler: Any, code: int, payload: dict) -> None:
    handler._json(code, payload)


def handle_get(handler: Any, path: str, settings: dict, _pack_dir: str) -> None:
    if path == "/api/maps/status":
        _json_response(
            handler,
            200,
            {
                "ok": True,
                "nominatim_base_url": _settings_str(settings, "nominatim_base_url", _DEFAULT_NOMINATIM),
                "osrm_base_url": _settings_str(settings, "osrm_base_url", _DEFAULT_OSRM),
                "attribution": _DEFAULT_ATTRIBUTION,
            },
        )
        return
    if path == "/api/maps/geocode":
        query = urllib.parse.parse_qs(urllib.parse.urlparse(handler.path).query)
        q = (query.get("q") or query.get("query") or [""])[0]
        body = {"query": q, "limit": int((query.get("limit") or ["5"])[0])}
        _geocode(handler, body, settings)
        return
    _json_response(handler, 404, {"error": "not found"})


def handle_post(handler: Any, path: str, body: dict, settings: dict, _pack_dir: str) -> None:
    if path == "/api/maps/geocode":
        _geocode(handler, body or {}, settings)
        return
    if path == "/api/maps/reverse":
        _reverse(handler, body or {}, settings)
        return
    if path == "/api/maps/route":
        _route(handler, body or {}, settings)
        return
    _json_response(handler, 404, {"error": "not found"})


def _geocode(handler: Any, body: dict, settings: dict) -> None:
    query = str(body.get("query") or body.get("q") or "").strip()
    if not query:
        _json_response(handler, 400, {"error": "query is required"})
        return
    try:
        limit = int(body.get("limit") or 5)
    except (TypeError, ValueError):
        limit = 5
    limit = max(1, min(limit, 10))

    viewbox = _viewbox_param(body)
    cache_key = f"{query.lower()}|{limit}|{viewbox}"
    with _geocode_cache_lock:
        cached = _geocode_cache.get(cache_key)
    if cached is not None:
        _json_response(handler, 200, cached)
        return

    base = _settings_str(settings, "nominatim_base_url", _DEFAULT_NOMINATIM).rstrip("/")
    ua = _settings_str(settings, "maps_user_agent", _DEFAULT_UA)
    params: dict[str, str] = {
        "q": query,
        "format": "json",
        "limit": str(limit),
        "addressdetails": "0",
    }
    if viewbox:
        params["viewbox"] = viewbox
        params["bounded"] = "0"
    params_qs = urllib.parse.urlencode(params)
    _throttle(settings)
    status, data = _http_get_json(f"{base}/search?{params_qs}", ua)
    if status == 429:
        _json_response(handler, 429, {"error": "Nominatim rate limit exceeded; retry shortly"})
        return
    if status >= 400:
        err = data.get("error") if isinstance(data, dict) else None
        _json_response(handler, status if status < 600 else 502, {"error": err or f"geocode failed ({status})"})
        return
    if not isinstance(data, list):
        _json_response(handler, 502, {"error": "unexpected Nominatim response"})
        return

    results = []
    for item in data:
        try:
            lat = float(item.get("lat"))
            lon = float(item.get("lon"))
        except (TypeError, ValueError):
            continue
        results.append(
            {
                "lat": lat,
                "lon": lon,
                "display_name": item.get("display_name") or query,
                "importance": item.get("importance"),
                "osm_type": item.get("osm_type"),
                "osm_id": item.get("osm_id"),
            }
        )
    payload = {
        "query": query,
        "results": results,
        "attribution": _DEFAULT_ATTRIBUTION,
    }
    with _geocode_cache_lock:
        if len(_geocode_cache) > 256:
            _geocode_cache.clear()
        _geocode_cache[cache_key] = payload
    _json_response(handler, 200, payload)


def _viewbox_param(body: dict) -> str:
    raw = body.get("viewbox")
    if isinstance(raw, str) and raw.strip():
        return raw.strip()
    near = body.get("near")
    if not isinstance(near, dict):
        return ""
    try:
        lat = float(near.get("lat"))
        lon = float(near.get("lon", near.get("lng")))
    except (TypeError, ValueError):
        return ""
    # ~15km box to bias Nominatim without hard-bounding the query.
    delta = 0.15
    min_lon = max(-180.0, lon - delta)
    max_lon = min(180.0, lon + delta)
    min_lat = max(-90.0, lat - delta)
    max_lat = min(90.0, lat + delta)
    return f"{min_lon},{max_lat},{max_lon},{min_lat}"


def _reverse(handler: Any, body: dict, settings: dict) -> None:
    try:
        lat = float(body.get("lat"))
        lon = float(body.get("lon", body.get("lng")))
    except (TypeError, ValueError):
        _json_response(handler, 400, {"error": "lat and lon are required numbers"})
        return
    if lat < -90 or lat > 90 or lon < -180 or lon > 180:
        _json_response(handler, 400, {"error": "lat/lon out of range"})
        return

    base = _settings_str(settings, "nominatim_base_url", _DEFAULT_NOMINATIM).rstrip("/")
    ua = _settings_str(settings, "maps_user_agent", _DEFAULT_UA)
    params = urllib.parse.urlencode(
        {
            "lat": f"{lat:.7f}",
            "lon": f"{lon:.7f}",
            "format": "json",
            "zoom": "14",
            "addressdetails": "1",
        }
    )
    _throttle(settings)
    status, data = _http_get_json(f"{base}/reverse?{params}", ua)
    if status == 429:
        _json_response(handler, 429, {"error": "Nominatim rate limit exceeded; retry shortly"})
        return
    if status >= 400:
        err = data.get("error") if isinstance(data, dict) else None
        _json_response(handler, status if status < 600 else 502, {"error": err or f"reverse geocode failed ({status})"})
        return
    if not isinstance(data, dict):
        _json_response(handler, 502, {"error": "unexpected Nominatim reverse response"})
        return
    display = str(data.get("display_name") or "").strip()
    addr = data.get("address") if isinstance(data.get("address"), dict) else {}
    city = ""
    if isinstance(addr, dict):
        city = str(
            addr.get("city")
            or addr.get("town")
            or addr.get("village")
            or addr.get("hamlet")
            or addr.get("county")
            or ""
        ).strip()
        region = str(addr.get("state") or addr.get("region") or "").strip()
        country = str(addr.get("country") or "").strip()
        parts = [p for p in (city, region, country) if p]
        if parts and not display:
            display = ", ".join(parts)
    _json_response(
        handler,
        200,
        {
            "lat": lat,
            "lon": lon,
            "display_name": display or f"{lat:.5f}, {lon:.5f}",
            "city": city,
            "attribution": _DEFAULT_ATTRIBUTION,
        },
    )


def _normalize_mode(mode: str) -> str:
    m = (mode or "walking").strip().lower()
    if m in ("walk", "walking", "foot", "pedestrian"):
        return "walking"
    if m in ("drive", "driving", "car", "auto", "automobile"):
        return "driving"
    return ""


def _osrm_profile(mode: str) -> str:
    return "foot" if mode == "walking" else "car"


def _route(handler: Any, body: dict, settings: dict) -> None:
    mode = _normalize_mode(str(body.get("mode") or "walking"))
    if not mode:
        _json_response(handler, 400, {"error": "mode must be walking or driving"})
        return

    waypoints = body.get("waypoints") or body.get("points") or []
    if not isinstance(waypoints, list) or len(waypoints) < 2:
        _json_response(handler, 400, {"error": "waypoints must be an array of at least 2 {lat, lon} points"})
        return

    coords: list[str] = []
    for wp in waypoints:
        if not isinstance(wp, dict):
            _json_response(handler, 400, {"error": "each waypoint must be an object with lat and lon"})
            return
        try:
            lat = float(wp.get("lat"))
            lon = float(wp.get("lon", wp.get("lng")))
        except (TypeError, ValueError):
            _json_response(handler, 400, {"error": "waypoint lat/lon must be numbers"})
            return
        coords.append(f"{lon},{lat}")

    base = _settings_str(settings, "osrm_base_url", _DEFAULT_OSRM).rstrip("/")
    ua = _settings_str(settings, "maps_user_agent", _DEFAULT_UA)
    profile = _osrm_profile(mode)
    path = f"{base}/route/v1/{profile}/{';'.join(coords)}"
    params = urllib.parse.urlencode({"overview": "full", "geometries": "geojson", "steps": "false"})
    _throttle(settings)
    status, data = _http_get_json(f"{path}?{params}", ua)
    if status == 429:
        _json_response(handler, 429, {"error": "OSRM rate limit exceeded; retry shortly or set osrm_base_url to a self-hosted instance"})
        return
    if status >= 400:
        err = data.get("message") if isinstance(data, dict) else None
        if not err and isinstance(data, dict):
            err = data.get("error")
        _json_response(handler, status if status < 600 else 502, {"error": err or f"route failed ({status})"})
        return
    if not isinstance(data, dict):
        _json_response(handler, 502, {"error": "unexpected OSRM response"})
        return
    if data.get("code") not in (None, "Ok"):
        _json_response(handler, 422, {"error": data.get("message") or data.get("code") or "no route found"})
        return
    routes = data.get("routes") or []
    if not routes:
        _json_response(handler, 422, {"error": "no route found"})
        return
    best = routes[0]
    geometry = best.get("geometry") or {"type": "LineString", "coordinates": []}
    payload = {
        "mode": mode,
        "distance_m": best.get("distance"),
        "duration_s": best.get("duration"),
        "geometry": geometry,
        "waypoints": [
            {"lat": float(wp["lat"]), "lon": float(wp.get("lon", wp.get("lng")))}
            for wp in waypoints
            if isinstance(wp, dict)
        ],
        "attribution": _DEFAULT_ATTRIBUTION,
        "tile_url_template": _settings_str(
            settings,
            "osm_tile_url_template",
            "https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png",
        ),
    }
    _json_response(handler, 200, payload)
