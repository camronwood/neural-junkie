type Props = {
  ascii?: string;
  fen?: string;
  legalMoves: string[];
  onMove: (move: string) => void;
  disabled?: boolean;
};

export function ChessBoard({ ascii, fen, legalMoves, onMove, disabled }: Props) {
  return (
    <div className="arena-chess-frame">
      {fen && (
        <div className="text-xs uppercase tracking-wider text-amber-200/70 break-all mb-2">
          FEN · {fen}
        </div>
      )}
      {ascii && <pre className="arena-chess-pre">{ascii}</pre>}
      <div className="arena-chess-moves">
        {legalMoves.map((move) => (
          <button
            key={move}
            type="button"
            disabled={disabled}
            onClick={() => onMove(move)}
            className="arena-chess-move-btn"
          >
            {move}
          </button>
        ))}
      </div>
    </div>
  );
}
