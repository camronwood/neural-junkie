/** Friendly labels for common Ollama / hub model tags (Agent Info, Settings hints). */

export type ModelDisplayRole = 'chat' | 'tool' | 'primary';

export function formatModelDisplayName(tag: string): string {
  const t = tag.trim().toLowerCase();
  if (!t) return 'Unknown model';

  if (t.includes('openbiollm') || t.includes('llama3-openbio')) {
    return 'OpenBioLLM 8B';
  }
  if (t === 'nj-bio:8b' || t.startsWith('nj-bio:')) {
    return 'OpenBio 8B (GGUF)';
  }
  if (t.startsWith('nj-biology:')) {
    return 'Biology LoRA 8B';
  }
  if (t.includes('qwen2.5-coder:14b')) {
    return 'Qwen 2.5 Coder 14B';
  }
  if (t.includes('qwen2.5-coder:7b')) {
    return 'Qwen 2.5 Coder 7B';
  }
  if (t === 'qwen2.5:7b' || t.startsWith('qwen2.5:7b')) {
    return 'Qwen 2.5 7B';
  }
  if (t.includes('qwen2.5:14b')) {
    return 'Qwen 2.5 14B';
  }
  if (t.startsWith('nj-cad:')) {
    return 'CAD LoRA 27B';
  }
  if (t.startsWith('nj-security:')) {
    return 'Security LoRA';
  }
  if (t.startsWith('nj-backend:') || t.startsWith('nj-frontend:')) {
    return 'Dev specialist LoRA';
  }
  if (t.includes('claude-sonnet')) {
    return 'Claude Sonnet';
  }
  if (t.includes('claude-haiku')) {
    return 'Claude Haiku';
  }

  return tag.trim();
}

export function formatModelWithRole(tag: string, role: ModelDisplayRole): string {
  const name = formatModelDisplayName(tag);
  if (role === 'chat') return `${name} (chat)`;
  if (role === 'tool') return `${name} (tools)`;
  return name;
}
