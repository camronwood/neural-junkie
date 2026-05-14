/** Keys must match internal/agent/prompt_context.go and internal/protocol user image keys */
export const USER_RULES_METADATA_KEY = 'user_rules_markdown';
export const PROMPT_ATTACHMENTS_METADATA_KEY = 'prompt_attachments';
/** Canonical multimodal image array on outbound messages */
export const USER_IMAGES_METADATA_KEY = 'user_images';

export interface PromptAttachmentPayload {
  path: string;
  language: string;
  content: string;
}

export interface UserImagePayload {
  mime: string;
  data: string;
}
