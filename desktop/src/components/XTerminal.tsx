import { useEffect, useRef, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { terminalAPI } from '../api/terminalAPI';
import { useSettingsStore } from '../stores/settingsStore';
import { useTerminalStore } from '../stores/terminalStore';
import { getTerminalTheme } from '../utils/editorThemes';
import '@xterm/xterm/css/xterm.css';

interface XTerminalProps {
  sessionId: string;
  cwd?: string;
  isActive: boolean;
}

export function XTerminal({ sessionId, cwd, isActive }: XTerminalProps) {
  const colorTheme = useSettingsStore((s) => s.settings.colorTheme ?? 'slack');
  const clearBufferNonce = useTerminalStore((s) => s.clearBufferNonce);
  const containerRef = useRef<HTMLDivElement>(null);
  const termRef = useRef<Terminal | null>(null);
  const fitRef = useRef<FitAddon | null>(null);
  const initializedRef = useRef(false);

  const writeToTerminal = useCallback((data: string) => {
    termRef.current?.write(data);
  }, []);

  useEffect(() => {
    if (!containerRef.current || initializedRef.current) return;
    initializedRef.current = true;

    const themeAtInit = useSettingsStore.getState().settings.colorTheme ?? 'slack';
    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'SF Mono', Menlo, Monaco, 'Courier New', monospace",
      theme: getTerminalTheme(themeAtInit),
      allowProposedApi: true,
      scrollback: 10000,
    });

    const fit = new FitAddon();
    const webLinks = new WebLinksAddon();
    term.loadAddon(fit);
    term.loadAddon(webLinks);

    termRef.current = term;
    fitRef.current = fit;

    term.open(containerRef.current);

    requestAnimationFrame(() => {
      fit.fit();
    });

    const cols = term.cols;
    const rows = term.rows;

    terminalAPI
      .createPtySession(sessionId, cwd, cols, rows)
      .catch((err) => {
        term.writeln(`\r\n\x1b[31mFailed to create PTY session: ${err}\x1b[0m`);
      });

    let unlistenPty: (() => void) | null = null;
    terminalAPI.onPtyOutput((payload) => {
      if (payload.id === sessionId) {
        term.write(payload.data);
      }
    }).then((unlisten) => {
      unlistenPty = unlisten;
    });


    const onDataDispose = term.onData((data) => {
      terminalAPI.writePtySession(sessionId, data).catch(() => {});
    });

    const resizeObserver = new ResizeObserver(() => {
      requestAnimationFrame(() => {
        if (fitRef.current) {
          fitRef.current.fit();
          if (termRef.current) {
            terminalAPI
              .resizePtySession(sessionId, termRef.current.cols, termRef.current.rows)
              .catch(() => {});
          }
        }
      });
    });
    resizeObserver.observe(containerRef.current);

    return () => {
      resizeObserver.disconnect();
      onDataDispose.dispose();
      unlistenPty?.();
      terminalAPI.closePtySession(sessionId).catch(() => {});
      term.dispose();
      termRef.current = null;
      fitRef.current = null;
      initializedRef.current = false;
    };
  }, [sessionId, cwd, writeToTerminal]);

  useEffect(() => {
    if (termRef.current) {
      termRef.current.options.theme = getTerminalTheme(colorTheme);
    }
  }, [colorTheme]);

  useEffect(() => {
    if (isActive && fitRef.current) {
      requestAnimationFrame(() => {
        fitRef.current?.fit();
        termRef.current?.focus();
      });
    }
  }, [isActive]);

  useEffect(() => {
    if (clearBufferNonce > 0) {
      termRef.current?.clear();
    }
  }, [clearBufferNonce]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full"
      style={{ padding: '4px 0 0 8px' }}
    />
  );
}
