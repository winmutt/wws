import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { workspaces, organizations, auth } from '../services/api';

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

interface VersionInfo {
  version: string;
  build_time: string;
}

function Dashboard() {
  const { isAuthenticated } = useAuth();
  const navigate = useNavigate();
  const [stats, setStats] = useState({
    totalWorkspaces: 0,
    runningWorkspaces: 0,
    totalOrganizations: 0,
  });
  const [recentWorkspaces, setRecentWorkspaces] = useState<Workspace[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [version, setVersion] = useState<VersionInfo | null>(null);
  const [lastChecked, setLastChecked] = useState<Date>(new Date());

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/login');
      return;
    }
    loadDashboard();
    loadVersion();
    const interval = setInterval(loadVersion, 60000);
    return () => clearInterval(interval);
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

  const loadVersion = async () => {
    try {
      const versionData = await auth.getVersion();
      setVersion(versionData);
      setLastChecked(new Date());
    } catch (error) {
      console.error('Failed to load version:', error);
    }
  };

  const handleLogout = async () => {
    navigate('/login');
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
              <h2 className="text-4xl font-bold bg-gradient-to-r from-cyan-400 to-blue-400 bg-clip-text text-transparent">Dashboard</h2>
              <p className="text-slate-400 mt-1">Welcome back! Here's what's happening.</p>
            </div>
          </div>
        </div>
        {version && (
          <div className="text-right">
            <div className="inline-flex items-center bg-slate-700/50 backdrop-blur-sm rounded-xl shadow-lg border border-slate-600 px-4 py-2">
              <span className="text-xs text-slate-400 mr-2 font-medium">Version</span>
              <span className="text-sm font-bold text-cyan-400">{version.version}</span>
              <span className="text-xs text-slate-500 ml-2">
                <svg className="w-3 h-3 inline mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4l3 3m6-3a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                {lastChecked.toLocaleTimeString()}
              </span>
            </div>
          </div>
        )}
      </div>

      {isLoading ? (
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
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-3 gap-6 mb-8">
            <div className="bg-indigo-100 rounded-2xl shadow border border-indigo-200 p-6 transform hover:scale-105 transition-all duration-300 cursor-pointer hover:shadow-lg">
              <div className="flex items-center gap-3 mb-3">
                <svg className="w-6 h-6 text-cyan-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M17 20h5v-2a3 3 0 00-5.356-1.857M17 20H7m10 0v-2c0-.656-.126-1.283-.356-1.857M7 20H2v-2a3 3 0 015.356-1.857M7 20v-2c0-.656.126-1.283.356-1.857m0 0a5.002 5.002 0 019.288 0M15 7a3 3 0 11-6 0 3 3 0 016 0zm6 3a2 2 0 11-4 0 2 2 0 014 0zM7 10a2 2 0 11-4 0 2 2 0 014 0z" />
                </svg>
                <h3 className="text-sm font-medium text-gray-800">Organizations</h3>
              </div>
              <p className="text-4xl font-bold">{stats.totalOrganizations}</p>
              <p className="text-xs text-cyan-200 mt-2 opacity-70">{stats.totalOrganizations === 1 ? 'Organization' : 'Organizations'} created</p>
              <button
                onClick={() => navigate('/organizations')}
                className="mt-4 inline-flex items-center text-sm font-medium text-white hover:text-cyan-200 transition-colors"
              >
                Manage <span className="ml-1 text-lg">→</span>
              </button>
            </div>
            <div className="bg-indigo-100 rounded-2xl shadow border border-indigo-200 p-6 transform hover:scale-105 transition-all duration-300 cursor-pointer hover:shadow-lg">
              <div className="flex items-center gap-3 mb-3">
                <svg className="w-6 h-6 text-purple-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                </svg>
                <h3 className="text-sm font-medium text-gray-800">Workspaces</h3>
              </div>
              <p className="text-4xl font-bold">{stats.totalWorkspaces}</p>
              <p className="text-xs text-purple-200 mt-2 opacity-70">{stats.totalWorkspaces === 1 ? 'Workspace' : 'Workspaces'} available</p>
              <button
                onClick={() => navigate('/workspaces')}
                className="mt-4 inline-flex items-center text-sm font-medium text-white hover:text-purple-200 transition-colors"
              >
                Manage <span className="ml-1 text-lg">→</span>
              </button>
            </div>
            <div className="bg-indigo-100 rounded-2xl shadow border border-indigo-200 p-6 transform hover:scale-105 transition-all duration-300 cursor-pointer hover:shadow-lg">
              <div className="flex items-center gap-3 mb-3">
                <svg className="w-6 h-6 text-emerald-200" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                <h3 className="text-sm font-medium text-gray-800">Running</h3>
              </div>
              <p className="text-4xl font-bold">{stats.runningWorkspaces}</p>
              <p className="text-xs text-emerald-200 mt-2 opacity-70">Active and ready</p>
              <button
                onClick={() => navigate('/workspaces')}
                className="mt-4 inline-flex items-center text-sm font-medium text-white hover:text-emerald-200 transition-colors"
              >
                View Active <span className="ml-1 text-lg">→</span>
              </button>
            </div>
          </div>

          <div className="bg-slate-800/50 backdrop-blur-sm rounded-2xl shadow-xl border border-slate-700 overflow-hidden">
            <div className="p-6 bg-gradient-to-r from-slate-800 to-slate-900 border-b border-slate-700">
              <div className="flex items-center gap-3">
                <svg className="w-6 h-6 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z" />
                </svg>
                <h3 className="text-2xl font-bold text-white">Recent Workspaces</h3>
              </div>
            </div>
            <div className="p-6">
              {recentWorkspaces.length === 0 ? (
                <div className="text-center py-16">
                  <div className="mb-6 opacity-50">
                    <svg className="w-20 h-20 mx-auto text-cyan-500" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                  </div>
                  <h3 className="text-xl font-semibold text-slate-300 mb-2">No workspaces yet</h3>
                  <p className="text-slate-500 mb-6">Get started by creating your first workspace</p>
                  <button
                    onClick={() => navigate('/workspaces')}
                    className="bg-gradient-to-r from-cyan-600 to-blue-600 text-white px-8 py-3 rounded-xl hover:from-cyan-700 hover:to-blue-700 shadow-lg shadow-cyan-500/30 transition-all transform hover:scale-105 font-medium"
                  >
                    Create Your First Workspace
                  </button>
                </div>
              ) : (
                <div className="space-y-4">
                  {recentWorkspaces.map((workspace) => (
                    <div key={workspace.id} className="flex justify-between items-center p-5 bg-slate-700/40 rounded-xl hover:bg-slate-700/60 transition-all border border-slate-600 hover:border-cyan-500/50">
                      <div>
                        <div className="flex items-center gap-2 mb-1">
                          <span className="px-2 py-0.5 rounded bg-slate-600 text-xs text-slate-300 font-mono">{workspace.tag}</span>
                          <p className="font-semibold text-lg text-white">{workspace.name}</p>
                        </div>
                        <div className="flex items-center gap-2 mt-2">
                          <span className={`px-3 py-1 rounded-full text-xs font-semibold ${
                            workspace.status === 'running'
                              ? 'bg-emerald-500/20 text-emerald-400 border border-emerald-500/30'
                              : workspace.status === 'stopped'
                              ? 'bg-slate-600 text-slate-300 border border-slate-500'
                              : 'bg-amber-500/20 text-amber-400 border border-amber-500/30'
                          }`}>
                            {workspace.status.toUpperCase()}
                          </span>
                        </div>
                      </div>
                      <div className="flex items-center space-x-3">
                        <button
                          onClick={() => navigate('/workspaces')}
                          className="p-2 bg-cyan-600/20 hover:bg-cyan-600/30 text-cyan-400 rounded-lg transition-colors"
                          title="View details"
                        >
                          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" />
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" />
                          </svg>
                        </button>
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
