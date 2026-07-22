import { useSettingsStore } from '../../stores/settingsStore';
import type { SettingsTabProps } from './settingsShared';

export function LayoutSettingsTab({ isActive }: SettingsTabProps) {
  const { layoutSettings, updateLayoutSettings } = useSettingsStore();

  if (!isActive) return null;

  return (
    <div className="space-y-6">
      <div className="mb-4">
        <h3 className="text-lg font-semibold text-slack-text mb-2">Layout preset</h3>
        <p className="text-sm text-slack-textMuted mb-3">
          IDE puts the editor and agent first; Team keeps the classic chat-first workspace.
        </p>
        <div className="flex gap-2">
          {(['team', 'ide'] as const).map((preset) => (
            <button
              key={preset}
              type="button"
              onClick={() => {
                void import('../../utils/layoutPresets').then(({ panelsForPreset }) =>
                  updateLayoutSettings(panelsForPreset(preset))
                );
              }}
              className={`px-3 py-1.5 text-sm rounded border ${
                (layoutSettings.layoutPreset ?? 'team') === preset
                  ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                  : 'border-slack-border text-slack-textMuted hover:text-slack-text'
              }`}
            >
              {preset === 'ide' ? 'IDE (project-first)' : 'Team (chat-first)'}
            </button>
          ))}
        </div>
      </div>

      <div className="mb-4">
        <h3 className="text-lg font-semibold text-slack-text mb-2">Toolbar actions</h3>
        <p className="text-sm text-slack-textMuted mb-3">
          On wide screens, show toolbar chips in the top bar or a right sidebar. Narrow windows always use the
          sidebar.
        </p>
        <div className="flex gap-2">
          {(['top', 'sidebar'] as const).map((placement) => (
            <button
              key={placement}
              type="button"
              onClick={() => updateLayoutSettings({ toolbarChipsPlacement: placement })}
              className={`px-3 py-1.5 text-sm rounded border ${
                (layoutSettings.toolbarChipsPlacement ?? 'top') === placement
                  ? 'border-slack-accent bg-slack-accent/20 text-slack-text'
                  : 'border-slack-border text-slack-textMuted hover:text-slack-text'
              }`}
            >
              {placement === 'top' ? 'Top bar' : 'Right sidebar'}
            </button>
          ))}
        </div>
      </div>

      <div className="mb-4">
        <h3 className="text-lg font-semibold text-slack-text mb-2">Panel Visibility</h3>
        <p className="text-sm text-slack-textMuted">
          Configure which panels are visible by default when the app starts. You can still toggle panels
          manually at any time.
        </p>
      </div>

      <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
        <div className="flex-1">
          <div className="font-medium text-slack-text">Files Panel</div>
          <div className="text-sm text-slack-textMuted">File explorer for browsing workspaces and files</div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={layoutSettings.filesPanelVisible}
            onChange={(e) => updateLayoutSettings({ filesPanelVisible: e.target.checked })}
            className="sr-only peer"
          />
          <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
        </label>
      </div>

      <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
        <div className="flex-1">
          <div className="font-medium text-slack-text">Editor Panel</div>
          <div className="text-sm text-slack-textMuted">Code editor for viewing and editing files</div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={layoutSettings.editorPanelVisible}
            onChange={(e) => updateLayoutSettings({ editorPanelVisible: e.target.checked })}
            className="sr-only peer"
          />
          <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
        </label>
      </div>

      <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
        <div className="flex-1">
          <div className="font-medium text-slack-text">Terminal Panel</div>
          <div className="text-sm text-slack-textMuted">Terminal for executing commands and viewing output</div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={layoutSettings.terminalPanelVisible}
            onChange={(e) => updateLayoutSettings({ terminalPanelVisible: e.target.checked })}
            className="sr-only peer"
          />
          <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
        </label>
      </div>

      <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
        <div className="flex-1">
          <div className="font-medium text-slack-text">My Agents Panel</div>
          <div className="text-sm text-slack-textMuted">Manage and view your custom repository agents</div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={layoutSettings.myAgentsPanelVisible}
            onChange={(e) => updateLayoutSettings({ myAgentsPanelVisible: e.target.checked })}
            className="sr-only peer"
          />
          <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
        </label>
      </div>

      <div className="flex items-center justify-between p-4 bg-slack-bgHover rounded-lg border border-slack-border">
        <div className="flex-1">
          <div className="font-medium text-slack-text">Sidebar agent shortcuts</div>
          <div className="text-sm text-slack-textMuted">
            Show agents without a DM under Direct Messages. Turn off for a cleaner list; open DMs stay.
          </div>
        </div>
        <label className="relative inline-flex items-center cursor-pointer">
          <input
            type="checkbox"
            checked={layoutSettings.sidebarAgentsVisible}
            onChange={(e) => updateLayoutSettings({ sidebarAgentsVisible: e.target.checked })}
            className="sr-only peer"
          />
          <div className="w-11 h-6 bg-gray-600 peer-focus:outline-none peer-focus:ring-4 peer-focus:ring-blue-800 rounded-full peer peer-checked:after:translate-x-full peer-checked:after:border-white after:content-[''] after:absolute after:top-[2px] after:left-[2px] after:bg-white after:border-gray-300 after:border after:rounded-full after:h-5 after:w-5 after:transition-all peer-checked:bg-blue-600"></div>
        </label>
      </div>
    </div>
  );
}
