import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { ChessBoard } from './arena/ChessBoard';
import { ConnectFourBoard } from './arena/ConnectFourBoard';
import { LogicPuzzlePanel } from './arena/LogicPuzzlePanel';
import { ArenaPlayLog, stepMetaToLogEntry, type ArenaPlayLogEntry } from './arena/ArenaPlayLog';
import {
  arenaCreateSession,
  arenaGetSession,
  arenaLeaderboard,
  arenaListChallenges,
  arenaMakeMove,
  arenaMatchStep,
  arenaSubmitAnswer,
  fetchInstalledOllamaModels,
  fetchProviders,
  type ArenaSession,
} from './arena/arenaSidecarApi';
import { useToastStore } from '../stores/toastStore';
import './arena/arenaRetro.css';

type ChallengeId = 'chess' | 'connect4' | 'logic';
type LogicPuzzleOption = {
  id: string;
  title?: string;
  difficulty?: string;
};

const CHALLENGE_MODES: Array<{ id: ChallengeId; label: string }> = [
  { id: 'connect4', label: 'Connect 4' },
  { id: 'chess', label: 'Chess' },
  { id: 'logic', label: 'Logic' },
];

/** Suggested arena models (14B cap friendly; avoid pulling provider's 27B default). */
const ARENA_SUGGESTED_MODELS = ['qwen2.5-coder:14b', 'qwen3.5:9b'] as const;
const ARENA_DEFAULT_MODEL = ARENA_SUGGESTED_MODELS[0];
const HUMAN_PLAYER = 'human';
const CUSTOM_LOGIC_PUZZLE = '__custom__';
const AUTO_RUN_DELAY_MS = 700;
/** Connect Four fills in ≤42 plies; chess often needs far more. */
const AUTO_RUN_MAX_STEPS: Record<ChallengeId, number> = {
  connect4: 42,
  chess: 160,
  logic: 4,
};

type MatchPauseReason = 'max_steps' | 'human_turn' | 'no_progress' | 'invalid_model_move' | null;

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => window.setTimeout(resolve, ms));
}

function autoRunMaxSteps(challenge: ChallengeId): number {
  return AUTO_RUN_MAX_STEPS[challenge] ?? 42;
}

function isSessionTerminal(state: Record<string, unknown>): boolean {
  const s = String(state.status ?? '').toLowerCase();
  return s !== '' && s !== 'active';
}

function sessionMoveCount(session: ArenaSession | null): number {
  return Array.isArray(session?.moves) ? session.moves.length : 0;
}

function playerRoleHint(challenge: ChallengeId, slot: 1 | 2): string {
  if (challenge === 'logic') {
    return slot === 1 ? 'Solver (home)' : 'Solver (away)';
  }
  if (challenge === 'connect4') {
    return slot === 1 ? 'Red' : 'Yellow';
  }
  return slot === 1 ? 'White' : 'Black';
}

function buildPlayerOptions(
  ollamaModels: string[],
  selected: string[] = [],
): Array<{ value: string; label: string }> {
  const models = new Set<string>();
  for (const m of ollamaModels) {
    const tag = m.trim();
    if (tag && tag !== HUMAN_PLAYER) models.add(tag);
  }
  for (const s of selected) {
    const tag = s.trim();
    if (tag && tag !== HUMAN_PLAYER && models.has(tag)) models.add(tag);
  }
  return [
    { value: HUMAN_PLAYER, label: 'Human (you)' },
    ...Array.from(models)
      .sort((a, b) => a.localeCompare(b))
      .map((m) => ({ value: m, label: m })),
  ];
}

function pickDefaultArenaModel(installed: string[], avoid?: string): string {
  const pick = installed.find((m) => m.trim() && m !== HUMAN_PLAYER && m !== avoid);
  return pick ?? ARENA_DEFAULT_MODEL;
}

function isInstalledModel(tag: string, installed: string[]): boolean {
  return tag === HUMAN_PLAYER || installed.includes(tag);
}

