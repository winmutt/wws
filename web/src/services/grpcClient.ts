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
import { GrpcWebFetchTransport } from '@protobuf-ts/grpcweb-transport';

// gRPC-Web endpoint (configure based on environment)
const GRPC_WEB_ENDPOINT = process.env.REACT_APP_GRPC_WEB_URL || 'http://localhost:9090';

// Create gRPC-Web transport
const transport = new GrpcWebFetchTransport({
  baseUrl: GRPC_WEB_ENDPOINT,
});

// Initialize gRPC-Web clients with transport
const organizationClient = new OrganizationServiceClient(transport);
const workspaceClient = new WorkspaceServiceClient(transport);
const userClient = new UserServiceClient(transport);
const authClient = new AuthServiceClient(transport);

// Organization gRPC client
const grpcOrganizations = {
  async list() {
    const request = ListOrganizationsRequest.create({});
    const response = await organizationClient.listOrganizations(request);
    return response.response.organizations.map(org => ({
      id: Number(org.id),
      name: org.name,
      owner_id: Number(org.ownerId),
      created_at: org.createdAt?.value || '',
      updated_at: org.updatedAt?.value || '',
    }));
  },

  async get(id: number) {
    const request = GetOrganizationRequest.create({ id });
    const response = await organizationClient.getOrganization(request);
    const org = response.response.organization;
    return {
      id: Number(org?.id),
      name: org?.name || '',
      owner_id: Number(org?.ownerId),
      created_at: org?.createdAt?.value || '',
      updated_at: org?.updatedAt?.value || '',
    };
  },

  async create(name: string) {
    const request = CreateOrganizationRequest.create({ name });
    const response = await organizationClient.createOrganization(request);
    const org = response.response.organization;
    return {
      id: Number(org?.id),
      name: org?.name || '',
      owner_id: Number(org?.ownerId),
      created_at: org?.createdAt?.value || '',
      updated_at: org?.updatedAt?.value || '',
    };
  },

  async update(id: number, name: string, description: string) {
    const request = UpdateOrganizationRequest.create({ id, name, description });
    const response = await organizationClient.updateOrganization(request);
    const org = response.response.organization;
    return {
      id: Number(org?.id),
      name: org?.name || '',
      description: org?.description || '',
      updated_at: org?.updatedAt?.value || '',
    };
  },

  async delete(id: number) {
    const request = DeleteOrganizationRequest.create({ id });
    await organizationClient.deleteOrganization(request);
  },
};

// Workspace gRPC client
const grpcWorkspaces = {
  async list(orgId: number) {
    const request = ListWorkspacesRequest.create({ organizationId: orgId });
    const response = await workspaceClient.listWorkspaces(request);
    return response.response.workspaces.map(ws => ({
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
    const request = GetWorkspaceRequest.create({ id });
    const response = await workspaceClient.getWorkspace(request);
    const ws = response.response.workspace;
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
    const request = CreateWorkspaceRequest.create({
      name: data.name,
      organizationId: data.organization_id,
      cpu: data.cpu,
      memory: data.memory,
      storage: data.storage,
      languages: data.languages,
      region: data.region,
    });
    const response = await workspaceClient.createWorkspace(request);
    const ws = response.response.workspace;
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
    const request = UpdateWorkspaceRequest.create({ id, ...data });
    const response = await workspaceClient.updateWorkspace(request);
    const ws = response.response.workspace;
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
    const request = DeleteWorkspaceRequest.create({ id });
    await workspaceClient.deleteWorkspace(request);
  },

  async start(id: number) {
    const request = StartWorkspaceRequest.create({ id });
    const response = await workspaceClient.startWorkspace(request);
    const ws = response.response.workspace;
    return {
      id: Number(ws?.id),
      status: ws?.status,
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async stop(id: number) {
    const request = StopWorkspaceRequest.create({ id });
    const response = await workspaceClient.stopWorkspace(request);
    const ws = response.response.workspace;
    return {
      id: Number(ws?.id),
      status: ws?.status,
      updated_at: ws?.updatedAt?.value || '',
    };
  },

  async restart(id: number) {
    const request = RestartWorkspaceRequest.create({ id });
    const response = await workspaceClient.restartWorkspace(request);
    const ws = response.response.workspace;
    return {
      id: Number(ws?.id),
      status: ws?.status,
      updated_at: ws?.updatedAt?.value || '',
    };
  },
};

export { grpcOrganizations, grpcWorkspaces };
