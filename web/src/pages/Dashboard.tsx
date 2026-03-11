import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { workspaces, organizations } from '../services/api';

interface Workspace {
  id: number;
  tag: string;
  name: string;
  organization_id: number;
  owner_id: number;
  provider: string;
  status: string;
  created_at: string;
}

interface Organization {
  id: number;
  name: string;
}

function Dashboard() {
  const { isAuthenticated, logout } = useAuth();
  const navigate = useNavigate();
  const [stats, setStats] = useState({
    totalWorkspaces: 0,
    runningWorkspaces: 0,
    totalOrganizations: 0,
  });
  const [recentWorkspaces, setRecentWorkspaces] = useState<Workspace[]>([]);
  const [isLoading, setIsLoading] = useState(true);

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/login');
      return;
    }
    loadDashboard();
  }, [isAuthenticated, navigate]);

  const loadDashboard = async () => {
    setIsLoading(true);
    try {
      const [orgs, allWorkspaces] = await Promise.all([
        organizations.list(),
        workspaces.list(0),
      ]);
      
      setStats({
        totalOrganizations: orgs.length,
        totalWorkspaces: allWorkspaces.length,
        runningWorkspaces: allWorkspaces.filter((w: Workspace) => w.status === 'running').length,
      });
      
      setRecentWorkspaces(allWorkspaces.slice(0, 5));
    } catch (error) {
      console.error('Failed to load dashboard:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  if (!isAuthenticated) {
    return null;
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Dashboard</h2>
        <button
          onClick={handleLogout}
          className="bg-gray-600 text-white px-4 py-2 rounded-md hover:bg-gray-700"
        >
          Logout
        </button>
      </div>

      {isLoading ? (
        <div className="text-center py-12">
          <p className="text-gray-600">Loading dashboard...</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Total Organizations</h3>
              <p className="text-3xl font-bold text-gray-900">{stats.totalOrganizations}</p>
              <a href="/organizations" className="text-blue-600 hover:text-blue-800 text-sm mt-2">
                View Organizations →
              </a>
            </div>
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Total Workspaces</h3>
              <p className="text-3xl font-bold text-gray-900">{stats.totalWorkspaces}</p>
              <a href="/workspaces" className="text-blue-600 hover:text-blue-800 text-sm mt-2">
                View Workspaces →
              </a>
            </div>
            <div className="bg-white rounded-lg shadow p-6">
              <h3 className="text-sm font-medium text-gray-500 mb-2">Running Workspaces</h3>
              <p className="text-3xl font-bold text-green-600">{stats.runningWorkspaces}</p>
              <a href="/workspaces" className="text-blue-600 hover:text-blue-800 text-sm mt-2">
                View Running →
              </a>
            </div>
          </div>

          <div className="bg-white rounded-lg shadow">
            <div className="p-6 border-b border-gray-200">
              <h3 className="text-lg font-semibold">Recent Workspaces</h3>
            </div>
            <div className="p-6">
              {recentWorkspaces.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-gray-600 mb-4">No workspaces yet</p>
                  <a
                    href="/workspaces"
                    className="inline-block bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700"
                  >
                    Create Your First Workspace
                  </a>
                </div>
              ) : (
                <div className="space-y-4">
                  {recentWorkspaces.map((workspace) => (
                    <div key={workspace.id} className="flex justify-between items-center py-2 border-b border-gray-100 last:border-0">
                      <div>
                        <p className="font-medium">{workspace.name}</p>
                        <p className="text-sm text-gray-500">{workspace.tag}</p>
                      </div>
                      <div className="flex items-center space-x-4">
                        <span
                          className={`px-2 py-1 rounded-full text-sm font-medium ${
                            workspace.status === 'running'
                              ? 'bg-green-100 text-green-800'
                              : workspace.status === 'stopped'
                              ? 'bg-gray-100 text-gray-800'
                              : 'bg-yellow-100 text-yellow-800'
                          }`}
                        >
                          {workspace.status}
                        </span>
                        <a
                          href="/workspaces"
                          className="text-blue-600 hover:text-blue-800 text-sm"
                        >
                          View
                        </a>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </>
      )}
    </div>
  );
}

export default Dashboard;
