package maps

import (
	"context"
	"fmt"
	"strings"

	mapssideecar "github.com/camronwood/neural-junkie/internal/maps"
)

// MediaType is the Neural Canvas media type for map artifacts.
const MediaType = "application/vnd.neural-junkie.map+json"

// DefaultAttribution is shown on OSM-backed maps.
const DefaultAttribution = "© OpenStreetMap contributors"

// BuildMapPayload constructs a map+json document, optionally computing a route.
func BuildMapPayload(ctx context.Context, client *mapssideecar.Client, args map[string]any) (map[string]any, error) {
	title, _ := args["title"].(string)
	title = strings.TrimSpace(title)
	if title == "" {
		title = "Map"
	}

	payload := map[string]any{
		"schema_version": 1,
		"title":          title,
		"basemap":        "osm",
		"attribution":    DefaultAttribution,
	}
	if zoom, ok := asFloat(args["zoom"]); ok {
		payload["zoom"] = zoom
	} else {
		payload["zoom"] = 14.0
	}
	if center := asLatLonMap(args["center"]); center != nil {
		payload["center"] = center
	}
	if tile, _ := args["tile_url_template"].(string); strings.TrimSpace(tile) != "" {
		payload["tile_url_template"] = strings.TrimSpace(tile)
	}

	markers := asObjectSlice(args["markers"])
	routes := asObjectSlice(args["routes"])
	waypoints := asObjectSlice(args["waypoints"])

	if len(routes) == 0 && len(waypoints) >= 2 && client != nil {
		mode, _ := args["mode"].(string)
		mode = strings.TrimSpace(mode)
		if mode == "" {
			mode = "walking"
		}
		routeReq := map[string]any{
			"mode":      mode,
			"waypoints": waypointsForRoute(waypoints),
		}
		out, err := client.Route(ctx, routeReq)
		if err != nil {
			return nil, err
		}
		route := map[string]any{
			"id":         "r1",
			"mode":       out["mode"],
			"distance_m": out["distance_m"],
			"duration_s": out["duration_s"],
			"geometry":   out["geometry"],
		}
		routes = []map[string]any{route}
		if tile, _ := out["tile_url_template"].(string); tile != "" {
			payload["tile_url_template"] = tile
		}
		if len(markers) == 0 {
			markers = markersFromWaypoints(waypoints)
		}
	}

	if len(markers) > 0 {
		normalized := make([]map[string]any, 0, len(markers))
		for i, m := range markers {
			lat, lon, ok := latLonFrom(m)
			if !ok {
				continue
			}
			id, _ := m["id"].(string)
			if strings.TrimSpace(id) == "" {
				id = fmt.Sprintf("m%d", i+1)
			}
			item := map[string]any{"id": id, "lat": lat, "lon": lon}
			if label, _ := m["label"].(string); strings.TrimSpace(label) != "" {
				item["label"] = strings.TrimSpace(label)
			}
			normalized = append(normalized, item)
		}
		payload["markers"] = normalized
		if payload["center"] == nil && len(normalized) > 0 {
			payload["center"] = map[string]any{
				"lat": normalized[0]["lat"],
				"lon": normalized[0]["lon"],
			}
		}
	}
	if len(routes) > 0 {
		payload["routes"] = routes
	}
	if payload["center"] == nil {
		return nil, fmt.Errorf("map needs center, markers, or a route")
	}
	return payload, nil
}

// StripMetaArgs removes tool-only keys from an update payload merge.
func StripMetaArgs(args map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range args {
		switch k {
		case "artifact_id", "expected_revision", "hint":
			continue
		default:
			out[k] = v
		}
	}
	return out
}

func asObjectSlice(v any) []map[string]any {
	arr, ok := v.([]any)
	if !ok {
		if typed, ok := v.([]map[string]any); ok {
			return typed
		}
		return nil
	}
	out := make([]map[string]any, 0, len(arr))
	for _, item := range arr {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}

func asLatLonMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	lat, lon, ok := latLonFrom(m)
	if !ok {
		return nil
	}
	return map[string]any{"lat": lat, "lon": lon}
}

func latLonFrom(m map[string]any) (float64, float64, bool) {
	lat, okLat := asFloat(m["lat"])
	lon, okLon := asFloat(m["lon"])
	if !okLon {
		lon, okLon = asFloat(m["lng"])
	}
	return lat, lon, okLat && okLon
}

func asFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case float32:
		return float64(n), true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	case jsonNumber:
		f, err := n.Float64()
		return f, err == nil
	default:
		return 0, false
	}
}

type jsonNumber interface {
	Float64() (float64, error)
}

func waypointsForRoute(waypoints []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(waypoints))
	for _, wp := range waypoints {
		lat, lon, ok := latLonFrom(wp)
		if !ok {
			continue
		}
		out = append(out, map[string]any{"lat": lat, "lon": lon})
	}
	return out
}

func markersFromWaypoints(waypoints []map[string]any) []map[string]any {
	out := make([]map[string]any, 0, len(waypoints))
	for i, wp := range waypoints {
		lat, lon, ok := latLonFrom(wp)
		if !ok {
			continue
		}
		id := fmt.Sprintf("m%d", i+1)
		item := map[string]any{"id": id, "lat": lat, "lon": lon}
		if label, _ := wp["label"].(string); strings.TrimSpace(label) != "" {
			item["label"] = strings.TrimSpace(label)
		} else if i == 0 {
			item["label"] = "Start"
		} else if i == len(waypoints)-1 {
			item["label"] = "End"
		}
		out = append(out, item)
	}
	return out
}
