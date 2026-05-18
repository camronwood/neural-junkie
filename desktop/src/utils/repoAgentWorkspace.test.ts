import { describe, expect, it } from 'vitest';
import {
  normalizeWorkspacePath,
  parseCreateRepoAgentCommand,
  isRepoAgentWorkspaceAction,
  REPO_AGENT_WORKSPACE_ACTION,
} from './repoAgentWorkspace';

describe('normalizeWorkspacePath', () => {
  it('strips trailing slashes and normalizes separators', () => {
    expect(normalizeWorkspacePath('/Users/me/proj/')).toBe('/Users/me/proj');
    expect(normalizeWorkspacePath('C:\\dev\\app\\')).toBe('C:/dev/app');
  });
});

describe('parseCreateRepoAgentCommand', () => {
  it('parses path and agent name', () => {
    expect(parseCreateRepoAgentCommand('/create-repo-agent /tmp/my-app MyExpert')).toEqual({
      repoPath: '/tmp/my-app',
      agentName: 'MyExpert',
    });
  });

  it('parses path when next token is provider', () => {
    expect(parseCreateRepoAgentCommand('/create-repo-agent /tmp/my-app ollama llama3.1')).toEqual({
      repoPath: '/tmp/my-app',
      agentName: undefined,
    });
  });
});

describe('isRepoAgentWorkspaceAction', () => {
  it('accepts valid client action metadata', () => {
    expect(
      isRepoAgentWorkspaceAction({
        type: REPO_AGENT_WORKSPACE_ACTION,
        path: '/tmp/repo',
        name: 'RepoExpert',
      })
    ).toBe(true);
  });
});
