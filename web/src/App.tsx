import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { AuthProvider, useAuth } from './context/AuthContext';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import Register from './pages/Register';
import OrganizationList from './pages/OrganizationList';
import WorkspaceList from './pages/WorkspaceList';

function NavBar() {
  const { isAuthenticated, user, logout } = useAuth();

  return (
    <nav className="bg-gray-800 text-white p-4">
      <div className="container mx-auto flex justify-between items-center">
        <a href="/" className="text-xl font-bold hover:text-gray-300">WWS - Winmutt's Work Spaces</a>
        <div className="flex items-center space-x-4">
          <a href="/" className="hover:text-gray-300">Dashboard</a>
          <a href="/organizations" className="hover:text-gray-300">Organizations</a>
          <a href="/workspaces" className="hover:text-gray-300">Workspaces</a>
          {isAuthenticated && user && (
            <div className="flex items-center space-x-3 border-l border-gray-600 pl-4">
              <span className="text-sm text-gray-300">{user.username || user.email}</span>
              <button
                onClick={() => logout()}
                className="bg-red-600 hover:bg-red-700 px-3 py-1 rounded text-sm"
              >
                Logout
              </button>
            </div>
          )}
        </div>
      </div>
    </nav>
  );
}

function App() {
  return (
    <AuthProvider>
      <div className="min-h-screen bg-gray-100">
        <NavBar />
        <main className="container mx-auto p-4">
          <Routes>
            <Route path="/login" element={<Login />} />
            <Route path="/register" element={<Register />} />
            <Route path="/" element={<Dashboard />} />
            <Route path="/organizations" element={<OrganizationList />} />
            <Route path="/workspaces" element={<WorkspaceList />} />
          </Routes>
        </main>
      </div>
    </AuthProvider>
  );
}

export default App;
