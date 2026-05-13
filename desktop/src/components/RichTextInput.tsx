import { useState, useRef, useEffect, forwardRef, useImperativeHandle, KeyboardEvent } from 'react';
import { MentionAutocomplete } from './MentionAutocomplete';
import type { AgentInfo } from '../types/protocol';
import { PROMPT_ATTACHMENTS_METADATA_KEY, type PromptAttachmentPayload } from '../constants/promptMetadata';

interface RichTextInputProps {
  onSend: (message: string, metadata?: Record<string, any>) => void;
  disabled?: boolean;
  placeholder?: string;
  agents?: AgentInfo[];
  onInsertMention?: (name: string) => void;
  onSlashTrigger?: (query: string) => void;
}

export const RichTextInput = forwardRef<HTMLTextAreaElement, RichTextInputProps>(
  function RichTextInput({
    onSend,
    disabled = false,
    placeholder = 'Type your message here...',
    agents = [],
    onInsertMention,
    onSlashTrigger,
  }, ref) {
  const [message, setMessage] = useState('');
  const [showMentionMenu, setShowMentionMenu] = useState(false);
  const [mentionQuery, setMentionQuery] = useState('');
  const [mentionStartPos, setMentionStartPos] = useState(0);
  const [selectedMentionIndex, setSelectedMentionIndex] = useState(0);
  const [selectedImage, setSelectedImage] = useState<{file: File, preview: string} | null>(null);
  const [isAnalyzing, setIsAnalyzing] = useState(false);
  const [pendingAttachments, setPendingAttachments] = useState<PromptAttachmentPayload[]>([]);
  const [dragActive, setDragActive] = useState(false);
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  // Filter agents for vision capability
  const visionAgents = agents.filter(agent => agent.supports_vision);
  const hasVisionAgents = visionAgents.length > 0;

  // Detect @ mentions
  useEffect(() => {
    if (!textareaRef.current) return;

    const cursorPos = textareaRef.current.selectionStart;
    const textBeforeCursor = message.substring(0, cursorPos);
    
    // Find the last @ symbol before cursor
    const lastAtIndex = textBeforeCursor.lastIndexOf('@');
    
    if (lastAtIndex !== -1) {
      const textAfterAt = textBeforeCursor.substring(lastAtIndex + 1);
      
      // Check if there's no space after @, which means we're still typing the mention
      if (!textAfterAt.includes(' ') && !textAfterAt.includes('\n')) {
        setShowMentionMenu(true);
        setMentionQuery(textAfterAt);
        setMentionStartPos(lastAtIndex);
        setSelectedMentionIndex(0);
        return;
      }
    }
    
    setShowMentionMenu(false);
  }, [message]);

  // Detect / commands at start of message
  const prevMessageRef = useRef('');
  useEffect(() => {
    if (!onSlashTrigger) return;
    const prev = prevMessageRef.current;
    prevMessageRef.current = message;

    // Only trigger when the user just typed '/' at position 0
    if (message.startsWith('/') && !prev.startsWith('/')) {
      onSlashTrigger(message.substring(1));
    }
  }, [message, onSlashTrigger]);

  // Filter agents for autocomplete
  const filteredAgents = agents.filter((agent) =>
    agent.name.toLowerCase().includes(mentionQuery.toLowerCase())
  );

  const insertMention = (agentName: string) => {
    if (!textareaRef.current) return;

    const beforeMention = message.substring(0, mentionStartPos);
    const cursorPos = textareaRef.current.selectionStart;
    const afterCursor = message.substring(cursorPos);
    
    const newMessage = `${beforeMention}@${agentName} ${afterCursor}`;
    setMessage(newMessage);
    setShowMentionMenu(false);
    
    // Set cursor position after the inserted mention
    setTimeout(() => {
      if (textareaRef.current) {
        const newCursorPos = mentionStartPos + agentName.length + 2; // +2 for @ and space
        textareaRef.current.selectionStart = newCursorPos;
        textareaRef.current.selectionEnd = newCursorPos;
        textareaRef.current.focus();
      }
    }, 0);

    // Notify parent if needed
    if (onInsertMention) {
      onInsertMention(agentName);
    }
  };

  const handleImageSelect = (event: React.ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    if (!file) return;

    // Validate file type
    const validTypes = ['image/png', 'image/jpeg', 'image/jpg', 'image/webp', 'image/gif'];
    if (!validTypes.includes(file.type)) {
      alert('Please select a valid image file (PNG, JPEG, WebP, or GIF)');
      return;
    }

    // Validate file size (5MB limit)
    if (file.size > 5 * 1024 * 1024) {
      alert('Image file is too large. Please select an image smaller than 5MB.');
      return;
    }

    // Create preview URL
    const preview = URL.createObjectURL(file);
    setSelectedImage({ file, preview });
  };

  const removeImage = () => {
    if (selectedImage) {
      URL.revokeObjectURL(selectedImage.preview);
      setSelectedImage(null);
    }
    if (fileInputRef.current) {
      fileInputRef.current.value = '';
    }
  };

  const inferLanguageFromPath = (path: string): string => {
    const ext = path.includes('.') ? path.slice(path.lastIndexOf('.') + 1).toLowerCase() : '';
    const m: Record<string, string> = {
      go: 'go',
      rs: 'rust',
      py: 'python',
      ts: 'typescript',
      tsx: 'tsx',
      js: 'javascript',
      jsx: 'jsx',
      md: 'markdown',
      json: 'json',
      yaml: 'yaml',
      yml: 'yaml',
      toml: 'toml',
      sql: 'sql',
      sh: 'bash',
      tf: 'hcl',
    };
    return m[ext] || 'text';
  };

  const binaryExt = new Set([
    'png', 'jpg', 'jpeg', 'gif', 'webp', 'ico', 'svg', 'bmp', 'zip', 'tar', 'gz', 'pdf', 'mp4', 'mp3', 'wav',
    'exe', 'dll', 'so', 'dylib', 'woff', 'woff2', 'ttf', 'eot', 'gguf', 'bin',
  ]);

  const isBinaryPath = (path: string) => {
    const ext = path.includes('.') ? path.slice(path.lastIndexOf('.') + 1).toLowerCase() : '';
    return binaryExt.has(ext);
  };

  const MAX_ATTACH_BYTES = 80_000;
  const MAX_ATTACH_COUNT = 12;
  const MAX_ATTACH_TOTAL = 350_000;

  const ingestDroppedFiles = async (files: FileList | null) => {
    if (!files?.length) return;
    const next: PromptAttachmentPayload[] = [...pendingAttachments];
    let total = next.reduce((s, x) => s + x.content.length, 0);
    for (let i = 0; i < files.length; i++) {
      if (next.length >= MAX_ATTACH_COUNT) break;
      const file = files[i];
      if (isBinaryPath(file.name)) continue;
      try {
        const text = await file.text();
        let slice = text;
        if (slice.length > MAX_ATTACH_BYTES) {
          slice = slice.slice(0, MAX_ATTACH_BYTES) + '\n[truncated client-side]';
        }
        if (total + slice.length > MAX_ATTACH_TOTAL) break;
        next.push({
          path: file.name,
          language: inferLanguageFromPath(file.name),
          content: slice,
        });
        total += slice.length;
      } catch {
        /* skip unreadable */
      }
    }
    setPendingAttachments(next);
  };

  const removeAttachmentAt = (idx: number) => {
    setPendingAttachments(prev => prev.filter((_, i) => i !== idx));
  };

  const handleAnalyzeDesign = async () => {
    if (!selectedImage) return;

    // Check if message contains @mentions
    const hasMentions = message.includes('@');
    if (!hasMentions) {
      alert('Please @mention which agent(s) should analyze the image. For example: "@FrontendAgent please analyze this design"');
      return;
    }

    setIsAnalyzing(true);
    try {
      // Convert file to base64
      const base64 = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => {
          const result = reader.result as string;
          // Remove data:image/...;base64, prefix
          const base64Data = result.split(',')[1];
          resolve(base64Data);
        };
        reader.onerror = reject;
        reader.readAsDataURL(selectedImage.file);
      });

      // Send message with image metadata and the message content (including @mentions)
      const metadata = {
        image_data: base64,
        image_type: selectedImage.file.type,
      };

      // Send the message content with @mentions along with the /analyze-design command
      onSend(`/analyze-design ${message}`, metadata);
      
      // Clear the image and message after sending
      removeImage();
      setMessage('');
    } catch (error) {
      console.error('Error processing image:', error);
      alert('Error processing image. Please try again.');
    } finally {
      setIsAnalyzing(false);
    }
  };

  const handleSend = () => {
    const trimmed = message.trim();
    if (trimmed && !disabled) {
      const composerMeta: Record<string, unknown> = {};
      if (pendingAttachments.length > 0) {
        composerMeta[PROMPT_ATTACHMENTS_METADATA_KEY] = pendingAttachments.map(({ path, language, content }) => ({
          path,
          language,
          content,
        }));
      }
      onSend(trimmed, Object.keys(composerMeta).length > 0 ? composerMeta : undefined);
      setMessage('');
      setPendingAttachments([]);
      setShowMentionMenu(false);
    }
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLTextAreaElement>) => {
    // Handle mention autocomplete navigation
    if (showMentionMenu && filteredAgents.length > 0) {
      if (e.key === 'ArrowDown') {
        e.preventDefault();
        setSelectedMentionIndex((prev) => 
          prev < filteredAgents.length - 1 ? prev + 1 : 0
        );
        return;
      }
      
      if (e.key === 'ArrowUp') {
        e.preventDefault();
        setSelectedMentionIndex((prev) => 
          prev > 0 ? prev - 1 : filteredAgents.length - 1
        );
        return;
      }
      
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault();
        insertMention(filteredAgents[selectedMentionIndex].name);
        return;
      }
      
      if (e.key === 'Escape') {
        e.preventDefault();
        setShowMentionMenu(false);
        return;
      }
    }

    // Send on Enter, new line on Shift+Enter
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  // Expose methods to parent via ref
  useImperativeHandle(ref, () => ({
    ...textareaRef.current!,
    clearInput: () => {
      setMessage('');
    },
    insertMentionText: (agentName: string) => {
      const cursorPos = textareaRef.current?.selectionStart || message.length;
      const beforeCursor = message.substring(0, cursorPos);
      const afterCursor = message.substring(cursorPos);
      
      // Check if we should add space before @mention
      const needsSpaceBefore = beforeCursor.length > 0 && !beforeCursor.endsWith(' ');
      const prefix = needsSpaceBefore ? ' ' : '';
      
      const newMessage = `${beforeCursor}${prefix}@${agentName} ${afterCursor}`;
      setMessage(newMessage);
      
      // Set cursor position after the inserted mention
      setTimeout(() => {
        if (textareaRef.current) {
          const newCursorPos = cursorPos + prefix.length + agentName.length + 2;
          textareaRef.current.selectionStart = newCursorPos;
          textareaRef.current.selectionEnd = newCursorPos;
          textareaRef.current.focus();
        }
      }, 0);
    },
  }), [message]);

  return (
    <div
      className={`relative flex flex-col gap-2 p-4 border-t border-slack-border bg-slack-bg rich-text-input ${dragActive ? 'ring-2 ring-slack-accent ring-inset rounded-lg' : ''}`}
      onDragEnter={(e) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(true);
      }}
      onDragLeave={(e) => {
        e.preventDefault();
        e.stopPropagation();
        if (e.currentTarget === e.target) setDragActive(false);
      }}
      onDragOver={(e) => {
        e.preventDefault();
        e.stopPropagation();
      }}
      onDrop={(e) => {
        e.preventDefault();
        e.stopPropagation();
        setDragActive(false);
        void ingestDroppedFiles(e.dataTransfer.files);
      }}
    >
      {/* Image Preview */}
      {selectedImage && (
        <div className="space-y-2">
          <div className="flex items-center gap-2 p-2 bg-slack-bgHover rounded-lg">
            <img 
              src={selectedImage.preview} 
              alt="Selected design mockup" 
              className="w-16 h-16 object-cover rounded border"
            />
            <div className="flex-1">
              <p className="text-sm text-slack-text font-medium">{selectedImage.file.name}</p>
              <p className="text-xs text-slack-textMuted">
                {(selectedImage.file.size / 1024 / 1024).toFixed(2)} MB
              </p>
            </div>
            <button
              onClick={removeImage}
              className="text-slack-textMuted hover:text-slack-text p-1"
              title="Remove image"
            >
              ✕
            </button>
          </div>
          <div className="text-xs text-slack-textMuted bg-blue-50 dark:bg-blue-900/20 p-2 rounded border-l-2 border-blue-400">
            💡 <strong>Tip:</strong> Type your message and @mention agent(s) to analyze the image, then click the 🎨 button.
            {visionAgents.length > 0 && (
              <span className="block mt-1">
                Available vision agents: {visionAgents.map(agent => `@${agent.name}`).join(', ')}
              </span>
            )}
          </div>
        </div>
      )}

      {pendingAttachments.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {pendingAttachments.map((a, idx) => (
            <div
              key={`${a.path}-${idx}`}
              className="flex items-center gap-1 px-2 py-1 rounded bg-slack-bgHover border border-slack-border text-xs text-slack-text max-w-full"
            >
              <span className="truncate" title={a.path}>
                {a.path}
              </span>
              <button
                type="button"
                onClick={() => removeAttachmentAt(idx)}
                className="text-slack-textMuted hover:text-slack-text shrink-0"
                aria-label={`Remove ${a.path}`}
              >
                ×
              </button>
            </div>
          ))}
        </div>
      )}

      {/* Mention Autocomplete */}
      {showMentionMenu && filteredAgents.length > 0 && (
        <MentionAutocomplete
          agents={filteredAgents}
          query={mentionQuery}
          selectedIndex={selectedMentionIndex}
          onSelect={(agent) => insertMention(agent.name)}
          position={{ top: 100, left: 20 }}
        />
      )}

      <div className="flex gap-2">
        <textarea
          ref={textareaRef}
          value={message}
          onChange={(e) => setMessage(e.target.value)}
          onKeyDown={handleKeyDown}
          disabled={disabled}
          placeholder={placeholder}
          className="flex-1 bg-slack-bgHover text-slack-text placeholder-slack-textMuted px-4 py-3 rounded-lg resize-none focus:outline-none focus:ring-2 focus:ring-slack-accent disabled:opacity-50 disabled:cursor-not-allowed"
          rows={3}
          style={{
            maxHeight: '200px',
            minHeight: '80px',
          }}
        />
        
        {/* Action Buttons */}
        <div className="flex flex-col gap-2">
          {/* Image Upload Button */}
          <input
            ref={fileInputRef}
            type="file"
            accept="image/png,image/jpeg,image/jpg,image/webp,image/gif"
            onChange={handleImageSelect}
            className="hidden"
          />
          <button
            onClick={() => fileInputRef.current?.click()}
            disabled={disabled || !hasVisionAgents}
            className="px-3 py-2 text-slack-textMuted hover:text-slack-text hover:bg-slack-bgHover rounded-lg transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
            title={hasVisionAgents ? `Upload design mockup (${visionAgents.length} vision-capable agent${visionAgents.length > 1 ? 's' : ''} available)` : "No vision-capable agents available"}
          >
            📷
          </button>

          {/* Analyze Design Button */}
          {selectedImage && (
            <button
              onClick={handleAnalyzeDesign}
              disabled={disabled || isAnalyzing}
              className="px-3 py-2 bg-slack-accent hover:bg-slack-accent/80 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
              title="Analyze design with @mentioned agents"
            >
              {isAnalyzing ? '⏳' : '🎨'}
            </button>
          )}

          {/* Send Button */}
          <button
            onClick={handleSend}
            disabled={disabled || !message.trim()}
            className="px-6 py-3 bg-slack-success hover:bg-slack-success/80 text-white rounded-lg font-medium transition-colors disabled:opacity-50 disabled:cursor-not-allowed"
          >
            Send
          </button>
        </div>
      </div>
    </div>
  );
});

