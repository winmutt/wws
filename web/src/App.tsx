import React from 'react';
import { Routes, Route } from 'react-router-dom';
import Dashboard from './pages/Dashboard';
import Login from './pages/Login';
import Register from './pages/Register';
import OrganizationList from './pages/OrganizationList';
import WorkspaceList from './pages/WorkspaceList';

function App() {
  return (
    <div className="min-h-screen bg-gray-100">
      <nav className="bg-gray-800 text-white p-4">
        <div className="container mx-auto">
          <h1 className="text-xl font-bold">WWS - Winmutt's Work Spaces</h1>
        </div>
      </nav>
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
  );
}

export default App;
