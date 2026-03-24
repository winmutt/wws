import React, { useState, useEffect } from 'react';

interface Team {
  id: number;
  organization_id: number;
  name: string;
  description: string;
  created_by: number;
  created_at: string;
}

interface TeamMember {
  id: number;
  team_id: number;
  user_id: number;
  username: string;
  email: string;
  role_name: string;
  joined_at: string;
  status: string;
}

interface TeamRole {
  id: number;
  name: string;
  description: string;
  permissions: string[];
  is_default: boolean;
}

function TeamManagement() {
  const [teams, setTeams] = useState<Team[]>([]);
  const [selectedTeam, setSelectedTeam] = useState<Team | null>(null);
  const [members, setMembers] = useState<TeamMember[]>([]);
  const [roles, setRoles] = useState<TeamRole[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newTeam, setNewTeam] = useState({ name: '', description: '' });

  useEffect(() => {
    loadTeams();
  }, []);

  useEffect(() => {
    if (selectedTeam) {
      loadTeamMembers(selectedTeam.id);
    }
  }, [selectedTeam]);

  const loadTeams = async () => {
    setIsLoading(true);
    try {
      const response = await fetch('/api/v1/teams', {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setTeams(data);
      }
    } catch (error) {
      console.error('Failed to load teams:', error);
    } finally {
      setIsLoading(false);
    }
  };

  const loadTeamMembers = async (teamId: number) => {
    try {
      const response = await fetch(`/api/v1/teams/members?team_id=${teamId}`, {
        credentials: 'include',
      });
      if (response.ok) {
        const data = await response.json();
        setMembers(data);
      }
    } catch (error) {
      console.error('Failed to load team members:', error);
    }
  };

  const createTeam = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const response = await fetch('/api/v1/teams', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(newTeam),
        credentials: 'include',
      });
      if (response.ok) {
        setShowCreateModal(false);
        setNewTeam({ name: '', description: '' });
        loadTeams();
      }
    } catch (error) {
      console.error('Failed to create team:', error);
    }
  };

  const removeMember = async (teamId: number, userId: number) => {
    try {
      const response = await fetch(`/api/v1/teams/members/remove?team_id=${teamId}&user_id=${userId}`, {
        method: 'DELETE',
        credentials: 'include',
      });
      if (response.ok) {
        loadTeamMembers(teamId);
      }
    } catch (error) {
      console.error('Failed to remove member:', error);
    }
  };

  if (isLoading) {
    return (
      <div className="p-4">
        <div className="text-center py-12">
          <p className="text-gray-600">Loading teams...</p>
        </div>
      </div>
    );
  }

  return (
    <div className="p-4">
      <div className="flex justify-between items-center mb-6">
        <h2 className="text-2xl font-bold">Team Management</h2>
        <button
          onClick={() => setShowCreateModal(true)}
          className="bg-blue-600 text-white px-4 py-2 rounded-md hover:bg-blue-700"
        >
          Create Team
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Teams List */}
        <div className="lg:col-span-1">
          <div className="bg-white rounded-lg shadow">
            <div className="p-6 border-b border-gray-200">
              <h3 className="text-lg font-semibold">Teams</h3>
            </div>
            <div className="p-4">
              {teams.length === 0 ? (
                <div className="text-center py-8">
                  <p className="text-gray-600">No teams yet</p>
                </div>
              ) : (
                <div className="space-y-2">
                  {teams.map((team) => (
                    <div
                      key={team.id}
                      onClick={() => setSelectedTeam(team)}
                      className={`p-4 rounded-lg cursor-pointer ${
                        selectedTeam?.id === team.id
                          ? 'bg-blue-50 border-blue-500 border'
                          : 'bg-gray-50 hover:bg-gray-100 border border-gray-200'
                      }`}
                    >
                      <h4 className="font-medium">{team.name}</h4>
                      <p className="text-sm text-gray-500">{team.description}</p>
                    </div>
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>

        {/* Team Details */}
        <div className="lg:col-span-2">
          {selectedTeam ? (
            <div className="bg-white rounded-lg shadow">
              <div className="p-6 border-b border-gray-200">
                <h3 className="text-lg font-semibold">{selectedTeam.name}</h3>
                <p className="text-sm text-gray-500">{selectedTeam.description}</p>
              </div>
              <div className="p-6">
                <h4 className="font-medium mb-4">Team Members ({members.length})</h4>
                {members.length === 0 ? (
                  <div className="text-center py-8">
                    <p className="text-gray-600">No members yet</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {members.map((member) => (
                      <div key={member.id} className="flex justify-between items-center p-4 bg-gray-50 rounded-lg">
                        <div>
                          <p className="font-medium">{member.username}</p>
                          <p className="text-sm text-gray-500">{member.email}</p>
                          <div className="flex items-center space-x-2 mt-1">
                            <span className="px-2 py-1 bg-blue-100 text-blue-800 text-xs rounded-full">
                              {member.role_name || 'Member'}
                            </span>
                            <span className={`px-2 py-1 text-xs rounded-full ${
                              member.status === 'active' ? 'bg-green-100 text-green-800' : 'bg-yellow-100 text-yellow-800'
                            }`}>
                              {member.status}
                            </span>
                          </div>
                        </div>
                        <button
                          onClick={() => removeMember(selectedTeam.id, member.user_id)}
                          className="px-3 py-1 bg-red-100 text-red-600 rounded-md hover:bg-red-200 text-sm"
                        >
                          Remove
                        </button>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          ) : (
            <div className="bg-white rounded-lg shadow p-6">
              <p className="text-gray-600 text-center py-12">Select a team to view details</p>
            </div>
          )}
        </div>
      </div>

      {/* Create Team Modal */}
      {showCreateModal && (
        <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
          <div className="bg-white rounded-lg p-6 w-96">
            <h3 className="text-lg font-semibold mb-4">Create New Team</h3>
            <form onSubmit={createTeam}>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">Team Name</label>
                <input
                  type="text"
                  value={newTeam.name}
                  onChange={(e) => setNewTeam({ ...newTeam, name: e.target.value })}
                  className="w-full border rounded-md px-3 py-2"
                  required
                />
              </div>
              <div className="mb-4">
                <label className="block text-sm font-medium text-gray-700 mb-1">Description</label>
                <textarea
                  value={newTeam.description}
                  onChange={(e) => setNewTeam({ ...newTeam, description: e.target.value })}
                  className="w-full border rounded-md px-3 py-2"
                  rows={3}
                />
              </div>
              <div className="flex justify-end space-x-3">
                <button
                  type="button"
                  onClick={() => setShowCreateModal(false)}
                  className="px-4 py-2 bg-gray-200 text-gray-700 rounded-md hover:bg-gray-300"
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
    </div>
  );
}

export default TeamManagement;
