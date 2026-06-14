import { useState, useEffect } from 'react';
import { usePacksStore } from '../../stores/packsStore';
import { PACK_CAP } from '../../stores/packCapabilities';
import { ChatAPI, type UserLearning } from '../../api/chatAPI';
import { mergeSettingsPut, type SettingsTabProps } from './settingsShared';

export function MemoryLearningSettingsTab({ hubHttp, isActive }: SettingsTabProps) {
  const hasPersonalLearning = usePacksStore((s) => s.hasCapability(PACK_CAP.PERSONAL_LEARNING));
  const hasLoRATraining = usePacksStore((s) => s.hasCapability(PACK_CAP.LORA_TRAINING));
  const [personalLearningEnabled, setPersonalLearningEnabled] = useState(false);
  const [personalLearningSuggestEnabled, setPersonalLearningSuggestEnabled] = useState(false);
  const [conversationMemoryEnabled, setConversationMemoryEnabled] = useState(true);
  const [conversationMemorySaving, setConversationMemorySaving] = useState(false);
  const [personalLearningSaving, setPersonalLearningSaving] = useState(false);
  const [personalLearningsOpen, setPersonalLearningsOpen] = useState(false);
  const [allLearnings, setAllLearnings] = useState<UserLearning[]>([]);
  const [allLearningsLoading, setAllLearningsLoading] = useState(false);
  const [allLearningsErr, setAllLearningsErr] = useState<string | null>(null);
  const [collabRoutingErr, setCollabRoutingErr] = useState<string | null>(null);

  useEffect(() => {
    if (!isActive) return;
    let cancelled = false;
    (async () => {
      try {
        const r = await fetch(`${hubHttp}/api/settings`);
        if (!r.ok) throw new Error(await r.text());
        const cfg = await r.json();
        if (!cancelled) {
          setPersonalLearningEnabled(!!cfg.features?.personal_learning_enabled);
          setPersonalLearningSuggestEnabled(!!cfg.features?.personal_learning_suggest_enabled);
          setConversationMemoryEnabled(cfg.features?.conversation_memory_enabled !== false);
        }
      } catch (e) {
        if (!cancelled) setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      }
    })();
    return () => { cancelled = true; };
  }, [isActive, hubHttp]);

useEffect(() => {
      if (!isActive || !personalLearningEnabled || !hasPersonalLearning) {
        return;
      }
      void refreshAllLearnings();
    }, [isActive, personalLearningEnabled, hasPersonalLearning, hubHttp]);

const handleConversationMemoryToggle = async (enabled: boolean) => {
      setConversationMemorySaving(true);
      setCollabRoutingErr(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          features: {
            ...(typeof cfg.features === 'object' && cfg.features ? cfg.features : {}),
            conversation_memory_enabled: enabled,
          },
        }));
        setConversationMemoryEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setConversationMemorySaving(false);
      }
    };

    const handlePersonalLearningToggle = async (enabled: boolean) => {
      setPersonalLearningSaving(true);
      setCollabRoutingErr(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          features: {
            ...(typeof cfg.features === 'object' && cfg.features ? cfg.features : {}),
            personal_learning_enabled: enabled,
          },
        }));
        setPersonalLearningEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setPersonalLearningSaving(false);
      }
    };

    const refreshAllLearnings = async () => {
      if (!personalLearningEnabled || !hasPersonalLearning) return;
      setAllLearningsLoading(true);
      setAllLearningsErr(null);
      try {
        const api = new ChatAPI(hubHttp);
        const rows = await api.fetchLearnings();
        setAllLearnings(rows);
      } catch (e) {
        setAllLearningsErr(e instanceof Error ? e.message : String(e));
      } finally {
        setAllLearningsLoading(false);
      }
    };

    const handlePersonalLearningSuggestToggle = async (enabled: boolean) => {
      setPersonalLearningSaving(true);
      setCollabRoutingErr(null);
      try {
        await mergeSettingsPut(hubHttp, (cfg) => ({
          ...cfg,
          features: {
            ...(typeof cfg.features === 'object' && cfg.features ? cfg.features : {}),
            personal_learning_suggest_enabled: enabled,
          },
        }));
        setPersonalLearningSuggestEnabled(enabled);
      } catch (e) {
        setCollabRoutingErr(e instanceof Error ? e.message : String(e));
      } finally {
        setPersonalLearningSaving(false);
      }
    };

  if (!isActive) return null;

  return (
    <div className="space-y-8">
<div className="border border-slack-border rounded-lg p-6">
      <h3 className="text-lg font-semibold text-slack-text mb-2">Conversation memory</h3>
      <p className="text-sm text-slack-textMuted mb-4">
        Index persisted chat and collab artifacts locally, then retrieve relevant past context when you ask
        about earlier decisions (requires Ollama embed model).
      </p>
      <label className="flex items-center gap-3 cursor-pointer">
        <input
          type="checkbox"
          checked={conversationMemoryEnabled}
          disabled={conversationMemorySaving}
          onChange={(e) => void handleConversationMemoryToggle(e.target.checked)}
          className="rounded border-slack-border"
        />
        <span className="text-slack-text">Retrieve relevant past messages</span>
      </label>
    </div>
      {collabRoutingErr && <p className="text-sm text-red-600">{collabRoutingErr}</p>}
{hasPersonalLearning && (
      <div className="border border-slack-border rounded-lg p-6">
        <h3 className="text-lg font-semibold text-slack-text mb-2">Personal learning</h3>
        <p className="text-sm text-slack-textMuted mb-4">
          Agents will ask before saving anything — each expert keeps its own learnings.
        </p>
        <label className="flex items-center gap-3 cursor-pointer">
          <input
            type="checkbox"
            checked={personalLearningEnabled}
            disabled={personalLearningSaving}
            onChange={(e) => void handlePersonalLearningToggle(e.target.checked)}
            className="rounded border-slack-border"
          />
          <span className="text-slack-text">Enable personal learning for experts</span>
        </label>

        {personalLearningEnabled && (
          <label className="flex items-center gap-3 cursor-pointer mt-3">
            <input
              type="checkbox"
              checked={personalLearningSuggestEnabled}
              disabled={personalLearningSaving}
              onChange={(e) => void handlePersonalLearningSuggestToggle(e.target.checked)}
              className="rounded border-slack-border"
            />
            <span className="text-slack-text">Allow agents to suggest learnings (still requires your approval)</span>
          </label>
        )}

        {personalLearningEnabled && (
          <div className="flex gap-2 mt-4">
            <button
              type="button"
              className="px-3 py-1.5 text-sm border border-slack-border rounded hover:bg-slack-bgHover"
              onClick={async () => {
                try {
                  const api = new ChatAPI(hubHttp);
                  const bundle = await api.exportLearnings();
                  const blob = new Blob([JSON.stringify(bundle, null, 2)], { type: 'application/json' });
                  const url = URL.createObjectURL(blob);
                  const a = document.createElement('a');
                  a.href = url;
                  a.download = 'neural-junkie-learnings.json';
                  a.click();
                  URL.revokeObjectURL(url);
                } catch (e) {
                  setAllLearningsErr(e instanceof Error ? e.message : String(e));
                }
              }}
            >
              Export learnings
            </button>
            <label className="px-3 py-1.5 text-sm border border-slack-border rounded hover:bg-slack-bgHover cursor-pointer">
              Import learnings
              <input
                type="file"
                accept="application/json,.json"
                className="hidden"
                onChange={async (e) => {
                  const file = e.target.files?.[0];
                  if (!file) return;
                  try {
                    const text = await file.text();
                    const bundle = JSON.parse(text) as { entries: UserLearning[] };
                    const api = new ChatAPI(hubHttp);
                    await api.importLearnings(bundle);
                    const rows = await api.fetchLearnings();
                    setAllLearnings(rows);
                    setAllLearningsErr(null);
                  } catch (err) {
                    setAllLearningsErr(err instanceof Error ? err.message : 'Import failed');
                  } finally {
                    e.target.value = '';
                  }
                }}
              />
            </label>
          </div>
        )}

        {personalLearningEnabled && (
          <details
            open={personalLearningsOpen}
            onToggle={(e) => setPersonalLearningsOpen(e.currentTarget.open)}
            className="mt-4"
          >
            <summary className="cursor-pointer text-sm font-medium text-slack-text">
              Saved learnings ({allLearnings.length})
            </summary>
            {allLearningsLoading && (
              <p className="text-sm text-slack-textMuted mt-2">Loading…</p>
            )}
            {allLearningsErr && (
              <p className="text-sm text-red-600 mt-2">{allLearningsErr}</p>
            )}
            {!allLearningsLoading && allLearnings.length === 0 && (
              <p className="text-sm text-slack-textMuted mt-2">No learnings saved yet.</p>
            )}
            {allLearnings.length > 0 && (
              <div className="mt-2 space-y-4 max-h-64 overflow-y-auto">
                {(['global', 'agent', 'collaboration'] as const).map((scopeKey) => {
                  const rows = allLearnings.filter((e) => (e.scope || 'agent') === scopeKey);
                  if (rows.length === 0) return null;
                  const title =
                    scopeKey === 'global'
                      ? 'All experts'
                      : scopeKey === 'collaboration'
                        ? 'By collaboration'
                        : 'By expert';
                  return (
                    <div key={scopeKey}>
                      <p className="text-xs font-semibold text-slack-textMuted uppercase mb-1">{title}</p>
                      <ul className="space-y-1">
                        {rows.map((e) => (
                          <li
                            key={e.id}
                            className="flex justify-between gap-2 text-sm text-slack-textMuted"
                          >
                            <span>
                              {scopeKey === 'agent' && (
                                <span className="text-slack-text mr-1">{e.agent_name || e.agent_id}:</span>
                              )}
                              [{e.category}] {e.content}
                            </span>
                            <button
                              type="button"
                              className="text-red-500 hover:text-red-400 shrink-0"
                              onClick={async () => {
                                try {
                                  const api = new ChatAPI(hubHttp);
                                  await api.deleteLearning(e.id);
                                  setAllLearnings((prev) => prev.filter((x) => x.id !== e.id));
                                } catch (err) {
                                  setAllLearningsErr(
                                    err instanceof Error ? err.message : 'Forget failed'
                                  );
                                }
                              }}
                            >
                              Forget
                            </button>
                          </li>
                        ))}
                      </ul>
                    </div>
                  );
                })}
              </div>
            )}
            {hasLoRATraining && (
              <p className="text-xs text-slack-textMuted mt-3">
                When an expert has 10+ chat turns, open agent info (ℹ️) → Train LoRA to bake history into weights.
              </p>
            )}
          </details>
        )}
      </div>
    )}
    </div>
  );
}
