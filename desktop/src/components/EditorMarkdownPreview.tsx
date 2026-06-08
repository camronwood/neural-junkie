import { RichMarkdownView } from './RichMarkdownView';

interface EditorMarkdownPreviewProps {
  content: string;
}

export function EditorMarkdownPreview({ content }: EditorMarkdownPreviewProps) {
  return (
    <div className="h-full overflow-auto bg-slack-bg">
      <div className="max-w-4xl mx-auto p-6">
        <RichMarkdownView content={content} />
      </div>
    </div>
  );
}
