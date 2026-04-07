import React, { useState } from 'react';
import { workspaces } from '../services/api';
import { WorkspaceExportModal, WorkspaceImportModal } from './WorkspaceExportModal';

interface Workspace {
  id: number;
  tag: string;
  name: string;
  organization_id: number;
  owner_id: number;
  provider: string;
  status: string;
  config?: string;
  region?: string;
  created_at: string;
  updated_at: string;
}

interface WorkspaceCardProps {
  workspace: Workspace;
  organizationId: number;
  onDelete: () => void;
  onStatusChange: () => void;
}

interface ActionButtonProps {
  onClick: () => void;
  disabled: boolean;
  loading: boolean;
  variant: 'start' | 'stop' | 'restart' | 'delete';
  children: React.ReactNode;
}

const ActionButton: React.FC<ActionButtonProps> = ({ onClick, disabled, loading, variant, children }) => {
  const variantClasses = {
    start: 'bg-green-600 hover:bg-green-700 focus:ring-green-500',
    stop: 'bg-gray-600 hover:bg-gray-700 focus:ring-gray-500',
    restart: 'bg-blue-600 hover:bg-blue-700 focus:ring-blue-500',
    delete: 'bg-red-600 hover:bg-red-700 focus:ring-red-500',
  };

  return (
    <button
      onClick={onClick}
      disabled={disabled || loading}
      className={`
        flex items-center justify-center px-4 py-2 text-sm font-medium text-white rounded-lg
        ${variantClasses[variant]}
        disabled:opacity-50 disabled:cursor-not-allowed
        focus:outline-none focus:ring-2 focus:ring-offset-2
        transition-all duration-200
      `}
    >
      {loading && (
        <svg className="animate-spin -ml-1 mr-2 h-4 w-4" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
          <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
          <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
        </svg>
      )}
      {children}
    </button>
  );
};

