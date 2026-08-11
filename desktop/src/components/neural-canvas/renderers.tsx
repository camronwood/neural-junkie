import { Background, Controls, ReactFlow, type Edge, type Node } from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { useMemo } from 'react';
import { MermaidCanvas } from '../MermaidCanvas';
import { DocumentArtifactRenderer } from './DocumentArtifactRenderer';
import { MapArtifactRenderer } from './MapArtifactRenderer';
import {
  EmptyArtifact,
  ImageArtifactRenderer,
  TableArtifactRenderer,
  textFrom,
} from './artifactViews';
import type { ArtifactRendererProps, ArtifactRendererRegistration } from './types';
import {
  ArenaArtifactRenderer,
  CadArtifactRenderer,
  ComparatorArtifactRenderer,
  KnowledgeGraphArtifactRenderer,
  MusicArtifactRenderer,
  StructureArtifactRenderer,
} from './specializedRenderers';

export { EmptyArtifact, ImageArtifactRenderer, TableArtifactRenderer, textFrom } from './artifactViews';

export function MarkdownArtifactRenderer(props: ArtifactRendererProps) {
  return <DocumentArtifactRenderer {...props} />;
}

export function DocumentPageArtifactRenderer(props: ArtifactRendererProps) {
  return <DocumentArtifactRenderer {...props} />;
}

export function MermaidArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  return (
    <div className={compact ? 'h-48' : 'flex h-full min-h-0 flex-col'}>
      <MermaidCanvas
        content={textFrom(artifact.data)}
        active
        showZoomControls={!compact}
        className="h-full w-full"
      />
    </div>
  );
}

interface CodeData {
  code?: unknown;
  language?: unknown;
}

export function CodeArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  const value = artifact.data as CodeData;
  const code = typeof value === 'object' && value !== null && 'code' in value
    ? textFrom(value.code)
    : textFrom(artifact.data);
  const language = typeof value?.language === 'string' ? value.language : 'text';

  return (
    <div className={`overflow-auto rounded border border-slate-700 bg-slate-950 ${compact ? '' : 'm-4 h-[calc(100%-2rem)]'}`}>
      {!compact && (
        <div className="border-b border-slate-800 px-3 py-1 text-[11px] text-slate-400">
          {language}
        </div>
      )}
      <pre className={`${compact ? 'max-h-40 p-2 text-xs' : 'p-4 text-sm'} overflow-auto`}>
        <code>{code}</code>
      </pre>
    </div>
  );
}

interface ChartPoint {
  x: number;
  y: number;
}

interface ChartSeries {
  name: string;
  color: string;
  points: ChartPoint[];
}

function finiteNumber(value: unknown): number | null {
  const number = typeof value === 'number' ? value : Number(value);
  return Number.isFinite(number) ? number : null;
}

function normalizeChart(data: unknown): ChartSeries[] {
  if (!data || typeof data !== 'object') return [];
  const source = data as { series?: unknown; points?: unknown; values?: unknown };
  const rawSeries = Array.isArray(source.series)
    ? source.series
    : [{ name: 'Series', points: source.points, values: source.values }];
  const palette = ['#8b5cf6', '#22c55e', '#38bdf8', '#f59e0b', '#f43f5e'];

  return rawSeries.flatMap((entry, seriesIndex): ChartSeries[] => {
    if (!entry || typeof entry !== 'object') return [];
    const item = entry as { name?: unknown; color?: unknown; points?: unknown; values?: unknown };
    const rawPoints = Array.isArray(item.points)
      ? item.points
      : Array.isArray(item.values)
        ? item.values.map((y, x) => ({ x, y }))
        : [];
    const points = rawPoints.flatMap((point, pointIndex): ChartPoint[] => {
      if (typeof point === 'number') return [{ x: pointIndex, y: point }];
      if (!point || typeof point !== 'object') return [];
      const candidate = point as { x?: unknown; y?: unknown };
      const x = finiteNumber(candidate.x ?? pointIndex);
      const y = finiteNumber(candidate.y);
      return x === null || y === null ? [] : [{ x, y }];
    });
    return [{
      name: typeof item.name === 'string' ? item.name : `Series ${seriesIndex + 1}`,
      color: typeof item.color === 'string' ? item.color : palette[seriesIndex % palette.length],
      points,
    }];
  });
}

