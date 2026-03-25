import React, { useState, useEffect } from 'react';

interface WorkspaceMember {
  id: number;
  user_id: number;
  username: string;
  email: string;
  role: string;
  added_at: string;
}

interface ShareWorkspaceRequest {
  workspace_id: number;
  user_id: number;
  role: string;
}

interface WorkspaceSharingProps {
  workspaceId: number;
  workspaceName: string;
}

const WorkspaceSharing: React.FC<WorkspaceSharingProps> = ({ workspaceId, workspaceName }) => {
  const [members, setMembers] = useState<WorkspaceMember[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [shareWithUserId, setShareWithUserId] = useState('');
  const [selectedRole, setSelectedRole] = useState('viewer');
  const [error, setError] = useState('');
  const [success, setSuccess] = useState('');

  // Fetch workspace members
  useEffect(() => {
    fetchMembers();
  }, [workspaceId]);

  const fetchMembers = async () => {
    try {
      setIsLoading(true);
      const response = await fetch(`/api/v1/workspaces/${workspaceId}/members`);
      if (response.ok) {
        const data = await response.json();
        setMembers(data);
      }
    } catch (err) {
      setError('Failed to load workspace members');
    } finally {
      setIsLoading(false);
    }
  };

  const shareWorkspace = async () => {
    if (!shareWithUserId) {
      setError('Please enter a user ID');
      return;
    }

    try {
      const response = await fetch(`/api/v1/workspaces/${workspaceId}/share`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({
          workspace_id: workspaceId,
          user_id: parseInt(shareWithUserId),
          role: selectedRole,
        }),
      });

      if (response.ok) {
        setSuccess('Workspace shared successfully');
        setShareWithUserId('');
        fetchMembers();
        setError('');
      } else {
        const data = await response.json();
        setError(data.error || 'Failed to share workspace');
      }
    } catch (err) {
      setError('Failed to share workspace');
    }
  };

  const removeMember = async (memberId: number) => {
    try {
      const response = await fetch(`/api/v1/workspaces/${workspaceId}/members/${memberId}`, {
        method: 'DELETE',
      });

      if (response.ok) {
        setSuccess('Member removed successfully');
        fetchMembers();
        setError('');
      } else {
        setError('Failed to remove member');
      }
    } catch (err) {
      setError('Failed to remove member');
    }
  };

  const updateMemberRole = async (memberId: number, newRole: string) => {
    try {
      const response = await fetch(`/api/v1/workspaces/${workspaceId}/members/${memberId}/role`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ role: newRole }),
      });

      if (response.ok) {
        setSuccess('Role updated successfully');
        fetchMembers();
        setError('');
      } else {
        setError('Failed to update role');
      }
    } catch (err) {
      setError('Failed to update role');
    }
  };

  const getRoleBadgeColor = (role: string) => {
    switch (role) {
      case 'admin':
        return 'bg-red-100 text-red-800';
      case 'editor':
        return 'bg-blue-100 text-blue-800';
      case 'viewer':
        return 'bg-gray-100 text-gray-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  const getRoleDescription = (role: string) => {
    switch (role) {
      case 'admin':
        return 'Full access - can manage workspace and members';
      case 'editor':
        return 'Can edit and run workspaces';
      case 'viewer':
        return 'View-only access';
      default:
        return role;
    }
  };

  return (
    <div className="bg-white rounded-lg shadow p-6">
      <h2 className="text-xl font-semibold mb-4">Workspace Sharing</h2>
      <p className="text-gray-600 mb-6">Share "{workspaceName}" with team members</p>

      {/* Error and Success Messages */}
      {error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-4 py-3 rounded mb-4">
          {error}
        </div>
      )}
      {success && (
        <div className="bg-green-50 border border-green-200 text-green-700 px-4 py-3 rounded mb-4">
          {success}
        </div>
      )}

      {/* Share with User */}
      <div className="mb-6 p-4 bg-gray-50 rounded-lg">
        <h3 className="text-lg font-medium mb-3">Invite Member</h3>
        <div className="flex flex-wrap gap-3">
          <input
            type="number"
            placeholder="User ID"
            value={shareWithUserId}
            onChange={(e) => setShareWithUserId(e.target.value)}
            className="flex-1 min-w-0 px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <select
            value={selectedRole}
            onChange={(e) => setSelectedRole(e.target.value)}
            className="px-3 py-2 border border-gray-300 rounded-md focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            <option value="viewer">Viewer</option>
            <option value="editor">Editor</option>
            <option value="admin">Admin</option>
          </select>
          <button
            onClick={shareWorkspace}
            className="px-4 py-2 bg-blue-600 text-white rounded-md hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Share
          </button>
        </div>
        <p className="text-sm text-gray-500 mt-2">
          Enter the user ID of the person you want to share this workspace with
        </p>
      </div>

      {/* Current Members */}
      <div>
        <h3 className="text-lg font-medium mb-3">Current Members ({members.length})</h3>
        
        {isLoading ? (
          <div className="text-center py-4 text-gray-500">Loading members...</div>
        ) : members.length === 0 ? (
          <div className="text-center py-4 text-gray-500">No members yet. Share this workspace to add members.</div>
        ) : (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    User
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Role
                  </th>
                  <th className="px-4 py-3 text-left text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Added
                  </th>
                  <th className="px-4 py-3 text-right text-xs font-medium text-gray-500 uppercase tracking-wider">
                    Actions
                  </th>
                </tr>
              </thead>
              <tbody className="bg-white divide-y divide-gray-200">
                {members.map((member) => (
                  <tr key={member.id}>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <div className="text-sm font-medium text-gray-900">
                        {member.username || `User ${member.user_id}`}
                      </div>
                      {member.email && (
                        <div className="text-sm text-gray-500">{member.email}</div>
                      )}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap">
                      <select
                        value={member.role}
                        onChange={(e) => updateMemberRole(member.id, e.target.value)}
                        className={`px-2 py-1 text-sm rounded-full ${getRoleBadgeColor(member.role)} focus:outline-none focus:ring-2 focus:ring-blue-500`}
                      >
                        <option value="viewer">Viewer</option>
                        <option value="editor">Editor</option>
                        <option value="admin">Admin</option>
                      </select>
                      <div className="text-xs text-gray-500 mt-1">
                        {getRoleDescription(member.role)}
                      </div>
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-sm text-gray-500">
                      {new Date(member.added_at).toLocaleDateString()}
                    </td>
                    <td className="px-4 py-3 whitespace-nowrap text-right text-sm font-medium">
                      <button
                        onClick={() => removeMember(member.id)}
                        className="text-red-600 hover:text-red-900"
                      >
                        Remove
                      </button>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>

      {/* Role Descriptions */}
      <div className="mt-6 p-4 bg-blue-50 rounded-lg">
        <h4 className="text-sm font-medium text-blue-900 mb-2">Role Permissions</h4>
        <ul className="text-sm text-blue-700 space-y-1">
          <li><strong>Admin:</strong> Full access including member management</li>
          <li><strong>Editor:</strong> Can edit, run, and manage workspace configurations</li>
          <li><strong>Viewer:</strong> View-only access to workspace</li>
        </ul>
      </div>
    </div>
  );
};

export default WorkspaceSharing;

