const API_BASE = process.env.REACT_APP_API_URL || 'http://localhost:8080/api/v1';

interface Session {
  user_id: number;
  expires_at: string;
}

interface User {
  id: number;
  github_id: string;
  username: string;
  email: string;
}

interface Organization {
  id: number;
  name: string;
  owner_id: number;
  created_at: string;
}

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

interface Member {
  id: number;
  user_id: number;
  organization_id: number;
  role: string;
  user?: User;
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

const auth = {
  async login(githubCode: string): Promise<{ redirect: string }> {
    const response = await fetch(`${API_BASE}/auth/github/callback?code=${githubCode}`, {
      method: 'GET',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Login failed');
    }
    
    return response.json();
  },

  async getSession(): Promise<Session | null> {
    const response = await fetch(`${API_BASE}/sessions`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      return null;
    }
    
    return response.json();
  },

  async refreshSession(): Promise<Session> {
    const response = await fetch(`${API_BASE}/sessions/refresh`, {
      method: 'POST',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Session refresh failed');
    }
    
    return response.json();
  },

  async logout(): Promise<void> {
    await fetch(`${API_BASE}/sessions/revoke`, {
      method: 'DELETE',
      credentials: 'include',
    });
  },

  async logoutAll(): Promise<void> {
    await fetch(`${API_BASE}/sessions/revoke-all`, {
      method: 'DELETE',
      credentials: 'include',
    });
  },
};

const users = {
  async list(): Promise<User[]> {
    const response = await fetch(`${API_BASE}/users`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch users');
    }
    
    return response.json();
  },

  async get(id: number): Promise<User> {
    const response = await fetch(`${API_BASE}/users/${id}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch user');
    }
    
    return response.json();
  },
};

const organizations = {
  async list(): Promise<Organization[]> {
    const response = await fetch(`${API_BASE}/organizations`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch organizations');
    }
    
    return response.json();
  },

  async create(name: string): Promise<Organization> {
    const response = await fetch(`${API_BASE}/organizations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name }),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to create organization');
    }
    
    return response.json();
  },

  async members(orgId: number): Promise<Member[]> {
    const response = await fetch(`${API_BASE}/organizations/members?organization_id=${orgId}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch members');
    }
    
    return response.json();
  },

  async removeMember(orgId: number, userId: number): Promise<void> {
    const response = await fetch(`${API_BASE}/organizations/members`, {
      method: 'DELETE',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ organization_id: orgId, user_id: userId }),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to remove member');
    }
  },

  async assignRole(orgId: number, userId: number, role: string): Promise<void> {
    const response = await fetch(`${API_BASE}/organizations/roles`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ organization_id: orgId, user_id: userId, role }),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to assign role');
    }
  },
};

const invitations = {
  async create(orgId: number, email: string): Promise<Invitation> {
    const response = await fetch(`${API_BASE}/organizations/invitations`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ organization_id: orgId, email }),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to create invitation');
    }
    
    return response.json();
  },

  async accept(token: string): Promise<void> {
    const response = await fetch(`${API_BASE}/organizations/invitations/accept`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ token }),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to accept invitation');
    }
  },
};

const workspaces = {
  async list(orgId: number): Promise<Workspace[]> {
    const response = await fetch(`${API_BASE}/workspaces?organization_id=${orgId}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch workspaces');
    }
    
    return response.json();
  },

  async get(id: number): Promise<Workspace> {
    const response = await fetch(`${API_BASE}/workspaces/${id}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to fetch workspace');
    }
    
    return response.json();
  },

  async create(data: { name: string; organization_id: number; provider?: string; config?: any }): Promise<Workspace> {
    const response = await fetch(`${API_BASE}/workspaces`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to create workspace');
    }
    
    return response.json();
  },

  async update(id: number, data: Partial<Workspace>): Promise<Workspace> {
    const response = await fetch(`${API_BASE}/workspaces/${id}`, {
      method: 'PUT',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to update workspace');
    }
    
    return response.json();
  },

  async delete(id: number): Promise<void> {
    const response = await fetch(`${API_BASE}/workspaces/${id}`, {
      method: 'DELETE',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete workspace');
    }
  },

  async start(id: number): Promise<void> {
    const response = await fetch(`${API_BASE}/workspaces/${id}/start`, {
      method: 'POST',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to start workspace');
    }
  },

  async stop(id: number): Promise<void> {
    const response = await fetch(`${API_BASE}/workspaces/${id}/stop`, {
      method: 'POST',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to stop workspace');
    }
  },

  async restart(id: number): Promise<void> {
    const response = await fetch(`${API_BASE}/workspaces/${id}/restart`, {
      method: 'POST',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to restart workspace');
    }
  },
};

export { auth, users, organizations, invitations, workspaces };
export type { Session, User, Organization, Workspace, Member, Invitation };
