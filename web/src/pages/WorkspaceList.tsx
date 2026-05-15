import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { workspaces, organizations } from '../services/api';
import WorkspaceCard from '../components/WorkspaceCard';
import CreateWorkspaceForm from '../components/CreateWorkspaceForm';
import { WorkspaceImportModal } from '../components/WorkspaceExportModal';

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
  const [showImportModal, setShowImportModal] = useState(false);
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

  const handleImportComplete = () => {
    setShowImportModal(false);
    loadData();
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="p-6 bg-gradient-to-br from-indigo-100 via-purple-50 to-pink-100 min-h-screen text-gray-900">
      <div className="flex justify-between items-center mb-8">
        <div>
          <div className="flex items-center gap-3">
            <div className="p-3 bg-gradient-to-br from-cyan-500 to-blue-600 rounded-xl shadow-lg shadow-cyan-500/30">
              <svg className="w-8 h-8 text-white" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
              </svg>
            </div>
            <div>
              <h2 className="text-4xl font-bold bg-gradient-to-r from-cyan-400 to-blue-400 bg-clip-text text-transparent">Workspaces</h2>
              <p className="text-slate-400 mt-1">Manage your development environments</p>
            </div>
          </div>
        </div>
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded-xl shadow-sm">
            <p className="font-medium">Error</p>
            <p className="text-sm">{error}</p>
          </div>
        )}
      </div>

      <div className="flex items-center gap-4 mb-8">
        {orgs.length > 0 && (
          <select
            value={selectedOrg || ''}
            onChange={(e) => setSelectedOrg(Number(e.target.value))}
            className="px-4 py-2 border border-slate-300 rounded-xl shadow-sm focus:ring-2 focus:ring-cyan-500 focus:border-cyan-500 bg-white"
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
          className="bg-gradient-to-r from-cyan-500 to-blue-600 text-white px-6 py-2 rounded-xl hover:from-cyan-600 hover:to-blue-700 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-cyan-500/30 transition-all"
        >
          Create Workspace
        </button>
        <button
          onClick={() => setShowImportModal(true)}
          disabled={!selectedOrg || isLoading}
          className="bg-gradient-to-r from-purple-500 to-pink-600 text-white px-6 py-2 rounded-xl hover:from-purple-600 hover:to-pink-700 disabled:opacity-50 disabled:cursor-not-allowed shadow-lg shadow-purple-500/30 transition-all"
        >
          Import Workspace
        </button>
        <button
          onClick={handleLogout}
          className="ml-auto bg-slate-700 text-white px-4 py-2 rounded-xl hover:bg-slate-800 transition-all"
        >
          Logout
        </button>
      </div>

      {!selectedOrg ? (
        <div className="bg-white rounded-2xl shadow-lg border border-slate-200 p-12 text-center">
          <svg className="w-16 h-16 text-slate-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
          </svg>
          <p className="text-slate-600 text-lg">No organizations found. Create an organization to get started.</p>
        </div>
      ) : isLoading ? (
        <div className="flex items-center justify-center py-12">
          <div className="relative">
            <div className="w-16 h-16 border-4 border-cyan-500/30 border-t-cyan-500 rounded-full animate-spin"></div>
            <div className="absolute top-1/2 left-1/2 transform -translate-x-1/2 -translate-y-1/2">
              <svg className="w-6 h-6 text-cyan-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
              </svg>
            </div>
          </div>
        </div>
      ) : workspaceList.length === 0 ? (
        <div className="bg-white rounded-2xl shadow-lg border border-slate-200 p-12 text-center">
          <svg className="w-16 h-16 text-slate-400 mx-auto mb-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <p className="text-slate-600 text-lg mb-6">No workspaces yet</p>
          <button
            onClick={handleCreateWorkspace}
            className="bg-gradient-to-r from-cyan-500 to-blue-600 text-white px-8 py-3 rounded-xl hover:from-cyan-600 hover:to-blue-700 shadow-lg shadow-cyan-500/30 transition-all"
          >
            Create Your First Workspace
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
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

      {showImportModal && selectedOrg && (
        <WorkspaceImportModal
          organizationId={selectedOrg}
          isOpen={showImportModal}
          onClose={() => setShowImportModal(false)}
          onImportComplete={handleImportComplete}
        />
      )}
    </div>
  );
}

export default WorkspaceList;
