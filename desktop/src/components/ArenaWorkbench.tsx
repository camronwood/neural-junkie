import { useCallback, useEffect, useMemo, useState } from 'react';
import { ChessBoard } from './arena/ChessBoard';
import { ConnectFourBoard } from './arena/ConnectFourBoard';
import { LogicPuzzlePanel } from './arena/LogicPuzzlePanel';
import {
  arenaCreateSession,
  arenaGetSession,
  arenaLeaderboard,
  arenaListChallenges,
  arenaMakeMove,
  arenaMatchRun,
  arenaMatchStep,
  arenaSubmitAnswer,
  fetchInstalledOllamaModels,
  fetchProviders,
  type ArenaSession,
} from './arena/arenaSidecarApi';
import { useToastStore } from '../stores/toastStore';
import './arena/arenaRetro.css';

type ChallengeId = 'chess' | 'connect4' | 'logic';

const CHALLENGE_MODES: Array<{ id: ChallengeId; label: string }> = [
  { id: 'connect4', label: 'Connect 4' },
  { id: 'chess', label: 'Chess' },
  { id: 'logic', label: 'Logic' },
];

/** Suggested arena models (14B cap friendly; avoid pulling provider's 27B default). */
const ARENA_SUGGESTED_MODELS = ['qwen2.5-coder:14b', 'qwen3.5:9b'] as const;
const ARENA_DEFAULT_MODEL = ARENA_SUGGESTED_MODELS[0];
const HUMAN_PLAYER = 'human';

function playerRoleHint(challenge: ChallengeId, slot: 1 | 2): string {
  if (challenge === 'logic') {
    return slot === 1 ? 'Solver' : 'Optional model';
  }
  if (challenge === 'connect4') {
    return slot === 1 ? 'Red' : 'Yellow';
  }
  return slot === 1 ? 'White' : 'Black';
}

function buildPlayerOptions(
  providers: Array<{ model?: string }>,
  ollamaModels: string[],
  leaderboardModels: string[] = [],
  selected: string[] = [],
): Array<{ value: string; label: string }> {
  const models = new Set<string>(ARENA_SUGGESTED_MODELS);
  for (const m of ollamaModels) {
    const tag = m.trim();
    if (tag && tag !== HUMAN_PLAYER) models.add(tag);
  }
  for (const m of leaderboardModels) {
    const tag = m.trim();
    if (tag && tag !== HUMAN_PLAYER) models.add(tag);
  }
  for (const p of providers) {
    const m = p.model?.trim();
    if (m && m !== HUMAN_PLAYER) models.add(m);
  }
  for (const s of selected) {
    const tag = s.trim();
    if (tag && tag !== HUMAN_PLAYER) models.add(tag);
  }
  return [
    { value: HUMAN_PLAYER, label: 'Human (you)' },
    ...Array.from(models)
      .sort((a, b) => a.localeCompare(b))
      .map((m) => ({ value: m, label: m })),
  ];
}

function resolveAiModelTag(white: string, black: string): string {
  if (white !== HUMAN_PLAYER) return white;
  if (black !== HUMAN_PLAYER) return black;
  return ARENA_DEFAULT_MODEL;
}

function rosterLabel(value: string): string {
  if (value === HUMAN_PLAYER) return 'HUMAN (YOU)';
  return value;
}

interface ArenaWorkbenchProps {
  workspaceId: string;
  sessionPath?: string;
  tabId: string;
  /** When false, skips the marquee header (modal supplies its own). */
  showHeader?: boolean;
}

function stateRecord(session: ArenaSession | null): Record<string, unknown> {
  return (session?.state as Record<string, unknown> | undefined) ?? {};
}

