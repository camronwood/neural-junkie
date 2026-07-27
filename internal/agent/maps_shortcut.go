package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	mapssideecar "github.com/camronwood/neural-junkie/internal/maps"
	"github.com/camronwood/neural-junkie/internal/protocol"
)

var (
	mapRouteNounRE = regexp.MustCompile(`\b(map|maps|route|routes|directions|itinerary|geocode|navigate|navigation)\b`)
	mapRouteVerbRE = regexp.MustCompile(`\b(draw|make|create|generate|show|get|plot|build|give|need|want|find|compute)\b`)
	// from A to B / A to B with place-like tokens
	mapFromToRE = regexp.MustCompile(`(?i)\bfrom\s+(.+?)\s+to\s+(.+)$`)
	mapToRE     = regexp.MustCompile(`(?i)\b(?:map|route|directions?|drive|walk|navigate)\b.{0,40}?\b(.+?)\s+to\s+(.+)$`)
	mapBareToRE = regexp.MustCompile(`(?i)^(.{3,160}?)\s+to\s+(.{3,160})$`)
	placeAdminRE = regexp.MustCompile(`(?i)\b(county|city|township|borough|parish|illinois|missouri|indiana|iowa|kentucky|tennessee|wisconsin|michigan|ohio|arkansas|texas|california|new\s+york|st\.?\s*louis|chicago)\b`)
	usStateCodeRE = regexp.MustCompile(`(?i)\b(A[LKSZRAEP]|C[AOT]|D[EC]|F[LM]|G[AU]|HI|I[ADLN]|K[SY]|LA|M[ADEHINOST]|N[CDEHJMVY]|O[HKR]|P[ARW]|RI|S[CD]|T[NX]|UT|V[AIT]|W[AIVY])\b`)
	nonPlaceToRE = regexp.MustCompile(`(?i)^(need|want|have|going|able|try|trying|used|ought|supposed)\s+to\b`)
)

// UserRequestsMapOrRoute detects geographic map / directions asks (not FLUX images).
func UserRequestsMapOrRoute(content string) bool {
	c := normalizeMapRequestText(strings.TrimSpace(content))
	if c == "" {
		return false
	}
	if mapFromToRE.MatchString(c) {
		return true
	}
	if strings.Contains(c, "canvas map") || strings.Contains(c, "neural canvas map") ||
		strings.Contains(c, "nj.map") || strings.Contains(c, "walking") || strings.Contains(c, "driving") {
		return true
	}
	if mapRouteNounRE.MatchString(c) && (mapRouteVerbRE.MatchString(c) || strings.Contains(c, " from ") || strings.Contains(c, " to ")) {
		return true
	}
	// Bare "Place A to Place B" (common after geocode display names are pasted back).
	if _, _, ok := ParseMapEndpoints(content); ok {
		return true
	}
	return false
}

func normalizeMapRequestText(content string) string {
	c := strings.ToLower(content)
	repl := strings.NewReplacer(
		"genereat", "generate",
		"genereate", "generate",
		"generete", "generate",
		"genrate", "generate",
		"cretae", "create",
	)
	return repl.Replace(c)
}

// ParseMapEndpoints extracts origin/destination place strings from common phrasings.
func ParseMapEndpoints(content string) (from, to string, ok bool) {
	c := strings.TrimSpace(content)
	for strings.HasPrefix(c, "@") {
		if i := strings.IndexByte(c, ' '); i > 0 {
			c = strings.TrimSpace(c[i+1:])
		} else {
			break
		}
	}
	if m := mapFromToRE.FindStringSubmatch(c); len(m) == 3 {
		from, to = cleanMapPlace(m[1]), cleanMapPlace(m[2])
		return from, to, from != "" && to != ""
	}
	if m := mapToRE.FindStringSubmatch(c); len(m) == 3 {
		from, to = cleanMapPlace(m[1]), cleanMapPlace(m[2])
		return from, to, from != "" && to != ""
	}
	if m := mapBareToRE.FindStringSubmatch(c); len(m) == 3 {
		from, to = cleanMapPlace(m[1]), cleanMapPlace(m[2])
		if from != "" && to != "" && looksLikePlace(from) && looksLikePlace(to) {
			return from, to, true
		}
	}
	return "", "", false
}

func looksLikePlace(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) < 2 || len(s) > 160 {
		return false
	}
	lower := strings.ToLower(s)
	if nonPlaceToRE.MatchString(lower) {
		return false
	}
	if strings.Contains(s, ",") {
		return true
	}
	if placeAdminRE.MatchString(s) || usStateCodeRE.MatchString(s) {
		return true
	}
	return false
}

func mapEndpointsLookGeographic(content string) bool {
	from, to, ok := ParseMapEndpoints(content)
	return ok && looksLikePlace(from) && looksLikePlace(to)
}

