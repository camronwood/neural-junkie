import { ArenaWorkbench } from '../ArenaWorkbench';
import { CadWorkbench } from '../CadWorkbench';
import { ComparatorAnalysisViewer } from '../ComparatorAnalysisViewer';
import { MusicWorkbench } from '../MusicWorkbench';
import { StructureWorkbench } from '../StructureWorkbench';
import { KnowledgeGraphWorkbench } from '../knowledge-graph/KnowledgeGraphWorkbench';
import type { ArtifactRendererProps } from './types';

type WorkbenchPayload = {
  workspace_id?: string;
  path?: string;
  content?: string;
  project_id?: string;
  repo_path?: string;
  analysis_dir?: string;
  audio_path?: string;
  project_path?: string;
  session_path?: string;
};

function payloadFrom(props: ArtifactRendererProps): WorkbenchPayload {
  return props.artifact.data && typeof props.artifact.data === 'object'
    ? props.artifact.data as WorkbenchPayload
    : {};
}

function MissingWorkbenchData({ label }: { label: string }) {
  return <p className="p-4 text-sm text-slack-textMuted">{label} artifact is missing its workspace or source path.</p>;
}

export function KnowledgeGraphArtifactRenderer(props: ArtifactRendererProps) {
  const data = payloadFrom(props);
  if (!data.workspace_id || !data.repo_path) return <MissingWorkbenchData label="Knowledge graph" />;
  return <KnowledgeGraphWorkbench workspaceId={data.workspace_id} repoPath={data.repo_path} />;
}

export function CadArtifactRenderer(props: ArtifactRendererProps) {
  const data = payloadFrom(props);
  if (!data.workspace_id || !data.path) return <MissingWorkbenchData label="CAD" />;
  return (
    <CadWorkbench
      workspaceId={data.workspace_id}
      scadPath={data.path}
      initialContent={data.content}
      projectId={data.project_id}
      tabId={`canvas-${props.artifact.id}`}
    />
  );
}

export function StructureArtifactRenderer(props: ArtifactRendererProps) {
  const data = payloadFrom(props);
  if (!data.workspace_id || !data.path) return <MissingWorkbenchData label="Structure" />;
  return (
    <StructureWorkbench
      workspaceId={data.workspace_id}
      structurePath={data.path}
      initialContent={data.content}
      tabId={`canvas-${props.artifact.id}`}
    />
  );
}

export function MusicArtifactRenderer(props: ArtifactRendererProps) {
  const data = payloadFrom(props);
  if (!data.workspace_id || (!data.audio_path && !data.project_path && !data.path)) {
    return <MissingWorkbenchData label="Music" />;
  }
  return (
    <MusicWorkbench
      workspaceId={data.workspace_id}
      audioPath={data.audio_path ?? (!data.project_path ? data.path : undefined)}
      projectPath={data.project_path}
      tabId={`canvas-${props.artifact.id}`}
    />
  );
}

export function ArenaArtifactRenderer(props: ArtifactRendererProps) {
  const data = payloadFrom(props);
  if (!data.workspace_id) return <MissingWorkbenchData label="Model Arena" />;
  return (
    <ArenaWorkbench
      workspaceId={data.workspace_id}
      sessionPath={data.session_path ?? data.path}
      tabId={`canvas-${props.artifact.id}`}
    />
  );
}

export function ComparatorArtifactRenderer(props: ArtifactRendererProps) {
  const data = payloadFrom(props);
  if (!data.workspace_id || !data.analysis_dir) return <MissingWorkbenchData label="Comparator analysis" />;
  return <ComparatorAnalysisViewer workspaceId={data.workspace_id} analysisDir={data.analysis_dir} />;
}
