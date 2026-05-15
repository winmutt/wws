const getApiBase = (): string => {
  if (process.env.REACT_APP_API_URL) {
    return process.env.REACT_APP_API_URL;
  }
  const hostname = window.location.hostname;
  const protocol = window.location.protocol;
  return `${protocol}//${hostname}:8080/api/v1`;
};

const API_BASE = getApiBase();

interface Session {
  user_id: number;
  expires_at: string;
}

const SESSION_TOKEN_KEY = 'session_token';
const GITHUB_USERNAME_KEY = 'github_username';

const getAuthHeaders = (): Headers => {
  const token = localStorage.getItem(SESSION_TOKEN_KEY);
  const headers = new Headers();
  if (token) {
    headers.set('Authorization', `Bearer ${token}`);
  }
  return headers;
};

const fetchWithAuth = async (url: string, options?: RequestInit): Promise<Response> => {
  const headers = getAuthHeaders();
  return fetch(url, { ...options, headers });
};

const logError = (method: string, endpoint: string, status: number, error: any) => {
  const debug = process.env.REACT_APP_DEBUG === '1' || window.location.search.includes('DEBUG=1');
  const message = error?.message || error?.error || 'Unknown error';
  const fullError = `${method} ${endpoint} - ${status}: ${message}`;
  
  if (debug) {
    console.error('[DEBUG]', fullError, error);
  }
  
  return message;
};

const request = async (method: string, endpoint: string, data?: any): Promise<any> => {
  const url = `${API_BASE}${endpoint}`;
  const headers = getAuthHeaders();
  headers.set('Content-Type', 'application/json');
  
  const init: RequestInit = {
    method,
    headers,
    credentials: 'include'
  };
  
  if (data) {
    init.body = JSON.stringify(data);
  }
  
  const response = await fetch(url, init);
  
  if (!response.ok) {
    let errorData: any = {};
    let errorMessage: string;
    let rawError: string = '';
    
    try {
      rawError = await response.text();
      try {
        errorData = JSON.parse(rawError);
      } catch {
        errorData = { message: rawError };
      }
    } catch {
      errorData = { message: 'Unknown error' };
    }
    
    errorMessage = errorData.message || errorData.error || rawError || `Request failed with status ${response.status}`;
    
    if (errorMessage === 'Unauthorized' || errorMessage.includes('Unauthorized')) {
      errorMessage = 'Authentication required. Please log in again.';
    } else if (errorMessage === 'missing session token') {
      errorMessage = 'Session expired. Please log in again.';
    } else if (errorMessage === 'owner_id is required') {
      errorMessage = 'Organization owner is required. Please contact your administrator.';
    } else if (errorMessage === 'organization_id parameter is required') {
      errorMessage = 'Organization ID is required.';
    } else if (errorMessage === 'Failed to create workspace') {
      errorMessage = 'Failed to create workspace. Please check your input and try again.';
    }
    
    logError(method, endpoint, response.status, errorData);
    throw new Error(errorMessage);
  }
  
  return response.json();
};

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

interface WorkspaceExport {
  id: number;
  workspace_id: number;
  export_path: string;
  format: string;
  file_size_mb?: number;
  status: string;
  error_message?: string;
  created_at: string;
  expires_at: string;
}

interface WorkspaceImport {
  id: number;
  export_id?: number;
  status: string;
  imported_workspace_id?: number;
  error?: string;
  created_at: string;
}

interface ExportRequest {
  format?: 'json' | 'tar' | 'zip';
  include_data?: boolean;
}

interface ImportRequest {
  export_id?: number;
  export_path?: string;
  format?: string;
  name?: string;
  organization_id: number;
}

const auth = {
  async login(githubCode: string): Promise<{ redirect: string }> {
    return request('GET', `/auth/github/callback?code=${encodeURIComponent(githubCode)}`);
  },

  async getSession(): Promise<Session | null> {
    try {
      return await request('GET', '/sessions');
    } catch {
      return null;
    }
  },

  async refreshSession(): Promise<Session> {
    return request('POST', '/sessions/refresh');
  },

  async logout(): Promise<void> {
    await request('DELETE', '/sessions/revoke');
  },

  async logoutAll(): Promise<void> {
    await request('DELETE', '/sessions/revoke-all');
  },

  async getCurrentUser(): Promise<User> {
    const session = await this.getSession();
    if (!session) {
      throw new Error('No active session');
    }
    
    return request('GET', `/users/${session.user_id}`);
  },

  async getVersion(): Promise<{ version: string; build_time: string }> {
    const response = await fetch(`${API_BASE}/version`);
    
    if (!response.ok) {
      throw new Error('Failed to fetch version');
    }
    
    return response.json();
  },
};

