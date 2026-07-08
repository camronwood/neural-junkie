import { useMemo, useState } from 'react';
import { useChatStore } from '../stores/chatStore';
import { usePacksStore } from '../stores/packsStore';
import { useRoomStore } from '../stores/roomStore';

interface RoomChatModalProps {
  isOpen: boolean;
  onClose: () => void;
}

type RoomTab = 'host' | 'join';

export function RoomChatModal({ isOpen, onClose }: RoomChatModalProps) {
  const hasRoomChat = usePacksStore((s) => s.hasCapability('room-chat'));
  const username = useChatStore((s) => s.username);

  const {
    mode,
    room,
    roomChannel,
    joinCode,
    hostHubUrlInput,
    joining,
    creating,
    error,
    setHostHubUrlInput,
    clearError,
    createRoom,
    joinRoom,
    leaveRoom,
    endRoom,
  } = useRoomStore();

  const [tab, setTab] = useState<RoomTab>('host');
  const [joinCodeInput, setJoinCodeInput] = useState('');
  const [joinUsernameInput, setJoinUsernameInput] = useState('');

  const effectiveJoinUsername = useMemo(() => {
    const v = joinUsernameInput.trim();
    if (v) return v;
    const u = username.trim();
    if (u) return u;
    return 'Anonymous';
  }, [joinUsernameInput, username]);

  if (!isOpen) return null;

  return (
    <div className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto py-6 px-4" role="presentation">
      <div
        className="fixed inset-0 bg-black/60"
        onClick={() => {
          clearError();
          onClose();
        }}
        aria-hidden
      />
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="room-chat-title"
        className="relative z-10 flex w-full max-w-2xl flex-col overflow-hidden rounded-xl border border-indigo-700/50 bg-slack-bg shadow-2xl max-h-[min(90vh,760px)]"
      >
        <div className="flex shrink-0 items-center justify-between border-b border-slack-border px-4 py-3">
          <div>
            <h2 id="room-chat-title" className="text-lg font-semibold text-indigo-200">
              Room chat
            </h2>
            <p className="text-xs text-gray-500 mt-0.5">Same-room LAN chat (room-chat)</p>
          </div>
          <button
            type="button"
            onClick={() => {
              clearError();
              onClose();
            }}
            className="text-slack-textMuted hover:text-slack-text px-2 py-1 rounded hover:bg-slack-bgHover"
          >
            ✕
          </button>
        </div>

        <div className="flex-1 overflow-y-auto p-4 space-y-4">
          {!hasRoomChat && (
            <p className="text-sm text-amber-400">
              Enable a domain pack with <code className="font-mono">room-chat</code> in Settings → Domain packs.
            </p>
          )}

          {hasRoomChat && (
            <>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setTab('host')}
                  className={`px-3 py-1.5 text-xs rounded-lg border ${
                    tab === 'host'
                      ? 'border-indigo-600 bg-indigo-600/20 text-indigo-200'
                      : 'border-slack-border text-gray-300 hover:bg-slack-bgHover'
                  }`}
                >
                  Host
                </button>
                <button
                  type="button"
                  onClick={() => setTab('join')}
                  className={`px-3 py-1.5 text-xs rounded-lg border ${
                    tab === 'join'
                      ? 'border-indigo-600 bg-indigo-600/20 text-indigo-200'
                      : 'border-slack-border text-gray-300 hover:bg-slack-bgHover'
                  }`}
                >
                  Join
                </button>
              </div>

              {error && (
                <div className="rounded-lg border border-red-900/40 bg-red-950/20 p-3 text-xs text-red-200">
                  {error}
                </div>
              )}

              {tab === 'host' && (
                <div className="space-y-3 rounded-lg border border-indigo-800/40 bg-indigo-950/20 p-4">
                  <h3 className="text-sm font-semibold text-indigo-200">Start a room</h3>
                  <p className="text-xs text-gray-400">
                    This creates an ephemeral room on your local hub. Others join by entering your hub URL and the join code.
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={creating || mode === 'host'}
                      onClick={() => void createRoom()}
                      className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 disabled:opacity-40"
                    >
                      {mode === 'host' ? 'Room active' : creating ? 'Starting…' : 'Start room'}
                    </button>
                    {mode === 'host' && (
                      <button
                        type="button"
                        onClick={() => void endRoom()}
                        className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover"
                      >
                        End room
                      </button>
                    )}
                  </div>
                  {mode === 'host' && room && (
                    <div className="space-y-1 text-xs text-gray-300">
                      <div>
                        <span className="text-gray-500">Join code:</span>{' '}
                        <span className="font-mono tracking-widest text-white">{joinCode}</span>
                      </div>
                      <div>
                        <span className="text-gray-500">Channel:</span>{' '}
                        <span className="font-mono text-white">{roomChannel ?? '(unknown)'}</span>
                      </div>
                    </div>
                  )}
                </div>
              )}

              {tab === 'join' && (
                <div className="space-y-3 rounded-lg border border-indigo-800/40 bg-indigo-950/20 p-4">
                  <h3 className="text-sm font-semibold text-indigo-200">Join a room</h3>
                  <div className="space-y-2">
                    <label className="block text-xs text-gray-400">
                      Host hub URL
                      <input
                        value={hostHubUrlInput}
                        onChange={(e) => setHostHubUrlInput(e.target.value)}
                        placeholder="http://192.168.1.10:18765"
                        className="mt-1 w-full rounded-md border border-slack-border bg-slack-bg px-2 py-1 text-xs text-gray-100 placeholder:text-gray-600"
                      />
                    </label>
                    <label className="block text-xs text-gray-400">
                      Join code
                      <input
                        value={joinCodeInput}
                        onChange={(e) => setJoinCodeInput(e.target.value)}
                        placeholder="ABC123"
                        className="mt-1 w-full rounded-md border border-slack-border bg-slack-bg px-2 py-1 text-xs font-mono tracking-widest text-gray-100 placeholder:tracking-normal placeholder:font-sans placeholder:text-gray-600"
                      />
                    </label>
                    <label className="block text-xs text-gray-400">
                      Username
                      <input
                        value={joinUsernameInput}
                        onChange={(e) => setJoinUsernameInput(e.target.value)}
                        placeholder={username?.trim() ? username : 'Anonymous'}
                        className="mt-1 w-full rounded-md border border-slack-border bg-slack-bg px-2 py-1 text-xs text-gray-100 placeholder:text-gray-600"
                      />
                    </label>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    <button
                      type="button"
                      disabled={joining || mode === 'guest'}
                      onClick={() =>
                        void joinRoom({
                          hostHubUrl: hostHubUrlInput,
                          joinCode: joinCodeInput,
                          username: effectiveJoinUsername,
                        })
                      }
                      className="px-3 py-1.5 text-xs font-medium rounded-lg bg-indigo-600 text-white hover:bg-indigo-500 disabled:opacity-40"
                    >
                      {mode === 'guest' ? 'In room' : joining ? 'Joining…' : 'Join room'}
                    </button>
                    {mode === 'guest' && (
                      <button
                        type="button"
                        onClick={() => void leaveRoom()}
                        className="px-3 py-1.5 text-xs rounded-lg border border-slack-border text-gray-300 hover:bg-slack-bgHover"
                      >
                        Leave room
                      </button>
                    )}
                  </div>
                  {mode === 'guest' && room && (
                    <div className="space-y-1 text-xs text-gray-300">
                      <div>
                        <span className="text-gray-500">Room:</span>{' '}
                        <span className="font-mono text-white">{room?.name ?? room?.id}</span>
                      </div>
                      <div>
                        <span className="text-gray-500">Channel:</span>{' '}
                        <span className="font-mono text-white">{roomChannel}</span>
                      </div>
                    </div>
                  )}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

