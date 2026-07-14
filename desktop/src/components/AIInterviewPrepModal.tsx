import { useCallback, useEffect, useState } from 'react';
import { ChatAPI, type AIInterviewProgressResponse } from '../api/chatAPI';
import { getHubBaseURL } from '../config/hubUrl';
import { usePacksStore } from '../stores/packsStore';

const api = new ChatAPI(getHubBaseURL());

interface AIInterviewPrepModalProps {
  isOpen: boolean;
  onClose: () => void;
}

export function AIInterviewPrepModal({ isOpen, onClose }: AIInterviewPrepModalProps) {
  const hasLauncher = usePacksStore(
    (s) =>
      s.hasCapability('ai-interview-launcher') || s.hasCapability('ai-interview-sidecar'),
  );

  const [data, setData] = useState<AIInterviewProgressResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [evalNotes, setEvalNotes] = useState('');
  const [mockNotes, setMockNotes] = useState('');

  const loadProgress = useCallback(async () => {
    if (!hasLauncher) return;
    setLoading(true);
    setError(null);
    try {
      setData(await api.fetchAIInterviewProgress());
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Failed to load progress';
      setError(msg);
      setData(null);
    } finally {
      setLoading(false);
    }
  }, [hasLauncher]);

  useEffect(() => {
    if (!isOpen) return;
    void loadProgress();
  }, [isOpen, loadProgress]);

  const runAction = async (fn: () => Promise<AIInterviewProgressResponse | unknown>) => {
    setBusy(true);
    setError(null);
    try {
      const result = await fn();
      if (result && typeof result === 'object' && 'progress' in result) {
        setData(result as AIInterviewProgressResponse);
      } else {
        await loadProgress();
      }
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Action failed');
    } finally {
      setBusy(false);
    }
  };

  if (!isOpen) return null;

  const progress = data?.progress;
  const today = data?.today;
  const day = today?.day ?? progress?.current_day ?? 1;
  const phase = progress?.phase ?? today?.phase ?? 1;
  const dayStatus = today?.day_status;
  const gates = progress?.gates ?? {};
  const cert = progress?.certification;

  // Prefer current phase gate; sidecar rejects if phase days are incomplete.
  const submitGateId = String(Math.min(Math.max(phase, 1), 3));

  return (
    <div
      className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4"
      role="presentation"
    >
      <div className="fixed inset-0 bg-black/60" onClick={onClose} aria-hidden />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="ai-interview-title"
        className="relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-slack-border bg-slack-bg shadow-2xl max-h-[min(90vh,760px)]"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-4 py-3">
          <div>
            <h2 id="ai-interview-title" className="text-lg font-semibold text-slack-text">
              AI Interview Prep
            </h2>
            <p className="text-xs text-gray-500 mt-0.5">90-day Applied AI Engineer track</p>
          </div>
          <button
            type="button"
            onClick={onClose}
            className="text-slack-textMuted hover:text-slack-text px-2 py-1 rounded hover:bg-slack-bgHover"
            aria-label="Close"
          >
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {!hasLauncher && (
            <p className="text-sm text-amber-400">
              Enable the <strong>AI Interview Prep</strong> pack in Settings → Domain packs.
            </p>
          )}

          {hasLauncher && (
            <>
              {loading && <p className="text-sm text-gray-400">Loading progress…</p>}
              {error && (
                <p className="text-sm text-amber-400 whitespace-pre-wrap">
                  {/403|forbidden|requires the ai interview/i.test(error)
                    ? 'Enable the AI Interview Prep pack in Settings → Domain packs.'
                    : error}
                </p>
              )}

              {data && (
                <>
                  <div className="grid grid-cols-2 sm:grid-cols-4 gap-3 text-sm">
                    <div>
                      <div className="text-xs text-gray-500">Day</div>
                      <div className="text-slack-text font-medium">{day}</div>
                    </div>
                    <div>
                      <div className="text-xs text-gray-500">Phase</div>
                      <div className="text-slack-text font-medium">{phase}</div>
                    </div>
                    <div>
                      <div className="text-xs text-gray-500">Streak</div>
                      <div className="text-slack-text font-medium">
                        {data.stats?.streak_days ?? progress?.streak_days ?? 0}
                      </div>
                    </div>
                    <div>
                      <div className="text-xs text-gray-500">Completed</div>
                      <div className="text-slack-text font-medium">
                        {data.stats?.completed_days ?? 0}/{data.stats?.total_days ?? 90}
                      </div>
                    </div>
                  </div>

                  <div className="rounded-lg border border-slack-border p-3 space-y-1">
                    <div className="text-sm font-medium text-slack-text">
                      Today · {today?.title ?? `Day ${day}`}
                    </div>
                    <p className="text-xs text-gray-400">{today?.summary ?? ''}</p>
                    <label className="flex items-center gap-2 text-sm text-gray-300 mt-2">
                      <input type="checkbox" checked={!!dayStatus?.concept} readOnly />
                      Concept done
                    </label>
                    <label className="flex items-center gap-2 text-sm text-gray-300">
                      <input type="checkbox" checked={!!dayStatus?.drill} readOnly />
                      Drill passed
                      {today?.has_drill === false ? ' (n/a)' : ''}
                    </label>
                    <div className="text-xs text-gray-500">
                      Status: {dayStatus?.status ?? 'pending'}
                      {today?.complete ? ' · complete' : ''}
                    </div>
                  </div>

                  <div className="rounded-lg border border-slack-border p-3 space-y-1 text-sm">
                    <div className="font-medium text-slack-text mb-1">Phase gates</div>
                    {(['1', '2', '3'] as const).map((id) => (
                      <div key={id} className="flex justify-between text-gray-300">
                        <span>Gate {id}</span>
                        <span className="text-xs text-gray-500">
                          {(gates[id] as { status?: string } | undefined)?.status ?? 'locked'}
                        </span>
                      </div>
                    ))}
                    <div className="flex justify-between text-gray-300 pt-1 border-t border-slack-border mt-2">
                      <span>Certification</span>
                      <span className="text-xs text-gray-500">
                        {(cert as { status?: string } | undefined)?.status ?? 'locked'}
                      </span>
                    </div>
                  </div>

                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() => void runAction(() => api.startAIInterviewDay())}
                      className="px-3 py-1.5 text-xs rounded-lg border border-indigo-600 bg-indigo-600/20 text-indigo-200 hover:bg-indigo-600/30 disabled:opacity-50"
                    >
                      Start today
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() =>
                        void runAction(() =>
                          api.completeAIInterviewDay(day, { concept: true }),
                        )
                      }
                      className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover disabled:opacity-50"
                    >
                      Mark concept done
                    </button>
                    <button
                      type="button"
                      disabled={busy || today?.has_drill === false}
                      onClick={() =>
                        void runAction(() => api.completeAIInterviewDay(day, { drill: true }))
                      }
                      className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover disabled:opacity-50"
                    >
                      Log drill pass
                    </button>
                    <button
                      type="button"
                      disabled={busy}
                      onClick={() =>
                        void runAction(() =>
                          api.completeAIInterviewDay(day, { complete: true }),
                        )
                      }
                      className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover disabled:opacity-50"
                    >
                      Complete day
                    </button>
                  </div>

                  <div className="space-y-2">
                    <label className="block text-xs text-gray-500">
                      Eval notes
                      <textarea
                        value={evalNotes}
                        onChange={(e) => setEvalNotes(e.target.value)}
                        rows={2}
                        className="mt-1 w-full rounded-lg border border-slack-border bg-slack-bgHover px-2 py-1.5 text-sm text-slack-text"
                        placeholder="Phase gate eval notes"
                      />
                    </label>
                    <label className="block text-xs text-gray-500">
                      Mock notes
                      <textarea
                        value={mockNotes}
                        onChange={(e) => setMockNotes(e.target.value)}
                        rows={2}
                        className="mt-1 w-full rounded-lg border border-slack-border bg-slack-bgHover px-2 py-1.5 text-sm text-slack-text"
                        placeholder="Mock panel notes"
                      />
                    </label>
                    <div className="flex flex-wrap gap-2">
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() =>
                          void runAction(() =>
                            api.submitAIInterviewGate(submitGateId, {
                              eval_notes: evalNotes || 'Submitted from desktop',
                              mock_notes: mockNotes || 'Submitted from desktop',
                            }),
                          )
                        }
                        className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover disabled:opacity-50"
                      >
                        Submit gate {submitGateId}
                      </button>
                      <button
                        type="button"
                        disabled={busy}
                        onClick={() => void runAction(() => api.unlockAIInterviewCert())}
                        className="px-3 py-1.5 text-xs rounded-lg border border-amber-600/60 text-amber-200 hover:bg-amber-600/10 disabled:opacity-50"
                      >
                        Unlock cert
                      </button>
                      <button
                        type="button"
                        disabled={busy || loading}
                        onClick={() => void loadProgress()}
                        className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover disabled:opacity-50"
                      >
                        Refresh
                      </button>
                    </div>
                  </div>
                </>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}
