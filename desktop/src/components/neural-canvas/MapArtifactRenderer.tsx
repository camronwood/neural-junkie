import L from 'leaflet';
import markerIcon2x from 'leaflet/dist/images/marker-icon-2x.png';
import markerIcon from 'leaflet/dist/images/marker-icon.png';
import markerShadow from 'leaflet/dist/images/marker-shadow.png';
import 'leaflet/dist/leaflet.css';
import { useEffect, useMemo, useRef } from 'react';
import type { ArtifactRendererProps } from './types';

// Vite breaks Leaflet's default relative icon URLs — pin them explicitly.
delete (L.Icon.Default.prototype as unknown as { _getIconUrl?: unknown })._getIconUrl;
L.Icon.Default.mergeOptions({
  iconRetinaUrl: markerIcon2x,
  iconUrl: markerIcon,
  shadowUrl: markerShadow,
});

const DEFAULT_TILE = 'https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png';
const DEFAULT_ATTRIBUTION = '© OpenStreetMap contributors';

type LatLon = { lat: number; lon: number };

type MapMarker = LatLon & {
  id?: string;
  label?: string;
};

type MapRoute = {
  id?: string;
  mode?: string;
  distance_m?: number;
  duration_s?: number;
  geometry?: {
    type?: string;
    coordinates?: unknown;
  };
};

type MapPayload = {
  title?: string;
  center?: LatLon;
  zoom?: number;
  markers?: MapMarker[];
  routes?: MapRoute[];
  tile_url_template?: string;
  attribution?: string;
};

function asNumber(v: unknown): number | null {
  return typeof v === 'number' && Number.isFinite(v) ? v : null;
}

function asLatLon(v: unknown): LatLon | null {
  if (!v || typeof v !== 'object') return null;
  const o = v as Record<string, unknown>;
  const lat = asNumber(o.lat);
  const lon = asNumber(o.lon ?? o.lng);
  if (lat == null || lon == null) return null;
  return { lat, lon };
}

function lineCoords(geometry: MapRoute['geometry']): LatLon[] {
  if (!geometry || geometry.type !== 'LineString' || !Array.isArray(geometry.coordinates)) {
    return [];
  }
  const out: LatLon[] = [];
  for (const pair of geometry.coordinates) {
    if (!Array.isArray(pair) || pair.length < 2) continue;
    const lon = asNumber(pair[0]);
    const lat = asNumber(pair[1]);
    if (lat == null || lon == null) continue;
    out.push({ lat, lon });
  }
  return out;
}

function formatMeta(routes: MapRoute[]): string {
  const parts: string[] = [];
  for (const r of routes) {
    const bits: string[] = [];
    if (r.mode) bits.push(String(r.mode));
    if (typeof r.distance_m === 'number') {
      bits.push(r.distance_m >= 1000 ? `${(r.distance_m / 1000).toFixed(1)} km` : `${Math.round(r.distance_m)} m`);
    }
    if (typeof r.duration_s === 'number') {
      const mins = Math.round(r.duration_s / 60);
      bits.push(mins >= 60 ? `${Math.floor(mins / 60)}h ${mins % 60}m` : `${mins} min`);
    }
    if (bits.length) parts.push(bits.join(' · '));
  }
  return parts.join(' · ');
}

export function MapArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  const containerRef = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<L.Map | null>(null);

  const payload = useMemo((): MapPayload => {
    if (artifact.data && typeof artifact.data === 'object') {
      return artifact.data as MapPayload;
    }
    return {};
  }, [artifact.data]);

  const markers = useMemo(() => {
    const out: Array<LatLon & { id?: string; label?: string }> = [];
    for (const m of payload.markers ?? []) {
      const ll = asLatLon(m);
      if (!ll) continue;
      out.push({ ...ll, id: m.id, label: m.label });
    }
    return out;
  }, [payload.markers]);

  const routes = payload.routes ?? [];
  const center = asLatLon(payload.center) ?? markers[0] ?? { lat: 0, lon: 0 };
  const zoom = asNumber(payload.zoom) ?? 14;
  const tileUrl = String(payload.tile_url_template || DEFAULT_TILE).trim() || DEFAULT_TILE;
  const attribution = String(payload.attribution || DEFAULT_ATTRIBUTION);
  const meta = formatMeta(routes);

  useEffect(() => {
    const el = containerRef.current;
    if (!el) return;

    if (mapRef.current) {
      mapRef.current.remove();
      mapRef.current = null;
    }

    const map = L.map(el, {
      zoomControl: !compact,
      attributionControl: true,
    }).setView([center.lat, center.lon], zoom);

    L.tileLayer(tileUrl, { attribution, maxZoom: 19 }).addTo(map);

    const bounds: L.LatLngExpression[] = [];

    for (const m of markers) {
      const marker = L.marker([m.lat, m.lon]);
      if (m.label) marker.bindPopup(m.label);
      marker.addTo(map);
      bounds.push([m.lat, m.lon]);
    }

    for (const route of routes) {
      const pts = lineCoords(route.geometry);
      if (pts.length < 2) continue;
      const latlngs = pts.map((p) => [p.lat, p.lon] as L.LatLngExpression);
      L.polyline(latlngs, { color: '#2563eb', weight: 4, opacity: 0.85 }).addTo(map);
      bounds.push(...latlngs);
    }

    if (bounds.length >= 2) {
      map.fitBounds(L.latLngBounds(bounds), { padding: [24, 24] });
    } else if (bounds.length === 1) {
      map.setView(bounds[0], zoom);
    }

    mapRef.current = map;
    // Leaflet needs a tick after mount in flex layouts.
    requestAnimationFrame(() => map.invalidateSize());

    return () => {
      map.remove();
      mapRef.current = null;
    };
  }, [attribution, center.lat, center.lon, compact, markers, routes, tileUrl, zoom]);

  return (
    <div className={`flex flex-col ${compact ? 'h-48' : 'h-full min-h-0'}`}>
      {meta ? (
        <div className="shrink-0 border-b border-slate-700 px-3 py-1.5 text-xs text-slate-300">
          {meta}
        </div>
      ) : null}
      <div ref={containerRef} className="min-h-0 flex-1" />
    </div>
  );
}
