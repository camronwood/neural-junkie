#!/usr/bin/env python3
"""Split AIProvidersSettingsTab and IntegrationsSettingsTab into focused tab files."""
from __future__ import annotations

import re
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SETTINGS = ROOT / "src" / "components" / "settings"

AI_SRC = (SETTINGS / "AIProvidersSettingsTab.tsx").read_text()
INT_SRC = (SETTINGS / "IntegrationsSettingsTab.tsx").read_text()


def extract_div_by_h3(src: str, title: str, *, class_prefix: str = "border border-slack-border rounded-lg p-6") -> str:
    """Extract first top-level section div whose h3 contains title."""
    pattern = rf'    <div className="{re.escape(class_prefix)}[^"]*">\s*\n(?:      <div[^>]*>\s*\n)?      <h3[^>]*>{re.escape(title)}</h3>'
    m = re.search(pattern, src)
    if not m:
        raise ValueError(f"section not found: {title!r}")
    start = m.start()
    # walk div depth from start
    i = start
    depth = 0
    while i < len(src):
        if src.startswith("<div", i):
            depth += 1
            i = src.find(">", i) + 1
            continue
        if src.startswith("</div>", i):
            depth -= 1
            i += 6
            if depth == 0:
                return src[start:i].strip()
            continue
        i += 1
    raise ValueError(f"unclosed div for {title!r}")


def extract_details_by_summary(src: str, summary_text: str) -> str:
    pattern = rf'    <details[^>]*>\s*\n      <summary[^>]*>\s*\n        {re.escape(summary_text)}'
    m = re.search(pattern, src)
    if not m:
        # try inline summary
        pattern = rf'    <details[^>]*>\s*\n      <summary[^>]*>{re.escape(summary_text)}'
        m = re.search(pattern, src)
    if not m:
        raise ValueError(f"details not found: {summary_text!r}")
    start = m.start()
    end = src.find("    </details>", start)
    if end < 0:
        raise ValueError(f"details close not found: {summary_text!r}")
    return src[start : end + len("    </details>")].strip()


def extract_comment_block(src: str, comment: str) -> str:
    pattern = rf"    /\* {re.escape(comment)} \*/\s*\n(    <div[\s\S]*?    </div>\s*\n)"
    m = re.search(pattern, src)
    if not m:
        raise ValueError(f"comment block not found: {comment!r}")
    return m.group(1).strip()


def extract_conditional_block(src: str, condition: str) -> str:
    pattern = rf"    \{{{re.escape(condition)} && \(\s*\n(    <div[\s\S]*?    </div>\s*\n    \)\}}"
    m = re.search(pattern, src)
    if not m:
        raise ValueError(f"conditional block not found: {condition!r}")
    return m.group(1).strip()


def extract_slack_section(src: str) -> str:
    pattern = r"    /\* Slack bridge \*/\s*\n(    <div[\s\S]*?    </div>\s*\n)(?=    /\* Confluence)"
    m = re.search(pattern, src)
    if not m:
        raise ValueError("slack section not found")
    return m.group(1).strip()


def write(name: str, content: str) -> None:
    path = SETTINGS / name
    path.write_text(content)
    print(f"wrote {name} ({len(content.splitlines())} lines)")


# --- ProvidersSettingsTab ---
providers_sections = [
    extract_comment_block(AI_SRC, "CLI agent install & auth"),
    extract_comment_block(AI_SRC, "Dynamic Provider Registry"),
    extract_comment_block(AI_SRC, "Ollama Settings (legacy)"),
    extract_comment_block(AI_SRC, "LM Studio Settings"),
    extract_comment_block(AI_SRC, "Global Provider Toggle"),
]