const users = {
  async list(): Promise<User[]> {
    return request('GET', '/users');
  },

  async get(id: number): Promise<User> {
    return request('GET', `/users/${id}`);
  },
};

const organizations = {
  async list(): Promise<Organization[]> {
    try {
      return await request('GET', '/organizations');
    } catch (error) {
      console.error('Failed to list organizations:', error);
      return [];
    }
  },

  async create(name: string): Promise<Organization> {
    return request('POST', '/organizations', { name });
  },

  async members(orgId: number): Promise<Member[]> {
    return request('GET', `/organizations/members?organization_id=${orgId}`);
  },

  async removeMember(orgId: number, userId: number): Promise<void> {
    await request('DELETE', '/organizations/members', { organization_id: orgId, user_id: userId });
  },

  async assignRole(orgId: number, userId: number, role: string): Promise<void> {
    await request('POST', '/organizations/roles', { organization_id: orgId, user_id: userId, role });
  },
};

const invitations = {
  async create(orgId: number, email: string): Promise<Invitation> {
    return request('POST', '/organizations/invitations', { organization_id: orgId, email });
  },

  async accept(token: string): Promise<void> {
    await request('POST', '/organizations/invitations/accept', { token });
  },
};

const workspaces = {
  async list(orgId: number): Promise<Workspace[]> {
    try {
      const response = await fetch(`${API_BASE}/workspaces?organization_id=${orgId}`, {
        credentials: 'include',
      });
      
      if (!response.ok) {
        throw new Error('Failed to fetch workspaces');
      }
      
      const data = await response.json();
      return data || [];
    } catch (error) {
      console.error('Failed to list workspaces:', error);
      return [];
    }
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

  async create(data: { name: string; organization_id: number; provider?: string; region?: string; config?: any }): Promise<Workspace> {
    return request('POST', '/workspaces', data);
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

  // Export/Import operations
  async exportWorkspace(workspaceId: number, options?: ExportRequest): Promise<WorkspaceExport> {
    const response = await fetch(`${API_BASE}/workspaces/export?workspace_id=${workspaceId}`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(options || {}),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to export workspace');
    }
    
    return response.json();
  },

  async importWorkspace(data: ImportRequest): Promise<WorkspaceImport> {
    const response = await fetch(`${API_BASE}/workspaces/import`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(data),
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to import workspace');
    }
    
    return response.json();
  },

  async getExportStatus(exportId: number): Promise<WorkspaceExport> {
    const response = await fetch(`${API_BASE}/workspaces/export/status?id=${exportId}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to get export status');
    }
    
    return response.json();
  },

  async getImportStatus(importId: number): Promise<WorkspaceImport> {
    const response = await fetch(`${API_BASE}/workspaces/import/status?id=${importId}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to get import status');
    }
    
    return response.json();
  },

  async listExports(): Promise<WorkspaceExport[]> {
    const response = await fetch(`${API_BASE}/workspaces/exports`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to list exports');
    }
    
    return response.json();
  },

  async listImports(orgId: number): Promise<WorkspaceImport[]> {
    const response = await fetch(`${API_BASE}/workspaces/imports?organization_id=${orgId}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to list imports');
    }
    
    return response.json();
  },

  async downloadExport(exportId: number): Promise<Blob> {
    const response = await fetch(`${API_BASE}/workspaces/export/download?id=${exportId}`, {
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to download export');
    }
    
    return response.blob();
  },

  async deleteExport(exportId: number): Promise<void> {
    const response = await fetch(`${API_BASE}/workspaces/export?id=${exportId}`, {
      method: 'DELETE',
      credentials: 'include',
    });
    
    if (!response.ok) {
      throw new Error('Failed to delete export');
    }
  },
};

export { auth, users, organizations, invitations, workspaces };
export type { Session, User, Organization, Workspace, Member, Invitation, WorkspaceExport, WorkspaceImport, ExportRequest, ImportRequest };
