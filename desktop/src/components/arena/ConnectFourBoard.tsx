type Props = {
  board: string[][];
  legalMoves: string[];
  onColumn: (col: number) => void;
  disabled?: boolean;
};

export function ConnectFourBoard({ board, legalMoves, onColumn, disabled }: Props) {
  const legalSet = new Set(legalMoves.map((m) => String(m)));
  return (
    <div className="arena-c4-cabinet">
      <div className="arena-c4-drop-row">
        {Array.from({ length: 7 }, (_, col) => (
          <button
            key={`col-${col}`}
            type="button"
            disabled={disabled || !legalSet.has(String(col))}
            onClick={() => onColumn(col)}
            className="arena-c4-drop-btn"
            aria-label={`Drop disc in column ${col}`}
          >
            {col}
          </button>
        ))}
      </div>
      <div className="arena-c4-grid">
        {board.flatMap((row, r) =>
          row.map((cell, c) => (
            <div
              key={`${r}-${c}`}
              className={`arena-c4-cell ${
                cell === 'R' ? 'red' : cell === 'Y' ? 'yellow' : 'empty'
              }`}
            />
          )),
        )}
      </div>
    </div>
  );
}
