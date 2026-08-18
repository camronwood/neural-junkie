import { describe, expect, it, vi } from 'vitest';
import { applyContextRequestToMetadata, loadRequestedDocumentBodies } from './contextRequestAttach';
import type { WorkspaceContext } from './workspaceContext';

describe('applyContextRequestToMetadata', () => {
  const full: WorkspaceContext = {
    workspace_id: 'ciso',
    workspace_name: 'CISO',
    workspace_path: '/Users/me/CISO',
    file_tree: 'internal/\n  a.md\ngilead-security/\n  b.md\n',
    open_files: [
      {
        path: '/Users/me/CISO/open.md',
        language: 'markdown',
        content: '# Open',
        is_active: true,
      },
    ],
  };

  it('stamps prepare_token and upgrades scope from context_request', async () => {
    const api = {
      fetchFileContent: vi.fn(),
    } as any;
    const meta = await applyContextRequestToMetadata({
      api,
      metadata: { context_scope: 'hint' },
      message: 'please reivew the documents in the workspace',
      contextRequest: {
        context_tier: 'outline',
        subject: 'workspace_documents',
        review_mode: 'workspace',
        include_workspace_identity: true,
        include_file_tree: true,
        include_document_bodies: false,
      },
      prepareToken: 'abc123',
      fullWorkspace: full,
    });
    expect(meta.prepare_token).toBe('abc123');
    expect(meta.context_scope).toBe('outline');
    expect((meta.workspace_context as WorkspaceContext).file_tree).toContain('internal/');
  });
});

describe('loadRequestedDocumentBodies', () => {
  it('prioritizes markdown under internal/ and gilead-security/', async () => {
    const api = {
      fetchFileContent: vi.fn(async (_ws: string, path: string) => `# ${path}`),
    } as any;
    const docs = await loadRequestedDocumentBodies(api, 'ciso', [
      {
        name: 'internal',
        is_dir: true,
        size: 0,
        mod_time: '',
        path: 'internal',
        children: [
          { name: 'plan.md', is_dir: false, size: 10, mod_time: '', path: 'internal/plan.md' },
        ],
      },
      {
        name: 'src',
        is_dir: true,
        size: 0,
        mod_time: '',
        path: 'src',
        children: [
          { name: 'index.js', is_dir: false, size: 10, mod_time: '', path: 'src/index.js' },
        ],
      },
    ]);
    expect(docs.map((d) => d.path)).toEqual(['internal/plan.md']);
    expect(docs[0].content).toContain('internal/plan.md');
  });
});
