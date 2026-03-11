import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { workspaces, organizations } from '../services/api';
import WorkspaceCard from '../components/WorkspaceCard';
import CreateWorkspaceForm from '../components/CreateWorkspaceForm';

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

interface Organization {
  id: number;
  name: string;
}

function WorkspaceList() {
  const { isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const [workspaceList, setWorkspaceList] = useState<Workspace[]>([]);
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<number | null>(null);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/login');
      return;
    }
    loadData();
  }, [isAuthenticated, navigate, selectedOrg]);

  const loadData = async () => {
    setIsLoading(true);
    setError('');
    try {
      const [orgs, ws] = await Promise.all([
        organizations.list(),
        selectedOrg ? workspaces.list(selectedOrg) : Promise.resolve([]),
      ]);
      setOrgs(orgs);
      setWorkspaceList(ws);
      if (!selectedOrg && orgs.length > 0) {
        setSelectedOrg(orgs[0].id);
      }
    } catch (err) {
      console.error('Failed to load data:', err);
      setError('Failed to load data. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  const handleCreateWorkspace = () => {
    setShowCreateForm(true);
  };

  const handleFormSubmit = () => {
    setShowCreateForm(false);
    loadData();
  };

  const handleDeleteWorkspace = () => {
    loadData();
  };

  const handleStatusChange = () => {
    loadData();
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Workspaces</h2>
        {error && (
          <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-2 rounded">
            {error}
          </div>
        )}
      </div>
        <div className="flex space-x-4">
          {orgs.length > 0 && (
            <select
              value={selectedOrg || ''}
              onChange={(e) => setSelectedOrg(Number(e.target.value))}
              className="px-3 py-2 border border-gray-300 rounded-md"
            >
              {orgs.map((org) => (
                <option key={org.id} value={org.id}>
                  {org.name}
                </option>
              ))}
            </select>
          )}
          <button
            onClick={handleCreateWorkspace}
            disabled={!selectedOrg || isLoading}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700 disabled:opacity-50"
          >
            Create Workspace
          </button>
          <button
            onClick={handleLogout}
            className="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700"
          >
            Logout
          </button>
        </div>
       {!selectedOrg ? (
        <div className="text-center py-12">
          <p className="text-gray-600">
            No organizations found. Create an organization to get started.
          </p>
        </div>
      ) : isLoading ? (
        <div className="text-center py-12">
          <p className="text-gray-600">Loading workspaces...</p>
        </div>
      ) : workspaceList.length === 0 ? (
        <div className="text-center py-12">
          <p className="text-gray-600 mb-4">No workspaces yet</p>
          <button
            onClick={handleCreateWorkspace}
            className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700"
          >
            Create Your First Workspace
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
          {workspaceList.map((workspace) => (
            <WorkspaceCard
              key={workspace.id}
              workspace={workspace}
              organizationId={selectedOrg}
              onDelete={handleDeleteWorkspace}
              onStatusChange={handleStatusChange}
            />
          ))}
        </div>
      )}

      {showCreateForm && selectedOrg && (
        <CreateWorkspaceForm
          organizationId={selectedOrg}
          onSubmit={handleFormSubmit}
          onCancel={() => setShowCreateForm(false)}
        />
      )}
    </div>
  );
}

export default WorkspaceList;
