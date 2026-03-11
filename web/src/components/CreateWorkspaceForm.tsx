import React, { useState } from 'react';
import { workspaces } from '../services/api';

interface CreateWorkspaceFormProps {
  organizationId: number;
  onSubmit: () => void;
  onCancel: () => void;
}

interface WorkspaceConfig {
  cpu: number;
  memory: number;
  storage: number;
}

function CreateWorkspaceForm({ organizationId, onSubmit, onCancel }: CreateWorkspaceFormProps) {
  const [name, setName] = useState('');
  const [provider, setProvider] = useState('podman');
  const [region, setRegion] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [config, setConfig] = useState<WorkspaceConfig>({
    cpu: 2,
    memory: 4,
    storage: 20,
  });

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    try {
      await workspaces.create({
        name,
        organization_id: organizationId,
        provider,
        region: region || undefined,
        config,
      });
      onSubmit();
    } catch (error) {
      console.error('Failed to create workspace:', error);
      alert('Failed to create workspace');
      setIsSubmitting(false);
    }
  };

  return (
    <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center">
      <div className="bg-white rounded-lg p-6 w-full max-w-md">
        <h2 className="text-2xl font-bold mb-4">Create Workspace</h2>
        
        <form onSubmit={handleSubmit}>
          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Name
            </label>
            <input
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="my-workspace"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Provider
            </label>
            <select
              value={provider}
              onChange={(e) => setProvider(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              <option value="podman">Podman</option>
            </select>
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Region (optional)
            </label>
            <input
              type="text"
              value={region}
              onChange={(e) => setRegion(e.target.value)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
              placeholder="us-east-1"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              CPU Cores
            </label>
            <input
              type="number"
              value={config.cpu}
              onChange={(e) => setConfig({ ...config, cpu: parseInt(e.target.value) })}
              min="1"
              max="16"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="mb-4">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Memory (GB)
            </label>
            <input
              type="number"
              value={config.memory}
              onChange={(e) => setConfig({ ...config, memory: parseInt(e.target.value) })}
              min="1"
              max="64"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="mb-6">
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Storage (GB)
            </label>
            <input
              type="number"
              value={config.storage}
              onChange={(e) => setConfig({ ...config, storage: parseInt(e.target.value) })}
              min="10"
              max="500"
              className="w-full px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
            />
          </div>

          <div className="flex space-x-3">
            <button
              type="button"
              onClick={onCancel}
              className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-md hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={isSubmitting}
              className="flex-1 px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 disabled:opacity-50"
            >
              {isSubmitting ? 'Creating...' : 'Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default CreateWorkspaceForm;
