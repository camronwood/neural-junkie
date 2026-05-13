/** Keys must match internal/agent/prompt_context.go */
export const USER_RULES_METADATA_KEY = 'user_rules_markdown';
export const PROMPT_ATTACHMENTS_METADATA_KEY = 'prompt_attachments';

export interface PromptAttachmentPayload {
  path: string;
  language: string;
  content: string;
}
