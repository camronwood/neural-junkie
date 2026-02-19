import { useState } from 'react';
import type { Message } from '../types/protocol';

interface DesignOutputProps {
  message: Message;
}

export function DesignOutput({ message }: DesignOutputProps) {
  const [showPreview, setShowPreview] = useState(false);
  const [previewError, setPreviewError] = useState<string | null>(null);

  const outputDir = message.metadata?.output_directory as string;
  const cssFile = message.metadata?.css_file as string;
  const htmlFile = message.metadata?.html_file as string;
  const markdownFile = message.metadata?.markdown_file as string;
  const zipFile = message.metadata?.zip_file as string;
  const analysis = message.metadata?.analysis as string;

  const handleDownload = async (filePath: string, filename: string) => {
    try {
      // In a real implementation, this would download from the server
      // For now, we'll show a placeholder
      alert(`Downloading ${filename}...\n\nFile path: ${filePath}\n\n(Download functionality will be implemented with server endpoints)`);
    } catch (error) {
      console.error('Download failed:', error);
      alert('Download failed. Please try again.');
    }
  };

  const handlePreview = () => {
    if (showPreview) {
      setShowPreview(false);
      setPreviewError(null);
    } else {
      setShowPreview(true);
      // In a real implementation, this would load the HTML preview
      // For now, we'll show a placeholder
      setPreviewError('Preview functionality will be implemented with server endpoints');
    }
  };

  return (
    <div className="bg-slack-bgHover rounded-lg p-4 border border-slack-border">
      <div className="flex items-center gap-2 mb-3">
        <span className="text-2xl">🎨</span>
        <h3 className="text-lg font-semibold text-slack-text">Design Analysis Complete</h3>
      </div>

      {/* Analysis Summary */}
      {analysis && (
        <div className="mb-4">
          <h4 className="text-sm font-medium text-slack-text mb-2">Analysis Summary:</h4>
          <div className="bg-slack-bg rounded p-3 text-sm text-slack-textMuted max-h-32 overflow-y-auto">
            {analysis.length > 200 ? `${analysis.substring(0, 200)}...` : analysis}
          </div>
        </div>
      )}

      {/* Generated Files */}
      <div className="mb-4">
        <h4 className="text-sm font-medium text-slack-text mb-2">Generated Files:</h4>
        <div className="grid grid-cols-2 gap-2">
          {htmlFile && (
            <button
              onClick={() => handleDownload(htmlFile, 'demo.html')}
              className="flex items-center gap-2 p-2 bg-slack-bg hover:bg-slack-accent/10 rounded text-sm text-slack-text transition-colors"
            >
              <span>📄</span>
              <span>demo.html</span>
            </button>
          )}
          {cssFile && (
            <button
              onClick={() => handleDownload(cssFile, 'style.css')}
              className="flex items-center gap-2 p-2 bg-slack-bg hover:bg-slack-accent/10 rounded text-sm text-slack-text transition-colors"
            >
              <span>🎨</span>
              <span>style.css</span>
            </button>
          )}
          {markdownFile && (
            <button
              onClick={() => handleDownload(markdownFile, 'style-guide.md')}
              className="flex items-center gap-2 p-2 bg-slack-bg hover:bg-slack-accent/10 rounded text-sm text-slack-text transition-colors"
            >
              <span>📝</span>
              <span>style-guide.md</span>
            </button>
          )}
          {zipFile && (
            <button
              onClick={() => handleDownload(zipFile, 'design-output.zip')}
              className="flex items-center gap-2 p-2 bg-slack-success/20 hover:bg-slack-success/30 rounded text-sm text-slack-text transition-colors"
            >
              <span>📦</span>
              <span>All Files (ZIP)</span>
            </button>
          )}
        </div>
      </div>

      {/* Preview Section */}
      {htmlFile && (
        <div className="mb-4">
          <div className="flex items-center gap-2 mb-2">
            <button
              onClick={handlePreview}
              className="flex items-center gap-2 px-3 py-1 bg-slack-accent hover:bg-slack-accent/80 text-white rounded text-sm transition-colors"
            >
              <span>{showPreview ? '👁️' : '👁️'}</span>
              <span>{showPreview ? 'Hide Preview' : 'Preview HTML'}</span>
            </button>
          </div>
          
          {showPreview && (
            <div className="bg-slack-bg rounded border border-slack-border p-4">
              {previewError ? (
                <div className="text-center text-slack-textMuted py-8">
                  <p>{previewError}</p>
                </div>
              ) : (
                <div className="text-center text-slack-textMuted py-8">
                  <p>HTML Preview will be displayed here</p>
                  <p className="text-xs mt-2">(Preview functionality coming soon)</p>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Design Tokens Preview */}
      <div className="text-xs text-slack-textMuted">
        <p>💡 <strong>Tip:</strong> Download the files to see the complete CSS style guide and HTML demo.</p>
        <p>📁 Files are stored in: <code className="bg-slack-bg px-1 rounded">{outputDir}</code></p>
      </div>
    </div>
  );
}
