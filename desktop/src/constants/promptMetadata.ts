/** Keys must match internal/agent/prompt_context.go and internal/protocol user image keys */
export const USER_RULES_METADATA_KEY = 'user_rules_markdown';
export const PROMPT_ATTACHMENTS_METADATA_KEY = 'prompt_attachments';
/** User-approved read of ~/.neural-junkie files/directories for this message only */
export const GRANTED_HUB_DATA_ACCESS_KEY = 'granted_hub_data_access';
/** Canonical multimodal image array on outbound messages */
export const USER_IMAGES_METADATA_KEY = 'user_images';
/** Tiered workspace attachment: none | hint | outline | focus | full */
export const CONTEXT_SCOPE_KEY = 'context_scope';
/** Dev UI: why Auto chose this scope */
export const CONTEXT_SCOPE_REASON_KEY = 'context_scope_reason';

export type ContextScope = 'none' | 'hint' | 'outline' | 'focus' | 'full';
export type WorkspaceContextMode = 'auto' | 'always' | 'off';

export interface PromptAttachmentPayload {
  path: string;
  language: string;
  content: string;
}

export interface UserImagePayload {
  mime: string;
  data: string;
}
