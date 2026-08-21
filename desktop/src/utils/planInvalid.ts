export function isPlanFormatInvalid(metadata?: Record<string, unknown> | null): boolean {
  return metadata?.plan_format_invalid === true;
}