function BaseChartArtifactRenderer({
  artifact,
  kind,
}: ArtifactRendererProps & { kind: 'line' | 'bar' | 'scatter' }) {
  const series = normalizeChart(artifact.data);
  const points = series.flatMap((item) => item.points);
  if (!points.length) return <EmptyArtifact message="No chart data" />;
  const width = 720;
  const height = 360;
  const padding = 42;
  const xs = points.map((point) => point.x);
  const ys = points.map((point) => point.y);
  const minX = Math.min(...xs);
  const maxX = Math.max(...xs);
  const minY = Math.min(0, ...ys);
  const maxY = Math.max(...ys);
  const xRange = maxX - minX || 1;
  const yRange = maxY - minY || 1;
  const x = (value: number) => padding + ((value - minX) / xRange) * (width - padding * 2);
  const y = (value: number) => height - padding - ((value - minY) / yRange) * (height - padding * 2);
  const barCount = Math.max(1, points.length);
  const barWidth = Math.max(3, (width - padding * 2) / barCount * 0.7);

  return (
    <svg
      viewBox={`0 0 ${width} ${height}`}
      role="img"
      aria-label={`${kind} chart`}
      className="h-auto max-h-full w-full rounded bg-slate-950 text-slate-300"
    >
      <line x1={padding} y1={height - padding} x2={width - padding} y2={height - padding} stroke="currentColor" opacity=".5" />
      <line x1={padding} y1={padding} x2={padding} y2={height - padding} stroke="currentColor" opacity=".5" />
      {series.map((item) => kind === 'line' ? (
        <polyline
          key={item.name}
          points={item.points.map((point) => `${x(point.x)},${y(point.y)}`).join(' ')}
          fill="none"
          stroke={item.color}
          strokeWidth="3"
          strokeLinejoin="round"
        />
      ) : item.points.map((point, pointIndex) => kind === 'scatter' ? (
        <circle key={`${item.name}-${pointIndex}`} cx={x(point.x)} cy={y(point.y)} r="5" fill={item.color} />
      ) : (
        <rect
          key={`${item.name}-${pointIndex}`}
          x={x(point.x) - barWidth / 2}
          y={y(Math.max(0, point.y))}
          width={barWidth}
          height={Math.abs(y(point.y) - y(0))}
          rx="2"
          fill={item.color}
        />
      )))}
      <text x={padding} y={22} fill="currentColor" fontSize="12">
        {maxY.toLocaleString()}
      </text>
      <text x={padding} y={height - 10} fill="currentColor" fontSize="12">
        {minX.toLocaleString()} – {maxX.toLocaleString()}
      </text>
    </svg>
  );
}

export const LineChartArtifactRenderer = (props: ArtifactRendererProps) => <BaseChartArtifactRenderer {...props} kind="line" />;
export const BarChartArtifactRenderer = (props: ArtifactRendererProps) => <BaseChartArtifactRenderer {...props} kind="bar" />;
export const ScatterChartArtifactRenderer = (props: ArtifactRendererProps) => <BaseChartArtifactRenderer {...props} kind="scatter" />;

interface TimelineItem {
  id?: unknown;
  date?: unknown;
  title?: unknown;
  description?: unknown;
}

export function TimelineArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  const value = artifact.data as { items?: unknown };
  const items = Array.isArray(value?.items) ? value.items as TimelineItem[] : [];
  const visible = compact ? items.slice(0, 3) : items;
  if (!visible.length) return <EmptyArtifact message="No timeline entries" />;

  return (
    <ol className="space-y-0">
      {visible.map((item, index) => (
        <li key={typeof item.id === 'string' ? item.id : index} className="relative border-l border-violet-500/50 pb-5 pl-5 last:pb-0">
          <span className="absolute -left-1.5 top-1 h-3 w-3 rounded-full bg-violet-400" />
          <time className="text-xs text-slate-400">{textFrom(item.date ?? '')}</time>
          <h3 className="font-medium text-slate-100">{textFrom(item.title ?? 'Untitled event')}</h3>
          {item.description !== undefined && <p className="mt-1 text-sm text-slate-300">{textFrom(item.description)}</p>}
        </li>
      ))}
    </ol>
  );
}

interface GraphNodeData extends Record<string, unknown> {
  label: string;
}

