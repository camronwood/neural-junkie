import type { SlackSidebarControlsState } from '../hooks/useSlackSidebarControls';
import { SlackIcon } from './icons/SlackIcon';

interface SlackSidebarChipProps {
  controls: SlackSidebarControlsState;
}

function chipBtnClass(active: boolean, accent: 'amber' | 'sky'): string {
  const base =
    'text-[10px] font-semibold uppercase tracking-wide px-1.5 py-0.5 rounded border disabled:opacity-50 transition-colors';
  if (active) {
    return accent === 'amber'
      ? `${base} border-amber-400/70 bg-amber-500/20 text-amber-100 hover:bg-amber-500/30`
      : `${base} border-sky-400/70 bg-sky-500/20 text-sky-100 hover:bg-sky-500/30`;
  }
  return `${base} border-white/20 text-white/55 hover:bg-white/10 hover:text-white/80`;
}

export function SlackSidebarChip({ controls }: SlackSidebarChipProps) {
  const {
    awayVisible,
    awayEnabled,
    awayMonitoringActive,
    forwardVisible,
    forwardEnabled,
    inboxEnabled,
    loading,
    awayToggling,
    forwardToggling,
    toggleAway,
    toggleForward,
  } = controls;

  if (!awayVisible && !forwardVisible) {
    return null;
  }

  return (
    <div
      className="inline-flex items-center gap-1 rounded-md border border-white/15 bg-white/5 px-1 py-0.5"
      title="Slack inbox controls"
    >
      <SlackIcon className="w-3.5 h-3.5 shrink-0" />
      {awayVisible ? (
        <button
          type="button"
          onClick={() => void toggleAway()}
          disabled={awayToggling || loading}
          className={chipBtnClass(
            awayEnabled,
            awayMonitoringActive && awayEnabled ? 'amber' : 'amber'
          )}
          title={
            awayEnabled
              ? awayMonitoringActive
                ? 'Away on — monitoring human Slack DMs. Click to turn off.'
                : 'Away on — click to turn off manual away mode.'
              : awayMonitoringActive
                ? 'Monitoring via schedule. Click to turn on manual away.'
                : 'Away off — click to monitor human Slack DMs while away.'
          }
        >
          {awayToggling ? '…' : awayEnabled ? 'Away' : 'Away off'}
        </button>
      ) : null}
      {forwardVisible ? (
        <button
          type="button"
          onClick={() => void toggleForward()}
          disabled={forwardToggling || loading || !inboxEnabled}
          className={chipBtnClass(forwardEnabled, 'sky')}
          title={
            !inboxEnabled
              ? 'Enable personal inbox in Settings → Slack first'
              : forwardEnabled
                ? 'Forward on — Slack channel messages appear in your inbox; reply here to post back to Slack. Click to turn off.'
                : 'Forward off — click to forward watched Slack channel messages into NJ so you can reply without switching apps.'
          }
        >
          {forwardToggling ? '…' : forwardEnabled ? 'Forward' : 'Forward off'}
        </button>
      ) : null}
    </div>
  );
}