const StatusBadge: React.FC<{ status: string }> = ({ status }) => {
  const statusInfo: Record<string, { color: string; icon: string }> = {
    running: { color: 'bg-green-100 text-green-800', icon: '●' },
    stopped: { color: 'bg-gray-100 text-gray-800', icon: '○' },
    pending: { color: 'bg-yellow-100 text-yellow-800', icon: '◷' },
    provisioning: { color: 'bg-blue-100 text-blue-800', icon: '◷' },
    error: { color: 'bg-red-100 text-red-800', icon: '✕' },
    deleted: { color: 'bg-red-100 text-red-800', icon: '✕' },
  };

  const info = statusInfo[status] || { color: 'bg-blue-100 text-blue-800', icon: '●' };

  return (
    <span className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${info.color}`}>
      <span className="mr-1.5">{info.icon}</span>
      {status.charAt(0).toUpperCase() + status.slice(1)}
    </span>
  );
};

function WorkspaceCard({ workspace, organizationId, onDelete, onStatusChange }: WorkspaceCardProps) {
  const [isDeleting, setIsDeleting] = useState(false);
  const [actionState, setActionState] = useState<{ action: string; loading: boolean }>({ action: '', loading: false });
  const [error, setError] = useState('');
  const [showExportModal, setShowExportModal] = useState(false);
  const [showImportModal, setShowImportModal] = useState(false);

  const executeAction = async (action: 'start' | 'stop' | 'restart', actionName: string) => {
    setError('');
    setActionState({ action: actionName, loading: true });

    try {
      switch (action) {
        case 'start':
          await workspaces.start(workspace.id);
          break;
        case 'stop':
          await workspaces.stop(workspace.id);
          break;
        case 'restart':
          await workspaces.restart(workspace.id);
          break;
      }
      onStatusChange();
    } catch (err) {
      console.error(`Failed to ${action} workspace:`, err);
      setError(`Failed to ${actionName} workspace. Please try again.`);
    } finally {
      setActionState({ action: '', loading: false });
      setTimeout(() => setError(''), 5000);
    }
  };

  const handleStart = () => executeAction('start', 'Start');
  const handleStop = () => {
    if (!confirm(`Are you sure you want to stop "${workspace.name}"?`)) return;
    executeAction('stop', 'Stop');
  };
  const handleRestart = () => {
    if (!confirm(`Are you sure you want to restart "${workspace.name}"?`)) return;
    executeAction('restart', 'Restart');
  };

  const handleDelete = async () => {
    if (!confirm(`Are you sure you want to delete "${workspace.name}"? This action cannot be undone.`)) {
      return;
    }

    setIsDeleting(true);
    try {
      await workspaces.delete(workspace.id);
      onDelete();
    } catch (err) {
      console.error('Failed to delete workspace:', err);
      alert('Failed to delete workspace. Please try again.');
      setIsDeleting(false);
    }
  };

  const handleExportComplete = () => {
    setShowExportModal(false);
  };

  const handleImportComplete = () => {
    setShowImportModal(false);
    onStatusChange();
  };

  const formatDate = (dateString: string) => {
    const date = new Date(dateString);
    return date.toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
      hour: '2-digit',
      minute: '2-digit',
    });
  };

  return (
    <div className="bg-white rounded-xl shadow-md hover:shadow-lg transition-shadow duration-200 overflow-hidden">
      <div className="p-5 border-b border-gray-100">
        <div className="flex justify-between items-start mb-2">
          <div className="flex-1">
            <h3 className="text-lg font-semibold text-gray-900 truncate" title={workspace.name}>
              {workspace.name}
            </h3>
            <p className="text-sm text-gray-500 font-mono">{workspace.tag}</p>
          </div>
          <StatusBadge status={workspace.status} />
        </div>

        {error && (
          <div className="mt-3 p-3 bg-red-50 border border-red-200 rounded-lg">
            <p className="text-sm text-red-700">{error}</p>
          </div>
        )}
      </div>

      <div className="px-5 py-4 space-y-2">
        <div className="flex justify-between text-sm">
          <span className="text-gray-500">Provider:</span>
          <span className="font-medium text-gray-900 capitalize">{workspace.provider}</span>
        </div>
        {workspace.region && (
          <div className="flex justify-between text-sm">
            <span className="text-gray-500">Region:</span>
            <span className="font-medium text-gray-900">{workspace.region}</span>
          </div>
        )}
        <div className="flex justify-between text-sm">
          <span className="text-gray-500">Created:</span>
          <span className="font-medium text-gray-900">{formatDate(workspace.created_at)}</span>
        </div>
      </div>

      <div className="px-5 py-4 bg-gray-50 border-t border-gray-100">
        <div className="flex flex-wrap gap-2">
          {workspace.status === 'stopped' && (
            <ActionButton
              onClick={handleStart}
              disabled={isDeleting}
              loading={actionState.loading && actionState.action === 'Starting'}
              variant="start"
            >
              {actionState.loading && actionState.action === 'Starting' ? 'Starting...' : 'Start'}
            </ActionButton>
          )}

          {workspace.status === 'running' && (
            <>
              <ActionButton
                onClick={handleStop}
                disabled={isDeleting}
                loading={actionState.loading && actionState.action === 'Stopping'}
                variant="stop"
              >
                {actionState.loading && actionState.action === 'Stopping' ? 'Stopping...' : 'Stop'}
              </ActionButton>
              <ActionButton
                onClick={handleRestart}
                disabled={isDeleting}
                loading={actionState.loading && actionState.action === 'Restarting'}
                variant="restart"
              >
                {actionState.loading && actionState.action === 'Restarting' ? 'Restarting...' : 'Restart'}
              </ActionButton>
            </>
          )}

          {workspace.status === 'pending' && (
            <div className="flex-1 text-center py-2 text-sm text-blue-600">
              <div className="flex items-center justify-center">
                <svg className="animate-spin h-4 w-4 mr-2" xmlns="http://www.w3.org/2000/svg" fill="none" viewBox="0 0 24 24">
                  <circle className="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" strokeWidth="4"></circle>
                  <path className="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"></path>
                </svg>
                Provisioning...
              </div>
            </div>
          )}

          {/* Export button - available for all workspaces */}
          <button
            onClick={() => setShowExportModal(true)}
            disabled={isDeleting || actionState.loading}
            className="px-4 py-2 text-sm font-medium text-gray-700 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 transition-colors"
          >
            Export
          </button>

          <ActionButton
            onClick={handleDelete}
            disabled={actionState.loading || workspace.status === 'pending'}
            loading={isDeleting}
            variant="delete"
          >
            {isDeleting ? 'Deleting...' : 'Delete'}
          </ActionButton>
        </div>
      </div>

      {/* Export Modal */}
      <WorkspaceExportModal
        workspaceId={workspace.id}
        workspaceName={workspace.name}
        isOpen={showExportModal}
        onClose={() => setShowExportModal(false)}
        onExportComplete={handleExportComplete}
      />

      {/* Import Modal - shown separately, typically on workspace list page */}
      <WorkspaceImportModal
        organizationId={organizationId}
        isOpen={showImportModal}
        onClose={() => setShowImportModal(false)}
        onImportComplete={handleImportComplete}
      />
    </div>
  );
}

export default WorkspaceCard;
