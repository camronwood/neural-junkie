import { useCallback, useMemo } from 'react';
import { parseCsvTable, serializeCsvTable } from '../utils/csvTable';

interface CsvTableViewerProps {
  content: string;
  onContentChange: (csv: string) => void;
}

export function CsvTableViewer({ content, onContentChange }: CsvTableViewerProps) {
  const rows = useMemo(() => parseCsvTable(content), [content]);

  const updateCell = useCallback(
    (rowIdx: number, colIdx: number, value: string) => {
      const next = rows.map((row, ri) =>
        ri === rowIdx ? row.map((cell, ci) => (ci === colIdx ? value : cell)) : [...row],
      );
      onContentChange(serializeCsvTable(next));
    },
    [rows, onContentChange],
  );

  const addRow = useCallback(() => {
    const colCount = rows[0]?.length ?? 1;
    onContentChange(serializeCsvTable([...rows, Array(colCount).fill('')]));
  }, [rows, onContentChange]);

  const addColumn = useCallback(() => {
    onContentChange(serializeCsvTable(rows.map((row) => [...row, ''])));
  }, [rows, onContentChange]);

  const deleteRow = useCallback(
    (rowIdx: number) => {
      if (rows.length <= 1) return;
      onContentChange(serializeCsvTable(rows.filter((_, i) => i !== rowIdx)));
    },
    [rows, onContentChange],
  );

  return (
    <div className="flex flex-col h-full min-h-0 bg-slack-bg">
      <div className="flex items-center gap-2 px-3 py-2 border-b border-slack-border bg-slack-bgHover/50 shrink-0">
        <span className="text-xs text-slack-textMuted">
          {rows.length} row{rows.length === 1 ? '' : 's'} · {rows[0]?.length ?? 0} column
          {(rows[0]?.length ?? 0) === 1 ? '' : 's'}
        </span>
        <button
          type="button"
          onClick={addRow}
          className="px-2 py-1 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover"
        >
          Add row
        </button>
        <button
          type="button"
          onClick={addColumn}
          className="px-2 py-1 text-xs rounded border border-slack-border text-slack-text hover:bg-slack-bgHover"
        >
          Add column
        </button>
      </div>
      <div className="flex-1 min-h-0 overflow-auto">
        <table className="w-full border-collapse text-sm">
          <tbody>
            {rows.map((row, rowIdx) => (
              <tr key={rowIdx} className="border-b border-slack-border/60 hover:bg-slack-bgHover/30">
                <td className="w-8 px-1 py-0.5 text-[10px] text-slack-textMuted text-right align-top sticky left-0 bg-slack-bg border-r border-slack-border/40">
                  <div className="flex flex-col items-end gap-0.5">
                    <span>{rowIdx + 1}</span>
                    {rows.length > 1 && (
                      <button
                        type="button"
                        title="Delete row"
                        onClick={() => deleteRow(rowIdx)}
                        className="text-red-400/70 hover:text-red-300 text-[9px]"
                      >
                        ×
                      </button>
                    )}
                  </div>
                </td>
                {row.map((cell, colIdx) => (
                  <td key={colIdx} className="p-0 min-w-[6rem] max-w-[20rem] border-r border-slack-border/30">
                    <input
                      type="text"
                      value={cell}
                      onChange={(e) => updateCell(rowIdx, colIdx, e.target.value)}
                      className="w-full px-2 py-1.5 bg-transparent text-slack-text text-xs font-mono outline-none focus:bg-slack-bgHover focus:ring-1 focus:ring-inset focus:ring-slack-accent/50"
                    />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
