import assert from 'node:assert/strict';
import test from 'node:test';
import {
  BOT_CALLBACK_PATH,
  OAUTH_STATE_PREFIX,
  buildRelayRedirectURL,
  isAllowedLocalOAuthCallback,
  parseOAuthState,
} from '../src/relay.ts';

function formatState(nonce: string, local: string): string {
  const enc = Buffer.from(local, 'utf8').toString('base64url');
  return `${OAUTH_STATE_PREFIX}${nonce}.${enc}`;
}

test('parseOAuthState round trip', () => {
  const local = 'http://127.0.0.1:18765/api/slack/oauth/callback';
  const state = formatState('abc123', local);
  const parsed = parseOAuthState(state);
  assert.equal(parsed.ok, true);
  if (!parsed.ok || 'legacy' in parsed) {
    assert.fail('expected relay state');
  }
  assert.equal(parsed.nonce, 'abc123');
  assert.equal(parsed.localReturn, local);
});

test('parseOAuthState legacy hex', () => {
  const legacy = 'deadbeefcafebabe0123456789abcdef';
  const parsed = parseOAuthState(legacy);
  assert.equal(parsed.ok, true);
  if (!parsed.ok) assert.fail();
  assert.equal(parsed.nonce, legacy);
  assert.equal(parsed.localReturn, '');
  assert.equal('legacy' in parsed && parsed.legacy, true);
});

test('isAllowedLocalOAuthCallback allows loopback paths', () => {
  assert.equal(
    isAllowedLocalOAuthCallback('http://localhost:18765/api/slack/oauth/callback'),
    true,
  );
  assert.equal(
    isAllowedLocalOAuthCallback('http://127.0.0.1:18766/api/slack/oauth/user-dm/callback'),
    true,
  );
  assert.equal(isAllowedLocalOAuthCallback('https://127.0.0.1:18765/api/slack/oauth/callback'), false);
  assert.equal(isAllowedLocalOAuthCallback('http://evil.example/api/slack/oauth/callback'), false);
  assert.equal(isAllowedLocalOAuthCallback('http://127.0.0.1:18765/api/other'), false);
});

test('buildRelayRedirectURL preserves query', () => {
  const local = `http://127.0.0.1:18765${BOT_CALLBACK_PATH}`;
  const params = new URLSearchParams({ code: 'test-code', state: 'nj1.x.y' });
  const got = buildRelayRedirectURL(local, params);
  assert.ok(got);
  const u = new URL(got!);
  assert.equal(u.searchParams.get('code'), 'test-code');
  assert.equal(u.searchParams.get('state'), 'nj1.x.y');
});