function resolveLogicModelTag(white: string, black: string): string {
  if (white !== HUMAN_PLAYER) return white;
  if (black !== HUMAN_PLAYER) return black;
  return ARENA_DEFAULT_MODEL;
}

function isHumanTurn(
  challenge: ChallengeId,
  state: Record<string, unknown>,
  white: string,
  black: string,
): boolean {
  const turn = String(state.turn ?? '').toLowerCase();
  if (challenge === 'connect4') {
    if (turn === 'red') return white === HUMAN_PLAYER;
    if (turn === 'yellow') return black === HUMAN_PLAYER;
  }
  if (challenge === 'chess') {
    if (turn === 'white') return white === HUMAN_PLAYER;
    if (turn === 'black') return black === HUMAN_PLAYER;
  }
  return false;
}

function activeChallengeId(session: ArenaSession | null, fallback: ChallengeId): ChallengeId {
  const raw = String(session?.challenge ?? fallback);
  if (raw === 'chess' || raw === 'connect4' || raw === 'logic') return raw;
  return fallback;
}

function rosterLabel(value: string): string {
  if (value === HUMAN_PLAYER) return 'HUMAN (YOU)';
  return value;
}

type ArenaOutcomeTone = 'win' | 'draw' | 'loss';

interface ArenaOutcome {
  over: boolean;
  tone: ArenaOutcomeTone;
  headline: string;
  detail?: string;
}

function seatModelLabel(seat: 'white' | 'black', white: string, black: string): string {
  return rosterLabel(seat === 'white' ? white : black);
}

