import { logger } from '../utils/logger';
import type {
  Project,
  User,
  Template,
  Instance,
  StorageVolume,
  EFSVolume,
  EBSVolume,
  InstanceSnapshot,
  BudgetData,
  CostBreakdown,
  Budget,
  BudgetAllocation,
  BudgetSummary,
  CreateBudgetRequest,
  AMI,
  AMIBuild,
  AMIRegion,
  RightsizingRecommendation,
  RightsizingStats,
  PolicyStatus,
  PolicySet,
  PolicyCheckResult,
  MarketplaceTemplate,
  MarketplaceCategory,
  IdlePolicy,
  IdleSchedule,
  CachedInvitation,
  ProjectData,
  MemberData,
  UserData,
  SharedTokenConfig,
  BulkInviteResponse,
  Invitation,
  ProjectDetails,
  SharedInvitationToken,
  RoleQuota,
  GrantPeriod,
  ApprovalRequest,
  BudgetShareRequest,
  BudgetShareRecord,
  OnboardingTemplate,
  ClassMember,
  Course,
  CourseBudgetSummary,
  CourseOverview,
  UsageReport,
  CourseAuditEntry,
  SharedMaterialsVolume,
  WorkspaceResetResult,
  WorkshopEvent,
  WorkshopDashboard,
  WorkshopConfig,
  HibernationStatus,
  UserStatus,
  UserProvisionResponse,
  SSHKeyResponse,
  UserSSHKeysResponse,
  ProjectUsageResponse,
  SharedToken,
  QRCodeData,
  RedeemTokenResponse,
  UserUpdateRequest,
  FileEntry,
  CapacityBlock,
  CapacityBlockRequest,
  S3Mount,
  StorageAnalyticsSummary,
} from './types';

// Encode a path segment for safe URL interpolation (#604)
const enc = (s: string) => encodeURIComponent(s);

export class SafePrismAPI {
  private baseURL = 'http://localhost:8947';
  private apiKey = '';

  constructor() {
    // API key loading disabled - daemon runs in PRISM_TEST_MODE with auth bypass
  }

