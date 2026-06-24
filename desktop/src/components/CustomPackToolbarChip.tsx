import type { PackToolbarAction } from '../stores/packCapabilityRegistry';

const iconBtn =
  'w-7 h-7 rounded transition-colors flex items-center justify-center shrink-0 focus-visible:outline focus-visible:outline-2 focus-visible:outline-offset-2';

interface CustomPackToolbarChipProps {
  action: PackToolbarAction;
  onClick: () => void;
}

export function CustomPackToolbarChip({ action, onClick }: CustomPackToolbarChipProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`${iconBtn} bg-teal-700 hover:bg-teal-600 text-white text-[10px] font-bold focus-visible:outline-teal-400 overflow-hidden`}
      title={action.title}
      aria-label={action.title}
    >
      {action.iconUrl ? (
        <img
          src={action.iconUrl}
          alt=""
          className="h-5 w-5 object-contain"
          draggable={false}
        />
      ) : (
        action.label
      )}
    </button>
  );
}