export function GraphArtifactRenderer({ artifact, compact }: ArtifactRendererProps) {
  const graph = artifact.data as {
    nodes?: Array<{ id?: unknown; label?: unknown; x?: unknown; y?: unknown }>;
    edges?: Array<{ id?: unknown; source?: unknown; target?: unknown; label?: unknown }>;
  };
  const nodes = useMemo<Node<GraphNodeData>[]>(() => (Array.isArray(graph?.nodes) ? graph.nodes : [])
    .flatMap((item, index): Node<GraphNodeData>[] => {
      if (typeof item.id !== 'string') return [];
      return [{
        id: item.id,
        position: {
          x: finiteNumber(item.x) ?? (index % 5) * 180,
          y: finiteNumber(item.y) ?? Math.floor(index / 5) * 120,
        },
        data: { label: typeof item.label === 'string' ? item.label : item.id },
        style: { color: '#0f172a', borderColor: '#8b5cf6', background: '#f8fafc' },
      }];
    }), [graph?.nodes]);
  const nodeIds = useMemo(() => new Set(nodes.map((node) => node.id)), [nodes]);
  const edges = useMemo<Edge[]>(() => (Array.isArray(graph?.edges) ? graph.edges : [])
    .flatMap((item, index): Edge[] => {
      if (typeof item.source !== 'string' || typeof item.target !== 'string') return [];
      if (!nodeIds.has(item.source) || !nodeIds.has(item.target)) return [];
      return [{
        id: typeof item.id === 'string' ? item.id : `${item.source}-${item.target}-${index}`,
        source: item.source,
        target: item.target,
        label: typeof item.label === 'string' ? item.label : undefined,
      }];
    }), [graph?.edges, nodeIds]);
  if (!nodes.length) return <EmptyArtifact message="No graph nodes" />;

  return (
    <div className={compact ? 'h-48' : 'h-full min-h-0'}>
      <ReactFlow
        nodes={nodes}
        edges={edges}
        fitView
        nodesDraggable={!compact}
        nodesConnectable={false}
        elementsSelectable={!compact}
        proOptions={{ hideAttribution: true }}
      >
        <Background />
        {!compact && <Controls />}
      </ReactFlow>
    </div>
  );
}

export function ChartArtifactRenderer(props: ArtifactRendererProps) {
  const data = props.artifact.data as { chart_type?: unknown; type?: unknown } | null;
  const type = String(data?.chart_type ?? data?.type ?? 'line').toLowerCase();
  if (type === 'bar') return <BarChartArtifactRenderer {...props} />;
  if (type === 'scatter') return <ScatterChartArtifactRenderer {...props} />;
  return <LineChartArtifactRenderer {...props} />;
}

export const BUILT_IN_RENDERERS: readonly ArtifactRendererRegistration[] = [
  { id: 'nj.document', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.document+json'], component: DocumentPageArtifactRenderer },
  { id: 'nj.markdown', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['text/markdown'], component: MarkdownArtifactRenderer },
  { id: 'nj.mermaid', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['text/vnd.mermaid', 'application/vnd.mermaid'], component: MermaidArtifactRenderer },
  { id: 'nj.code', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['text/plain', 'application/vnd.neural-canvas.code+json', 'application/vnd.neural-junkie.code+json'], component: CodeArtifactRenderer },
  { id: 'nj.table', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-canvas.table+json', 'application/vnd.neural-junkie.table+json', 'text/csv'], component: TableArtifactRenderer },
  { id: 'nj.chart', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.chart+json', 'application/vnd.neural-canvas.line-chart+json', 'application/vnd.neural-canvas.bar-chart+json', 'application/vnd.neural-canvas.scatter-chart+json'], component: ChartArtifactRenderer },
  { id: 'nj.timeline', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-canvas.timeline+json', 'application/vnd.neural-junkie.timeline+json'], component: TimelineArtifactRenderer },
  { id: 'nj.image', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['image/png', 'image/jpeg', 'image/gif', 'image/webp', 'image/svg+xml', 'application/vnd.neural-canvas.image+json', 'application/vnd.neural-junkie.image+json'], component: ImageArtifactRenderer },
  { id: 'nj.graph', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-canvas.graph+json', 'application/vnd.neural-junkie.graph+json'], component: GraphArtifactRenderer },
  { id: 'nj.map', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.map+json'], component: MapArtifactRenderer },
  { id: 'nj.knowledge-graph', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.knowledge-graph+json'], component: KnowledgeGraphArtifactRenderer },
  { id: 'nj.cad', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.cad+json', 'model/stl'], component: CadArtifactRenderer },
  { id: 'nj.structure', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['chemical/x-pdb', 'chemical/x-mmcif'], component: StructureArtifactRenderer },
  { id: 'nj.music', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.music+json', 'audio/wav', 'audio/mpeg'], component: MusicArtifactRenderer },
  { id: 'nj.arena', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.arena+json'], component: ArenaArtifactRenderer },
  { id: 'nj.comparator-analysis', apiVersions: ['1', '1.0', 'v1'], mediaTypes: ['application/vnd.neural-junkie.comparator+json'], component: ComparatorArtifactRenderer },
];
