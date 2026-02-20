import { useEffect, useRef, useCallback } from 'react';
import { Terminal } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { terminalAPI } from '../api/terminalAPI';
import '@xterm/xterm/css/xterm.css';

interface XTerminalProps {
  sessionId: string;
  cwd?: string;
  isActive: boolean;
}

export function XTerminal({ sessionId, cwd, isActive }: XTerminalProps) {
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

    const term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: "'JetBrains Mono', 'Fira Code', 'SF Mono', Menlo, Monaco, 'Courier New', monospace",
      theme: {
        background: '#1a1b26',
        foreground: '#c0caf5',
        cursor: '#c0caf5',
        selectionBackground: '#33467c',
        black: '#15161e',
        red: '#f7768e',
        green: '#9ece6a',
        yellow: '#e0af68',
        blue: '#7aa2f7',
        magenta: '#bb9af7',
        cyan: '#7dcfff',
        white: '#a9b1d6',
        brightBlack: '#414868',
        brightRed: '#f7768e',
        brightGreen: '#9ece6a',
        brightYellow: '#e0af68',
        brightBlue: '#7aa2f7',
        brightMagenta: '#bb9af7',
        brightCyan: '#7dcfff',
        brightWhite: '#c0caf5',
      },
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

    term.attachCustomKeyEventHandler((event) => {
      if ((event.metaKey || event.ctrlKey) && event.key === 'k' && event.type === 'keydown') {
        term.clear();
        return false;
      }
      return true;
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
    if (isActive && fitRef.current) {
      requestAnimationFrame(() => {
        fitRef.current?.fit();
        termRef.current?.focus();
      });
    }
  }, [isActive]);

  return (
    <div
      ref={containerRef}
      className="w-full h-full"
      style={{ padding: '4px 0 0 8px' }}
    />
  );
}
