import { describe, expect, it } from 'vitest';
import {
  buildConnectorConfig,
  buildEmailConfig,
  buildSmsConfig,
  connectorListDetail,
  validateConnectorForm,
  type ConnectorFormFields,
} from './connectorFormUtils';

function base(overrides: Partial<ConnectorFormFields> = {}): ConnectorFormFields {
  return {
    type: 'webhook',
    brokerUrl: '',
    clientId: '',
    username: '',
    brokers: '',
    groupId: '',
    webhookUrl: '',
    smtpHost: '',
    smtpPort: '587',
    smtpFrom: '',
    smsUrl: '',
    smsFrom: '',
    smsFormat: '',
    secret: '',
    ...overrides,
  };
}

describe('buildEmailConfig', () => {
  it('maps host, port, username, from', () => {
    expect(
      buildEmailConfig({
        smtpHost: ' smtp.example.com ',
        smtpPort: '465',
        username: 'nj@example.com',
        smtpFrom: 'alerts@example.com',
      })
    ).toEqual({
      host: 'smtp.example.com',
      port: '465',
      username: 'nj@example.com',
      from: 'alerts@example.com',
    });
  });

  it('defaults port to 587 when host set and port empty', () => {
    expect(
      buildEmailConfig({
        smtpHost: 'mail.local',
        smtpPort: '  ',
        username: '',
        smtpFrom: '',
      })
    ).toEqual({ host: 'mail.local', port: '587' });
  });

  it('omits host/port when host missing (validation blocks save)', () => {
    expect(
      buildEmailConfig({ smtpHost: '', smtpPort: '587', username: 'u', smtpFrom: 'f' })
    ).toEqual({ username: 'u', from: 'f' });
  });
});

describe('buildSmsConfig', () => {
  it('maps url, from, format', () => {
    expect(
      buildSmsConfig({
        smsUrl: ' https://sms.example/send ',
        smsFrom: '+15551212',
        smsFormat: 'json',
      })
    ).toEqual({
      url: 'https://sms.example/send',
      from: '+15551212',
      format: 'json',
    });
  });

  it('omits url when missing (validation blocks save)', () => {
    expect(buildSmsConfig({ smsUrl: '', smsFrom: 'x', smsFormat: 'form' })).toEqual({
      from: 'x',
      format: 'form',
    });
  });
});

describe('buildConnectorConfig type isolation', () => {
  it('email does not include sms url fields', () => {
    const cfg = buildConnectorConfig(
      base({
        type: 'email',
        smtpHost: 'smtp.example.com',
        username: 'u',
        smsUrl: 'https://should-not-appear',
        smsFrom: 'ignored',
      })
    );
    expect(cfg).toEqual({ host: 'smtp.example.com', port: '587', username: 'u' });
    expect(cfg).not.toHaveProperty('url');
  });

  it('sms does not include smtp host fields', () => {
    const cfg = buildConnectorConfig(
      base({
        type: 'sms',
        smsUrl: 'https://gateway/sms',
        smtpHost: 'smtp.should-not-appear',
        username: 'smtp-user',
      })
    );
    expect(cfg).toEqual({ url: 'https://gateway/sms' });
    expect(cfg).not.toHaveProperty('host');
  });
});

describe('validateConnectorForm', () => {
  it('requires SMTP host and password for email', () => {
    expect(validateConnectorForm(base({ type: 'email', secret: 'pw' }))).toBe(
      'SMTP host is required'
    );
    expect(
      validateConnectorForm(base({ type: 'email', smtpHost: 'smtp.example.com', secret: '' }))
    ).toBe('SMTP password is required');
    expect(
      validateConnectorForm(
        base({ type: 'email', smtpHost: 'smtp.example.com', secret: 'pw' })
      )
    ).toBe('');
  });

  it('requires SMS URL', () => {
    expect(validateConnectorForm(base({ type: 'sms' }))).toBe('SMS gateway URL is required');
    expect(
      validateConnectorForm(base({ type: 'sms', smsUrl: 'https://gateway/sms' }))
    ).toBe('');
  });

  it('allows webhook without url', () => {
    expect(validateConnectorForm(base({ type: 'webhook' }))).toBe('');
  });
});

describe('connectorListDetail', () => {
  it('formats email host:port', () => {
    expect(connectorListDetail({ host: 'smtp.example.com', port: '465' }, 'email')).toBe(
      'smtp.example.com:465'
    );
    expect(connectorListDetail({ host: 'smtp.example.com' }, 'email')).toBe(
      'smtp.example.com:587'
    );
  });

  it('shows sms/webhook url', () => {
    expect(connectorListDetail({ url: 'https://x' }, 'sms')).toBe('https://x');
    expect(connectorListDetail({ url: 'https://y' }, 'webhook')).toBe('https://y');
  });
});
