import React, { useState } from 'react';
import { workspaces } from '../services/api';

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

function WorkspaceCard({ workspace, organizationId, onDelete, onStatusChange }: WorkspaceCardProps) {
  const [isDeleting, setIsDeleting] = useState(false);
  const [isActionLoading, setIsActionLoading] = useState(false);

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'running':
        return 'bg-green-100 text-green-800';
      case 'stopped':
        return 'bg-gray-100 text-gray-800';
      case 'pending':
        return 'bg-yellow-100 text-yellow-800';
      case 'deleted':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-blue-100 text-blue-800';
    }
  };

  const handleStart = async () => {
    setIsActionLoading(true);
    try {
      await workspaces.start(workspace.id);
      onStatusChange();
    } catch (error) {
      console.error('Failed to start workspace:', error);
      alert('Failed to start workspace');
    } finally {
      setIsActionLoading(false);
    }
  };

  const handleStop = async () => {
    setIsActionLoading(true);
    try {
      await workspaces.stop(workspace.id);
      onStatusChange();
    } catch (error) {
      console.error('Failed to stop workspace:', error);
      alert('Failed to stop workspace');
    } finally {
      setIsActionLoading(false);
    }
  };

  const handleRestart = async () => {
    setIsActionLoading(true);
    try {
      await workspaces.restart(workspace.id);
      onStatusChange();
    } catch (error) {
      console.error('Failed to restart workspace:', error);
      alert('Failed to restart workspace');
    } finally {
      setIsActionLoading(false);
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this workspace?')) {
      return;
    }
    
    setIsDeleting(true);
    try {
      await workspaces.delete(workspace.id);
      onDelete();
    } catch (error) {
      console.error('Failed to delete workspace:', error);
      alert('Failed to delete workspace');
      setIsDeleting(false);
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <div className="flex justify-between items-start mb-4">
        <div>
          <h3 className="text-lg font-semibold text-gray-900">{workspace.name}</h3>
          <p className="text-sm text-gray-500">{workspace.tag}</p>
        </div>
        <span className={`px-2 py-1 rounded-full text-sm font-medium ${getStatusColor(workspace.status)}`}>
          {workspace.status}
        </span>
      </div>
      
      <div className="space-y-2 mb-4">
        <div className="flex justify-between">
          <span className="text-sm text-gray-500">Provider:</span>
          <span className="text-sm font-medium">{workspace.provider}</span>
        </div>
        {workspace.region && (
          <div className="flex justify-between">
            <span className="text-sm text-gray-500">Region:</span>
            <span className="text-sm font-medium">{workspace.region}</span>
          </div>
        )}
        <div className="flex justify-between">
          <span className="text-sm text-gray-500">Created:</span>
          <span className="text-sm font-medium">{new Date(workspace.created_at).toLocaleDateString()}</span>
        </div>
      </div>
      
      <div className="flex space-x-2">
        {workspace.status === 'stopped' && (
          <button
            onClick={handleStart}
            disabled={isActionLoading}
            className="flex-1 bg-green-600 text-white px-4 py-2 rounded hover:bg-green-700 disabled:opacity-50"
          >
            {isActionLoading ? 'Starting...' : 'Start'}
          </button>
        )}
        {workspace.status === 'running' && (
          <>
            <button
              onClick={handleStop}
              disabled={isActionLoading}
              className="flex-1 bg-gray-600 text-white px-4 py-2 rounded hover:bg-gray-700 disabled:opacity-50"
            >
              {isActionLoading ? 'Stopping...' : 'Stop'}
            </button>
            <button
              onClick={handleRestart}
              disabled={isActionLoading}
              className="flex-1 bg-blue-600 text-white px-4 py-2 rounded hover:bg-blue-700 disabled:opacity-50"
            >
              {isActionLoading ? 'Restarting...' : 'Restart'}
            </button>
          </>
        )}
        <button
          onClick={handleDelete}
          disabled={isDeleting}
          className="flex-1 bg-red-600 text-white px-4 py-2 rounded hover:bg-red-700 disabled:opacity-50"
        >
          {isDeleting ? 'Deleting...' : 'Delete'}
        </button>
      </div>
    </div>
  );
}

export default WorkspaceCard;