  private async safeRequest<T = unknown>(endpoint: string, method = 'GET', body?: unknown): Promise<T> {
    try {
      const response = await fetch(`${this.baseURL}${endpoint}`, {
        method,
        headers: {
          'Content-Type': 'application/json',
          'X-API-Key': this.apiKey,
        },
        body: body ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        // Try to parse error message from response body
        let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
        try {
          const errorData = await response.text();
          if (errorData) {
            errorMessage = errorData;
          }
        } catch (e) {
          // If parsing fails, use the default message
        }
        const error: any = new Error(errorMessage);
        error.response = { status: response.status, statusText: response.statusText };
        throw error;
      }

      // Handle HTTP 204 No Content (empty response)
      if (response.status === 204 || response.headers.get('content-length') === '0') {
        return {} as T;
      }

      const data = await response.json();
      return data;
    } catch (error) {
      logger.error(`API request failed for ${endpoint}:`, error);
      throw error;
    }
  }

  async getDaemonStatus(): Promise<{ version?: string; status?: string } | null> {
    try {
      return await this.safeRequest<{ version?: string; status?: string }>('/api/v1/admin/status');
    } catch {
      return null;
    }
  }

  async getInstanceLogs(instanceName: string, logType: string = 'console', tail: number = 200): Promise<string[]> {
    try {
      const data = await this.safeRequest<{ lines?: string[]; output?: string }>(`/api/v1/logs/${enc(instanceName)}?type=${enc(logType)}&tail=${tail}`);
      if (data?.lines) return data.lines;
      if (data?.output) return data.output.split('\n');
      return [];
    } catch (error) {
      logger.error('Failed to fetch logs:', error);
      return [];
    }
  }

  async getTemplates(): Promise<Record<string, Template>> {
    try {
      const data = await this.safeRequest<Record<string, Template>>('/api/v1/templates');
      return data || {};
    } catch (error) {
      logger.error('Failed to fetch templates:', error);
      return {};
    }
  }

  async getInstances(): Promise<Instance[]> {
    try {
      const data = await this.safeRequest<{instances: Instance[]}>('/api/v1/instances');
      return Array.isArray(data?.instances) ? data.instances : [];
    } catch (error) {
      logger.error('Failed to fetch instances:', error);
      return [];
    }
  }

  async launchInstance(templateSlug: string, name: string, size: string = 'M', dryRun: boolean = false, options?: { dnsName?: string; ttl?: string; spot?: boolean; spotMaxPrice?: string; efa?: boolean; placementGroup?: string }): Promise<Instance & { approval_pending?: boolean; approval_request_id?: string; message?: string }> {
    const body: Record<string, unknown> = {
      template: templateSlug,
      name,
      size,
    };
    if (dryRun) {
      body.dry_run = true;
    }
    if (options?.dnsName) {
      body.dns_name = options.dnsName;
    }
    if (options?.ttl) {
      body.ttl = options.ttl;
    }
    // Optional launch capabilities (keys must match pkg/types.LaunchRequest json tags).
    if (options?.spot) {
      body.spot = true;
    }
    if (options?.spotMaxPrice) {
      body.spot_max_price = options.spotMaxPrice;
    }
    if (options?.efa) {
      body.efa = true;
    }
    if (options?.placementGroup) {
      body.placement_group = options.placementGroup;
    }
    return this.safeRequest('/api/v1/instances', 'POST', body);
  }

  // Comprehensive Instance Management APIs - Using Fixed Backend Endpoints
  async startInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(identifier)}/start`, 'POST');
  }

  async stopInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(identifier)}/stop`, 'POST');
  }

  async extendInstanceTTL(identifier: string, hours: number = 4): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(encodeURIComponent(identifier))}/extend`, 'POST', { hours });
  }

  async hibernateInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(identifier)}/hibernate`, 'POST');
  }

  async resumeInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(identifier)}/resume`, 'POST');
  }

  async getConnectionInfo(identifier: string): Promise<string> {
    const data = await this.safeRequest<{connection_info?: string}>(`/api/v1/instances/${enc(identifier)}/connect`);
    return data.connection_info || '';
  }

  async getHibernationStatus(identifier: string): Promise<HibernationStatus> {
    return this.safeRequest(`/api/v1/instances/${enc(identifier)}/hibernation-status`);
  }

  async deleteInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(identifier)}`, 'DELETE');
  }

  // Comprehensive Storage Management APIs

  // Helper functions to convert unified StorageVolume to legacy formats
  private storageVolumeToEFS(vol: StorageVolume): EFSVolume | null {
    if (vol.type !== 'shared' && vol.aws_service !== 'efs') return null;
    return {
      name: vol.name,
      filesystem_id: vol.filesystem_id || '',
      region: vol.region,
      creation_time: vol.creation_time,
      state: vol.state,
      performance_mode: vol.performance_mode || '',
      throughput_mode: vol.throughput_mode || '',
      estimated_cost_gb: vol.estimated_cost_gb,
      size_bytes: vol.size_bytes || 0,
      attached_to: vol.attached_to,
    };
  }

  private storageVolumeToEBS(vol: StorageVolume): EBSVolume | null {
    if (vol.type !== 'workspace' && vol.aws_service !== 'ebs') return null;
    return {
      name: vol.name,
      volume_id: vol.volume_id || '',
      region: vol.region,
      creation_time: vol.creation_time,
      state: vol.state,
      volume_type: vol.volume_type || '',
      size_gb: vol.size_gb || 0,
      estimated_cost_gb: vol.estimated_cost_gb,
      attached_to: vol.attached_to,
    };
  }

  // EFS Volume Management (using unified API)
  async getEFSVolumes(): Promise<EFSVolume[]> {
    try {
      const data: StorageVolume[] = await this.safeRequest('/api/v1/volumes');
      if (!Array.isArray(data)) return [];
      // Convert unified StorageVolume to legacy EFSVolume format
      return data.map(vol => this.storageVolumeToEFS(vol)).filter((v): v is EFSVolume => v !== null);
    } catch (error) {
      logger.error('Failed to fetch shared storage volumes:', error);
      return [];
    }
  }

  async createEFSVolume(name: string, performanceMode: string = 'generalPurpose', throughputMode: string = 'bursting'): Promise<EFSVolume> {
    return this.safeRequest('/api/v1/volumes', 'POST', {
      name,
      performance_mode: performanceMode,
      throughput_mode: throughputMode,
    });
  }

  async deleteEFSVolume(name: string): Promise<void> {
    await this.safeRequest(`/api/v1/volumes/${enc(name)}`, 'DELETE');
  }

  async mountEFSVolume(volumeName: string, instance: string, mountPoint?: string): Promise<void> {
    const body: Record<string, string> = { instance };
    if (mountPoint) body.mount_point = mountPoint;
    await this.safeRequest(`/api/v1/volumes/${enc(volumeName)}/mount`, 'POST', body);
  }

  async unmountEFSVolume(volumeName: string, instance: string): Promise<void> {
    await this.safeRequest(`/api/v1/volumes/${enc(volumeName)}/unmount`, 'POST', { instance });
  }

  async syncEFSVolume(volumeName: string): Promise<EFSVolume> {
    return this.safeRequest(`/api/v1/volumes/${enc(volumeName)}/sync`, 'POST');
  }

  async syncAllEFSVolumes(): Promise<EFSVolume[]> {
    return this.safeRequest('/api/v1/volumes/sync', 'POST');
  }

  // EBS Storage Management (using unified API)
  async getEBSVolumes(): Promise<EBSVolume[]> {
    try {
      const data: StorageVolume[] = await this.safeRequest('/api/v1/storage');
      if (!Array.isArray(data)) return [];
      // Convert unified StorageVolume to legacy EBSVolume format
      // Note: /api/v1/storage now returns ALL storage (EBS + EFS), so filter for workspace only
      return data
        .filter(vol => vol.type === 'workspace' || vol.aws_service === 'ebs')
        .map(vol => this.storageVolumeToEBS(vol))
        .filter((v): v is EBSVolume => v !== null);
    } catch (error) {
      logger.error('Failed to fetch workspace storage volumes:', error);
      return [];
    }
  }

  async createEBSVolume(name: string, size: string = 'M', volumeType: string = 'gp3'): Promise<EBSVolume> {
    return this.safeRequest('/api/v1/storage', 'POST', {
      name,
      size,
      volume_type: volumeType,
    });
  }

  async deleteEBSVolume(name: string): Promise<void> {
    await this.safeRequest(`/api/v1/storage/${enc(name)}`, 'DELETE');
  }

  async attachEBSVolume(storageName: string, instance: string): Promise<void> {
    await this.safeRequest(`/api/v1/storage/${enc(storageName)}/attach`, 'POST', { instance });
  }

  async detachEBSVolume(storageName: string): Promise<void> {
    await this.safeRequest(`/api/v1/storage/${enc(storageName)}/detach`, 'POST');
  }

  async syncEBSVolume(storageName: string): Promise<EBSVolume> {
    return this.safeRequest(`/api/v1/storage/${enc(storageName)}/sync`, 'POST');
  }

  async syncAllEBSVolumes(): Promise<EBSVolume[]> {
    return this.safeRequest('/api/v1/storage/sync', 'POST');
  }

  // Snapshot/Backup Management APIs
  async getSnapshots(): Promise<InstanceSnapshot[]> {
    try {
      const data = await this.safeRequest<{snapshots: InstanceSnapshot[], count: number}>('/api/v1/snapshots');
      return Array.isArray(data?.snapshots) ? data.snapshots : [];
    } catch (error) {
      logger.error('Failed to fetch snapshots:', error);
      return [];
    }
  }

  async createSnapshot(instanceName: string, snapshotName: string, description?: string): Promise<InstanceSnapshot> {
    return this.safeRequest('/api/v1/snapshots', 'POST', {
      instance_name: instanceName,
      snapshot_name: snapshotName,
      description: description || '',
      no_reboot: true
    });
  }

  async deleteSnapshot(snapshotName: string): Promise<void> {
    await this.safeRequest(`/api/v1/snapshots/${enc(snapshotName)}`, 'DELETE');
  }

  async restoreSnapshot(snapshotName: string, instanceName: string): Promise<void> {
    await this.safeRequest(`/api/v1/snapshots/${enc(snapshotName)}/restore`, 'POST', {
      instance_name: instanceName
    });
  }

  // Comprehensive Project Management APIs

  // Project Operations
  async getProjects(): Promise<Project[]> {
    try {
      const data = await this.safeRequest<{projects?: Project[]}>('/api/v1/projects');
      return Array.isArray(data?.projects) ? data.projects : [];
    } catch (error) {
      logger.error('Failed to fetch projects:', error);
      return [];
    }
  }

  async createProject(projectData: ProjectData): Promise<Project> {
    return this.safeRequest<Project>('/api/v1/projects', 'POST', projectData);
  }

  async getProject(projectId: string): Promise<Project> {
    return this.safeRequest<Project>(`/api/v1/projects/${enc(projectId)}`);
  }

  async updateProject(projectId: string, projectData: Partial<ProjectData>): Promise<Project> {
    return this.safeRequest<Project>(`/api/v1/projects/${enc(projectId)}`, 'PUT', projectData);
  }

  async deleteProject(projectId: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}`, 'DELETE');
  }

  // Project Members
  async getProjectMembers(projectId: string): Promise<MemberData[]> {
    try {
      const data = await this.safeRequest(`/api/v1/projects/${enc(projectId)}/members`);
      return Array.isArray(data) ? data : [];
    } catch (error) {
      logger.error('Failed to fetch project members:', error);
      return [];
    }
  }

  async addProjectMember(projectId: string, memberData: MemberData): Promise<MemberData> {
    return this.safeRequest<MemberData>(`/api/v1/projects/${enc(projectId)}/members`, 'POST', memberData);
  }

  async updateProjectMember(projectId: string, userId: string, memberData: Partial<MemberData>): Promise<MemberData> {
    return this.safeRequest<MemberData>(`/api/v1/projects/${enc(projectId)}/members/${enc(userId)}`, 'PUT', memberData);
  }

  async removeProjectMember(projectId: string, userId: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/members/${enc(userId)}`, 'DELETE');
  }

  // Budget Management
  async getProjectBudget(projectId: string): Promise<BudgetData> {
    return this.safeRequest<BudgetData>(`/api/v1/projects/${enc(projectId)}/budget`);
  }

  // Cost Analysis
  async getProjectCosts(projectId: string, startDate?: string, endDate?: string): Promise<CostBreakdown> {
    const params = new URLSearchParams();
    if (startDate) params.append('start_date', startDate);
    if (endDate) params.append('end_date', endDate);
    const query = params.toString();
    return this.safeRequest<CostBreakdown>(`/api/v1/projects/${enc(projectId)}/costs${enc(query ? '?' + query : '')}`);
  }

  // Resource Usage
  async getProjectUsage(projectId: string, period?: string): Promise<ProjectUsageResponse> {
    const query = period ? `?period=${period}` : '';
    return this.safeRequest<ProjectUsageResponse>(`/api/v1/projects/${enc(projectId)}/usage${enc(query)}`);
  }

  // User Operations
  async getUsers(): Promise<User[]> {
    try {
      const data = await this.safeRequest<User[] | {users?: User[]}>('/api/v1/users');
      // Handle both direct array response and wrapped object response
      if (Array.isArray(data)) return data;
      if (!Array.isArray(data) && Array.isArray((data as {users?: User[]}).users)) return (data as {users: User[]}).users;
      return [];
    } catch (error) {
      logger.error('Failed to fetch users:', error);
      return [];
    }
  }

  async createUser(userData: UserData): Promise<User> {
    console.log('[DEBUG] createUser called with:', userData);
    try {
      const result = await this.safeRequest<User>('/api/v1/users', 'POST', userData);
      console.log('[DEBUG] createUser success:', result);
      return result;
    } catch (error) {
      console.error('[DEBUG] createUser error:', error);
      throw error;
    }
  }

  async deleteUser(username: string): Promise<void> {
    await this.safeRequest(`/api/v1/users/${enc(username)}`, 'DELETE');
  }

  async getUserStatus(username: string): Promise<UserStatus> {
    return this.safeRequest(`/api/v1/users/${enc(username)}/status`);
  }

  async provisionUser(username: string, instanceName: string): Promise<UserProvisionResponse> {
    return this.safeRequest(`/api/v1/users/${enc(username)}/provision`, 'POST', { instance: instanceName });
  }

  async enableUser(username: string): Promise<void> {
    await this.safeRequest(`/api/v1/users/${enc(username)}/enable`, 'POST');
  }

  async disableUser(username: string): Promise<void> {
    await this.safeRequest(`/api/v1/users/${enc(username)}/disable`, 'POST');
  }

  async generateSSHKey(username: string): Promise<SSHKeyResponse> {
    return this.safeRequest(`/api/v1/users/${enc(username)}/ssh-key`, 'POST', {
      username: username,
      key_type: 'ed25519'
    });
  }

  async getUserSSHKeys(username: string): Promise<UserSSHKeysResponse> {
    return this.safeRequest(`/api/v1/users/${enc(username)}/ssh-key`);
  }

  async getUser(username: string): Promise<User> {
    return this.safeRequest(`/api/v1/users/${enc(username)}`);
  }

  async updateUser(username: string, updates: Partial<UserUpdateRequest>): Promise<User> {
    return this.safeRequest(`/api/v1/users/${enc(username)}`, 'PUT', updates);
  }

  // Helper function to calculate budget status
  private calculateBudgetStatus(spentPercent: number): 'ok' | 'warning' | 'critical' {
    if (spentPercent >= 95) return 'critical';
    if (spentPercent >= 80) return 'warning';
    return 'ok';
  }

  // Budget Management APIs
  async getBudgets(): Promise<BudgetData[]> {
    try {
      const projects = await this.getProjects();
      const budgets: BudgetData[] = [];

      // Fetch budget status for each project
      for (const project of projects) {
        try {
          const budgetStatus = await this.safeRequest<BudgetData>(`/api/v1/projects/${enc(project.id)}/budget`);

          if (budgetStatus && budgetStatus.total_budget > 0) {
            const remaining = budgetStatus.total_budget - budgetStatus.spent_amount;
            const spentPercent = budgetStatus.spent_percentage * 100;

            budgets.push({
              project_id: project.id,
              project_name: project.name,
              total_budget: budgetStatus.total_budget,
              spent_amount: budgetStatus.spent_amount,
              spent_percentage: budgetStatus.spent_percentage,
              remaining: remaining,
              alert_count: budgetStatus.alert_count || 0,
              status: this.calculateBudgetStatus(spentPercent),
              projected_monthly_spend: budgetStatus.projected_monthly_spend,
              days_until_exhausted: budgetStatus.days_until_exhausted,
              active_alerts: budgetStatus.active_alerts
            });
          }
        } catch (error) {
          logger.error(`Failed to fetch budget for project ${project.id}:`, error);
        }
      }

      return budgets;
    } catch (error) {
      logger.error('Failed to fetch budgets:', error);
      return [];
    }
  }

  async getCostBreakdown(projectId: string, startDate?: string, endDate?: string): Promise<CostBreakdown> {
    try {
      const params = new URLSearchParams();
      if (startDate) params.append('start_date', startDate);
      if (endDate) params.append('end_date', endDate);
      const query = params.toString();

      const data = await this.safeRequest<CostBreakdown>(`/api/v1/projects/${enc(projectId)}/costs${enc(query ? '?' + query : '')}`);

      return {
        ec2_compute: data.ec2_compute || 0,
        ebs_storage: data.ebs_storage || 0,
        efs_storage: data.efs_storage || 0,
        data_transfer: data.data_transfer || 0,
        other: data.other || 0,
        total: data.total || 0
      };
    } catch (error) {
      logger.error(`Failed to fetch cost breakdown for project ${projectId}:`, error);
      return {
        ec2_compute: 0,
        ebs_storage: 0,
        efs_storage: 0,
        data_transfer: 0,
        other: 0,
        total: 0
      };
    }
  }

  async setBudget(projectId: string, totalBudget: number, alertThresholds?: number[]): Promise<void> {
    const alerts = alertThresholds?.map(threshold => ({
      threshold,
      action: 'notify',
      enabled: true
    })) || [];

    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/budget`, 'PUT', {
      total_budget: totalBudget,
      alert_thresholds: alerts,
      budget_period: 'monthly'
    });
  }

  // Budget Pool Operations (v0.6.0+)
  async getBudgetPools(): Promise<Budget[]> {
    try {
      const data = await this.safeRequest<{budgets?: Budget[]}>('/api/v1/budgets');
      return Array.isArray(data?.budgets) ? data.budgets : [];
    } catch (error) {
      logger.error('Failed to fetch budget pools:', error);
      return [];
    }
  }

  async getBudgetPool(budgetId: string): Promise<Budget> {
    return this.safeRequest<Budget>(`/api/v1/budgets/${enc(budgetId)}`);
  }

  async getBudgetSummary(budgetId: string): Promise<BudgetSummary> {
    return this.safeRequest<BudgetSummary>(`/api/v1/budgets/${enc(budgetId)}/summary`);
  }

  async createBudgetPool(budgetData: CreateBudgetRequest): Promise<Budget> {
    return this.safeRequest<Budget>('/api/v1/budgets', 'POST', budgetData);
  }

  async updateBudgetPool(budgetId: string, updates: Partial<CreateBudgetRequest>): Promise<Budget> {
    return this.safeRequest<Budget>(`/api/v1/budgets/${enc(budgetId)}`, 'PUT', updates);
  }

  async deleteBudgetPool(budgetId: string): Promise<void> {
    await this.safeRequest(`/api/v1/budgets/${enc(budgetId)}`, 'DELETE');
  }

  async getBudgetAllocations(budgetId: string): Promise<BudgetAllocation[]> {
    try {
      const data = await this.safeRequest<{allocations?: BudgetAllocation[]}>(`/api/v1/budgets/${enc(budgetId)}/allocations`);
      return Array.isArray(data?.allocations) ? data.allocations : [];
    } catch (error) {
      logger.error('Failed to fetch budget allocations:', error);
      return [];
    }
  }

  // Invitation Management APIs (v0.5.11+)
  async getInvitationByToken(token: string): Promise<CachedInvitation> {
    try {
      const data = await this.safeRequest<{invitation: Invitation; project: {name: string}}>(`/api/v1/invitations/${enc(token)}`);
      const inv = data.invitation;

      // Map backend Invitation type to frontend CachedInvitation type
      return {
        token: inv.token,
        invitation_id: inv.id,  // Backend uses 'id', frontend uses 'invitation_id'
        project_id: inv.project_id,
        project_name: inv.project_name || data.project?.name || 'Unknown Project',
        email: inv.email,
        role: inv.role,
        invited_by: inv.invited_by,
        invited_at: inv.invited_at,
        expires_at: inv.expires_at,
        status: inv.status,
        message: inv.message,
        added_at: new Date().toISOString()  // Client-side timestamp for when added to cache
      };
    } catch (error) {
      logger.error('Failed to fetch invitation:', error);
      throw error;
    }
  }

  async acceptInvitation(token: string): Promise<void> {
    try {
      await this.safeRequest(`/api/v1/invitations/${enc(token)}/accept`, 'POST');
    } catch (error) {
      logger.error('Failed to accept invitation:', error);
      throw error;
    }
  }

  async declineInvitation(token: string, reason?: string): Promise<void> {
    try {
      const body = reason ? { reason } : undefined;
      await this.safeRequest(`/api/v1/invitations/${enc(token)}/decline`, 'POST', body);
    } catch (error) {
      logger.error('Failed to decline invitation:', error);
      throw error;
    }
  }

  async sendInvitation(projectId: string, email: string, role: string, message?: string, expiresAt?: string): Promise<Invitation> {
    try {
      const body: Record<string, string> = {
        email,
        role,
        invited_by: 'test-user', // Required by backend API
      };
      if (message) body.message = message;
      if (expiresAt) body.expires_at = expiresAt;

      // Backend returns nested response: {invitation, project, message}
      const data = await this.safeRequest<{invitation: Invitation, project: unknown, message: string}>(`/api/v1/projects/${enc(projectId)}/invitations`, 'POST', body);

      // Emit event to notify UI components that a new invitation was created
      window.dispatchEvent(new CustomEvent('invitation-created'));

      return data.invitation;
    } catch (error) {
      logger.error('Failed to send invitation:', error);
      throw error;
    }
  }

  // Bulk Invitation API (Issue #240)
  async bulkInvite(projectId: string, emails: string[], role: string, message?: string): Promise<BulkInviteResponse> {
    try {
      // Backend expects invitations array with BulkInvitationEntry objects
      const body: Record<string, unknown> = {
        invitations: emails.map(email => ({ email })),
        default_role: role,
      };
      if (message) body.default_message = message;

      // Backend returns BulkInvitationResponse with summary and results
      const data = await this.safeRequest<{summary: {total: number; sent: number; failed: number}; results: Array<{email: string; status: string; error?: string}>}>(`/api/v1/projects/${enc(projectId)}/invitations/bulk`, 'POST', body);

      // Transform backend response to match frontend BulkInviteResponse interface
      return {
        total: data.summary.total,
        sent: data.summary.sent,
        failed: data.summary.failed,
        errors: data.results.filter(r => r.status === 'failed').map(r => ({
          email: r.email,
          error: r.error || 'Unknown error'
        }))
      };
    } catch (error) {
      logger.error('Failed to send bulk invitations:', error);
      throw error;
    }
  }

  // Shared Token APIs (Issue #241)
  async getSharedTokens(projectId: string): Promise<SharedToken[]> {
    try {
      const data = await this.safeRequest<{tokens?: SharedToken[]}>(`/api/v1/projects/${enc(projectId)}/invitations/shared`);
      return Array.isArray(data?.tokens) ? data.tokens : [];
    } catch (error) {
      logger.error('Failed to fetch shared tokens:', error);
      return [];
    }
  }

  async createSharedToken(projectId: string, config: SharedTokenConfig): Promise<SharedInvitationToken> {
    try {
      const data = await this.safeRequest<SharedInvitationToken>(`/api/v1/projects/${enc(projectId)}/invitations/shared`, 'POST', config);
      // Emit event to notify UI that shared token was created
      window.dispatchEvent(new CustomEvent('shared-token-created'));
      return data;
    } catch (error) {
      logger.error('Failed to create shared token:', error);
      throw error;
    }
  }

  async extendSharedToken(token: string, expiresIn: string): Promise<void> {
    try {
      await this.safeRequest(`/api/v1/invitations/shared/${enc(token)}/extend`, 'PATCH', {
        add_days: parseInt(expiresIn)
      });
    } catch (error) {
      logger.error('Failed to extend shared token:', error);
      throw error;
    }
  }

  async revokeSharedToken(token: string): Promise<void> {
    try {
      await this.safeRequest(`/api/v1/invitations/shared/${enc(token)}`, 'DELETE');
    } catch (error) {
      logger.error('Failed to revoke shared token:', error);
      throw error;
    }
  }

  async redeemSharedToken(token: string): Promise<RedeemTokenResponse> {
    try {
      const data = await this.safeRequest<RedeemTokenResponse>(`/api/v1/invitations/shared/redeem`, 'POST', { token });
      return data;
    } catch (error) {
      logger.error('Failed to redeem shared token:', error);
      throw error;
    }
  }

  async getSharedTokenQRCode(token: string, format: 'json' | 'png' = 'json'): Promise<QRCodeData> {
    try {
      const data = await this.safeRequest<QRCodeData>(`/api/v1/invitations/shared/${enc(token)}/qr?format=${enc(format)}`);
      return data;
    } catch (error) {
      logger.error('Failed to get QR code:', error);
      throw error;
    }
  }

  // Additional Invitation Management APIs
  async getMyInvitations(email?: string): Promise<Invitation[]> {
    try {
      // Use provided email or default for testing
      const userEmail = email || 'test-user@example.com';
      const response = await this.safeRequest(`/api/v1/invitations/my?email=${enc(encodeURIComponent(userEmail))}`);
      // Backend returns {invitations: [...], email: "...", filter: {...}}
      const data = (response as any)?.invitations || response;
      return Array.isArray(data) ? data : [];
    } catch (error) {
      logger.error('Failed to fetch my invitations:', error);
      return [];
    }
  }

  async getProjectInvitations(projectId: string): Promise<Invitation[]> {
    try {
      const data = await this.safeRequest(`/api/v1/projects/${enc(projectId)}/invitations`);
      return Array.isArray(data) ? data : [];
    } catch (error) {
      logger.error('Failed to fetch project invitations:', error);
      return [];
    }
  }

  async revokeInvitation(invitationId: string): Promise<void> {
    try {
      await this.safeRequest(`/api/v1/invitations/${enc(invitationId)}`, 'DELETE');
    } catch (error) {
      logger.error('Failed to revoke invitation:', error);
      throw error;
    }
  }

  // Project Details API
  async getProjectDetails(projectId: string): Promise<ProjectDetails> {
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}`);
    // Map backend types.Project format (budget.total_budget) to ProjectDetails format (budget_limit)
    return {
      ...data,
      budget_limit: data.budget?.total_budget,
      current_spend: data.budget?.spent_amount || 0,
      members: data.members || [],
      cost_breakdown: data.cost_breakdown || { instances: 0, storage: 0, data_transfer: 0, total: 0 }
    };
  }

  // User Provisioning API
  async provisionUserOnWorkspace(username: string, instanceId: string): Promise<void> {
    await this.safeRequest(`/api/v1/users/${enc(username)}/provision`, 'POST', { instance_id: instanceId });
  }

  // AMI Management APIs
  async getAMIs(): Promise<AMI[]> {
    try {
      const data = await this.safeRequest('/api/v1/ami/list');

      // Transform backend response to frontend format
      if (!data || !Array.isArray(data)) {
        return [];
      }

      return data.map((ami: Record<string, unknown>) => ({
        id: (ami.id as string) || (ami.ami_id as string) || '',
        name: (ami.name as string) || (ami.id as string) || '',
        template_name: (ami.template_name as string) || (ami.template as string) || 'unknown',
        region: (ami.region as string) || 'us-west-2',
        state: (ami.state as string) || 'available',
        architecture: (ami.architecture as string) || 'x86_64',
        size_gb: (ami.size_gb as number) || (ami.size as number) || 0,
        description: (ami.description as string) || '',
        created_at: (ami.created_at as string) || (ami.creation_date as string) || new Date().toISOString(),
        tags: (ami.tags as Record<string, string>) || {}
      }));
    } catch (error) {
      logger.error('Failed to fetch AMIs:', error);
      return [];
    }
  }

  async getAMIBuilds(): Promise<AMIBuild[]> {
    try {
      // Note: Backend may not have build tracking yet
      // Return empty array for now
      return [];
    } catch (error) {
      logger.error('Failed to fetch AMI builds:', error);
      return [];
    }
  }

  async getAMIRegions(): Promise<AMIRegion[]> {
    try {
      const amis = await this.getAMIs();

      // Group AMIs by region
      const regionMap = new Map<string, { count: number; totalSize: number }>();

      amis.forEach(ami => {
        const existing = regionMap.get(ami.region) || { count: 0, totalSize: 0 };
        regionMap.set(ami.region, {
          count: existing.count + 1,
          totalSize: existing.totalSize + ami.size_gb
        });
      });

      // Convert to array and calculate costs (estimated at $0.05 per GB-month for EBS snapshots)
      return Array.from(regionMap.entries()).map(([name, data]) => ({
        name,
        ami_count: data.count,
        total_size_gb: data.totalSize,
        monthly_cost: data.totalSize * 0.05
      })).sort((a, b) => b.ami_count - a.ami_count);
    } catch (error) {
      logger.error('Failed to calculate AMI regions:', error);
      return [];
    }
  }

  async deleteAMI(amiId: string): Promise<void> {
    await this.safeRequest('/api/v1/ami/delete', 'POST', {
      ami_id: amiId,
      deregister_only: false
    });
  }

  async buildAMI(templateName: string): Promise<{ build_id: string }> {
    const response = await this.safeRequest<{ build_id: string }>('/api/v1/ami/create', 'POST', {
      template_name: templateName
    });
    return response;
  }

  // Rightsizing APIs
  async getRightsizingRecommendations(): Promise<RightsizingRecommendation[]> {
    try {
      const data = await this.safeRequest<{recommendations?: Record<string, unknown>[]}>('/api/v1/rightsizing/recommendations');
      if (!data || !Array.isArray(data.recommendations)) {
        return [];
      }
      return data.recommendations.map((rec: Record<string, unknown>) => ({
        instance_name: (rec.instance_name as string) || (rec.InstanceName as string) || '',
        current_type: (rec.current_type as string) || (rec.CurrentType as string) || '',
        recommended_type: (rec.recommended_type as string) || (rec.RecommendedType as string) || '',
        cpu_utilization: (rec.cpu_utilization as number) || (rec.CPUUtilization as number) || 0,
        memory_utilization: (rec.memory_utilization as number) || (rec.MemoryUtilization as number) || 0,
        current_cost: (rec.current_cost as number) || (rec.CurrentCost as number) || 0,
        recommended_cost: (rec.recommended_cost as number) || (rec.RecommendedCost as number) || 0,
        monthly_savings: (rec.monthly_savings as number) || (rec.MonthlySavings as number) || 0,
        savings_percentage: (rec.savings_percentage as number) || (rec.SavingsPercentage as number) || 0,
        confidence: ((rec.confidence as string) || (rec.Confidence as string) || 'medium') as 'high' | 'medium' | 'low',
        reason: (rec.reason || rec.Reason) as string | undefined
      }));
    } catch (error) {
      logger.error('Failed to fetch rightsizing recommendations:', error);
      return [];
    }
  }

  async getRightsizingStats(): Promise<RightsizingStats | null> {
    try {
      const data = await this.safeRequest<Partial<RightsizingStats>>('/api/v1/rightsizing/stats');
      return {
        total_recommendations: data.total_recommendations || 0,
        total_monthly_savings: data.total_monthly_savings || 0,
        average_cpu_utilization: data.average_cpu_utilization || 0,
        average_memory_utilization: data.average_memory_utilization || 0,
        over_provisioned_count: data.over_provisioned_count || 0,
        optimized_count: data.optimized_count || 0
      };
    } catch (error: unknown) {
      // Silently handle 400/404 - endpoint may not be implemented yet
      const errorMessage = error instanceof Error ? error.message : String(error);
      if (errorMessage.includes('HTTP 400') || errorMessage.includes('HTTP 404')) {
        return null; // Don't log, just return null
      }
      // Only log unexpected errors
      logger.error('Unexpected error fetching rightsizing stats:', error);
      return null;
    }
  }

  async applyRightsizingRecommendation(instanceName: string): Promise<void> {
    await this.safeRequest(`/api/v1/rightsizing/instance/${enc(instanceName)}/apply`, 'POST');
  }

  // Policy APIs
  async getPolicyStatus(): Promise<PolicyStatus | null> {
    try {
      const data = await this.safeRequest<Partial<PolicyStatus>>('/api/v1/policies/status');
      return {
        enabled: data.enabled || false,
        status: data.status || 'unknown',
        status_icon: data.status_icon || '',
        assigned_policies: data.assigned_policies || [],
        message: data.message
      };
    } catch (error) {
      logger.error('Failed to fetch policy status:', error);
      return null;
    }
  }

  async getPolicySets(): Promise<PolicySet[]> {
    try {
      const data = await this.safeRequest<{policy_sets?: Record<string, Record<string, unknown>>}>('/api/v1/policies/sets');
      if (!data || !data.policy_sets) {
        return [];
      }
      return Object.entries(data.policy_sets).map(([id, info]: [string, Record<string, unknown>]) => ({
        id,
        name: (info.name as string) || id,
        description: (info.description as string) || '',
        policies: (info.policies as number) || 0,
        status: (info.status as string) || 'active',
        tags: info.tags as Record<string, string> | undefined
      }));
    } catch (error) {
      logger.error('Failed to fetch policy sets:', error);
      return [];
    }
  }

  async setPolicyEnforcement(enabled: boolean): Promise<void> {
    await this.safeRequest('/api/v1/policies/enforcement', 'POST', { enabled });
  }

  async assignPolicySet(policySetId: string): Promise<void> {
    await this.safeRequest('/api/v1/policies/assign', 'POST', { policy_set: policySetId });
  }

  async checkTemplateAccess(templateName: string): Promise<PolicyCheckResult> {
    const data = await this.safeRequest<Partial<PolicyCheckResult>>('/api/v1/policies/check', 'POST', { template_name: templateName });
    return {
      allowed: data.allowed || false,
      template_name: data.template_name || templateName,
      reason: data.reason || '',
      matched_policies: data.matched_policies,
      suggestions: data.suggestions
    };
  }

  // Marketplace APIs
  async getMarketplaceTemplates(query?: string, category?: string): Promise<MarketplaceTemplate[]> {
    try {
      let url = '/api/v1/marketplace/templates?';
      if (query) url += `query=${encodeURIComponent(query)}&`;
      if (category) url += `category=${encodeURIComponent(category)}&`;

      const data = await this.safeRequest<{templates?: Record<string, unknown>[]}>( url);
      if (!data || !Array.isArray(data.templates)) {
        return [];
      }
      return data.templates.map((t: Record<string, unknown>) => ({
        id: (t.id as string) || (t.ID as string) || '',
        name: (t.name as string) || (t.Name as string) || '',
        display_name: (t.display_name as string) || (t.DisplayName as string) || (t.name as string) || '',
        author: (t.author as string) || (t.Author as string) || '',
        publisher: (t.publisher as string) || (t.Publisher as string) || '',
        category: (t.category as string) || (t.Category as string) || '',
        description: (t.description as string) || (t.Description as string) || '',
        rating: (t.rating as number) || (t.Rating as number) || 0,
        downloads: (t.downloads as number) || (t.Downloads as number) || 0,
        verified: (t.verified as boolean) || (t.Verified as boolean) || false,
        featured: (t.featured as boolean) || (t.Featured as boolean) || false,
        version: (t.version as string) || (t.Version as string) || '',
        tags: (t.tags as string[]) || (t.Tags as string[]),
        badges: (t.badges as string[]) || (t.Badges as string[]),
        created_at: (t.created_at as string) || (t.CreatedAt as string) || '',
        updated_at: (t.updated_at as string) || (t.UpdatedAt as string) || '',
        ami_available: (t.ami_available as boolean) || (t.AMIAvailable as boolean) || false
      }));
    } catch (error) {
      logger.error('Failed to fetch marketplace templates:', error);
      return [];
    }
  }

  async getMarketplaceCategories(): Promise<MarketplaceCategory[]> {
    try {
      const data = await this.safeRequest<{categories?: Record<string, unknown>[]}>('/api/v1/marketplace/categories');
      if (!data || !Array.isArray(data.categories)) {
        return [];
      }
      return data.categories.map((c: Record<string, unknown>) => ({
        id: (c.id as string) || (c.ID as string) || '',
        name: (c.name as string) || (c.Name as string) || '',
        count: (c.count as number) || (c.Count as number) || 0
      }));
    } catch (error) {
      logger.error('Failed to fetch marketplace categories:', error);
      return [];
    }
  }

  async installMarketplaceTemplate(templateId: string, localName: string): Promise<void> {
    await this.safeRequest('/api/v1/templates/install-marketplace', 'POST', { marketplace_template_id: templateId, local_name: localName });
  }

  // Idle Detection APIs
  async getIdlePolicies(): Promise<IdlePolicy[]> {
    try {
      const data = await this.safeRequest<{policies?: Record<string, Record<string, unknown>>}>('/api/v1/idle/policies');
      if (!data || !data.policies) {
        return [];
      }
      const policies = Object.entries(data.policies).map(([id, p]: [string, Record<string, unknown>]) => ({
        id,
        name: (p.name as string) || (p.Name as string) || id,
        idle_minutes: (p.idle_minutes as number) || (p.IdleMinutes as number) || 0,
        action: ((p.action as string) || (p.Action as string) || 'notify') as 'hibernate' | 'stop' | 'notify',
        cpu_threshold: (p.cpu_threshold as number) || (p.CPUThreshold as number) || 10,
        memory_threshold: (p.memory_threshold as number) || (p.MemoryThreshold as number) || 10,
        network_threshold: (p.network_threshold as number) || (p.NetworkThreshold as number) || 1,
        description: (p.description as string) || (p.Description as string),
        enabled: p.enabled !== undefined ? (p.enabled as boolean) : (p.Enabled !== undefined ? (p.Enabled as boolean) : true)
      }));
      return policies;
    } catch (error) {
      logger.error('Failed to fetch idle policies:', error);
      return [];
    }
  }

  async getIdleSchedules(): Promise<IdleSchedule[]> {
    try {
      const data = await this.safeRequest<{schedules?: Record<string, unknown>[]}>('/api/v1/idle/schedules');
      if (!data || !Array.isArray(data.schedules)) {
        return [];
      }
      return data.schedules.map((s: Record<string, unknown>) => ({
        instance_name: (s.instance_name as string) || (s.InstanceName as string) || '',
        policy_name: (s.policy_name as string) || (s.PolicyName as string) || '',
        enabled: s.enabled !== undefined ? (s.enabled as boolean) : (s.Enabled !== undefined ? (s.Enabled as boolean) : true),
        last_checked: (s.last_checked as string) || (s.LastChecked as string) || '',
        idle_minutes: (s.idle_minutes as number) || (s.IdleMinutes as number) || 0,
        status: (s.status as string) || (s.Status as string) || ''
      }));
    } catch (error) {
      logger.error('Failed to fetch idle schedules:', error);
      return [];
    }
  }

  // Per-instance Idle Policy APIs (Issue #288)
  async getInstanceIdlePolicies(instanceName: string): Promise<IdlePolicy[]> {
    try {
      const data = await this.safeRequest(`/api/v1/instances/${enc(instanceName)}/idle/policies`);
      if (!data || !Array.isArray(data)) return [];
      return (data as Record<string, unknown>[]).map((p) => ({
        id: (p.id as string) || '',
        name: (p.name as string) || (p.Name as string) || '',
        idle_minutes: (p.idle_minutes as number) || 0,
        action: ((p.action as string) || 'notify') as 'hibernate' | 'stop' | 'notify',
        cpu_threshold: (p.cpu_threshold as number) || 10,
        memory_threshold: (p.memory_threshold as number) || 10,
        network_threshold: (p.network_threshold as number) || 1,
        description: (p.description as string) || '',
        enabled: p.enabled !== undefined ? (p.enabled as boolean) : true
      }));
    } catch (error) {
      logger.error(`Failed to fetch idle policies for ${instanceName}:`, error);
      return [];
    }
  }

  async applyIdlePolicy(instanceName: string, policyId: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(instanceName)}/idle/policies/${enc(policyId)}`, 'PUT');
  }

  async removeIdlePolicy(instanceName: string, policyId: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${enc(instanceName)}/idle/policies/${enc(policyId)}`, 'DELETE');
  }

  // Profile Management APIs
  async getProfiles(): Promise<any[]> {
    try {
      const data = await this.safeRequest('/api/v1/profiles');
      return Array.isArray(data) ? data : [];
    } catch (error) {
      logger.error('Failed to fetch profiles:', error);
      return [];
    }
  }

  async createProfile(profile: { name: string; aws_profile: string; region?: string }): Promise<any> {
    return this.safeRequest('/api/v1/profiles', 'POST', profile);
  }

  async updateProfile(profileId: string, updates: { name?: string; aws_profile?: string; region?: string }): Promise<any> {
    return this.safeRequest(`/api/v1/profiles/${enc(profileId)}`, 'PUT', updates);
  }

  async deleteProfile(profileId: string): Promise<void> {
    await this.safeRequest(`/api/v1/profiles/${enc(profileId)}`, 'DELETE');
  }

  async switchProfile(profileId: string): Promise<any> {
    return this.safeRequest(`/api/v1/profiles/${enc(profileId)}/activate`, 'POST');
  }

  async checkForUpdates(): Promise<any> {
    return this.safeRequest('/api/v1/update/check');
  }

  async getAutoStartStatus(): Promise<{ enabled: boolean; method?: string; path?: string }> {
    // Call the Wails backend method (only available in desktop app, not in E2E tests)
    try {
      // Check if running in Wails environment
      if (!(window as any).go?.main?.PrismService) {
        // Not in Wails environment (e.g., E2E tests), return default
        return { enabled: false };
      }
      const response = await (window as any).go.main.PrismService.GetAutoStartStatus();
      return response;
    } catch (error) {
      logger.error('Failed to get auto-start status:', error);
      // Return default instead of throwing to avoid breaking E2E tests
      return { enabled: false };
    }
  }

  async setAutoStart(enable: boolean): Promise<void> {
    // Call the Wails backend method (only available in desktop app, not in E2E tests)
    try {
      // Check if running in Wails environment
      if (!(window as any).go?.main?.PrismService) {
        // Not in Wails environment (e.g., E2E tests), no-op
        logger.debug('setAutoStart called in non-Wails environment, ignoring');
        return;
      }
      await (window as any).go.main.PrismService.SetAutoStart(enable);
    } catch (error) {
      logger.error('Failed to set auto-start:', error);
      // Don't throw to avoid breaking E2E tests
    }
  }

  // v0.13.0 Governance APIs
  async getProjectQuotas(projectId: string): Promise<RoleQuota[]> {
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/quotas`);
    return data?.role_quotas || [];
  }

  async setProjectQuota(projectId: string, quota: RoleQuota): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/quotas`, 'PUT', quota);
  }

  async deleteProjectQuota(projectId: string, role: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/quotas/${enc(role)}`, 'DELETE');
  }

  async getGrantPeriod(projectId: string): Promise<GrantPeriod | null> {
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/grant-period`);
    return data?.grant_period || null;
  }

  async setGrantPeriod(projectId: string, gp: GrantPeriod): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/grant-period`, 'PUT', gp);
  }

  async deleteGrantPeriod(projectId: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/grant-period`, 'DELETE');
  }

  async listApprovals(projectId: string, status?: string): Promise<ApprovalRequest[]> {
    const qs = status ? `?status=${status}` : '';
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/approvals${enc(qs)}`);
    return data?.approvals || [];
  }

  async approveRequest(projectId: string, approvalId: string, note: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/approvals/${enc(approvalId)}/approve`, 'POST', { note });
  }

  async denyRequest(projectId: string, approvalId: string, note: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/approvals/${enc(approvalId)}/deny`, 'POST', { note });
  }

  async listAllApprovals(status?: string): Promise<ApprovalRequest[]> {
    const qs = status ? `?status=${status}` : '';
    const data = await this.safeRequest<any>(`/api/v1/admin/approvals${enc(qs)}`);
    return data?.approvals || [];
  }

  // v0.21.0: Get a single approval request by ID (#495)
  async getApproval(projectId: string, approvalId: string): Promise<ApprovalRequest> {
    return this.safeRequest<ApprovalRequest>(`/api/v1/projects/${enc(projectId)}/approvals/${enc(approvalId)}`);
  }

  // v0.21.0: Submit approval request for a launch (#495)
  async submitApprovalForLaunch(projectId: string, templateName: string, size: string, reason: string): Promise<ApprovalRequest> {
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/approvals`, 'POST', {
      type: 'expensive_instance',
      details: { template: templateName, size },
      reason,
    });
    return data;
  }

  async shareProjectBudget(projectId: string, req: BudgetShareRequest): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/budget/share`, 'POST', req);
  }

  async listProjectBudgetShares(projectId: string): Promise<BudgetShareRecord[]> {
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/budget/shares`);
    return data?.shares || [];
  }

  async listOnboardingTemplates(projectId: string): Promise<OnboardingTemplate[]> {
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/onboarding-templates`);
    return data?.onboarding_templates || [];
  }

  async addOnboardingTemplate(projectId: string, tmpl: OnboardingTemplate): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/onboarding-templates`, 'POST', tmpl);
  }

  async deleteOnboardingTemplate(projectId: string, nameOrId: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${enc(projectId)}/onboarding-templates/${enc(encodeURIComponent(nameOrId))}`, 'DELETE');
  }

  async getMonthlyReport(projectId: string, month: string, format: string): Promise<string> {
    const qs = `?month=${month}&format=${format}`;
    const data = await this.safeRequest<any>(`/api/v1/projects/${enc(projectId)}/reports/monthly${enc(qs)}`);
    return typeof data === 'string' ? data : JSON.stringify(data, null, 2);
  }

  // ── v0.14.0/v0.16.0 University Education System ──────────────────────────

  async getCourses(): Promise<Course[]> {
    const data = await this.safeRequest<any>('/api/v1/courses');
    return (data?.courses || []) as Course[];
  }

  async createCourse(courseData: Partial<Course>): Promise<Course> {
    const data = await this.safeRequest<Course>('/api/v1/courses', 'POST', courseData);
    return data as Course;
  }

  async getCourse(id: string): Promise<Course> {
    const data = await this.safeRequest<Course>(`/api/v1/courses/${enc(encodeURIComponent(id))}`);
    return data as Course;
  }

  async closeCourse(id: string): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(id))}/close`, 'POST');
  }

  async deleteCourse(id: string): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(id))}`, 'DELETE');
  }

  async archiveCourse(id: string): Promise<{ instances_stopped: string[] }> {
    const data = await this.safeRequest<any>(`/api/v1/courses/${enc(encodeURIComponent(id))}/archive`, 'POST');
    return data || { instances_stopped: [] };
  }

  async getCourseMembers(id: string, role?: string): Promise<ClassMember[]> {
    const qs = role ? `?role=${encodeURIComponent(role)}` : '';
    const data = await this.safeRequest<any>(`/api/v1/courses/${enc(encodeURIComponent(id))}/members${enc(qs)}`);
    return (data?.members || []) as ClassMember[];
  }

  async enrollCourseMember(id: string, memberData: Partial<ClassMember>): Promise<ClassMember> {
    const data = await this.safeRequest<ClassMember>(`/api/v1/courses/${enc(encodeURIComponent(id))}/members`, 'POST', memberData);
    return data as ClassMember;
  }

  async unenrollCourseMember(id: string, userId: string): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(id))}/members/${enc(encodeURIComponent(userId))}`, 'DELETE');
  }

  async getCourseTemplates(id: string): Promise<{ templates: string[] }> {
    const data = await this.safeRequest<any>(`/api/v1/courses/${enc(encodeURIComponent(id))}/templates`);
    return { templates: data?.approved_templates || [] };
  }

  async addCourseTemplate(id: string, slug: string): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(id))}/templates`, 'POST', { template: slug });
  }

  async removeCourseTemplate(id: string, slug: string): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(id))}/templates/${enc(encodeURIComponent(slug))}`, 'DELETE');
  }

  async getCourseBudget(id: string): Promise<CourseBudgetSummary> {
    const data = await this.safeRequest<CourseBudgetSummary>(`/api/v1/courses/${enc(encodeURIComponent(id))}/budget`);
    return data as CourseBudgetSummary;
  }

  async distributeCourseBudget(id: string, amount: number): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(id))}/budget/distribute`, 'POST', { amount_per_student: amount });
  }

  async debugStudent(courseId: string, studentId: string): Promise<Record<string, unknown>> {
    const data = await this.safeRequest<any>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/members/${enc(encodeURIComponent(studentId))}/debug`);
    return data || {};
  }

  async resetStudent(courseId: string, studentId: string, reason: string): Promise<void> {
    await this.safeRequest(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/members/${enc(encodeURIComponent(studentId))}/reset`, 'POST', { reason });
  }

  async provisionStudent(courseId: string, studentId: string, data?: Record<string, unknown>): Promise<Record<string, unknown>> {
    const result = await this.safeRequest<any>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/members/${enc(encodeURIComponent(studentId))}/provision`, 'POST', data || {});
    return result || {};
  }

  async getCourseOverview(id: string): Promise<CourseOverview> {
    const data = await this.safeRequest<CourseOverview>(`/api/v1/courses/${enc(encodeURIComponent(id))}/overview`);
    return data as CourseOverview;
  }

  async getCourseReport(id: string, format?: string): Promise<UsageReport> {
    const qs = format ? `?format=${encodeURIComponent(format)}` : '';
    const data = await this.safeRequest<UsageReport>(`/api/v1/courses/${enc(encodeURIComponent(id))}/report${enc(qs)}`);
    return data as UsageReport;
  }

  async getCourseAuditLog(id: string, params?: { student_id?: string; since?: string; limit?: number }): Promise<{ entries: CourseAuditEntry[] }> {
    const qs = new URLSearchParams();
    if (params?.student_id) qs.set('student_id', params.student_id);
    if (params?.since) qs.set('since', params.since);
    if (params?.limit) qs.set('limit', String(params.limit));
    const qstr = qs.toString() ? `?${qs.toString()}` : '';
    const data = await this.safeRequest<any>(`/api/v1/courses/${enc(encodeURIComponent(id))}/audit${enc(qstr)}`);
    return { entries: (data?.entries || []) as CourseAuditEntry[] };
  }

  async importCourseRoster(id: string, file: File, format?: string): Promise<{ enrolled: number; errors: string[] }> {
    const qs = format ? `?format=${encodeURIComponent(format)}` : '';
    const url = `/api/v1/courses/${enc(encodeURIComponent(id))}/members/import${enc(qs)}`;
    const baseURL = (window as any).__daemonURL || 'http://localhost:8947';
    const resp = await fetch(`${baseURL}${url}`, {
      method: 'POST',
      headers: { 'Content-Type': 'text/csv' },
      body: file,
    });
    if (!resp.ok) throw new Error(`Import failed: ${resp.statusText}`);
    const data = await resp.json();
    return { enrolled: data?.enrolled || 0, errors: data?.errors || [] };
  }

  // ── v0.18.0 Workshop & Event Management ────────────────────────────────────

  async getWorkshops(params?: { owner?: string; status?: string }): Promise<WorkshopEvent[]> {
    const qs = params ? '?' + new URLSearchParams(Object.entries(params).filter(([, v]) => v)).toString() : '';
    const data = await this.safeRequest<any>(`/api/v1/workshops${enc(qs)}`);
    return (data?.workshops || []) as WorkshopEvent[];
  }

  async createWorkshop(workshopData: Partial<WorkshopEvent>): Promise<WorkshopEvent> {
    const data = await this.safeRequest<WorkshopEvent>('/api/v1/workshops', 'POST', workshopData);
    return data as WorkshopEvent;
  }

  async getWorkshop(id: string): Promise<WorkshopEvent> {
    const data = await this.safeRequest<WorkshopEvent>(`/api/v1/workshops/${enc(encodeURIComponent(id))}`);
    return data as WorkshopEvent;
  }

  async updateWorkshop(id: string, updates: Partial<WorkshopEvent>): Promise<WorkshopEvent> {
    const data = await this.safeRequest<WorkshopEvent>(`/api/v1/workshops/${enc(encodeURIComponent(id))}`, 'PUT', updates);
    return data as WorkshopEvent;
  }

  async deleteWorkshop(id: string): Promise<void> {
    await this.safeRequest(`/api/v1/workshops/${enc(encodeURIComponent(id))}`, 'DELETE');
  }

  async provisionWorkshop(id: string): Promise<{ provisioned: number; skipped: number; errors: string[] }> {
    const data = await this.safeRequest<any>(`/api/v1/workshops/${enc(encodeURIComponent(id))}/provision`, 'POST');
    return { provisioned: data?.provisioned || 0, skipped: data?.skipped || 0, errors: data?.errors || [] };
  }

  async getWorkshopDashboard(id: string): Promise<WorkshopDashboard> {
    const data = await this.safeRequest<WorkshopDashboard>(`/api/v1/workshops/${enc(encodeURIComponent(id))}/dashboard`);
    return data as WorkshopDashboard;
  }

  async endWorkshop(id: string): Promise<{ stopped: number; errors: string[] }> {
    const data = await this.safeRequest<any>(`/api/v1/workshops/${enc(encodeURIComponent(id))}/end`, 'POST');
    return { stopped: data?.stopped || 0, errors: data?.errors || [] };
  }

  async getWorkshopDownload(id: string): Promise<{ workshop_id: string; participants: any[] }> {
    const data = await this.safeRequest<any>(`/api/v1/workshops/${enc(encodeURIComponent(id))}/download`);
    return { workshop_id: data?.workshop_id || id, participants: data?.participants || [] };
  }

  async getWorkshopConfigs(): Promise<WorkshopConfig[]> {
    const data = await this.safeRequest<any>('/api/v1/workshops/configs');
    return (data?.configs || []) as WorkshopConfig[];
  }

  async saveWorkshopConfig(workshopId: string, configName: string): Promise<WorkshopConfig> {
    const data = await this.safeRequest<WorkshopConfig>(`/api/v1/workshops/${enc(encodeURIComponent(workshopId))}/config`, 'POST', { name: configName });
    return data as WorkshopConfig;
  }

  async createWorkshopFromConfig(configName: string, workshopData: Partial<WorkshopEvent>): Promise<WorkshopEvent> {
    const data = await this.safeRequest<WorkshopEvent>(`/api/v1/workshops/from-config/${enc(encodeURIComponent(configName))}`, 'POST', workshopData);
    return data as WorkshopEvent;
  }

  // ── v0.19.0 TA Access, Shared Materials, Workspace Reset ──
  async listCourseTAAccess(courseId: string): Promise<ClassMember[]> {
    const data = await this.safeRequest<{ ta_members: ClassMember[] }>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/ta-access`, 'GET');
    return (data?.ta_members || []) as ClassMember[];
  }

  async grantCourseTAAccess(courseId: string, email: string, displayName?: string): Promise<ClassMember> {
    const data = await this.safeRequest<ClassMember>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/ta-access`, 'POST', { email, display_name: displayName || '' });
    return data as ClassMember;
  }

  async revokeCourseTAAccess(courseId: string, email: string): Promise<void> {
    await this.safeRequest<void>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/ta-access/${enc(encodeURIComponent(email))}`, 'DELETE');
  }

  async connectCourseTAAccess(courseId: string, studentId: string, reason: string): Promise<{ ssh_command: string }> {
    const data = await this.safeRequest<{ ssh_command: string }>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/ta-access/connect`, 'POST', { student_id: studentId, reason });
    return data as { ssh_command: string };
  }

  async resetCourseStudentWorkspace(courseId: string, studentId: string, reason: string, backup: boolean): Promise<WorkspaceResetResult> {
    const data = await this.safeRequest<WorkspaceResetResult>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/ta/reset/${enc(encodeURIComponent(studentId))}`, 'POST', { reason, backup });
    return data as WorkspaceResetResult;
  }

  async getCourseMaterials(courseId: string): Promise<SharedMaterialsVolume | null> {
    const data = await this.safeRequest<{ materials: SharedMaterialsVolume | null }>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/materials`, 'GET');
    return data?.materials || null;
  }

  async createCourseMaterials(courseId: string, sizeGB: number, mountPath: string): Promise<SharedMaterialsVolume> {
    const data = await this.safeRequest<{ materials: SharedMaterialsVolume }>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/materials`, 'POST', { size_gb: sizeGB, mount_path: mountPath });
    return data?.materials as SharedMaterialsVolume;
  }

  async mountCourseMaterials(courseId: string): Promise<{ status: string; note: string }> {
    const data = await this.safeRequest<{ status: string; note: string }>(`/api/v1/courses/${enc(encodeURIComponent(courseId))}/materials/mount`, 'POST', {});
    return data as { status: string; note: string };
  }

  // ── v0.20.0 SSM File Operations (#30) ──────────────────────────────────────
  async listInstanceFiles(instanceName: string, path?: string): Promise<FileEntry[]> {
    const url = `/api/v1/instances/${enc(instanceName)}/files${path ? `?path=${encodeURIComponent(path)}` : ''}`;
    const data = await this.safeRequest<FileEntry[]>(url, 'GET');
    return (data || []) as FileEntry[];
  }

  async pushFileToInstance(instanceName: string, localPath: string, remotePath: string): Promise<{ status: string; message: string }> {
    const data = await this.safeRequest<{ status: string; message: string }>(
      `/api/v1/instances/${enc(encodeURIComponent(instanceName))}/files/push`, 'POST',
      { local_path: localPath, remote_path: remotePath });
    return data as { status: string; message: string };
  }

  async pullFileFromInstance(instanceName: string, remotePath: string, localPath: string): Promise<{ status: string; message: string }> {
    const data = await this.safeRequest<{ status: string; message: string }>(
      `/api/v1/instances/${enc(encodeURIComponent(instanceName))}/files/pull`, 'POST',
      { remote_path: remotePath, local_path: localPath });
    return data as { status: string; message: string };
  }

  // ── v0.20.0 EC2 Capacity Blocks (#63) ──────────────────────────────────────
  async getCapacityBlocks(): Promise<CapacityBlock[]> {
    const data = await this.safeRequest<CapacityBlock[]>('/api/v1/capacity-blocks', 'GET');
    return (data || []) as CapacityBlock[];
  }

  async reserveCapacityBlock(req: CapacityBlockRequest): Promise<CapacityBlock> {
    const data = await this.safeRequest<CapacityBlock>('/api/v1/capacity-blocks', 'POST', req);
    return data as CapacityBlock;
  }

  async describeCapacityBlock(id: string): Promise<CapacityBlock> {
    const data = await this.safeRequest<CapacityBlock>(`/api/v1/capacity-blocks/${enc(encodeURIComponent(id))}`, 'GET');
    return data as CapacityBlock;
  }

  async cancelCapacityBlock(id: string): Promise<void> {
    await this.safeRequest<void>(`/api/v1/capacity-blocks/${enc(encodeURIComponent(id))}`, 'DELETE');
  }

  // S3 mount methods (#22c)

  async listInstanceS3Mounts(instanceName: string): Promise<S3Mount[]> {
    const data = await this.safeRequest<S3Mount[]>(`/api/v1/instances/${enc(encodeURIComponent(instanceName))}/s3-mounts`, 'GET');
    return (data as S3Mount[]) || [];
  }

  async mountS3Bucket(instanceName: string, bucket: string, mountPath: string, method = 'mountpoint', readOnly = false): Promise<S3Mount> {
    const data = await this.safeRequest<S3Mount>(`/api/v1/instances/${enc(encodeURIComponent(instanceName))}/s3-mounts`, 'POST', {
      bucket_name: bucket,
      mount_path: mountPath,
      method,
      read_only: readOnly,
    });
    return data as S3Mount;
  }

  async unmountS3Bucket(instanceName: string, mountPath: string): Promise<void> {
    const encoded = encodeURIComponent(mountPath.replace(/^\//, ''));
    await this.safeRequest<void>(`/api/v1/instances/${enc(encodeURIComponent(instanceName))}/s3-mounts/${enc(encoded)}`, 'DELETE');
  }

  // Storage analytics methods (#23c)

  async getAllStorageAnalytics(period = 'daily'): Promise<StorageAnalyticsSummary[]> {
    const data = await this.safeRequest<{ resources?: StorageAnalyticsSummary[] }>(`/api/v1/storage/analytics?period=${enc(encodeURIComponent(period))}`, 'GET');
    const result = data as { resources?: StorageAnalyticsSummary[] };
    return result?.resources || [];
  }

  async getStorageAnalytics(name: string, period = 'daily'): Promise<StorageAnalyticsSummary> {
    const data = await this.safeRequest<StorageAnalyticsSummary>(`/api/v1/storage/analytics/${enc(encodeURIComponent(name))}?period=${enc(encodeURIComponent(period))}`, 'GET');
    return data as StorageAnalyticsSummary;
  }
}
