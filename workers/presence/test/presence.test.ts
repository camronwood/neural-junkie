import assert from 'node:assert/strict';
import test from 'node:test';
import {
  authorizeStats,
  countActiveInWindow,
  dayOffsetUtc,
  parsePingBody,
  parseStatsCache,
  parseUniqueCount,
  utcDayString,
} from '../src/presence.ts';

test('parsePingBody accepts only id uuid', () => {
  const ok = parsePingBody({ id: '550e8400-e29b-41d4-a716-446655440000' });
  assert.equal(ok.ok, true);
  if (ok.ok) assert.equal(ok.id, '550e8400-e29b-41d4-a716-446655440000');
});

test('parsePingBody rejects extra keys', () => {
  const bad = parsePingBody({
    id: '550e8400-e29b-41d4-a716-446655440000',
    os: 'mac',
  });
  assert.equal(bad.ok, false);
});

test('parsePingBody rejects invalid uuid', () => {
  assert.equal(parsePingBody({ id: 'not-a-uuid' }).ok, false);
  assert.equal(parsePingBody({ id: 123 }).ok, false);
  assert.equal(parsePingBody(null).ok, false);
  assert.equal(parsePingBody([]).ok, false);
});

test('countActiveInWindow uses inclusive 30-day window', () => {
  const today = '2026-11-16';
  const cutoff = dayOffsetUtc(today, 29);
  assert.equal(cutoff, '2026-10-18');
  const days = [today, cutoff, '2026-10-17', '2026-11-01'];
  assert.equal(countActiveInWindow(days, today), 3);
});

test('parseUniqueCount and stats cache ttl', () => {
  assert.equal(parseUniqueCount(null), 0);
  assert.equal(parseUniqueCount('12'), 12);
  assert.equal(parseUniqueCount('x'), 0);

  const now = Date.parse('2026-11-16T12:00:00.000Z');
  const fresh = JSON.stringify({
    unique_installs: 5,
    active_30d: 3,
    computed_at: '2026-11-16T11:30:00.000Z',
  });
  assert.deepEqual(parseStatsCache(fresh, now), { unique_installs: 5, active_30d: 3 });

  const stale = JSON.stringify({
    unique_installs: 5,
    active_30d: 3,
    computed_at: '2026-11-16T10:00:00.000Z',
  });
  assert.equal(parseStatsCache(stale, now), null);
});

test('authorizeStats bearer and query token', () => {
  const token = 'secret-token';
  const bearer = new Request('https://example/v1/stats', {
    headers: { authorization: 'Bearer secret-token' },
  });
  assert.equal(authorizeStats(bearer, token), true);
  assert.equal(authorizeStats(bearer, undefined), false);

  const query = new Request('https://example/v1/stats?token=secret-token');
  assert.equal(authorizeStats(query, token), true);

  const bad = new Request('https://example/v1/stats?token=wrong');
  assert.equal(authorizeStats(bad, token), false);
});

test('utcDayString is YYYY-MM-DD', () => {
  assert.match(utcDayString(new Date('2026-11-16T23:59:59.000Z')), /^\d{4}-\d{2}-\d{2}$/);
  assert.equal(utcDayString(new Date('2026-11-16T23:59:59.000Z')), '2026-11-16');
});
