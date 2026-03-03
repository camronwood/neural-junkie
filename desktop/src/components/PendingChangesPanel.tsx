import { useEffect, useState } from 'react';
import { useFileChangeStore } from '../stores/fileChangeStore';
import { FileChangeCard } from './FileChangeCard';
import { FileChangePreview } from './FileChangePreview';
import type { FileChange } from '../types/protocol';

interface PendingChangesPanelProps {
  onClose: () => void;
}

export function PendingChangesPanel({ onClose }: PendingChangesPanelProps) {
  const {
    pendingChanges,
    loading,
    error,
    fetchPendingChanges,
    approveChange,
    rejectChange,
    selectChange,
    clearError,
    refreshChanges,
  } = useFileChangeStore();

  const [selectedChange, setSelectedChange] = useState<FileChange | null>(null);

  // Fetch changes on mount
  useEffect(() => {
    fetchPendingChanges();
  }, [fetchPendingChanges]);

  // Auto-refresh every 30 seconds
  useEffect(() => {
    const interval = setInterval(() => {
      refreshChanges();
    }, 30000);

    return () => clearInterval(interval);
  }, [refreshChanges]);

  const handlePreview = (change: FileChange) => {
    setSelectedChange(change);
    selectChange(change.id);
  };

  const handleClosePreview = () => {
    setSelectedChange(null);
    selectChange(null);
  };

  const handleApprove = async (changeId: string) => {
    try {
      await approveChange(changeId);
      if (selectedChange?.id === changeId) {
        handleClosePreview();
      }
    } catch (error) {
      console.error('Failed to approve change:', error);
    }
  };

  const handleReject = async (changeId: string, reason: string) => {
    try {
      await rejectChange(changeId, reason);
      if (selectedChange?.id === changeId) {
        handleClosePreview();
      }
    } catch (error) {
      console.error('Failed to reject change:', error);
    }
  };

  const getOperationCounts = () => {
    // Ensure pendingChanges is always an array
    const changes = Array.isArray(pendingChanges) ? pendingChanges : [];
    
    const counts = {
      create: 0,
      edit: 0,
      delete: 0,
      move: 0,
      total: changes.length,
    };

    changes.forEach(change => {
      counts[change.operation as keyof typeof counts]++;
    });

    return counts;
  };

  const counts = getOperationCounts();

  return (
    <div 
      className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50"
      onClick={(e) => {
        if (e.target === e.currentTarget) {
          onClose();
        }
      }}
    >
      <div className="bg-white rounded-lg shadow-xl max-w-6xl w-full mx-4 max-h-[90vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between p-4 border-b">
          <div>
            <h2 className="text-xl font-semibold">Pending File Changes</h2>
            <p className="text-sm text-gray-600">
              {counts.total} pending changes
              {counts.create > 0 && ` • ${counts.create} create`}
              {counts.edit > 0 && ` • ${counts.edit} edit`}
              {counts.delete > 0 && ` • ${counts.delete} delete`}
              {counts.move > 0 && ` • ${counts.move} move`}
            </p>
          </div>
          <div className="flex items-center space-x-2">
            <button
              onClick={refreshChanges}
              disabled={loading}
              className="px-3 py-1 text-sm bg-blue-600 text-white rounded hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500 disabled:opacity-50"
            >
              {loading ? '⟳' : '🔄'} Refresh
            </button>
            <button
              onClick={onClose}
              className="p-1.5 rounded text-gray-500 hover:text-gray-700 hover:bg-gray-100 text-xl leading-none"
              aria-label="Close pending changes panel"
            >
              ×
            </button>
          </div>
        </div>

        {/* Error Message */}
        {error && (
          <div className="mx-4 mt-4 p-3 bg-red-50 border border-red-200 rounded-lg">
            <div className="flex items-center justify-between">
              <div className="flex items-center">
                <span className="text-red-600 mr-2">⚠️</span>
                <span className="text-red-800">{error}</span>
              </div>
              <button
                onClick={clearError}
                className="text-red-600 hover:text-red-800"
              >
                ×
              </button>
            </div>
          </div>
        )}

        {/* Content */}
        <div className="flex-1 overflow-y-auto p-4">
          {loading && (Array.isArray(pendingChanges) ? pendingChanges.length === 0 : true) ? (
            <div className="flex items-center justify-center p-8">
              <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
              <span className="ml-2 text-gray-600">Loading changes...</span>
            </div>
          ) : (Array.isArray(pendingChanges) ? pendingChanges.length === 0 : true) ? (
            <div className="text-center text-gray-500 p-8">
              <div className="text-4xl mb-4">📝</div>
              <h3 className="text-lg font-medium mb-2">No Pending Changes</h3>
              <p className="text-sm">All file changes have been processed.</p>
            </div>
          ) : (
            <div className="space-y-3">
              {(Array.isArray(pendingChanges) ? pendingChanges : []).map((change) => (
                <FileChangeCard
                  key={change.id}
                  change={change}
                  onPreview={handlePreview}
                  onApprove={handleApprove}
                  onReject={handleReject}
                />
              ))}
            </div>
          )}
        </div>

        {/* Footer */}
        <div className="border-t p-4 bg-gray-50">
          <div className="flex items-center justify-between text-sm text-gray-600">
            <div>
              💡 <strong>Tip:</strong> Click "Preview" to see detailed changes before approving
            </div>
            <div>
              Auto-refresh: 30s
            </div>
          </div>
        </div>
      </div>

      {/* Preview Modal */}
      {selectedChange && (
        <FileChangePreview
          change={selectedChange}
          onClose={handleClosePreview}
          onApprove={handleApprove}
          onReject={handleReject}
        />
      )}
    </div>
  );
}