export function ArenaWorkbench({ workspaceId: _workspaceId, sessionPath, tabId: _tabId, showHeader = true }: ArenaWorkbenchProps) {
  const { addToast } = useToastStore();
  const [challenge, setChallenge] = useState<ChallengeId>('connect4');
  const [whiteModel, setWhiteModel] = useState<string>(HUMAN_PLAYER);
  const [blackModel, setBlackModel] = useState<string>(ARENA_DEFAULT_MODEL);
  const [providerId, setProviderId] = useState('');
  const [session, setSession] = useState<ArenaSession | null>(null);
  const [logicAnswer, setLogicAnswer] = useState('');
  const [leaderboard, setLeaderboard] = useState<Record<string, Record<string, number>>>({});
  const [busy, setBusy] = useState(false);
  const [providers, setProviders] = useState<Array<{ id: string; name: string; model?: string }>>([]);
  const [ollamaModels, setOllamaModels] = useState<string[]>([]);
  const [chessAvailable, setChessAvailable] = useState(true);

  const refreshLeaderboard = useCallback(async () => {
    try {
      const lb = await arenaLeaderboard();
      setLeaderboard(lb.models ?? {});
    } catch {
      /* optional */
    }
  }, []);

  useEffect(() => {
    void Promise.all([fetchProviders(), fetchInstalledOllamaModels(), arenaListChallenges()])
      .then(([rows, tags, challenges]) => {
        setProviders(rows);
        setOllamaModels(tags);
        if (rows[0]?.id) setProviderId(rows[0].id);
        const chess = challenges.challenges?.find((c) => String(c.id) === 'chess');
        setChessAvailable(chess?.available !== false);
      })
      .catch(() => undefined);
    void refreshLeaderboard();
  }, [refreshLeaderboard]);

  const leaderboardModelIds = useMemo(() => Object.keys(leaderboard), [leaderboard]);
  const playerOptions = useMemo(
    () => buildPlayerOptions(providers, ollamaModels, leaderboardModelIds, [whiteModel, blackModel]),
    [providers, ollamaModels, leaderboardModelIds, whiteModel, blackModel],
  );
  const modelTag = useMemo(() => resolveAiModelTag(whiteModel, blackModel), [whiteModel, blackModel]);

  const st = useMemo(() => stateRecord(session), [session]);
  const legalMoves = useMemo(() => {
    const raw = st.legal_moves;
    if (!Array.isArray(raw)) return [] as string[];
    return raw.map(String);
  }, [st.legal_moves]);

  const startSession = useCallback(async () => {
    setBusy(true);
    try {
      const created = await arenaCreateSession({
        challenge,
        white: whiteModel,
        black: blackModel,
      });
      setSession(created);
      setLogicAnswer('');
      addToast({ type: 'success', title: 'Arena', message: `Session ${created.id.slice(0, 8)} started` });
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [addToast, blackModel, challenge, whiteModel]);

  const refreshSession = useCallback(async () => {
    if (!session?.id) return;
    try {
      setSession(await arenaGetSession(session.id));
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    }
  }, [addToast, session?.id]);

  const applyMove = useCallback(
    async (body: Record<string, unknown>) => {
      if (!session?.id) return;
      setBusy(true);
      try {
        const next = await arenaMakeMove(session.id, { ...body, by: 'human' });
        setSession(next);
        void refreshLeaderboard();
      } catch (err) {
        addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
      } finally {
        setBusy(false);
      }
    },
    [addToast, refreshLeaderboard, session?.id],
  );

  const aiStep = useCallback(async () => {
    if (!session?.id) return;
    setBusy(true);
    try {
      const next = await arenaMatchStep({
        session_id: session.id,
        provider_id: providerId,
        model: modelTag,
      });
      setSession(next);
      void refreshLeaderboard();
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [addToast, modelTag, providerId, refreshLeaderboard, session?.id]);

  const autoRun = useCallback(async () => {
    if (!session?.id) return;
    setBusy(true);
    try {
      const next = await arenaMatchRun({
        session_id: session.id,
        provider_id: providerId,
        model: modelTag,
        max_steps: 30,
      });
      setSession(next);
      void refreshLeaderboard();
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [addToast, modelTag, providerId, refreshLeaderboard, session?.id]);

  const submitLogic = useCallback(async () => {
    if (!session?.id) return;
    setBusy(true);
    try {
      const next = await arenaSubmitAnswer(session.id, logicAnswer, modelTag);
      setSession(next);
      void refreshLeaderboard();
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [addToast, logicAnswer, modelTag, refreshLeaderboard, session?.id]);

  const board = (st.board as string[][] | undefined) ?? [];
  const status = String(st.status ?? session?.status ?? '');
  const isActive = status === 'active' || status === '';
  const liveLabel = session ? (isActive ? 'LIVE' : status.toUpperCase() || 'FINAL') : 'PRE-GAME';

  return (
    <div className="arena-retro flex h-full min-h-0 flex-col overflow-auto">
      {showHeader && (
        <header className="arena-retro-marquee shrink-0">
          <div>
            <div className="arena-retro-title">MODEL ARENA</div>
            <div className="arena-retro-subtitle">Model vs model · sports night</div>
          </div>
          <span className={`arena-retro-badge ${session ? '' : 'off'}`}>{liveLabel}</span>
        </header>
      )}

      {!showHeader && (
        <div className="shrink-0 flex justify-end px-3 pt-2">
          <span className={`arena-retro-badge ${session ? '' : 'off'}`}>{liveLabel}</span>
        </div>
      )}

      <div className="arena-retro-body flex min-h-0 flex-1 flex-col">
        <div className="arena-retro-mode-tabs" role="tablist" aria-label="Challenge mode">
          {CHALLENGE_MODES.map((mode) => (
            <button
              key={mode.id}
              type="button"
              role="tab"
              aria-selected={challenge === mode.id}
              disabled={busy && !!session}
              className={`arena-retro-mode-tab ${challenge === mode.id ? 'active' : ''}`}
              onClick={() => setChallenge(mode.id)}
            >
              {mode.label}
            </button>
          ))}
        </div>

        {challenge === 'chess' && !chessAvailable && (
          <div className="arena-retro-hint mb-3 border border-amber-500/50 bg-amber-500/10 px-3 py-2 text-amber-100">
            Chess needs <strong>python-chess</strong>. Open <strong>Settings → Domain packs → Tools → Model Arena</strong> and
            click <strong>Install chess dependencies</strong>.
          </div>
        )}

        <div className="arena-retro-matchup">
          <div className="arena-retro-team home">
            <div className="arena-retro-team-label">HOME</div>
            <div className="arena-retro-team-role">{playerRoleHint(challenge, 1)}</div>
            <label className="sr-only" htmlFor="arena-player-1">
              Player 1
            </label>
            <select
              id="arena-player-1"
              value={whiteModel}
              onChange={(e) => setWhiteModel(e.target.value)}
              className="arena-retro-select"
            >
              {playerOptions.map((opt) => (
                <option key={`p1-${opt.value}`} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
          <div className="arena-retro-vs" aria-hidden>
            VS
          </div>
          <div className="arena-retro-team away">
            <div className="arena-retro-team-label">AWAY</div>
            <div className="arena-retro-team-role">{playerRoleHint(challenge, 2)}</div>
            <label className="sr-only" htmlFor="arena-player-2">
              Player 2
            </label>
            <select
              id="arena-player-2"
              value={blackModel}
              onChange={(e) => setBlackModel(e.target.value)}
              className="arena-retro-select"
            >
              {playerOptions.map((opt) => (
                <option key={`p2-${opt.value}`} value={opt.value}>
                  {opt.label}
                </option>
              ))}
            </select>
          </div>
        </div>

        {ollamaModels.length === 0 && (
          <p className="arena-retro-hint">
            Pull models in the model library — roster fills from installed Ollama tags.
          </p>
        )}

        <div className="arena-retro-toolbar">
          <div>
            <span className="arena-retro-label">AI provider</span>
            <select
              value={providerId}
              onChange={(e) => setProviderId(e.target.value)}
              className="arena-retro-select min-w-[10rem]"
            >
              {providers.length === 0 && <option value="">No providers</option>}
              {providers.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name || p.id}
                </option>
              ))}
            </select>
          </div>
          <button type="button" disabled={busy} onClick={() => void startSession()} className="arena-retro-btn">
            Kick off
          </button>
          {session?.id && (
            <>
              <button
                type="button"
                disabled={busy || !isActive}
                onClick={() => void aiStep()}
                className="arena-retro-btn action"
              >
                AI play
              </button>
              <button
                type="button"
                disabled={busy || !isActive}
                onClick={() => void autoRun()}
                className="arena-retro-btn action"
              >
                Auto-run
              </button>
              <button
                type="button"
                disabled={busy}
                onClick={() => void refreshSession()}
                className="arena-retro-btn secondary"
              >
                Refresh
              </button>
            </>
          )}
        </div>

        {sessionPath && (
          <div className="arena-retro-hint mb-2">Save · {sessionPath}</div>
        )}

        <div className="arena-retro-grid flex-1 min-h-0">
          <div className="arena-retro-field min-h-0 overflow-auto">
            <div className="arena-retro-field-inner">
              {!session && (
                <div className="arena-retro-empty">
                  <strong>INSERT COIN</strong>
                  Pick your roster, hit <em>Kick off</em>, and play Connect Four, chess, or logic puzzles head-to-head.
                </div>
              )}
              {session && challenge === 'connect4' && (
                <ConnectFourBoard
                  board={board.length ? board : Array.from({ length: 6 }, () => Array(7).fill(''))}
                  legalMoves={legalMoves}
                  disabled={busy || !isActive}
                  onColumn={(col) => void applyMove({ column: col })}
                />
              )}
              {session && challenge === 'chess' && (
                <ChessBoard
                  ascii={String(st.ascii ?? '')}
                  fen={String(st.fen ?? '')}
                  legalMoves={legalMoves}
                  disabled={busy || !isActive}
                  onMove={(move) => void applyMove({ move })}
                />
              )}
              {session && challenge === 'logic' && (
                <LogicPuzzlePanel
                  prompt={String(st.prompt ?? session.puzzle?.prompt ?? '')}
                  title={session.puzzle?.title}
                  difficulty={String(st.difficulty ?? session.puzzle?.difficulty ?? '')}
                  answer={logicAnswer}
                  onAnswerChange={setLogicAnswer}
                  onSubmit={() => void submitLogic()}
                  onAskModel={() => void aiStep()}
                  result={String(st.result ?? session.result ?? '')}
                  explanation={String((session as { answer?: { explanation?: string } }).answer?.explanation ?? '')}
                  busy={busy}
                />
              )}
            </div>
            {session && (
              <div className="arena-retro-ticker">
                {rosterLabel(whiteModel)} vs {rosterLabel(blackModel)} ·{' '}
                {status || 'active'} · {String(st.result ?? session.result ?? 'in progress')}
              </div>
            )}
          </div>

          <aside className="arena-retro-standings">
            <h3>Standings</h3>
            {Object.keys(leaderboard).length === 0 && (
              <p className="text-sm text-slate-500 text-center">No scores yet</p>
            )}
            <ul className="space-y-0">
              {Object.entries(leaderboard).map(([model, row]) => (
                <li key={model} className="arena-retro-standing-row">
                  <span className="arena-retro-standing-name" title={model}>
                    {model}
                  </span>
                  <span className="arena-retro-standing-stats">
                    {row.wins ?? 0}W {row.losses ?? 0}L
                  </span>
                </li>
              ))}
            </ul>
          </aside>
        </div>
      </div>
    </div>
  );
}
