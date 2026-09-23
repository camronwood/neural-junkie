/** Form fields used when adding a connector in Settings → Connectors. */
export interface ConnectorFormFields {
  type: string;
  brokerUrl: string;
  clientId: string;
  username: string;
  brokers: string;
  groupId: string;
  webhookUrl: string;
  /** SMTP host when type is email. */
  smtpHost: string;
  smtpPort: string;
  smtpFrom: string;
  smsUrl: string;
  smsFrom: string;
  smsFormat: string;
  secret: string;
}

export function buildEmailConfig(
  fields: Pick<ConnectorFormFields, 'smtpHost' | 'smtpPort' | 'username' | 'smtpFrom'>
): Record<string, string> {
  const cfg: Record<string, string> = {};
  const host = fields.smtpHost.trim();
  if (host) cfg.host = host;
  const port = fields.smtpPort.trim() || '587';
  if (host) cfg.port = port;
  if (fields.username.trim()) cfg.username = fields.username.trim();
  if (fields.smtpFrom.trim()) cfg.from = fields.smtpFrom.trim();
  return cfg;
}

export function buildSmsConfig(
  fields: Pick<ConnectorFormFields, 'smsUrl' | 'smsFrom' | 'smsFormat'>
): Record<string, string> {
  const cfg: Record<string, string> = {};
  if (fields.smsUrl.trim()) cfg.url = fields.smsUrl.trim();
  if (fields.smsFrom.trim()) cfg.from = fields.smsFrom.trim();
  if (fields.smsFormat.trim()) cfg.format = fields.smsFormat.trim();
  return cfg;
}

export function buildConnectorConfig(fields: ConnectorFormFields): Record<string, string> {
  if (fields.type === 'mqtt') {
    const cfg: Record<string, string> = {};
    if (fields.brokerUrl.trim()) cfg.broker_url = fields.brokerUrl.trim();
    if (fields.clientId.trim()) cfg.client_id = fields.clientId.trim();
    if (fields.username.trim()) cfg.username = fields.username.trim();
    return cfg;
  }
  if (fields.type === 'kafka') {
    const cfg: Record<string, string> = {};
    if (fields.brokers.trim()) cfg.brokers = fields.brokers.trim();
    if (fields.groupId.trim()) cfg.group_id = fields.groupId.trim();
    if (fields.username.trim()) cfg.username = fields.username.trim();
    return cfg;
  }
  if (fields.type === 'email') {
    return buildEmailConfig(fields);
  }
  if (fields.type === 'sms') {
    return buildSmsConfig(fields);
  }
  if ((fields.type === 'webhook' || fields.type === 'http_auth') && fields.webhookUrl.trim()) {
    return { url: fields.webhookUrl.trim() };
  }
  return {};
}

/** Returns an error message when the form cannot be saved; empty string when valid. */
export function validateConnectorForm(fields: ConnectorFormFields): string {
  if (fields.type === 'email') {
    if (!fields.smtpHost.trim()) return 'SMTP host is required';
    if (!fields.secret.trim()) return 'SMTP password is required';
    return '';
  }
  if (fields.type === 'sms') {
    if (!fields.smsUrl.trim()) return 'SMS gateway URL is required';
    return '';
  }
  return '';
}

export function connectorListDetail(
  config: Record<string, string> | undefined,
  type: string
): string | null {
  if (!config) return null;
  if (type === 'email' && config.host) {
    const port = config.port || '587';
    return `${config.host}:${port}`;
  }
  if ((type === 'sms' || type === 'webhook' || type === 'http_auth') && config.url) {
    return config.url;
  }
  if (config.broker_url) return config.broker_url;
  if (config.brokers) return config.brokers;
  return null;
}