func cleanMapPlace(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"'`)
	lower := strings.ToLower(s)
	for _, prefix := range []string{
		"a map from ", "a map of ", "map from ", "map of ", "canvas map from ",
		"canvas map for ", "route from ", "directions from ", "driving from ",
		"walking from ", "me a map from ", "me from ", "for ",
	} {
		if strings.HasPrefix(lower, prefix) {
			s = strings.TrimSpace(s[len(prefix):])
			lower = strings.ToLower(s)
		}
	}
	for _, suffix := range []string{" please", " thanks", " thank you", "?", "!", "."} {
		if strings.HasSuffix(lower, suffix) {
			s = strings.TrimSpace(s[:len(s)-len(suffix)])
			lower = strings.ToLower(s)
		}
	}
	// Trim trailing sentence punctuation only (keep "St." in place names).
	s = strings.TrimRight(s, "?! ")
	return strings.TrimSpace(s)
}

func mapModeFromMessage(content string) string {
	c := strings.ToLower(content)
	if strings.Contains(c, "walk") || strings.Contains(c, "pedestrian") || strings.Contains(c, "foot") {
		return "walking"
	}
	if strings.Contains(c, "drive") || strings.Contains(c, "car") || strings.Contains(c, "highway") {
		return "driving"
	}
	// Intercity defaults to driving; short "walk" phrases already caught above.
	return "driving"
}

// tryMapsRouteShortcut geocodes A→B and publishes an nj.map artifact without waiting on the LLM tool loop.
func (a *Agent) tryMapsRouteShortcut(ctx context.Context, msg *protocol.Message) (string, bool) {
	if a == nil || msg == nil || a.Info.Type != protocol.AgentTypeMaps {
		return "", false
	}
	if !a.mapsToolsEnabledForMessage(msg) {
		return "", false
	}
	from, to, ok := ParseMapEndpoints(msg.Content)
	if !ok {
		return "", false
	}
	// Accept explicit map/route phrasing OR bare place→place (MapsExpert DMs often paste geocode labels).
	if !UserRequestsMapOrRoute(msg.Content) && !(looksLikePlace(from) && looksLikePlace(to)) {
		return "", false
	}
	client := mapssideecar.DefaultSidecarClient
	if client == nil {
		return "Maps sidecar is not running. Enable the Maps pack and restart the hub.", true
	}

	origin, err := geocodeFirstResult(ctx, client, from)
	if err != nil {
		return fmt.Sprintf("I couldn't geocode %q: %v", from, err), true
	}
	dest, err := geocodeFirstResult(ctx, client, to)
	if err != nil {
		return fmt.Sprintf("I couldn't geocode %q: %v", to, err), true
	}

	mode := mapModeFromMessage(msg.Content)
	title := fmt.Sprintf("%s to %s", displayName(origin, from), displayName(dest, to))
	args := map[string]any{
		"title": title,
		"mode":  mode,
		"waypoints": []any{
			map[string]any{"lat": origin["lat"], "lon": origin["lon"], "label": displayName(origin, from)},
			map[string]any{"lat": dest["lat"], "lon": dest["lon"], "label": displayName(dest, to)},
		},
	}
	raw, err := json.Marshal(args)
	if err != nil {
		return "", false
	}
	out, err := a.executeMapsCreateTool(ctx, msg, raw)
	if err != nil {
		return fmt.Sprintf("I couldn't create the canvas map: %v", err), true
	}

	summary := summarizeRouteFromCreateArgs(args, out)
	return summary, true
}

func geocodeFirstResult(ctx context.Context, client *mapssideecar.Client, query string) (map[string]any, error) {
	out, err := client.Geocode(ctx, map[string]any{"query": query, "limit": 1})
	if err != nil {
		return nil, err
	}
	results, _ := out["results"].([]any)
	if len(results) == 0 {
		return nil, fmt.Errorf("no results")
	}
	first, ok := results[0].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("unexpected geocode payload")
	}
	return first, nil
}

func displayName(place map[string]any, fallback string) string {
	if s, _ := place["display_name"].(string); strings.TrimSpace(s) != "" {
		// Prefer a shorter label for markers when Nominatim returns a long address.
		parts := strings.Split(s, ",")
		if len(parts) > 0 && strings.TrimSpace(parts[0]) != "" {
			label := strings.TrimSpace(parts[0])
			if len(parts) > 1 {
				label = label + "," + parts[1]
			}
			return strings.TrimSpace(label)
		}
		return s
	}
	return fallback
}

func summarizeRouteFromCreateArgs(args map[string]any, createResult string) string {
	mode, _ := args["mode"].(string)
	title, _ := args["title"].(string)
	var b strings.Builder
	b.WriteString("Posted an interactive Neural Canvas map")
	if strings.TrimSpace(title) != "" {
		b.WriteString(" for **")
		b.WriteString(strings.TrimSpace(title))
		b.WriteString("**")
	}
	if mode != "" {
		b.WriteString(" (")
		b.WriteString(mode)
		b.WriteString(")")
	}
	b.WriteString(". ")
	b.WriteString(createResult)
	b.WriteString(" Open it from the artifact card to pan/zoom the route.")
	return b.String()
}
