import type { Message } from '../types/protocol';
import {
  TELEMETRY_KIND_METADATA_KEY,
  TELEMETRY_PAYLOAD_METADATA_KEY,
  THINKING_ACTIVITY_DETAIL_KEY,
} from '../types/protocol';
import { useChatStore } from '../stores/chatStore';

/** Record agent_status telemetry for the turn debug drawer. */
export function appendTurnTelemetryFromAgentStatus(channel: string, message: Message): void {
  const meta = message.metadata;
  if (!meta) return;

  const telemetryKind =
    typeof meta[TELEMETRY_KIND_METADATA_KEY] === 'string'
      ? (meta[TELEMETRY_KIND_METADATA_KEY] as string)
      : undefined;
  const activity =
    typeof meta.thinking_activity === 'string' ? (meta.thinking_activity as string) : undefined;
  if (!telemetryKind && !activity) return;

  const activityDetail =
    typeof meta[THINKING_ACTIVITY_DETAIL_KEY] === 'string'
      ? (meta[THINKING_ACTIVITY_DETAIL_KEY] as string)
      : typeof meta.thinking_activity_detail === 'string'
        ? (meta.thinking_activity_detail as string)
        : '';

  const kind = telemetryKind ?? activity ?? 'activity';
  const detail = activityDetail || kind;
  const payloadRaw = meta[TELEMETRY_PAYLOAD_METADATA_KEY];
  const payload =
    payloadRaw && typeof payloadRaw === 'object' && !Array.isArray(payloadRaw)
      ? (payloadRaw as Record<string, unknown>)
      : undefined;

  useChatStore.getState().appendTurnTelemetryEvent(channel, {
    agentId: message.from.id,
    agentName: message.from.name,
    kind,
    detail,
    payload,
  });
}

/** Record stream_delta tool steps in the telemetry drawer. */
export function appendTurnTelemetryFromToolStep(channel: string, message: Message): void {
  const meta = message.metadata;
  if (!meta || typeof meta.tool_step !== 'string') return;
  const name = typeof meta.tool_name === 'string' ? meta.tool_name : 'tool';
  const preview = typeof meta.tool_preview === 'string' ? meta.tool_preview : '';
  useChatStore.getState().appendTurnTelemetryEvent(channel, {
    agentId: message.from.id,
    agentName: message.from.name,
    kind: 'tool',
    detail: preview || `${name} (${meta.tool_step})`,
    payload: {
      name,
      kind: meta.tool_step,
      preview,
    },
  });
}
