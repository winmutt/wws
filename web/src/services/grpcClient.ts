// gRPC-Web API Client for WWS
// This file provides gRPC-Web clients for all services
// Generated from protobuf definitions

import { OrganizationServiceClient } from '../proto/organization/organization.client';
import { WorkspaceServiceClient } from '../proto/workspace/workspace.client';
import { UserServiceClient } from '../proto/user/user.client';
import { AuthServiceClient } from '../proto/auth/auth.client';
import {
  ListOrganizationsRequest,
  GetOrganizationRequest,
  CreateOrganizationRequest,
  UpdateOrganizationRequest,
  DeleteOrganizationRequest,
} from '../proto/organization/organization';
import {
  ListWorkspacesRequest,
  GetWorkspaceRequest,
  CreateWorkspaceRequest,
  UpdateWorkspaceRequest,
  DeleteWorkspaceRequest,
  StartWorkspaceRequest,
  StopWorkspaceRequest,
  RestartWorkspaceRequest,
} from '../proto/workspace/workspace';

// gRPC-Web endpoint (configure based on environment)
const GRPC_WEB_ENDPOINT = process.env.REACT_APP_GRPC_WEB_URL || 'http://localhost:9090';

// Initialize gRPC-Web clients
const organizationClient = new OrganizationServiceClient(GRPC_WEB_ENDPOINT);
const workspaceClient = new WorkspaceServiceClient(GRPC_WEB_ENDPOINT);
const userClient = new UserServiceClient(GRPC_WEB_ENDPOINT);
const authClient = new AuthServiceClient(GRPC_WEB_ENDPOINT);

// Organization gRPC client
const grpcOrganizations = {
  async list() {
    const request = ListOrganizationsRequest.fromPartial({});
    const response = await organizationClient.listOrganizations(request);
    return response.organizations.map(org => ({
      id: Number(org.id),
      name: org.name,
      owner_id: Number(org.ownerId),
      created_at: org.createdAt?.value || '',
      updated_at: org.updatedAt?.value || '',
    }));
  },

  async get(id: number) {
    const request = GetOrganizationRequest.fromPartial({ id });
    const response = await organizationClient.getOrganization(request);
    const org = response.organization;
    return {
      id: Number(org?.id),
      name: org?.name || '',
      owner_id: Number(org?.ownerId),
      created_at: org?.createdAt?.value || '',
      updated_at: org?.updatedAt?.value || '',
    };
  },

  async create(name: string) {
    const request = CreateOrganizationRequest.fromPartial({ name });
    const response = await organizationClient.createOrganization(request);
    const org = response.organization;
    return {
      id: Number(org?.id),
      name: org?.name || '',
      owner_id: Number(org?.ownerId),
      created_at: org?.createdAt?.value || '',
      updated_at: org?.updatedAt?.value || '',
    };
  },

  async update(id: number, name: string, description: string) {
    const request = UpdateOrganizationRequest.fromPartial({ id, name, description });
    const response = await organizationClient.updateOrganization(request);
    const org = response.organization;
    return {
      id: Number(org?.id),
      name: org?.name || '',
      description: org?.description || '',
      updated_at: org?.updatedAt?.value || '',
    };
  },

  async delete(id: number) {
    const request = DeleteOrganizationRequest.fromPartial({ id });
    await organizationClient.deleteOrganization(request);
  },
};

// Workspace gRPC client
const grpcWorkspaces = {
  async list(orgId: number) {
    const request = ListWorkspacesRequest.fromPartial({ organizationId: orgId });
    const response = await workspaceClient.listWorkspaces(request);
    return response.workspaces.map(ws => ({
      id: Number(ws.id),
      tag: ws.tag,
      name: ws.name,
      organization_id: Number(ws.organizationId),
      owner_id: Number(ws.ownerId),
      provider: ws.provider,
      status: ws.status,
      created_at: ws.createdAt?.value || '',
      updated_at: ws.updatedAt?.value || '',
    }));
  },

  async get(id: number) {
    const request = GetWorkspaceRequest.fromPartial({ id });
    const response = await workspaceClient.getWorkspace(request);
    const ws = response.workspace;
    return {
      id: Number(ws?.id),
      tag: ws?.tag || '',
      name: ws?.name || '',
      organization_id: Number(ws?.organizationId),
      owner_id: Number(ws?.ownerId),
      provider: ws?.provider || '',
      status: ws?.status,
      created_at: ws?.createdAt?.value || '',
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async create(data: { name: string; organization_id: number; cpu?: number; memory?: number; storage?: number; languages?: string[]; region?: string }) {
    const request = CreateWorkspaceRequest.fromPartial({
      name: data.name,
      organizationId: data.organization_id,
      cpu: data.cpu,
      memory: data.memory,
      storage: data.storage,
      languages: data.languages,
      region: data.region,
    });
    const response = await workspaceClient.createWorkspace(request);
    const ws = response.workspace;
    return {
      id: Number(ws?.id),
      tag: ws?.tag || '',
      name: ws?.name || '',
      organization_id: Number(ws?.organizationId),
      owner_id: Number(ws?.ownerId),
      provider: ws?.provider || '',
      status: ws?.status,
      created_at: ws?.createdAt?.value || '',
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async update(id: number, data: Partial<{ name: string; cpu: number; memory: number; storage: number; languages: string[]; region: string }>) {
    const request = UpdateWorkspaceRequest.fromPartial({ id, ...data });
    const response = await workspaceClient.updateWorkspace(request);
    const ws = response.workspace;
    return {
      id: Number(ws?.id),
      tag: ws?.tag || '',
      name: ws?.name || '',
      organization_id: Number(ws?.organizationId),
      owner_id: Number(ws?.ownerId),
      provider: ws?.provider || '',
      status: ws?.status,
      created_at: ws?.createdAt?.value || '',
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async delete(id: number) {
    const request = DeleteWorkspaceRequest.fromPartial({ id });
    await workspaceClient.deleteWorkspace(request);
  },

  async start(id: number) {
    const request = StartWorkspaceRequest.fromPartial({ id });
    const response = await workspaceClient.startWorkspace(request);
    const ws = response.workspace;
    return {
      id: Number(ws?.id),
      status: ws?.status,
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async stop(id: number) {
    const request = StopWorkspaceRequest.fromPartial({ id });
    const response = await workspaceClient.stopWorkspace(request);
    const ws = response.workspace;
    return {
      id: Number(ws?.id),
      status: ws?.status,
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async restart(id: number) {
    const request = RestartWorkspaceRequest.fromPartial({ id });
    const response = await workspaceClient.restartWorkspace(request);
    const ws = response.workspace;
    return {
      id: Number(ws?.id),
      status: ws?.status,
      updated_at: ws?.updatedAt?.value || '',
    };
  },
};

export { grpcOrganizations, grpcWorkspaces };