/** Resolve a human-readable game-over result (winner, draw, or logic correctness). */
function resolveArenaOutcome(
  challenge: ChallengeId,
  state: Record<string, unknown>,
  session: ArenaSession | null,
  white: string,
  black: string,
): ArenaOutcome {
  const status = String(state.status ?? session?.status ?? '').toLowerCase();
  const result = String(state.result ?? session?.result ?? '').toLowerCase();
  if (!session || status === '' || status === 'active') {
    return { over: false, tone: 'draw', headline: '' };
  }
  if (status === 'draw' || result === 'draw') {
    const duelKind = String(
      (session as { answer?: { duel_kind?: string } } | null)?.answer?.duel_kind ?? '',
    );
    if (duelKind === 'both_correct') {
      return { over: true, tone: 'draw', headline: 'Draw', detail: 'Both models answered correctly.' };
    }
    if (duelKind === 'both_incorrect') {
      return { over: true, tone: 'draw', headline: 'Draw', detail: 'Both models missed the answer.' };
    }
    return { over: true, tone: 'draw', headline: 'Draw', detail: 'No winner this round.' };
  }
  if (status === 'correct' || status === 'incorrect') {
    const solver = white !== HUMAN_PLAYER ? white : black;
    const correct = status === 'correct';
    return {
      over: true,
      tone: correct ? 'win' : 'loss',
      headline: correct ? 'Correct' : 'Incorrect',
      detail: `${rosterLabel(solver)} ${correct ? 'solved the puzzle' : 'got it wrong'}.`,
    };
  }
  let winnerSeat: 'white' | 'black' | '' = '';
  if (result === 'white' || result === 'red') winnerSeat = 'white';
  else if (result === 'black' || result === 'yellow') winnerSeat = 'black';
  if (winnerSeat) {
    const seatColor =
      challenge === 'connect4'
        ? winnerSeat === 'white'
          ? 'Red'
          : 'Yellow'
        : winnerSeat === 'white'
          ? 'White'
          : 'Black';
    let detail = `Playing ${seatColor}.`;
    if (challenge === 'chess') detail = `Checkmate — ${seatColor}.`;
    if (challenge === 'logic') detail = 'Only correct answer in the duel.';
    return {
      over: true,
      tone: 'win',
      headline: `${seatModelLabel(winnerSeat, white, black)} wins`,
      detail,
    };
  }
  // finished/incorrect-style status without a mapped seat — still show something useful
  if (status === 'finished') {
    return {
      over: true,
      tone: 'draw',
      headline: 'Match finished',
      detail: result ? `Result · ${result}` : 'No winner recorded.',
    };
  }
  return { over: true, tone: 'draw', headline: (status || 'final').toUpperCase() };
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
  const [logicPuzzles, setLogicPuzzles] = useState<LogicPuzzleOption[]>([]);
  const [logicPuzzleId, setLogicPuzzleId] = useState('');
  const [customLogicPrompt, setCustomLogicPrompt] = useState('');
  const [customLogicAnswer, setCustomLogicAnswer] = useState('');
  const [playLog, setPlayLog] = useState<ArenaPlayLogEntry[]>([]);
  const [autoRunning, setAutoRunning] = useState(false);
  const [matchPause, setMatchPause] = useState<MatchPauseReason>(null);
  const autoRunAbort = useRef(false);

  const refreshLeaderboard = useCallback(async () => {
    try {
      const lb = await arenaLeaderboard();
      setLeaderboard(lb.models ?? {});
    } catch {
      /* optional */
    }
  }, []);

  const refreshOllamaModels = useCallback(async () => {
    try {
      const tags = await fetchInstalledOllamaModels();
      setOllamaModels(tags);
      return tags;
    } catch {
      return [];
    }
  }, []);

  useEffect(() => {
    void Promise.all([fetchProviders(), refreshOllamaModels(), arenaListChallenges()])
      .then(([rows, tags, challenges]) => {
        setProviders(rows);
        if (tags.length > 0) {
          setBlackModel((prev) => {
            if (prev !== HUMAN_PLAYER && tags.includes(prev)) return prev;
            return pickDefaultArenaModel(tags);
          });
        }
        if (rows[0]?.id) setProviderId(rows[0].id);
        const chess = challenges.challenges?.find((c) => String(c.id) === 'chess');
        setChessAvailable(chess?.available !== false);
        const puzzles = (challenges.puzzles ?? [])
          .map((p) => ({
            id: String(p.id ?? ''),
            title: typeof p.title === 'string' ? p.title : undefined,
            difficulty: typeof p.difficulty === 'string' ? p.difficulty : undefined,
          }))
          .filter((p) => p.id);
        setLogicPuzzles(puzzles);
        setLogicPuzzleId((prev) =>
          prev === CUSTOM_LOGIC_PUZZLE || (prev && puzzles.some((p) => p.id === prev))
            ? prev
            : puzzles[0]?.id ?? '',
        );
      })
      .catch(() => undefined);
    void refreshLeaderboard();
  }, [refreshLeaderboard, refreshOllamaModels]);

  const playerOptions = useMemo(
    () => buildPlayerOptions(ollamaModels, [whiteModel, blackModel]),
    [ollamaModels, whiteModel, blackModel],
  );
  const logicPuzzleIds = useMemo(
    () => [...logicPuzzles.map((p) => p.id), CUSTOM_LOGIC_PUZZLE],
    [logicPuzzles],
  );
  const logicPuzzleIndex = Math.max(0, logicPuzzleIds.indexOf(logicPuzzleId));
  const cycleLogicPuzzle = useCallback(
    (direction: -1 | 1) => {
      if (logicPuzzleIds.length === 0) return;
      const next = (logicPuzzleIndex + direction + logicPuzzleIds.length) % logicPuzzleIds.length;
      setLogicPuzzleId(logicPuzzleIds[next]);
    },
    [logicPuzzleIds, logicPuzzleIndex],
  );

  useEffect(() => {
    if (ollamaModels.length === 0) return;
    setWhiteModel((prev) => {
      if (prev === HUMAN_PLAYER || isInstalledModel(prev, ollamaModels)) return prev;
      return HUMAN_PLAYER;
    });
    setBlackModel((prev) => {
      if (prev === HUMAN_PLAYER) return prev;
      if (isInstalledModel(prev, ollamaModels)) return prev;
      return pickDefaultArenaModel(ollamaModels);
    });
  }, [ollamaModels]);
  const logicModelTag = useMemo(
    () => resolveLogicModelTag(whiteModel, blackModel),
    [whiteModel, blackModel],
  );
  const activeChallenge = useMemo(
    () => activeChallengeId(session, challenge),
    [session, challenge],
  );
  const rosterWhite = session?.players?.white ?? whiteModel;
  const rosterBlack = session?.players?.black ?? blackModel;

  const st = useMemo(() => stateRecord(session), [session]);
  const legalMoves = useMemo(() => {
    const raw = st.legal_moves;
    if (!Array.isArray(raw)) return [] as string[];
    return raw.map(String);
  }, [st.legal_moves]);
  const status = String(st.status ?? session?.status ?? '');
  const isActive = status === 'active' || status === '';

  useEffect(() => {
    return () => {
      autoRunAbort.current = true;
    };
  }, []);

  const appendStepFromResult = useCallback((result: ArenaSession) => {
    const entry = stepMetaToLogEntry(result._arena_step, activeChallengeId(result, challenge));
    if (entry) {
      setPlayLog((prev) => [...prev, entry]);
    }
    return entry;
  }, [challenge]);

  const startSession = useCallback(async () => {
    autoRunAbort.current = true;
    setAutoRunning(false);
    setMatchPause(null);
    setPlayLog([]);
    if (whiteModel !== HUMAN_PLAYER && !isInstalledModel(whiteModel, ollamaModels)) {
      addToast({
        type: 'error',
        title: 'Arena',
        message: `${whiteModel} is not installed in Ollama. Pick an installed tag or pull it in the model library.`,
      });
      return;
    }
    if (blackModel !== HUMAN_PLAYER && !isInstalledModel(blackModel, ollamaModels)) {
      addToast({
        type: 'error',
        title: 'Arena',
        message: `${blackModel} is not installed in Ollama. Pick an installed tag or pull it in the model library.`,
      });
      return;
    }
    if (
      challenge === 'logic' &&
      logicPuzzleId === CUSTOM_LOGIC_PUZZLE &&
      (!customLogicPrompt.trim() || !customLogicAnswer.trim())
    ) {
      addToast({
        type: 'error',
        title: 'Arena',
        message: 'Custom logic puzzles require both the puzzle text and expected answer.',
      });
      return;
    }
    setBusy(true);
    try {
      const created = await arenaCreateSession({
        challenge,
        white: whiteModel,
        black: blackModel,
        ...(challenge === 'logic' && logicPuzzleId !== CUSTOM_LOGIC_PUZZLE
          ? { puzzle_id: logicPuzzleId }
          : {}),
        ...(challenge === 'logic' && logicPuzzleId === CUSTOM_LOGIC_PUZZLE
          ? {
              custom_prompt: customLogicPrompt.trim(),
              custom_answer: customLogicAnswer.trim(),
            }
          : {}),
      });
      setSession(created);
      setLogicAnswer('');
      addToast({ type: 'success', title: 'Arena', message: `Session ${created.id.slice(0, 8)} started` });
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [
    addToast,
    blackModel,
    challenge,
    customLogicAnswer,
    customLogicPrompt,
    logicPuzzleId,
    ollamaModels,
    whiteModel,
  ]);

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

  const humanTurn = useMemo(
    () => isHumanTurn(activeChallenge, st, rosterWhite, rosterBlack),
    [activeChallenge, st, rosterWhite, rosterBlack],
  );
  const canModelStep = useMemo(() => {
    if (!session || !isActive) return false;
    if (activeChallenge === 'logic') return true;
    return !humanTurn;
  }, [activeChallenge, humanTurn, isActive, session]);

  const aiStep = useCallback(async () => {
    if (!session?.id) return;
    setBusy(true);
    setMatchPause(null);
    try {
      const next = await arenaMatchStep({
        session_id: session.id,
        provider_id: providerId,
      });
      appendStepFromResult(next);
      setSession(next);
      const meta = next._arena_step;
      if (meta?.skipped) {
        if (meta.reason === 'human_turn') setMatchPause('human_turn');
        else if (meta.reason === 'invalid_model_move') setMatchPause('invalid_model_move');
        else setMatchPause('no_progress');
      }
      void refreshLeaderboard();
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [addToast, appendStepFromResult, providerId, refreshLeaderboard, session?.id]);

  const autoRun = useCallback(async () => {
    if (!session?.id) return;
    autoRunAbort.current = false;
    setMatchPause(null);
    setAutoRunning(true);
    setBusy(true);
    setPlayLog([]);
    const maxSteps = autoRunMaxSteps(challenge);
    let pause: MatchPauseReason = null;
    let stepsTaken = 0;
    try {
      let current: ArenaSession = session;
      for (let step = 0; step < maxSteps; step++) {
        if (autoRunAbort.current) break;

        const challengeId = activeChallengeId(current, challenge);
        const movesBefore = sessionMoveCount(current);
        const next = await arenaMatchStep({
          session_id: current.id,
          provider_id: providerId,
        });
        appendStepFromResult(next);
        setSession(next);
        current = next;
        stepsTaken = step + 1;

        const st = stateRecord(next);
        if (isSessionTerminal(st)) {
          pause = null;
          break;
        }

        const meta = next._arena_step;
        if (meta?.skipped) {
          if (meta.reason === 'human_turn') pause = 'human_turn';
          else if (meta.reason === 'invalid_model_move') pause = 'invalid_model_move';
          else pause = 'no_progress';
          break;
        }

        if (challengeId !== 'logic' && sessionMoveCount(next) === movesBefore) {
          pause = 'no_progress';
          break;
        }

        if (step === maxSteps - 1 && !isSessionTerminal(stateRecord(next))) {
          pause = 'max_steps';
        }

        await sleep(AUTO_RUN_DELAY_MS);
      }
      if (pause === null && stepsTaken >= maxSteps && !isSessionTerminal(stateRecord(current))) {
        pause = 'max_steps';
      }
      setMatchPause(pause);
      if (pause === 'max_steps') {
        addToast({
          type: 'info',
          title: 'Match paused',
          message: `Auto-run stopped after ${stepsTaken} moves — game still in progress. Hit Auto-run again to continue.`,
        });
      }
      void refreshLeaderboard();
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setAutoRunning(false);
      setBusy(false);
    }
  }, [addToast, appendStepFromResult, challenge, providerId, refreshLeaderboard, session]);

  const submitLogic = useCallback(async () => {
    if (!session?.id) return;
    setBusy(true);
    try {
      const next = await arenaSubmitAnswer(session.id, logicAnswer, logicModelTag);
      setSession(next);
      void refreshLeaderboard();
    } catch (err) {
      addToast({ type: 'error', title: 'Arena', message: err instanceof Error ? err.message : String(err) });
    } finally {
      setBusy(false);
    }
  }, [addToast, logicAnswer, logicModelTag, refreshLeaderboard, session?.id]);

  const board = (st.board as string[][] | undefined) ?? [];
  const outcome = useMemo(
    () => resolveArenaOutcome(activeChallenge, st, session, rosterWhite, rosterBlack),
    [activeChallenge, st, session, rosterWhite, rosterBlack],
  );
  const liveLabel = session
    ? isActive
      ? matchPause === 'max_steps'
        ? 'PAUSED'
        : 'LIVE'
      : outcome.over
        ? 'GAME OVER'
        : status.toUpperCase() || 'FINAL'
    : 'PRE-GAME';
  const canHumanMove = isActive && (activeChallenge === 'logic' || humanTurn);

  const announcedEndRef = useRef<string>('');
  useEffect(() => {
    if (!session?.id || !outcome.over) return;
    if (announcedEndRef.current === session.id) return;
    announcedEndRef.current = session.id;
    setMatchPause(null);
    addToast({
      type: outcome.tone === 'loss' ? 'error' : outcome.tone === 'draw' ? 'info' : 'success',
      title: 'Game over',
      message: outcome.detail ? `${outcome.headline} — ${outcome.detail}` : outcome.headline,
    });
  }, [session?.id, outcome.over, outcome.tone, outcome.headline, outcome.detail, addToast]);

  return (
    <div className="arena-retro flex h-full min-h-0 flex-col overflow-hidden">
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

      <div className="arena-retro-body flex min-h-0 flex-1 flex-col overflow-y-auto overscroll-contain">
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

        {challenge === 'logic' && logicPuzzles.length > 0 && (
          <div className="arena-retro-puzzle-picker">
            <button
              type="button"
              className="arena-retro-btn secondary"
              onClick={() => cycleLogicPuzzle(-1)}
              disabled={busy}
              aria-label="Previous logic puzzle"
            >
              ←
            </button>
            <label>
              <span className="arena-retro-label">
                {logicPuzzleId === CUSTOM_LOGIC_PUZZLE
                  ? 'Custom puzzle'
                  : `Puzzle ${logicPuzzleIndex + 1} of ${logicPuzzles.length}`}
              </span>
              <select
                value={logicPuzzleId}
                onChange={(e) => setLogicPuzzleId(e.target.value)}
                className="arena-retro-select"
                disabled={busy}
              >
                {logicPuzzles.map((p, index) => (
                  <option key={p.id} value={p.id}>
                    {index + 1}. {p.title || p.id}
                    {p.difficulty ? ` · ${p.difficulty}` : ''}
                  </option>
                ))}
                <option value={CUSTOM_LOGIC_PUZZLE}>Custom puzzle…</option>
              </select>
            </label>
            <button
              type="button"
              className="arena-retro-btn secondary"
              onClick={() => cycleLogicPuzzle(1)}
              disabled={busy}
              aria-label="Next logic puzzle"
            >
              →
            </button>
          </div>
        )}

        {challenge === 'logic' && logicPuzzleId === CUSTOM_LOGIC_PUZZLE && (
          <div className="arena-retro-custom-puzzle">
            <label>
              <span className="arena-retro-label">Your logic puzzle</span>
              <textarea
                value={customLogicPrompt}
                onChange={(e) => setCustomLogicPrompt(e.target.value)}
                placeholder="Describe the puzzle and ask one clear question…"
                className="arena-retro-custom-puzzle-text"
                rows={4}
                disabled={busy}
              />
            </label>
            <label>
              <span className="arena-retro-label">Expected answer</span>
              <input
                type="text"
                value={customLogicAnswer}
                onChange={(e) => setCustomLogicAnswer(e.target.value)}
                placeholder="Answer used to score both models"
                className="arena-retro-select"
                disabled={busy}
              />
            </label>
            <p className="arena-retro-hint">
              Answers are compared case-insensitively after trimming whitespace.
            </p>
          </div>
        )}

        <div className="arena-retro-matchup">
          <div className="arena-retro-team home">
            <div className="arena-retro-team-label">HOME</div>
            <div className="arena-retro-team-role">{playerRoleHint(activeChallenge, 1)}</div>
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
            <div className="arena-retro-team-role">{playerRoleHint(activeChallenge, 2)}</div>
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
            Pull models in the model library (⇧⌘M) — the roster lists only installed Ollama tags.
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
          <button
            type="button"
            disabled={busy}
            onClick={() => void refreshOllamaModels()}
            className="arena-retro-btn secondary"
            title="Refresh installed Ollama model tags"
          >
            Refresh models
          </button>
          <button type="button" disabled={busy} onClick={() => void startSession()} className="arena-retro-btn">
            Kick off
          </button>
          {session?.id && (
            <>
              <button
                type="button"
                disabled={busy || !canModelStep}
                onClick={() => void aiStep()}
                className="arena-retro-btn action"
                title={humanTurn ? 'Waiting for your move' : 'Prompt the active model seat'}
              >
                AI play
              </button>
              <button
                type="button"
                disabled={busy || !canModelStep || autoRunning}
                onClick={() => void autoRun()}
                className="arena-retro-btn action"
                title={
                  humanTurn
                    ? 'Waiting for your move'
                    : `Run model moves until a human turn, game end, or ${autoRunMaxSteps(challenge)} plies`
                }
              >
                {autoRunning ? 'Running…' : 'Auto-run'}
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
          <div className="arena-retro-field min-h-0">
            <div className="arena-retro-field-inner">
              {session && outcome.over && (
                <div className={`arena-retro-gameover tone-${outcome.tone}`} role="status" aria-live="assertive">
                  <span className="arena-retro-gameover-tag">Game Over</span>
                  <span className="arena-retro-gameover-headline">{outcome.headline}</span>
                  {outcome.detail && <span className="arena-retro-gameover-detail">{outcome.detail}</span>}
                </div>
              )}
              {session && !outcome.over && matchPause === 'max_steps' && (
                <div className="arena-retro-gameover tone-draw" role="status" aria-live="polite">
                  <span className="arena-retro-gameover-tag">Match paused</span>
                  <span className="arena-retro-gameover-headline">No winner yet</span>
                  <span className="arena-retro-gameover-detail">
                    Auto-run hit the move limit while the game was still active. Hit Auto-run again to continue.
                  </span>
                </div>
              )}
              {session && !outcome.over && matchPause === 'no_progress' && (
                <div className="arena-retro-gameover tone-draw" role="status" aria-live="polite">
                  <span className="arena-retro-gameover-tag">Match paused</span>
                  <span className="arena-retro-gameover-headline">Could not continue</span>
                  <span className="arena-retro-gameover-detail">
                    The last model reply did not produce a legal move. Try AI play or Auto-run again.
                  </span>
                </div>
              )}
              {session && !outcome.over && matchPause === 'invalid_model_move' && (
                <div className="arena-retro-gameover tone-draw" role="status" aria-live="polite">
                  <span className="arena-retro-gameover-tag">Match paused</span>
                  <span className="arena-retro-gameover-headline">Invalid model move</span>
                  <span className="arena-retro-gameover-detail">
                    Re-prompted the model for a legal UCI move (e2e4-style, not Nf3), but it still failed. Hit AI
                    play or Auto-run to try again.
                  </span>
                </div>
              )}
              {!session && (
                <div className="arena-retro-empty">
                  <strong>INSERT COIN</strong>
                  Pick your roster, hit <em>Kick off</em>, and play Connect Four, chess, or logic puzzles head-to-head.
                </div>
              )}
              {session && activeChallenge === 'connect4' && (
                <ConnectFourBoard
                  board={board.length ? board : Array.from({ length: 6 }, () => Array(7).fill(''))}
                  legalMoves={legalMoves}
                  disabled={busy || !canHumanMove}
                  onColumn={(col) => void applyMove({ column: col })}
                />
              )}
              {session && activeChallenge === 'chess' && (
                <ChessBoard
                  ascii={String(st.ascii ?? '')}
                  fen={String(st.fen ?? '')}
                  legalMoves={legalMoves}
                  disabled={busy || !canHumanMove}
                  onMove={(move) => void applyMove({ move })}
                />
              )}
              {session && activeChallenge === 'logic' && (
                <LogicPuzzlePanel
                  prompt={String(st.prompt ?? session.puzzle?.prompt ?? '')}
                  title={session.puzzle?.title}
                  difficulty={String(st.difficulty ?? session.puzzle?.difficulty ?? '')}
                  answer={logicAnswer}
                  onAnswerChange={setLogicAnswer}
                  onSubmit={() => void submitLogic()}
                  onAskModel={() => void aiStep()}
                  result={String(st.result ?? session.result ?? '')}
                  explanation={String(
                    (session as { answer?: { explanation?: string; duel?: boolean } }).answer?.explanation ??
                      '',
                  )}
                  duelAnswers={
                    (session as { answers?: Record<string, { answer?: string; correct?: boolean; model?: string }> })
                      .answers
                  }
                  busy={busy}
                />
              )}
            </div>
            {session && (
              <div className={`arena-retro-ticker ${outcome.over ? `over tone-${outcome.tone}` : ''}`}>
                {rosterLabel(rosterWhite)} vs {rosterLabel(rosterBlack)} ·{' '}
                {outcome.over ? `RESULT · ${outcome.headline}` : status ? status.toUpperCase() : 'IN PROGRESS'}
              </div>
            )}
          </div>

          <aside className="arena-retro-standings">
            <ArenaPlayLog entries={playLog} autoRunning={autoRunning} />
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