providers = f"""import {{ useState, useEffect }} from 'react';
import {{ shallow }} from 'zustand/shallow';
import {{ useSettingsStore }} from '../../stores/settingsStore';
import {{ useChatStore }} from '../../stores/chatStore';
import {{ ProviderManager }} from '../ProviderManager';
import {{ CLIAgentsManager }} from '../CLIAgentsManager';
import {{
  fetchHardwareSnapshot,
  fetchModelLookup,
  formatModelResourceHint,
  type HardwareSnapshot,
  type ModelLookup,
}} from '../../utils/hardwareRecommendations';
import type {{ OllamaSettings, LMStudioSettings }} from '../../types/protocol';
import type {{ SettingsTabProps }} from './settingsShared';

export function ProvidersSettingsTab({{ hubHttp, isActive }}: SettingsTabProps) {{
  const {{
    integrations,
    loadIntegrations,
    updateOllamaSettings,
    updateLMStudioSettings,
    fetchOllamaModels,
    fetchLMStudioModels,
    testOllamaConnection,
    testLMStudioConnection,
  }} = useSettingsStore();
  const {{ switchAllAgentProviders }} = useChatStore(
    (s) => ({{ switchAllAgentProviders: s.switchAllAgentProviders }}),
    shallow
  );

  const [ollamaForm, setOllamaForm] = useState<OllamaSettings>(integrations.ollama);
  const [hardwareSnapshot, setHardwareSnapshot] = useState<HardwareSnapshot | null>(null);
  const [defaultModelLookup, setDefaultModelLookup] = useState<ModelLookup | null>(null);
  const [lmstudioForm, setLMStudioForm] = useState<LMStudioSettings>(integrations.lmstudio);
  const [testResults, setTestResults] = useState<Record<string, {{ success: boolean; message: string }}>>({{}});
  const [isSwitching, setIsSwitching] = useState(false);

  useEffect(() => {{
    if (!isActive) return;
    loadIntegrations();
  }}, [isActive, loadIntegrations]);

  useEffect(() => {{
    setOllamaForm(integrations.ollama);
    setLMStudioForm(integrations.lmstudio);
  }}, [integrations]);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    const loadModels = async () => {{
      try {{
        const ollamaModels = await fetchOllamaModels();
        if (!cancelled) setOllamaForm((prev) => ({{ ...prev, availableModels: ollamaModels }}));
      }} catch {{ /* Ollama may not be running */ }}
      try {{
        const lmModels = await fetchLMStudioModels();
        if (!cancelled) setLMStudioForm((prev) => ({{ ...prev, availableModels: lmModels }}));
      }} catch {{ /* LM Studio may not be running */ }}
    }};
    void loadModels();
    return () => {{ cancelled = true; }};
  }}, [isActive, fetchOllamaModels, fetchLMStudioModels]);

  useEffect(() => {{
    if (!isActive) return;
    let cancelled = false;
    void fetchHardwareSnapshot(hubHttp).then((snap) => {{
      if (!cancelled) setHardwareSnapshot(snap);
    }});
    return () => {{ cancelled = true; }};
  }}, [isActive, hubHttp]);

  useEffect(() => {{
    if (!isActive) return;
    const model = ollamaForm.defaultModel?.trim();
    if (!model) {{
      setDefaultModelLookup(null);
      return;
    }}
    let cancelled = false;
    void fetchModelLookup(hubHttp, model).then((row) => {{
      if (!cancelled) setDefaultModelLookup(row);
    }});
    return () => {{ cancelled = true; }};
  }}, [isActive, hubHttp, ollamaForm.defaultModel]);

  const handleOllamaChange = (field: keyof OllamaSettings, value: string | string[]) => {{
    setOllamaForm((prev) => ({{ ...prev, [field]: value }}));
  }};

  const handleLMStudioChange = (field: keyof LMStudioSettings, value: string | string[]) => {{
    setLMStudioForm((prev) => ({{ ...prev, [field]: value }}));
  }};

  const saveOllamaSettings = async () => {{
    try {{
      await updateOllamaSettings(ollamaForm);
      setTestResults((prev) => ({{ ...prev, ollama: {{ success: true, message: 'Settings saved successfully!' }} }}));
    }} catch (error) {{
      setTestResults((prev) => ({{
        ...prev,
        ollama: {{
          success: false,
          message: error instanceof Error ? error.message : 'Failed to save settings',
        }},
      }}));
    }}
  }};

  const saveLMStudioSettings = async () => {{
    try {{
      await updateLMStudioSettings(lmstudioForm);
      setTestResults((prev) => ({{ ...prev, lmstudio: {{ success: true, message: 'Settings saved successfully!' }} }}));
    }} catch (error) {{
      setTestResults((prev) => ({{
        ...prev,
        lmstudio: {{
          success: false,
          message: error instanceof Error ? error.message : 'Failed to save settings',
        }},
      }}));
    }}
  }};

  const testConnection = async (service: string) => {{
    setTestResults((prev) => ({{ ...prev, [service]: {{ success: false, message: 'Testing...' }} }}));
    try {{
      let result = false;
      if (service === 'ollama') result = await testOllamaConnection();
      else if (service === 'lmstudio') result = await testLMStudioConnection();
      setTestResults((prev) => ({{
        ...prev,
        [service]: {{
          success: result,
          message: result ? 'Connection successful!' : 'Connection failed. Check your credentials.',
        }},
      }}));
    }} catch (error) {{
      setTestResults((prev) => ({{
        ...prev,
        [service]: {{
          success: false,
          message: `Error: ${{error instanceof Error ? error.message : 'Unknown error'}}`,
        }},
      }}));
    }}
  }};

  const handleSwitchAllToClaude = async () => {{
    setIsSwitching(true);
    try {{
      await switchAllAgentProviders('claude', 'claude-sonnet');
      setTestResults((prev) => ({{
        ...prev,
        providerSwitch: {{ success: true, message: 'All agents switched to Claude successfully!' }},
      }}));
    }} catch (error) {{
      setTestResults((prev) => ({{
        ...prev,
        providerSwitch: {{
          success: false,
          message: error instanceof Error ? error.message : 'Failed to switch all agents to Claude',
        }},
      }}));
    }} finally {{
      setIsSwitching(false);
    }}
  }};

  const handleSwitchAllToOllama = async () => {{
    setIsSwitching(true);
    try {{
      const model = ollamaForm.defaultModel || 'llama3.1';
      await switchAllAgentProviders('ollama', model);
      setTestResults((prev) => ({{
        ...prev,
        providerSwitch: {{ success: true, message: `All agents switched to Ollama (${{model}}) successfully!` }},
      }}));
    }} catch (error) {{
      setTestResults((prev) => ({{
        ...prev,
        providerSwitch: {{
          success: false,
          message: error instanceof Error ? error.message : 'Failed to switch all agents to Ollama',
        }},
      }}));
    }} finally {{
      setIsSwitching(false);
    }}
  }};

  const handleSwitchAllToLMStudio = async () => {{
    setIsSwitching(true);
    try {{
      const model = lmstudioForm.defaultModel || (lmstudioForm.availableModels[0] ?? '');
      await switchAllAgentProviders('lmstudio', model);
      setTestResults((prev) => ({{
        ...prev,
        providerSwitch: {{
          success: true,
          message: `All agents switched to LM Studio${{model ? ` (${{model}})` : ''}} successfully!`,
        }},
      }}));
    }} catch (error) {{
      setTestResults((prev) => ({{
        ...prev,
        providerSwitch: {{
          success: false,
          message: error instanceof Error ? error.message : 'Failed to switch all agents to LM Studio',
        }},
      }}));
    }} finally {{
      setIsSwitching(false);
    }}
  }};

  if (!isActive) return null;

  return (
    <div className="space-y-8">
{chr(10).join(providers_sections)}
    </div>
  );
}}
"""
write("ProvidersSettingsTab.tsx", providers)

print("ProvidersSettingsTab done — run manual generation for remaining tabs via full file copy")
