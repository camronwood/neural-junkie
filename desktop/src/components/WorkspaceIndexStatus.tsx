import { useEffect, useState } from 'react';
import { ChatAPI } from '../api/chatAPI';

export function useWorkspaceIndexLabel(repoPath: string | undefined): string {
  const [label, setLabel] = useState('');

  useEffect(() => {
    if (!repoPath) {
      setLabel('');
      return;
    }
    let cancelled = false;
    const api = new ChatAPI();
    const poll = async () => {
      try {
        const meta = await api.repoIndexStatus(repoPath);
        if (cancelled) return;
        if (meta.ready) {
          setLabel('Index ready');
        } else if (meta.building) {
          setLabel('Indexing…');
        } else {
          setLabel('Index pending');
        }
      } catch {
        if (!cancelled) setLabel('');
      }
    };
    void poll();
    const id = window.setInterval(poll, 15000);
    return () => {
      cancelled = true;
      window.clearInterval(id);
    };
  }, [repoPath]);

  return label;
}

interface WorkspaceIndexStatusProps {
  repoPath: string | undefined;
}

export function WorkspaceIndexStatus({ repoPath }: WorkspaceIndexStatusProps) {
  const label = useWorkspaceIndexLabel(repoPath);
  if (!label) return null;

  return (
    <span
      className="px-2 py-0.5 text-[10px] uppercase tracking-wide rounded bg-slack-bgHover text-slack-textMuted whitespace-nowrap flex-shrink-0"
      title="Semantic codebase index for @codebase and specialist consult"
    >
      {label}
    </span>
  );
}
