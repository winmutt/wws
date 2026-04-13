import React, { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { organizations, invitations, workspaces } from '../services/api';

interface Organization {
  id: number;
  name: string;
  owner_id: number;
  created_at: string;
}

interface Member {
  id: number;
  user_id: number;
  organization_id: number;
  role: string;
  user?: {
    id: number;
    github_id: string;
    username: string;
    email: string;
  };
}

interface Invitation {
  id: number;
  organization_id: number;
  email: string;
  token: string;
  status: string;
  created_by: number;
  created_at: string;
  expires_at: string;
}

interface Workspace {
  id: number;
  tag: string;
  name: string;
  organization_id: number;
  status: string;
  created_at: string;
}

function OrganizationList() {
  const { isAuthenticated, user } = useAuth();
  const navigate = useNavigate();
  const [orgs, setOrgs] = useState<Organization[]>([]);
  const [selectedOrg, setSelectedOrg] = useState<Organization | null>(null);
  const [members, setMembers] = useState<Member[]>([]);
  const [invites, setInvites] = useState<Invitation[]>([]);
  const [orgWorkspaces, setOrgWorkspaces] = useState<Workspace[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [showMemberModal, setShowMemberModal] = useState(false);
  const [showInviteModal, setShowInviteModal] = useState(false);
  const [newOrgName, setNewOrgName] = useState('');
  const [inviteEmail, setInviteEmail] = useState('');
  const [selectedMember, setSelectedMember] = useState<Member | null>(null);

  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/login');
      return;
    }
    loadOrganizations();
  }, [isAuthenticated, navigate]);

  const loadOrganizations = async () => {
    setIsLoading(true);
    setError('');
    try {
      const orgList = await organizations.list();
      setOrgs(orgList);
      if (orgList.length > 0 && !selectedOrg) {
        setSelectedOrg(orgList[0]);
        loadOrgDetails(orgList[0].id);
      }
    } catch (err) {
      console.error('Failed to load organizations:', err);
      setError('Failed to load organizations. Please try again.');
    } finally {
      setIsLoading(false);
    }
  };

  const loadOrgDetails = async (orgId: number) => {
    try {
      const [membersList, invitesList, workspacesList] = await Promise.all([
        organizations.members(orgId),
        getOrganizationInvites(orgId),
        workspaces.list(orgId),
      ]);
      setMembers(membersList);
      setInvites(invitesList);
      setOrgWorkspaces(workspacesList);
    } catch (err) {
      console.error('Failed to load organization details:', err);
    }
  };

  const getOrganizationInvites = async (orgId: number): Promise<Invitation[]> => {
    try {
      const response = await fetch(
        `${process.env.REACT_APP_API_URL || 'http://localhost:8080'}/api/v1/organizations/invitations?organization_id=${orgId}`,
        { credentials: 'include' }
      );
      if (!response.ok) return [];
      return response.json();
    } catch {
      return [];
    }
  };

  const handleCreateOrganization = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newOrgName.trim()) {
      setError('Organization name is required');
      return;
    }
    try {
      const newOrg = await organizations.create(newOrgName.trim());
      setOrgs([...orgs, newOrg]);
      setNewOrgName('');
      setShowCreateModal(false);
      setSelectedOrg(newOrg);
      setSuccess(`Organization "${newOrg.name}" created successfully`);
      loadOrgDetails(newOrg.id);
    } catch (err) {
      console.error('Failed to create organization:', err);
      setError('Failed to create organization. Please try again.');
    }
  };

  const handleDeleteOrganization = async () => {
    if (!selectedOrg) return;
    if (!window.confirm(`Are you sure you want to delete "${selectedOrg.name}"? This cannot be undone.`)) {
      return;
    }
    try {
      const response = await fetch(
        `${process.env.REACT_APP_API_URL || 'http://localhost:8080'}/api/v1/organizations/${selectedOrg.id}`,
        { method: 'DELETE', credentials: 'include' }
      );
      if (!response.ok) throw new Error('Failed to delete');
      setOrgs(orgs.filter((o) => o.id !== selectedOrg.id));
      setSelectedOrg(null);
      setMembers([]);
      setInvites([]);
      setOrgWorkspaces([]);
      setSuccess(`Organization "${selectedOrg.name}" deleted successfully`);
    } catch (err) {
      console.error('Failed to delete organization:', err);
      setError('Failed to delete organization. Please try again.');
    }
  };

  const handleInviteMember = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!inviteEmail.trim() || !selectedOrg) {
      setError('Email and organization are required');
      return;
    }
    try {
      await invitations.create(selectedOrg.id, inviteEmail.trim());
      setInviteEmail('');
      setShowInviteModal(false);
      setSuccess('Invitation sent successfully');
      loadOrgDetails(selectedOrg.id);
    } catch (err) {
      console.error('Failed to send invitation:', err);
      setError('Failed to send invitation. Please try again.');
    }
  };

  const handleRevokeInvitation = async (inviteId: number) => {
    if (!selectedOrg) return;
    try {
      const response = await fetch(
        `${process.env.REACT_APP_API_URL || 'http://localhost:8080'}/api/v1/organizations/invitations?invite_id=${inviteId}`,
        { method: 'DELETE', credentials: 'include' }
      );
      if (!response.ok) throw new Error('Failed to revoke');
      setSuccess('Invitation revoked');
      loadOrgDetails(selectedOrg.id);
    } catch (err) {
      console.error('Failed to revoke invitation:', err);
      setError('Failed to revoke invitation.');
    }
  };

  const handleRemoveMember = async (userId: number) => {
    if (!selectedOrg) return;
    if (!window.confirm('Remove this member from the organization?')) return;
    try {
      await organizations.removeMember(selectedOrg.id, userId);
      setSuccess('Member removed successfully');
      loadOrgDetails(selectedOrg.id);
    } catch (err) {
      console.error('Failed to remove member:', err);
      setError('Failed to remove member.');
    }
  };

  const handleRoleChange = async (userId: number, newRole: string) => {
    if (!selectedOrg) return;
    try {
      await organizations.assignRole(selectedOrg.id, userId, newRole);
      setSuccess('Role updated successfully');
      loadOrgDetails(selectedOrg.id);
    } catch (err) {
      console.error('Failed to update role:', err);
      setError('Failed to update role.');
    }
  };

  const copyInviteLink = (token: string) => {
    const link = `${window.location.origin}/invite/${token}`;
    navigator.clipboard.writeText(link);
    setSuccess('Invite link copied to clipboard');
  };

  if (!isAuthenticated) return null;

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Organizations</h2>
        <div className="flex space-x-2">
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
          >
            Create Organization
          </button>
        </div>
      </div>

      {error && (
        <div className="bg-red-100 border border-red-400 text-red-700 px-4 py-3 rounded mb-4">
          {error}
          <button onClick={() => setError('')} className="float-right font-bold">&times;</button>
        </div>
      )}
      {success && (
        <div className="bg-green-100 border border-green-400 text-green-700 px-4 py-3 rounded mb-4">
          {success}
          <button onClick={() => setSuccess('')} className="float-right font-bold">&times;</button>
        </div>
      )}

      {isLoading ? (
        <div className="text-center py-12">
          <p className="text-gray-600">Loading organizations...</p>
        </div>
      ) : orgs.length === 0 ? (
        <div className="text-center py-12 bg-white rounded-lg shadow">
          <p className="text-gray-600 mb-4">No organizations yet</p>
          <button
            onClick={() => setShowCreateModal(true)}
            className="bg-blue-600 text-white px-6 py-2 rounded-md hover:bg-blue-700"
          >
            Create Your First Organization
          </button>
        </div>
      ) : (
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <div className="lg:col-span-1">
            <div className="bg-white rounded-lg shadow">
              <div className="p-4 border-b border-gray-200">
                <h3 className="font-semibold">Your Organizations</h3>
              </div>
              <div className="divide-y divide-gray-200">
                {orgs.map((org) => (
                  <div
                    key={org.id}
                    onClick={() => {
                      setSelectedOrg(org);
                      loadOrgDetails(org.id);
                    }}
                    className={`p-4 cursor-pointer hover:bg-gray-50 ${
                      selectedOrg?.id === org.id ? 'bg-blue-50' : ''
                    }`}
                  >
                    <div className="flex justify-between items-center">
                      <div>
                        <p className="font-medium">{org.name}</p>
                        <p className="text-sm text-gray-500">
                          {orgs.filter((o) => o.id === org.id).length} workspace(s)
                        </p>
                      </div>
                      <span className="text-xs text-gray-400">
                        {new Date(org.created_at).toLocaleDateString()}
                      </span>
                    </div>
                  </div>
                ))}
              </div>
            </div>
          </div>

          <div className="lg:col-span-2">
            {selectedOrg ? (
              <div className="space-y-6">
                <div className="bg-white rounded-lg shadow">
                  <div className="p-4 border-b border-gray-200 flex justify-between items-center">
                    <h3 className="font-semibold">Organization: {selectedOrg.name}</h3>
                    <div className="flex space-x-2">
                      <button
                        onClick={() => setShowInviteModal(true)}
                        className="bg-green-600 text-white px-3 py-1 rounded text-sm hover:bg-green-700"
                      >
                        Invite Member
                      </button>
                      <button
                        onClick={handleDeleteOrganization}
                        className="bg-red-600 text-white px-3 py-1 rounded text-sm hover:bg-red-700"
                      >
                        Delete Org
                      </button>
                    </div>
                  </div>
                  <div className="p-4">
                    <p className="text-sm text-gray-600">
                      Owner: @{user?.username || 'Unknown'}
                    </p>
                    <p className="text-sm text-gray-500">
                      Created: {new Date(selectedOrg.created_at).toLocaleString()}
                    </p>
                  </div>
                </div>

                <div className="bg-white rounded-lg shadow">
                  <div className="p-4 border-b border-gray-200 flex justify-between items-center">
                    <h3 className="font-semibold">Members ({members.length})</h3>
                  </div>
                  <div className="divide-y divide-gray-200">
                    {members.length === 0 ? (
                      <p className="p-4 text-gray-500">No members yet</p>
                    ) : (
                      members.map((member) => (
                        <div key={member.id} className="p-4 flex justify-between items-center">
                          <div>
                            <p className="font-medium">
                              {member.user?.username || `User ${member.user_id}`}
                            </p>
                            <p className="text-sm text-gray-500">{member.user?.email}</p>
                          </div>
                          <div className="flex items-center space-x-2">
                            <select
                              value={member.role}
                              onChange={(e) => handleRoleChange(member.user_id, e.target.value)}
                              className="px-2 py-1 border rounded text-sm"
                            >
                              <option value="admin">Admin</option>
                              <option value="member">Member</option>
                              <option value="viewer">Viewer</option>
                            </select>
                            {member.user_id !== user?.id && (
                              <button
                                onClick={() => handleRemoveMember(member.user_id)}
                                className="text-red-600 hover:text-red-800 text-sm"
                              >
                                Remove
                              </button>
                            )}
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                <div className="bg-white rounded-lg shadow">
                  <div className="p-4 border-b border-gray-200 flex justify-between items-center">
                    <h3 className="font-semibold">Pending Invitations ({invites.length})</h3>
                  </div>
                  <div className="divide-y divide-gray-200">
                    {invites.length === 0 ? (
                      <p className="p-4 text-gray-500">No pending invitations</p>
                    ) : (
                      invites.map((invite) => (
                        <div key={invite.id} className="p-4 flex justify-between items-center">
                          <div>
                            <p className="font-medium">{invite.email}</p>
                            <p className="text-sm text-gray-500">
                              Expires: {new Date(invite.expires_at).toLocaleString()}
                            </p>
                          </div>
                          <div className="flex items-center space-x-2">
                            <button
                              onClick={() => copyInviteLink(invite.token)}
                              className="text-blue-600 hover:text-blue-800 text-sm"
                            >
                              Copy Link
                            </button>
                            <button
                              onClick={() => handleRevokeInvitation(invite.id)}
                              className="text-red-600 hover:text-red-800 text-sm"
                            >
                              Revoke
                            </button>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </div>

                <div className="bg-white rounded-lg shadow">
                  <div className="p-4 border-b border-gray-200">
                    <h3 className="font-semibold">Workspaces ({orgWorkspaces.length})</h3>
                  </div>
                  <div className="divide-y divide-gray-200">
                    {orgWorkspaces.length === 0 ? (
                      <p className="p-4 text-gray-500">No workspaces in this organization</p>
                    ) : (
                      orgWorkspaces.map((ws) => (
                        <div key={ws.id} className="p-4 flex justify-between items-center">
                          <div>
                            <p className="font-medium">{ws.name}</p>
                            <p className="text-sm text-gray-500">{ws.tag}</p>
                          </div>
                          <span
                            className={`px-2 py-1 rounded-full text-sm ${
                              ws.status === 'running'
                                ? 'bg-green-100 text-green-800'
                                : ws.status === 'stopped'
                                ? 'bg-gray-100 text-gray-800'
                                : 'bg-yellow-100 text-yellow-800'
                            }`}
                          >
                            {ws.status}
                          </span>
                        </div>
                      ))
                    )}
                  </div>
                </div>
              </div>
            ) : (
              <div className="text-center py-12 bg-white rounded-lg shadow">
                <p className="text-gray-600">Select an organization to view details</p>
              </div>
            )}
          </div>
        </div>
      )}

      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-xl font-bold mb-4">Create Organization</h3>
            <form onSubmit={handleCreateOrganization}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-2">Organization Name</label>
                <input
                  type="text"
                  value={newOrgName}
                  onChange={(e) => setNewOrgName(e.target.value)}
                  className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="My Organization"
                  autoFocus
                />
              </div>
              <div className="flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowCreateModal(false);
                    setNewOrgName('');
                  }}
                  className="px-4 py-2 text-gray-600 hover:text-gray-800"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700"
                >
                  Create
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {showInviteModal && selectedOrg && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-full max-w-md">
            <h3 className="text-xl font-bold mb-4">Invite Member to {selectedOrg.name}</h3>
            <form onSubmit={handleInviteMember}>
              <div className="mb-4">
                <label className="block text-sm font-medium mb-2">Email Address</label>
                <input
                  type="email"
                  value={inviteEmail}
                  onChange={(e) => setInviteEmail(e.target.value)}
                  className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
                  placeholder="user@example.com"
                  autoFocus
                />
              </div>
              <div className="flex justify-end space-x-2">
                <button
                  type="button"
                  onClick={() => {
                    setShowInviteModal(false);
                    setInviteEmail('');
                  }}
                  className="px-4 py-2 text-gray-600 hover:text-gray-800"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  className="px-4 py-2 bg-green-600 text-white rounded-md hover:bg-green-700"
                >
                  Send Invitation
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}

export default OrganizationList;
