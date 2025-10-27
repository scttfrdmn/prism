// Prism GUI - Bulletproof AWS Integration
// Complete error handling, real API integration, professional UX

import React, { useState, useEffect } from 'react';
import '@cloudscape-design/global-styles/index.css';
import './index.css';
import Terminal from './Terminal';
import WebView from './WebView';

import {
  AppLayout,
  SideNavigation,
  Container,
  Header,
  SpaceBetween,
  Button,
  Cards,
  StatusIndicator,
  Badge,
  Table,
  Modal,
  Form,
  FormField,
  Input,
  Select,
  Alert,
  Flashbar,
  Spinner,
  Box,
  ColumnLayout,
  Link,
  ButtonDropdown,
  Tabs,
  PropertyFilter,
  Wizard,
  ProgressBar,
  TextContent
} from '@cloudscape-design/components';

// Type definitions
interface Project {
  id: string;
  name: string;
  description: string;
  budget_limit: number;
  current_spend: number;
  owner_id: string;
  owner_email: string;
  created_at: string;
  updated_at: string;
  status: string;
  member_count?: number;
}

interface User {
  username: string;
  uid: number;
  full_name: string;
  email: string;
  ssh_keys: number;
  created_at: string;
  provisioned_instances?: string[];
  status?: string;
}

interface Template {
  Name: string;  // API returns capital N
  Slug: string;  // API returns capital S
  Description?: string;  // API returns capital D
  name?: string;  // Keep lowercase for backward compatibility
  slug?: string;  // Keep lowercase for backward compatibility
  description?: string;  // Keep lowercase for backward compatibility
  category?: string;
  complexity?: string;
  package_manager?: string;
  features?: string[];
  // Additional fields that might come from API
  [key: string]: any;
}

interface Instance {
  id: string;
  name: string;
  template: string;
  state: string;
  public_ip?: string;
  instance_type?: string;
  launch_time?: string;
  region?: string;
}

// Unified StorageVolume interface matching backend API
interface StorageVolume {
  name: string;
  type: 'workspace' | 'shared' | 'cloud';
  aws_service: 'ebs' | 'efs' | 's3';
  region: string;
  state: string;
  creation_time: string;

  // Size fields (varies by type)
  size_gb?: number;      // EBS
  size_bytes?: number;   // EFS

  // EBS-specific fields
  volume_id?: string;
  volume_type?: string;
  iops?: number;
  throughput?: number;
  attached_to?: string;

  // EFS-specific fields
  filesystem_id?: string;
  mount_targets?: string[];
  performance_mode?: string;
  throughput_mode?: string;

  // Cost
  estimated_cost_gb: number;
}

// Legacy interfaces for backward compatibility
interface EFSVolume {
  name: string;
  filesystem_id: string;
  region: string;
  creation_time: string;
  state: string;
  performance_mode: string;
  throughput_mode: string;
  estimated_cost_gb: number;
  size_bytes: number;
}

interface EBSVolume {
  name: string;
  volume_id: string;
  region: string;
  creation_time: string;
  state: string;
  volume_type: string;
  size_gb: number;
  estimated_cost_gb: number;
  attached_to?: string;
}

interface Project {
  id: string;
  name: string;
  description?: string;
  owner: string;
  status: string;
  member_count: number;
  active_instances: number;
  total_cost: number;
  budget_status?: {
    total_budget: number;
    spent_amount: number;
    spent_percentage: number;
    alert_count: number;
  };
  created_at: string;
  last_activity: string;
}

interface BudgetData {
  project_id: string;
  project_name: string;
  total_budget: number;
  spent_amount: number;
  spent_percentage: number;
  remaining: number;
  alert_count: number;
  status: 'ok' | 'warning' | 'critical';
  projected_monthly_spend?: number;
  days_until_exhausted?: number;
  active_alerts?: Array<{
    threshold: number;
    action: string;
    triggered_at: string;
  }>;
}

interface CostBreakdown {
  ec2_compute: number;
  ebs_storage: number;
  efs_storage: number;
  data_transfer: number;
  other: number;
  total: number;
}

interface AMI {
  id: string;
  name: string;
  template_name: string;
  region: string;
  state: string;
  architecture: string;
  size_gb: number;
  description?: string;
  created_at: string;
  tags?: Record<string, string>;
}

interface AMIBuild {
  id: string;
  template_name: string;
  status: string;
  progress: number;
  current_step?: string;
  error?: string;
  started_at: string;
  completed_at?: string;
}

interface AMIRegion {
  name: string;
  ami_count: number;
  total_size_gb: number;
  monthly_cost: number;
}

interface RightsizingRecommendation {
  instance_name: string;
  current_type: string;
  recommended_type: string;
  cpu_utilization: number;
  memory_utilization: number;
  current_cost: number;
  recommended_cost: number;
  monthly_savings: number;
  savings_percentage: number;
  confidence: 'high' | 'medium' | 'low';
  reason?: string;
}

interface RightsizingStats {
  total_recommendations: number;
  total_monthly_savings: number;
  average_cpu_utilization: number;
  average_memory_utilization: number;
  over_provisioned_count: number;
  optimized_count: number;
}

interface PolicyStatus {
  enabled: boolean;
  status: string;
  status_icon: string;
  assigned_policies: string[];
  message?: string;
}

interface PolicySet {
  id: string;
  name: string;
  description: string;
  policies: number;
  status: string;
  tags?: Record<string, string>;
}

interface PolicyCheckResult {
  allowed: boolean;
  template_name: string;
  reason: string;
  matched_policies?: string[];
  suggestions?: string[];
}

interface MarketplaceTemplate {
  id: string;
  name: string;
  display_name: string;
  author: string;
  publisher: string;
  category: string;
  description: string;
  rating: number;
  downloads: number;
  verified: boolean;
  featured: boolean;
  version: string;
  tags?: string[];
  badges?: string[];
  created_at: string;
  updated_at: string;
  ami_available?: boolean;
}

interface MarketplaceCategory {
  id: string;
  name: string;
  count: number;
}

interface IdlePolicy {
  id: string;
  name: string;
  idle_minutes: number;
  action: 'hibernate' | 'stop' | 'notify';
  cpu_threshold: number;
  memory_threshold: number;
  network_threshold: number;
  description?: string;
  enabled: boolean;
}

interface IdleSchedule {
  instance_name: string;
  policy_name: string;
  enabled: boolean;
  last_checked: string;
  idle_minutes: number;
  status: string;
}

interface AppState {
  activeView: 'dashboard' | 'templates' | 'instances' | 'storage' | 'projects' | 'project-detail' | 'users' | 'ami' | 'rightsizing' | 'policy' | 'marketplace' | 'idle' | 'logs' | 'settings' | 'terminal' | 'webview';
  templates: Record<string, Template>;
  instances: Instance[];
  efsVolumes: EFSVolume[];
  ebsVolumes: EBSVolume[];
  projects: Project[];
  users: User[];
  budgets: BudgetData[];
  amis: AMI[];
  amiBuilds: AMIBuild[];
  amiRegions: AMIRegion[];
  rightsizingRecommendations: RightsizingRecommendation[];
  rightsizingStats: RightsizingStats | null;
  policyStatus: PolicyStatus | null;
  policySets: PolicySet[];
  marketplaceTemplates: MarketplaceTemplate[];
  marketplaceCategories: MarketplaceCategory[];
  idlePolicies: IdlePolicy[];
  idleSchedules: IdleSchedule[];
  selectedTemplate: Template | null;
  selectedProject: Project | null;
  selectedTerminalInstance: string;
  loading: boolean;
  notifications: any[];
  connected: boolean;
  error: string | null;
}

// Safe API Service with comprehensive error handling
class SafePrismAPI {
  private baseURL = 'http://localhost:8947';
  private apiKey = '';

  constructor() {
    // Load API key on initialization
    this.loadAPIKey();
  }

  private async loadAPIKey() {
    try {
      const response = await fetch('http://localhost:8948/api-key');
      const data = await response.json();
      this.apiKey = data.api_key;
      console.log('✅ API key loaded successfully');
    } catch (error) {
      console.error('❌ Failed to load API key:', error);
    }
  }

  private async safeRequest(endpoint: string, method = 'GET', body?: any): Promise<any> {
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
        throw new Error(`HTTP ${response.status}: ${response.statusText}`);
      }

      const data = await response.json();
      return data;
    } catch (error) {
      console.error(`API request failed for ${endpoint}:`, error);
      throw error;
    }
  }

  async getTemplates(): Promise<Record<string, Template>> {
    try {
      const data = await this.safeRequest('/api/v1/templates');
      return data || {};
    } catch (error) {
      console.error('Failed to fetch templates:', error);
      return {};
    }
  }

  async getInstances(): Promise<Instance[]> {
    try {
      const data = await this.safeRequest('/api/v1/instances');
      return Array.isArray(data?.instances) ? data.instances : [];
    } catch (error) {
      console.error('Failed to fetch instances:', error);
      return [];
    }
  }

  async launchInstance(templateSlug: string, name: string, size: string = 'M'): Promise<any> {
    return this.safeRequest('/api/v1/instances', 'POST', {
      template: templateSlug,
      name,
      size,
    });
  }

  // Comprehensive Instance Management APIs - Using Fixed Backend Endpoints
  async startInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${identifier}/start`, 'POST');
  }

  async stopInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${identifier}/stop`, 'POST');
  }

  async hibernateInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${identifier}/hibernate`, 'POST');
  }

  async resumeInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${identifier}/resume`, 'POST');
  }

  async getConnectionInfo(identifier: string): Promise<string> {
    const data = await this.safeRequest(`/api/v1/instances/${identifier}/connect`);
    return data.connection_info || '';
  }

  async getHibernationStatus(identifier: string): Promise<any> {
    return this.safeRequest(`/api/v1/instances/${identifier}/hibernation-status`);
  }

  async deleteInstance(identifier: string): Promise<void> {
    await this.safeRequest(`/api/v1/instances/${identifier}`, 'DELETE');
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
      console.error('Failed to fetch shared storage volumes:', error);
      return [];
    }
  }

  async createEFSVolume(name: string, performanceMode: string = 'generalPurpose', throughputMode: string = 'bursting'): Promise<any> {
    return this.safeRequest('/api/v1/volumes', 'POST', {
      name,
      performance_mode: performanceMode,
      throughput_mode: throughputMode,
    });
  }

  async deleteEFSVolume(name: string): Promise<void> {
    await this.safeRequest(`/api/v1/volumes/${name}`, 'DELETE');
  }

  async mountEFSVolume(volumeName: string, instance: string, mountPoint?: string): Promise<void> {
    const body: any = { instance };
    if (mountPoint) body.mount_point = mountPoint;
    await this.safeRequest(`/api/v1/volumes/${volumeName}/mount`, 'POST', body);
  }

  async unmountEFSVolume(volumeName: string, instance: string): Promise<void> {
    await this.safeRequest(`/api/v1/volumes/${volumeName}/unmount`, 'POST', { instance });
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
      console.error('Failed to fetch workspace storage volumes:', error);
      return [];
    }
  }

  async createEBSVolume(name: string, size: string = 'M', volumeType: string = 'gp3'): Promise<any> {
    return this.safeRequest('/api/v1/storage', 'POST', {
      name,
      size,
      volume_type: volumeType,
    });
  }

  async deleteEBSVolume(name: string): Promise<void> {
    await this.safeRequest(`/api/v1/storage/${name}`, 'DELETE');
  }

  async attachEBSVolume(storageName: string, instance: string): Promise<void> {
    await this.safeRequest(`/api/v1/storage/${storageName}/attach`, 'POST', { instance });
  }

  async detachEBSVolume(storageName: string): Promise<void> {
    await this.safeRequest(`/api/v1/storage/${storageName}/detach`, 'POST');
  }

  // Comprehensive Project Management APIs

  // Project Operations
  async getProjects(): Promise<any[]> {
    try {
      const data = await this.safeRequest('/api/v1/projects');
      return Array.isArray(data?.projects) ? data.projects : [];
    } catch (error) {
      console.error('Failed to fetch projects:', error);
      return [];
    }
  }

  async createProject(projectData: any): Promise<any> {
    return this.safeRequest('/api/v1/projects', 'POST', projectData);
  }

  async getProject(projectId: string): Promise<any> {
    return this.safeRequest(`/api/v1/projects/${projectId}`);
  }

  async updateProject(projectId: string, projectData: any): Promise<any> {
    return this.safeRequest(`/api/v1/projects/${projectId}`, 'PUT', projectData);
  }

  async deleteProject(projectId: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${projectId}`, 'DELETE');
  }

  // Project Members
  async getProjectMembers(projectId: string): Promise<any[]> {
    try {
      const data = await this.safeRequest(`/api/v1/projects/${projectId}/members`);
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to fetch project members:', error);
      return [];
    }
  }

  async addProjectMember(projectId: string, memberData: any): Promise<any> {
    return this.safeRequest(`/api/v1/projects/${projectId}/members`, 'POST', memberData);
  }

  async updateProjectMember(projectId: string, userId: string, memberData: any): Promise<any> {
    return this.safeRequest(`/api/v1/projects/${projectId}/members/${userId}`, 'PUT', memberData);
  }

  async removeProjectMember(projectId: string, userId: string): Promise<void> {
    await this.safeRequest(`/api/v1/projects/${projectId}/members/${userId}`, 'DELETE');
  }

  // Budget Management
  async getProjectBudget(projectId: string): Promise<any> {
    return this.safeRequest(`/api/v1/projects/${projectId}/budget`);
  }

  // Cost Analysis
  async getProjectCosts(projectId: string, startDate?: string, endDate?: string): Promise<any> {
    const params = new URLSearchParams();
    if (startDate) params.append('start_date', startDate);
    if (endDate) params.append('end_date', endDate);
    const query = params.toString();
    return this.safeRequest(`/api/v1/projects/${projectId}/costs${query ? '?' + query : ''}`);
  }

  // Resource Usage
  async getProjectUsage(projectId: string, period?: string): Promise<any> {
    const query = period ? `?period=${period}` : '';
    return this.safeRequest(`/api/v1/projects/${projectId}/usage${query}`);
  }

  // User Operations
  async getUsers(): Promise<any[]> {
    try {
      const data = await this.safeRequest('/api/v1/users');
      return Array.isArray(data?.users) ? data.users : [];
    } catch (error) {
      console.error('Failed to fetch users:', error);
      return [];
    }
  }

  async createUser(userData: any): Promise<any> {
    return this.safeRequest('/api/v1/users', 'POST', userData);
  }

  async deleteUser(username: string): Promise<void> {
    await this.safeRequest(`/api/v1/users/${username}`, 'DELETE');
  }

  async getUserStatus(username: string): Promise<any> {
    return this.safeRequest(`/api/v1/users/${username}/status`);
  }

  async provisionUser(username: string, instanceName: string): Promise<any> {
    return this.safeRequest(`/api/v1/users/${username}/provision`, 'POST', { instance: instanceName });
  }

  async generateSSHKey(username: string): Promise<any> {
    return this.safeRequest(`/api/v1/users/${username}/ssh-keys`, 'POST');
  }

  // Budget Management APIs
  async getBudgets(): Promise<BudgetData[]> {
    try {
      const projects = await this.getProjects();
      const budgets: BudgetData[] = [];

      // Fetch budget status for each project
      for (const project of projects) {
        try {
          const budgetStatus = await this.safeRequest(`/api/v1/projects/${project.id}/budget`);

          if (budgetStatus && budgetStatus.total_budget > 0) {
            const remaining = budgetStatus.total_budget - budgetStatus.spent_amount;
            const spentPercent = budgetStatus.spent_percentage * 100;

            let status: 'ok' | 'warning' | 'critical' = 'ok';
            if (spentPercent >= 95) {
              status = 'critical';
            } else if (spentPercent >= 80) {
              status = 'warning';
            }

            budgets.push({
              project_id: project.id,
              project_name: project.name,
              total_budget: budgetStatus.total_budget,
              spent_amount: budgetStatus.spent_amount,
              spent_percentage: budgetStatus.spent_percentage,
              remaining: remaining,
              alert_count: budgetStatus.alert_count || 0,
              status: status,
              projected_monthly_spend: budgetStatus.projected_monthly_spend,
              days_until_exhausted: budgetStatus.days_until_exhausted,
              active_alerts: budgetStatus.active_alerts
            });
          }
        } catch (error) {
          console.error(`Failed to fetch budget for project ${project.id}:`, error);
        }
      }

      return budgets;
    } catch (error) {
      console.error('Failed to fetch budgets:', error);
      return [];
    }
  }

  async getCostBreakdown(projectId: string, startDate?: string, endDate?: string): Promise<CostBreakdown> {
    try {
      const params = new URLSearchParams();
      if (startDate) params.append('start_date', startDate);
      if (endDate) params.append('end_date', endDate);
      const query = params.toString();

      const data = await this.safeRequest(`/api/v1/projects/${projectId}/costs${query ? '?' + query : ''}`);

      return {
        ec2_compute: data.ec2_compute || 0,
        ebs_storage: data.ebs_storage || 0,
        efs_storage: data.efs_storage || 0,
        data_transfer: data.data_transfer || 0,
        other: data.other || 0,
        total: data.total || 0
      };
    } catch (error) {
      console.error(`Failed to fetch cost breakdown for project ${projectId}:`, error);
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

    await this.safeRequest(`/api/v1/projects/${projectId}/budget`, 'PUT', {
      total_budget: totalBudget,
      alert_thresholds: alerts,
      budget_period: 'monthly'
    });
  }

  // AMI Management APIs
  async getAMIs(): Promise<AMI[]> {
    try {
      const data = await this.safeRequest('/api/v1/ami/list');

      // Transform backend response to frontend format
      if (!data || !Array.isArray(data)) {
        return [];
      }

      return data.map((ami: any) => ({
        id: ami.id || ami.ami_id || '',
        name: ami.name || ami.id || '',
        template_name: ami.template_name || ami.template || 'unknown',
        region: ami.region || 'us-west-2',
        state: ami.state || 'available',
        architecture: ami.architecture || 'x86_64',
        size_gb: ami.size_gb || ami.size || 0,
        description: ami.description || '',
        created_at: ami.created_at || ami.creation_date || new Date().toISOString(),
        tags: ami.tags || {}
      }));
    } catch (error) {
      console.error('Failed to fetch AMIs:', error);
      return [];
    }
  }

  async getAMIBuilds(): Promise<AMIBuild[]> {
    try {
      // Note: Backend may not have build tracking yet
      // Return empty array for now
      return [];
    } catch (error) {
      console.error('Failed to fetch AMI builds:', error);
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
      console.error('Failed to calculate AMI regions:', error);
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
    const response = await this.safeRequest('/api/v1/ami/create', 'POST', {
      template_name: templateName
    });
    return response;
  }

  // Rightsizing APIs
  async getRightsizingRecommendations(): Promise<RightsizingRecommendation[]> {
    try {
      const data = await this.safeRequest('/api/v1/rightsizing/recommendations');
      if (!data || !Array.isArray(data.recommendations)) {
        return [];
      }
      return data.recommendations.map((rec: any) => ({
        instance_name: rec.instance_name || rec.InstanceName || '',
        current_type: rec.current_type || rec.CurrentType || '',
        recommended_type: rec.recommended_type || rec.RecommendedType || '',
        cpu_utilization: rec.cpu_utilization || rec.CPUUtilization || 0,
        memory_utilization: rec.memory_utilization || rec.MemoryUtilization || 0,
        current_cost: rec.current_cost || rec.CurrentCost || 0,
        recommended_cost: rec.recommended_cost || rec.RecommendedCost || 0,
        monthly_savings: rec.monthly_savings || rec.MonthlySavings || 0,
        savings_percentage: rec.savings_percentage || rec.SavingsPercentage || 0,
        confidence: rec.confidence || rec.Confidence || 'medium',
        reason: rec.reason || rec.Reason
      }));
    } catch (error) {
      console.error('Failed to fetch rightsizing recommendations:', error);
      return [];
    }
  }

  async getRightsizingStats(): Promise<RightsizingStats | null> {
    try {
      const data = await this.safeRequest('/api/v1/rightsizing/stats');
      return {
        total_recommendations: data.total_recommendations || 0,
        total_monthly_savings: data.total_monthly_savings || 0,
        average_cpu_utilization: data.average_cpu_utilization || 0,
        average_memory_utilization: data.average_memory_utilization || 0,
        over_provisioned_count: data.over_provisioned_count || 0,
        optimized_count: data.optimized_count || 0
      };
    } catch (error: any) {
      // Silently handle 400/404 - endpoint may not be implemented yet
      const errorMessage = error?.message || String(error);
      if (errorMessage.includes('HTTP 400') || errorMessage.includes('HTTP 404')) {
        return null; // Don't log, just return null
      }
      // Only log unexpected errors
      console.error('Unexpected error fetching rightsizing stats:', error);
      return null;
    }
  }

  async applyRightsizingRecommendation(instanceName: string): Promise<void> {
    await this.safeRequest(`/api/v1/rightsizing/instance/${instanceName}/apply`, 'POST');
  }

  // Policy APIs
  async getPolicyStatus(): Promise<PolicyStatus | null> {
    try {
      const data = await this.safeRequest('/api/v1/policies/status');
      return {
        enabled: data.enabled || false,
        status: data.status || 'unknown',
        status_icon: data.status_icon || '',
        assigned_policies: data.assigned_policies || [],
        message: data.message
      };
    } catch (error) {
      console.error('Failed to fetch policy status:', error);
      return null;
    }
  }

  async getPolicySets(): Promise<PolicySet[]> {
    try {
      const data = await this.safeRequest('/api/v1/policies/sets');
      if (!data || !data.policy_sets) {
        return [];
      }
      return Object.entries(data.policy_sets).map(([id, info]: [string, any]) => ({
        id,
        name: info.name || id,
        description: info.description || '',
        policies: info.policies || 0,
        status: info.status || 'active',
        tags: info.tags
      }));
    } catch (error) {
      console.error('Failed to fetch policy sets:', error);
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
    const data = await this.safeRequest('/api/v1/policies/check', 'POST', { template_name: templateName });
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

      const data = await this.safeRequest(url);
      if (!data || !Array.isArray(data.templates)) {
        return [];
      }
      return data.templates.map((t: any) => ({
        id: t.id || t.ID || '',
        name: t.name || t.Name || '',
        display_name: t.display_name || t.DisplayName || t.name || '',
        author: t.author || t.Author || '',
        publisher: t.publisher || t.Publisher || '',
        category: t.category || t.Category || '',
        description: t.description || t.Description || '',
        rating: t.rating || t.Rating || 0,
        downloads: t.downloads || t.Downloads || 0,
        verified: t.verified || t.Verified || false,
        featured: t.featured || t.Featured || false,
        version: t.version || t.Version || '',
        tags: t.tags || t.Tags,
        badges: t.badges || t.Badges,
        created_at: t.created_at || t.CreatedAt || '',
        updated_at: t.updated_at || t.UpdatedAt || '',
        ami_available: t.ami_available || t.AMIAvailable || false
      }));
    } catch (error) {
      console.error('Failed to fetch marketplace templates:', error);
      return [];
    }
  }

  async getMarketplaceCategories(): Promise<MarketplaceCategory[]> {
    try {
      const data = await this.safeRequest('/api/v1/marketplace/categories');
      if (!data || !Array.isArray(data.categories)) {
        return [];
      }
      return data.categories.map((c: any) => ({
        id: c.id || c.ID || '',
        name: c.name || c.Name || '',
        count: c.count || c.Count || 0
      }));
    } catch (error) {
      console.error('Failed to fetch marketplace categories:', error);
      return [];
    }
  }

  async installMarketplaceTemplate(templateId: string): Promise<void> {
    await this.safeRequest('/api/v1/templates/install-marketplace', 'POST', { template_id: templateId });
  }

  // Idle Detection APIs
  async getIdlePolicies(): Promise<IdlePolicy[]> {
    try {
      const data = await this.safeRequest('/api/v1/idle/policies');
      if (!data || !data.policies) {
        return [];
      }
      const policies = Object.entries(data.policies).map(([id, p]: [string, any]) => ({
        id,
        name: p.name || p.Name || id,
        idle_minutes: p.idle_minutes || p.IdleMinutes || 0,
        action: p.action || p.Action || 'notify',
        cpu_threshold: p.cpu_threshold || p.CPUThreshold || 10,
        memory_threshold: p.memory_threshold || p.MemoryThreshold || 10,
        network_threshold: p.network_threshold || p.NetworkThreshold || 1,
        description: p.description || p.Description,
        enabled: p.enabled !== undefined ? p.enabled : (p.Enabled !== undefined ? p.Enabled : true)
      }));
      return policies;
    } catch (error) {
      console.error('Failed to fetch idle policies:', error);
      return [];
    }
  }

  async getIdleSchedules(): Promise<IdleSchedule[]> {
    try {
      const data = await this.safeRequest('/api/v1/idle/schedules');
      if (!data || !Array.isArray(data.schedules)) {
        return [];
      }
      return data.schedules.map((s: any) => ({
        instance_name: s.instance_name || s.InstanceName || '',
        policy_name: s.policy_name || s.PolicyName || '',
        enabled: s.enabled !== undefined ? s.enabled : (s.Enabled !== undefined ? s.Enabled : true),
        last_checked: s.last_checked || s.LastChecked || '',
        idle_minutes: s.idle_minutes || s.IdleMinutes || 0,
        status: s.status || s.Status || ''
      }));
    } catch (error) {
      console.error('Failed to fetch idle schedules:', error);
      return [];
    }
  }
}

export default function PrismApp() {
  const api = new SafePrismAPI();

  const [state, setState] = useState<AppState>({
    activeView: 'dashboard',
    templates: {},
    instances: [],
    efsVolumes: [],
    ebsVolumes: [],
    projects: [],
    users: [],
    budgets: [],
    amis: [],
    amiBuilds: [],
    amiRegions: [],
    rightsizingRecommendations: [],
    rightsizingStats: null,
    policyStatus: null,
    policySets: [],
    marketplaceTemplates: [],
    marketplaceCategories: [],
    idlePolicies: [],
    idleSchedules: [],
    selectedTemplate: null,
    selectedProject: null,
    selectedTerminalInstance: '',
    loading: true,
    notifications: [],
    connected: false,
    error: null
  });

  const [navigationOpen, setNavigationOpen] = useState(true);
  const [launchModalVisible, setLaunchModalVisible] = useState(false);
  const [launchConfig, setLaunchConfig] = useState({
    name: '',
    size: 'M'
  });

  // Delete confirmation modal state
  const [deleteModalVisible, setDeleteModalVisible] = useState(false);
  const [deleteModalConfig, setDeleteModalConfig] = useState<{
    type: 'instance' | 'efs-volume' | 'ebs-volume' | 'project' | 'user' | null;
    name: string;
    requireNameConfirmation: boolean;
    onConfirm: () => Promise<void>;
  }>({
    type: null,
    name: '',
    requireNameConfirmation: false,
    onConfirm: async () => {}
  });
  const [deleteConfirmationText, setDeleteConfirmationText] = useState('');

  // Onboarding wizard state
  const [onboardingVisible, setOnboardingVisible] = useState(false);
  const [onboardingStep, setOnboardingStep] = useState(0);
  const [onboardingComplete, setOnboardingComplete] = useState(() => {
    // Check if user has completed onboarding before
    const completed = localStorage.getItem('cws_onboarding_complete');
    return completed === 'true';
  });

  // Quick Start Wizard state
  const [quickStartWizardVisible, setQuickStartWizardVisible] = useState(false);
  const [quickStartActiveStepIndex, setQuickStartActiveStepIndex] = useState(0);
  const [quickStartConfig, setQuickStartConfig] = useState({
    selectedTemplate: null as Template | null,
    workspaceName: '',
    size: 'M',
    launchInProgress: false,
    launchedWorkspaceId: null as string | null
  });

  // Bulk selection state for instances
  const [selectedInstances, setSelectedInstances] = useState<Instance[]>([]);

  // Filtering state for instances table
  const [instancesFilterQuery, setInstancesFilterQuery] = useState({ tokens: [], operation: 'and' as const });

  // Safe data loading with comprehensive error handling
  const loadApplicationData = async () => {
    try {
      setState(prev => ({ ...prev, loading: true, error: null }));

      console.log('Loading Prism data...');

      const [templatesData, instancesData, efsVolumesData, ebsVolumesData, projectsData, usersData, budgetsData, amisData, amiBuildsData, amiRegionsData, rightsizingRecommendationsData, rightsizingStatsData, policyStatusData, policySetsData, marketplaceTemplatesData, marketplaceCategoriesData, idlePoliciesData, idleSchedulesData] = await Promise.all([
        api.getTemplates(),
        api.getInstances(),
        api.getEFSVolumes(),
        api.getEBSVolumes(),
        api.getProjects(),
        api.getUsers(),
        api.getBudgets(),
        api.getAMIs(),
        api.getAMIBuilds(),
        api.getAMIRegions(),
        api.getRightsizingRecommendations(),
        api.getRightsizingStats(),
        api.getPolicyStatus(),
        api.getPolicySets(),
        api.getMarketplaceTemplates(),
        api.getMarketplaceCategories(),
        api.getIdlePolicies(),
        api.getIdleSchedules()
      ]);

      console.log('Templates loaded:', Object.keys(templatesData).length);
      console.log('Instances loaded:', instancesData.length);
      console.log('EFS Volumes loaded:', efsVolumesData.length);
      console.log('EBS Volumes loaded:', ebsVolumesData.length);
      console.log('Projects loaded:', projectsData.length);
      console.log('Users loaded:', usersData.length);
      console.log('Budgets loaded:', budgetsData.length);
      console.log('AMIs loaded:', amisData.length);
      console.log('AMI Builds loaded:', amiBuildsData.length);
      console.log('AMI Regions loaded:', amiRegionsData.length);

      setState(prev => ({
        ...prev,
        templates: templatesData,
        instances: instancesData,
        efsVolumes: efsVolumesData,
        ebsVolumes: ebsVolumesData,
        projects: projectsData,
        users: usersData,
        budgets: budgetsData,
        amis: amisData,
        amiBuilds: amiBuildsData,
        amiRegions: amiRegionsData,
        rightsizingRecommendations: rightsizingRecommendationsData,
        rightsizingStats: rightsizingStatsData,
        policyStatus: policyStatusData,
        policySets: policySetsData,
        marketplaceTemplates: marketplaceTemplatesData,
        marketplaceCategories: marketplaceCategoriesData,
        idlePolicies: idlePoliciesData,
        idleSchedules: idleSchedulesData,
        loading: false,
        connected: true,
        error: null
      }));

      // Clear connection error notifications
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n =>
          n.type !== 'error' || !n.content.includes('daemon')
        )
      }));

    } catch (error) {
      console.error('Failed to load application data:', error);

      setState(prev => ({
        ...prev,
        loading: false,
        connected: false,
        error: error instanceof Error ? error.message : 'Unknown error',
        notifications: [
          {
            type: 'error',
            header: 'Connection Error',
            content: `Failed to connect to Prism daemon: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  // Utility function to get accessible status labels (WCAG 1.1.1)
  const getStatusLabel = (context: string, status: string, additionalInfo?: string): string => {
    const labels: Record<string, Record<string, string>> = {
      instance: {
        running: 'Workspace running',
        stopped: 'Workspace stopped',
        pending: 'Workspace pending',
        stopping: 'Workspace stopping',
        terminated: 'Workspace terminated',
        hibernated: 'Workspace hibernated'
      },
      volume: {
        available: 'Volume available',
        'in-use': 'Volume in use',
        creating: 'Volume creating',
        deleting: 'Volume deleting'
      },
      project: {
        active: 'Project active',
        suspended: 'Project suspended',
        archived: 'Project archived'
      },
      user: {
        active: 'User active',
        inactive: 'User inactive'
      },
      connection: {
        success: 'Connected to daemon',
        error: 'Disconnected from daemon'
      },
      ami: {
        available: 'AMI available',
        pending: 'AMI pending',
        failed: 'AMI failed'
      },
      build: {
        completed: 'Build completed',
        failed: 'Build failed',
        'in-progress': 'Build in progress'
      },
      budget: {
        ok: 'Budget OK',
        warning: 'Budget warning',
        critical: 'Budget critical'
      },
      policy: {
        enabled: 'Policy enabled',
        disabled: 'Policy disabled'
      },
      marketplace: {
        verified: 'Verified publisher',
        community: 'Community template'
      },
      idle: {
        enabled: 'Idle detection enabled',
        disabled: 'Idle detection disabled'
      },
      auth: {
        authenticated: 'Authenticated'
      }
    };
    const label = labels[context]?.[status] || `${context} ${status}`;
    return additionalInfo ? `${label}: ${additionalInfo}` : label;
  };

  // Load data on mount and refresh periodically
  useEffect(() => {
    loadApplicationData();
    const interval = setInterval(loadApplicationData, 30000);
    return () => clearInterval(interval);
  }, []);

  // Show onboarding for first-time users
  useEffect(() => {
    if (!onboardingComplete && state.connected && !state.loading) {
      // Show onboarding after a short delay to let the UI settle
      const timer = setTimeout(() => {
        setOnboardingVisible(true);
      }, 1000);
      return () => clearTimeout(timer);
    }
  }, [onboardingComplete, state.connected, state.loading]);

  // Keyboard shortcuts for common actions
  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      // Skip if user is typing in an input field
      const target = event.target as HTMLElement;
      if (target.tagName === 'INPUT' || target.tagName === 'TEXTAREA' || target.isContentEditable) {
        return;
      }

      // Cmd/Ctrl + R: Refresh data
      if ((event.metaKey || event.ctrlKey) && event.key === 'r') {
        event.preventDefault();
        loadApplicationData();
        setState(prev => ({
          ...prev,
          notifications: [...prev.notifications, {
            type: 'success',
            content: 'Data refreshed',
            dismissible: true,
            id: Date.now().toString()
          }]
        }));
      }

      // Cmd/Ctrl + K: Focus search/filter
      if ((event.metaKey || event.ctrlKey) && event.key === 'k') {
        event.preventDefault();
        // Focus first search input if available
        const searchInput = document.querySelector('input[type="search"]') as HTMLInputElement;
        if (searchInput) searchInput.focus();
      }

      // Number keys 1-9: Navigate to views
      if (!event.metaKey && !event.ctrlKey && !event.altKey) {
        const viewMap: Record<string, string> = {
          '1': 'dashboard',
          '2': 'templates',
          '3': 'instances',
          '4': 'storage',
          '5': 'projects',
          '6': 'users',
          '7': 'settings'
        };
        if (viewMap[event.key]) {
          event.preventDefault();
          setState(prev => ({ ...prev, activeView: viewMap[event.key] }));
        }
      }

      // ? : Show keyboard shortcuts help
      if (event.key === '?' && !event.shiftKey) {
        setState(prev => ({
          ...prev,
          notifications: [...prev.notifications, {
            type: 'info',
            header: 'Keyboard Shortcuts',
            content: 'Cmd/Ctrl+R: Refresh | Cmd/Ctrl+K: Search | 1-7: Navigate views | ?: Help',
            dismissible: true,
            id: Date.now().toString()
          }]
        }));
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => window.removeEventListener('keydown', handleKeyPress);
  }, [state.activeView]);

  // Safe template selection
  const handleTemplateSelection = (template: Template) => {
    try {
      setState(prev => ({ ...prev, selectedTemplate: template }));
      setLaunchConfig({ name: '', size: 'M' });
      setLaunchModalVisible(true);
    } catch (error) {
      console.error('Template selection failed:', error);
    }
  };

  // Handle modal dismissal
  const handleModalDismiss = () => {
    setLaunchModalVisible(false);
    setState(prev => ({ ...prev, selectedTemplate: null }));
  };

  // Safe instance launch
  const handleLaunchInstance = async () => {
    if (!state.selectedTemplate || !launchConfig.name.trim()) {
      return;
    }

    try {
      const templateSlug = getTemplateSlug(state.selectedTemplate);
      const templateName = getTemplateName(state.selectedTemplate);

      await api.launchInstance(templateSlug, launchConfig.name, launchConfig.size);

      setState(prev => ({
        ...prev,
        notifications: [
          {
            type: 'success',
            header: 'Workspace Launched',
            content: `Successfully launched ${launchConfig.name} using ${templateName}`,
            dismissible: true,
            id: Date.now().toString()
          },
          ...prev.notifications
        ]
      }));

      handleModalDismiss();
      loadApplicationData();

    } catch (error) {
      setState(prev => ({
        ...prev,
        notifications: [
          {
            type: 'error',
            header: 'Launch Failed',
            content: `Failed to launch workspace: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          },
          ...prev.notifications
        ]
      }));
    }
  };

  // Dashboard View
  const DashboardView = () => (
    <SpaceBetween size="l">
      {/* Hero Section with Quick Start CTA */}
      <Container>
        <SpaceBetween size="l">
          <Box textAlign="center" padding={{ top: 'xl', bottom: 'l' }}>
            <SpaceBetween size="m">
              <TextContent>
                <h1>Welcome to Prism</h1>
                <p>
                  <Box variant="p" fontSize="heading-m" color="text-body-secondary">
                    Launch your research workspace in seconds
                  </Box>
                </p>
              </TextContent>
              <Button
                variant="primary"
                iconName="add-plus"
                onClick={() => setQuickStartWizardVisible(true)}
              >
                Quick Start - Launch Workspace
              </Button>
              <Box color="text-body-secondary">
                Pre-configured environments for ML, Data Science, Bioinformatics, and more
              </Box>
            </SpaceBetween>
          </Box>
        </SpaceBetween>
      </Container>

      <Header
        variant="h1"
        description="Prism research computing platform - manage your cloud environments"
        actions={
          <Button onClick={loadApplicationData} disabled={state.loading}>
            {state.loading ? <Spinner size="normal" /> : 'Refresh'}
          </Button>
        }
      >
        Dashboard
      </Header>

      <ColumnLayout columns={3} variant="text-grid">
        <Container header={<Header variant="h2">Research Templates</Header>}>
          <SpaceBetween size="s">
            <Box>
              <Box variant="awsui-key-label">Available Templates</Box>
              <Box fontSize="display-l" fontWeight="bold" color={state.connected ? 'text-status-success' : 'text-status-error'}>
                {Object.keys(state.templates).length}
              </Box>
            </Box>
            <Button
              variant="primary"
              onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
            >
              Browse Templates
            </Button>
          </SpaceBetween>
        </Container>

        <Container header={<Header variant="h2">Active Workspaces</Header>}>
          <SpaceBetween size="s">
            <Box>
              <Box variant="awsui-key-label">Running Workspaces</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
                {state.instances.filter(i => i.state === 'running').length}
              </Box>
            </Box>
            <Button
              onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
            >
              Manage Workspaces
            </Button>
          </SpaceBetween>
        </Container>

        <Container header={<Header variant="h2">System Status</Header>}>
          <SpaceBetween size="s">
            <Box>
              <Box variant="awsui-key-label">Connection</Box>
              <StatusIndicator
                type={state.connected ? 'success' : 'error'}
                ariaLabel={getStatusLabel('connection', state.connected ? 'success' : 'error')}
              >
                {state.connected ? 'Connected' : 'Disconnected'}
              </StatusIndicator>
            </Box>
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? 'Checking...' : 'Test Connection'}
            </Button>
          </SpaceBetween>
        </Container>
      </ColumnLayout>

      <Container header={<Header variant="h2">Quick Actions</Header>}>
        <SpaceBetween direction="horizontal" size="s">
          <Button
            variant="primary"
            onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
            disabled={Object.keys(state.templates).length === 0}
          >
            Launch New Workspace
          </Button>
          <Button
            onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
            disabled={state.instances.length === 0}
          >
            View Workspaces ({state.instances.length})
          </Button>
          <Button onClick={() => setState(prev => ({ ...prev, activeView: 'storage' }))}>
            Storage Management
          </Button>
        </SpaceBetween>
      </Container>
    </SpaceBetween>
  );

  // Safe accessors for template data
  const getTemplateName = (template: Template): string => {
    return template.Name || template.name || 'Unnamed Template';
  };

  const getTemplateSlug = (template: Template): string => {
    return template.Slug || template.slug || '';
  };

  const getTemplateDescription = (template: Template): string => {
    return template.Description || template.description || 'Professional research computing environment';
  };

  // Templates View
  const TemplateSelectionView = () => {
    const templateList = Object.values(state.templates);

    if (state.loading) {
      return (
        <Container>
          <Box textAlign="center" padding="xl">
            <Spinner size="large" />
            <Box variant="p" color="text-body-secondary">
              Loading templates from AWS...
            </Box>
          </Box>
        </Container>
      );
    }

    if (templateList.length === 0) {
      return (
        <Container>
          <Box textAlign="center" padding="xl">
            <Box variant="strong">No templates available</Box>
            <Box variant="p">Unable to load templates. Check your connection.</Box>
            <Button onClick={loadApplicationData}>Retry</Button>
          </Box>
        </Container>
      );
    }

    return (
      <SpaceBetween size="l">
        <Container
          header={
            <Header
              variant="h1"
              description={`${templateList.length} pre-configured research environments ready to launch`}
              counter={`(${templateList.length} templates)`}
              actions={
                <SpaceBetween direction="horizontal" size="xs">
                  <Button onClick={loadApplicationData} disabled={state.loading}>
                    {state.loading ? <Spinner /> : 'Refresh'}
                  </Button>
                  <Button
                    variant="primary"
                    disabled={!state.selectedTemplate}
                    onClick={() => setLaunchModalVisible(true)}
                  >
                    Launch Selected
                  </Button>
                </SpaceBetween>
              }
            >
              Research Templates
            </Header>
          }
        >
          {/* Working Template Cards Implementation */}
          <SpaceBetween size="m" data-testid="cards">
            {templateList.map((template, index) => (
              <Container
                key={getTemplateName(template)}
                data-testid="card"
              >
                <SpaceBetween size="s">
                  <Box>
                    <Box variant="h3">{getTemplateName(template)}</Box>
                    <Box variant="small" color="text-body-secondary">
                      {getTemplateDescription(template)}
                    </Box>
                  </Box>
                  <Box>
                    <Button
                      variant="primary"
                      onClick={() => handleTemplateSelection(template)}
                    >
                      Launch Template
                    </Button>
                  </Box>
                </SpaceBetween>
              </Container>
            ))}
          </SpaceBetween>
        </Container>
      </SpaceBetween>
    );
  };

  // Comprehensive Instance Action Handler
  const handleInstanceAction = async (action: string, instance: Instance) => {
    try {
      setState(prev => ({ ...prev, loading: true }));

      let actionMessage = '';

      switch (action) {
        case 'start':
          await api.startInstance(instance.name);
          actionMessage = `Started instance ${instance.name}`;
          break;
        case 'stop':
          await api.stopInstance(instance.name);
          actionMessage = `Stopped instance ${instance.name}`;
          break;
        case 'hibernate':
          await api.hibernateInstance(instance.name);
          actionMessage = `Hibernated instance ${instance.name}`;
          break;
        case 'resume':
          await api.resumeInstance(instance.name);
          actionMessage = `Resumed instance ${instance.name}`;
          break;
        case 'connect':
          const connectionInfo = await api.getConnectionInfo(instance.name);
          // Copy to clipboard and show notification
          navigator.clipboard.writeText(connectionInfo);
          actionMessage = `Connection command copied to clipboard: ${connectionInfo}`;
          break;
        case 'delete':
          // Show confirmation modal instead of deleting immediately
          setState(prev => ({ ...prev, loading: false }));
          setDeleteModalConfig({
            type: 'instance',
            name: instance.name,
            requireNameConfirmation: true,
            onConfirm: async () => {
              try {
                await api.deleteInstance(instance.name);
                setState(prev => ({
                  ...prev,
                  notifications: [
                    ...prev.notifications,
                    {
                      type: 'success',
                      header: 'Workspace Deleted',
                      content: `Successfully deleted workspace ${instance.name}`,
                      dismissible: true,
                      id: Date.now().toString()
                    }
                  ]
                }));
                setDeleteModalVisible(false);
                setDeleteConfirmationText('');
                setTimeout(loadApplicationData, 1000);
              } catch (error) {
                setState(prev => ({
                  ...prev,
                  notifications: [
                    ...prev.notifications,
                    {
                      type: 'error',
                      header: 'Delete Failed',
                      content: `Failed to delete workspace: ${error instanceof Error ? error.message : 'Unknown error'}`,
                      dismissible: true,
                      id: Date.now().toString()
                    }
                  ]
                }));
              }
            }
          });
          setDeleteModalVisible(true);
          return; // Don't continue with normal flow
        default:
          throw new Error(`Unknown action: ${action}`);
      }

      // Add success notification
      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: 'success',
            header: 'Action Successful',
            content: actionMessage,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));

      // Refresh instances after action
      setTimeout(loadApplicationData, 1000);

    } catch (error) {
      console.error(`Failed to ${action} workspace ${instance.name}:`, error);

      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: 'error',
            header: 'Action Failed',
            content: `Failed to ${action} workspace ${instance.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  // Bulk action handlers for multiple instances
  const handleBulkAction = async (action: 'start' | 'stop' | 'hibernate' | 'delete') => {
    if (selectedInstances.length === 0) {
      setState(prev => ({
        ...prev,
        notifications: [
          ...prev.notifications,
          {
            type: 'warning',
            header: 'No Workspaces Selected',
            content: 'Please select one or more workspaces to perform bulk actions.',
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
      return;
    }

    // For delete, show confirmation modal
    if (action === 'delete') {
      setDeleteModalConfig({
        type: 'instance',
        name: `${selectedInstances.length} workspace${selectedInstances.length > 1 ? 's' : ''}`,
        requireNameConfirmation: false,
        onConfirm: async () => {
          await executeBulkAction('delete');
          setDeleteModalVisible(false);
        }
      });
      setDeleteModalVisible(true);
      return;
    }

    // Execute non-delete bulk actions immediately
    await executeBulkAction(action);
  };

  const executeBulkAction = async (action: 'start' | 'stop' | 'hibernate' | 'delete') => {
    try {
      setState(prev => ({ ...prev, loading: true }));

      // Execute action on all selected instances
      const results = await Promise.allSettled(
        selectedInstances.map(async (instance) => {
          switch (action) {
            case 'start':
              return await api.startInstance(instance.name);
            case 'stop':
              return await api.stopInstance(instance.name);
            case 'hibernate':
              return await api.hibernateInstance(instance.name);
            case 'delete':
              return await api.deleteInstance(instance.name);
            default:
              throw new Error(`Unknown action: ${action}`);
          }
        })
      );

      // Count successes and failures
      const successes = results.filter(r => r.status === 'fulfilled').length;
      const failures = results.filter(r => r.status === 'rejected').length;

      // Show notification with results
      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: failures > 0 ? 'warning' : 'success',
            header: `Bulk ${action.charAt(0).toUpperCase() + action.slice(1)} ${failures > 0 ? 'Partially Complete' : 'Complete'}`,
            content: `Successfully ${action}ed ${successes} workspace${successes !== 1 ? 's' : ''}${failures > 0 ? `, failed to ${action} ${failures} workspace${failures !== 1 ? 's' : ''}` : ''}.`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));

      // Clear selection and refresh data
      setSelectedInstances([]);
      setTimeout(loadApplicationData, 1000);

    } catch (error) {
      console.error(`Failed to execute bulk ${action}:`, error);

      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: 'error',
            header: 'Bulk Action Failed',
            content: `Failed to ${action} workspaces: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  // Filter instances based on PropertyFilter query
  const getFilteredInstances = () => {
    if (!instancesFilterQuery.tokens || instancesFilterQuery.tokens.length === 0) {
      return state.instances;
    }

    return state.instances.filter((instance) => {
      return instancesFilterQuery.tokens.every((token: any) => {
        const { propertyKey, value, operator } = token;

        if (!propertyKey) {
          // Free text search across all fields
          const searchValue = value.toLowerCase();
          return (
            instance.name.toLowerCase().includes(searchValue) ||
            instance.template.toLowerCase().includes(searchValue) ||
            instance.state.toLowerCase().includes(searchValue) ||
            (instance.public_ip && instance.public_ip.toLowerCase().includes(searchValue))
          );
        }

        // Property-specific filtering
        const instanceValue = instance[propertyKey as keyof Instance];
        if (!instanceValue) return false;

        const stringValue = String(instanceValue).toLowerCase();
        const filterValue = value.toLowerCase();

        switch (operator) {
          case '=':
            return stringValue === filterValue;
          case '!=':
            return stringValue !== filterValue;
          case ':':
            return stringValue.includes(filterValue);
          case '!:':
            return !stringValue.includes(filterValue);
          default:
            return stringValue.includes(filterValue);
        }
      });
    });
  };

  // Instances View
  const InstanceManagementView = () => (
    <SpaceBetween size="l">
      <Container
        header={
          <Header
            variant="h1"
            description="Monitor and manage your research computing environments"
            counter={`(${state.instances.length})`}
            actions={
              <SpaceBetween direction="horizontal" size="xs">
                <Button onClick={loadApplicationData} disabled={state.loading}>
                  {state.loading ? <Spinner /> : 'Refresh'}
                </Button>
                <Button
                  variant="primary"
                  onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
                >
                  Launch New Workspace
                </Button>
              </SpaceBetween>
            }
          >
            My Workspaces
          </Header>
        }
      >
        {/* Advanced Filtering */}
        <PropertyFilter
          query={instancesFilterQuery}
          onChange={({ detail }) => setInstancesFilterQuery(detail)}
          filteringPlaceholder="Search workspaces"
          filteringProperties={[
            {
              key: 'name',
              propertyLabel: 'Workspace Name',
              operators: [':', '!:', '=', '!=']
            },
            {
              key: 'template',
              propertyLabel: 'Template',
              operators: [':', '!:', '=', '!=']
            },
            {
              key: 'state',
              propertyLabel: 'Status',
              operators: ['=', '!=']
            },
            {
              key: 'public_ip',
              propertyLabel: 'Public IP',
              operators: [':', '!:', '=', '!=']
            }
          ]}
          filteringOptions={[
            { propertyKey: 'state', value: 'running', label: 'Status: Running' },
            { propertyKey: 'state', value: 'stopped', label: 'Status: Stopped' },
            { propertyKey: 'state', value: 'hibernated', label: 'Status: Hibernated' },
            { propertyKey: 'state', value: 'pending', label: 'Status: Pending' }
          ]}
        />

        {/* Bulk Actions Toolbar */}
        {selectedInstances.length > 0 && (
          <SpaceBetween direction="horizontal" size="xs">
            <Box variant="awsui-key-label">
              {selectedInstances.length} workspace{selectedInstances.length !== 1 ? 's' : ''} selected
            </Box>
            <Button
              onClick={() => handleBulkAction('start')}
              disabled={state.loading || selectedInstances.every(i => i.state === 'running')}
            >
              Start Selected
            </Button>
            <Button
              onClick={() => handleBulkAction('stop')}
              disabled={state.loading || selectedInstances.every(i => i.state !== 'running')}
            >
              Stop Selected
            </Button>
            <Button
              onClick={() => handleBulkAction('hibernate')}
              disabled={state.loading || selectedInstances.every(i => i.state !== 'running')}
            >
              Hibernate Selected
            </Button>
            <Button
              onClick={() => handleBulkAction('delete')}
              disabled={state.loading}
            >
              Delete Selected
            </Button>
            <Button
              variant="link"
              onClick={() => setSelectedInstances([])}
            >
              Clear Selection
            </Button>
          </SpaceBetween>
        )}
        <Table
          selectionType="multi"
          selectedItems={selectedInstances}
          onSelectionChange={({ detail }) => setSelectedInstances(detail.selectedItems)}
          columnDefinitions={[
            {
              id: "name",
              header: "Workspace Name",
              cell: (item: Instance) => <Link fontSize="body-m">{item.name}</Link>,
              sortingField: "name"
            },
            {
              id: "template",
              header: "Template",
              cell: (item: Instance) => item.template
            },
            {
              id: "status",
              header: "Status",
              cell: (item: Instance) => (
                <StatusIndicator
                  type={
                    item.state === 'running' ? 'success' :
                    item.state === 'stopped' ? 'stopped' :
                    item.state === 'hibernated' ? 'pending' :
                    item.state === 'pending' ? 'in-progress' : 'error'
                  }
                  ariaLabel={getStatusLabel('instance', item.state)}
                >
                  {item.state}
                </StatusIndicator>
              )
            },
            {
              id: "public_ip",
              header: "Public IP",
              cell: (item: Instance) => item.public_ip || 'Not assigned'
            },
            {
              id: "actions",
              header: "Actions",
              cell: (item: Instance) => (
                <ButtonDropdown
                  expandToViewport
                  items={[
                    { text: 'Connect', id: 'connect', disabled: item.state !== 'running' },
                    { text: 'Stop', id: 'stop', disabled: item.state !== 'running' },
                    { text: 'Start', id: 'start', disabled: item.state === 'running' },
                    { text: 'Hibernate', id: 'hibernate', disabled: item.state !== 'running' },
                    { text: 'Resume', id: 'resume', disabled: item.state !== 'stopped' },
                    { text: 'Delete', id: 'delete', disabled: item.state === 'running' || item.state === 'pending' }
                  ]}
                  onItemClick={({ detail }) => {
                    handleInstanceAction(detail.id, item);
                  }}
                >
                  Actions
                </ButtonDropdown>
              )
            }
          ]}
          items={getFilteredInstances()}
          loadingText="Loading workspaces from AWS"
          loading={state.loading}
          trackBy="id"
          empty={
            <Box textAlign="center" color="inherit">
              <Box variant="strong" textAlign="center" color="inherit">
                No workspaces running
              </Box>
              <Box variant="p" padding={{ bottom: 's' }} color="inherit">
                Launch your first research environment to get started.
              </Box>
              <Button
                variant="primary"
                onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
              >
                Browse Templates
              </Button>
            </Box>
          }
          sortingDisabled={false}
        />
      </Container>
    </SpaceBetween>
  );

  // Comprehensive Storage Action Handler
  const handleStorageAction = async (action: string, volume: any, volumeType: 'efs' | 'ebs') => {
    try {
      setState(prev => ({ ...prev, loading: true }));

      let actionMessage = '';

      if (volumeType === 'efs') {
        switch (action) {
          case 'delete':
            // Show confirmation modal instead of deleting immediately
            setState(prev => ({ ...prev, loading: false }));
            setDeleteModalConfig({
              type: 'efs-volume',
              name: volume.name,
              requireNameConfirmation: false,
              onConfirm: async () => {
                try {
                  await api.deleteEFSVolume(volume.name);
                  setState(prev => ({
                    ...prev,
                    notifications: [
                      ...prev.notifications,
                      {
                        type: 'success',
                        header: 'EFS Volume Deleted',
                        content: `Successfully deleted EFS volume ${volume.name}`,
                        dismissible: true,
                        id: Date.now().toString()
                      }
                    ]
                  }));
                  setDeleteModalVisible(false);
                  setTimeout(loadApplicationData, 1000);
                } catch (error) {
                  setState(prev => ({
                    ...prev,
                    notifications: [
                      ...prev.notifications,
                      {
                        type: 'error',
                        header: 'Delete Failed',
                        content: `Failed to delete EFS volume: ${error instanceof Error ? error.message : 'Unknown error'}`,
                        dismissible: true,
                        id: Date.now().toString()
                      }
                    ]
                  }));
                }
              }
            });
            setDeleteModalVisible(true);
            return;
          case 'mount':
            // For demo, mount to first available instance
            if (state.instances.length > 0) {
              const instance = state.instances[0].name;
              await api.mountEFSVolume(volume.name, instance);
              actionMessage = `Mounted EFS volume ${volume.name} to ${instance}`;
            } else {
              throw new Error('No running workspaces available for mounting');
            }
            break;
          case 'unmount':
            if (state.instances.length > 0) {
              const instance = state.instances[0].name;
              await api.unmountEFSVolume(volume.name, instance);
              actionMessage = `Unmounted EFS volume ${volume.name} from ${instance}`;
            } else {
              throw new Error('No workspaces to unmount from');
            }
            break;
          default:
            throw new Error(`Unknown EFS action: ${action}`);
        }
      } else if (volumeType === 'ebs') {
        switch (action) {
          case 'delete':
            // Show confirmation modal instead of deleting immediately
            setState(prev => ({ ...prev, loading: false }));
            setDeleteModalConfig({
              type: 'ebs-volume',
              name: volume.name,
              requireNameConfirmation: false,
              onConfirm: async () => {
                try {
                  await api.deleteEBSVolume(volume.name);
                  setState(prev => ({
                    ...prev,
                    notifications: [
                      ...prev.notifications,
                      {
                        type: 'success',
                        header: 'EBS Volume Deleted',
                        content: `Successfully deleted EBS volume ${volume.name}`,
                        dismissible: true,
                        id: Date.now().toString()
                      }
                    ]
                  }));
                  setDeleteModalVisible(false);
                  setTimeout(loadApplicationData, 1000);
                } catch (error) {
                  setState(prev => ({
                    ...prev,
                    notifications: [
                      ...prev.notifications,
                      {
                        type: 'error',
                        header: 'Delete Failed',
                        content: `Failed to delete EBS volume: ${error instanceof Error ? error.message : 'Unknown error'}`,
                        dismissible: true,
                        id: Date.now().toString()
                      }
                    ]
                  }));
                }
              }
            });
            setDeleteModalVisible(true);
            return;
          case 'attach':
            if (state.instances.length > 0) {
              const instance = state.instances[0].name;
              await api.attachEBSVolume(volume.name, instance);
              actionMessage = `Attached EBS volume ${volume.name} to ${instance}`;
            } else {
              throw new Error('No running workspaces available for attachment');
            }
            break;
          case 'detach':
            await api.detachEBSVolume(volume.name);
            actionMessage = `Detached EBS volume ${volume.name}`;
            break;
          default:
            throw new Error(`Unknown EBS action: ${action}`);
        }
      }

      // Add success notification
      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: 'success',
            header: 'Storage Action Successful',
            content: actionMessage,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));

      // Refresh data after action
      setTimeout(loadApplicationData, 1000);

    } catch (error) {
      console.error(`Failed to ${action} ${volumeType} volume ${volume.name}:`, error);

      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: 'error',
            header: 'Storage Action Failed',
            content: `Failed to ${action} ${volumeType.toUpperCase()} volume ${volume.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  // Storage Management View
  const StorageManagementView = () => {
    const [activeTabId, setActiveTabId] = React.useState('shared');

    return (
      <SpaceBetween size="l">
        <Header
          variant="h1"
          description="Manage shared and workspace-specific storage for your research computing environments"
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button onClick={loadApplicationData} disabled={state.loading}>
                {state.loading ? <Spinner /> : 'Refresh'}
              </Button>
            </SpaceBetween>
          }
        >
          Storage
        </Header>

        {/* Educational Overview */}
        <Container>
          <ColumnLayout columns={2} variant="text-grid">
            <SpaceBetween size="s">
              <Box variant="h3">📁 Shared Storage (EFS)</Box>
              <Box color="text-body-secondary">
                <strong>Use for:</strong> Data shared across multiple workspaces, collaborative projects, persistent datasets
              </Box>
              <Box color="text-body-secondary">
                <strong>Cost:</strong> ~$0.30/GB/month (pay for what you use)
              </Box>
              <Box color="text-body-secondary">
                <strong>Performance:</strong> Elastic, scalable file system with automatic capacity
              </Box>
            </SpaceBetween>
            <SpaceBetween size="s">
              <Box variant="h3">💾 Private Storage (EBS)</Box>
              <Box color="text-body-secondary">
                <strong>Use for:</strong> Workspace-specific data, high-performance applications, temporary processing
              </Box>
              <Box color="text-body-secondary">
                <strong>Cost:</strong> ~$0.10/GB/month (fixed allocation, pay for provisioned size)
              </Box>
              <Box color="text-body-secondary">
                <strong>Performance:</strong> High IOPS, low latency, best for compute-intensive work
              </Box>
            </SpaceBetween>
          </ColumnLayout>
        </Container>

        {/* Cost Comparison Alert */}
        <Alert
          type="info"
          header="💡 Storage Selection Guide"
        >
          <ColumnLayout columns={2}>
            <Box>
              <strong>Choose Shared (EFS) when:</strong>
              <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                <li>Multiple workspaces need access to the same data</li>
                <li>Collaborating with other researchers</li>
                <li>Data needs to persist across workspace lifecycles</li>
                <li>Total data size is unpredictable or grows over time</li>
              </ul>
            </Box>
            <Box>
              <strong>Choose Private (EBS) when:</strong>
              <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                <li>Data is workspace-specific and not shared</li>
                <li>Need maximum I/O performance for databases or processing</li>
                <li>Working with large temporary datasets</li>
                <li>Data is tied to a single workspace's lifecycle</li>
              </ul>
            </Box>
          </ColumnLayout>
        </Alert>

        {/* Tabbed Storage Interface */}
        <Tabs
          activeTabId={activeTabId}
          onChange={({ detail }) => setActiveTabId(detail.activeTabId)}
          tabs={[
            {
              id: 'shared',
              label: `Shared (EFS) - ${state.efsVolumes.length}`,
              content: (
                <Container
                  header={
                    <Header
                      variant="h2"
                      description="Elastic File System volumes for multi-workspace data sharing and collaboration"
                      counter={`(${state.efsVolumes.length} volumes)`}
                      actions={
                        <Button variant="primary">
                          Create Shared Storage
                        </Button>
                      }
                      info={
                        <Link variant="info" onFollow={() => {}}>
                          Learn more about EFS
                        </Link>
                      }
                    >
                      Shared Storage Volumes
                    </Header>
                  }
                >
                  <Table
                    columnDefinitions={[
                      {
                        id: "name",
                        header: "Volume Name",
                        cell: (item: EFSVolume) => <Link fontSize="body-m">{item.name}</Link>,
                        sortingField: "name"
                      },
                      {
                        id: "filesystem_id",
                        header: "File System ID",
                        cell: (item: EFSVolume) => item.filesystem_id
                      },
                      {
                        id: "status",
                        header: "Status",
                        cell: (item: EFSVolume) => (
                          <StatusIndicator
                            type={
                              item.state === 'available' ? 'success' :
                              item.state === 'creating' ? 'in-progress' :
                              item.state === 'deleting' ? 'warning' : 'error'
                            }
                            ariaLabel={getStatusLabel('volume', item.state)}
                          >
                            {item.state}
                          </StatusIndicator>
                        )
                      },
                      {
                        id: "size",
                        header: "Size",
                        cell: (item: EFSVolume) => `${Math.round(item.size_bytes / (1024 * 1024 * 1024))} GB`
                      },
                      {
                        id: "cost",
                        header: "Est. Cost/Month",
                        cell: (item: EFSVolume) => {
                          const sizeGB = Math.round(item.size_bytes / (1024 * 1024 * 1024));
                          const monthlyCost = sizeGB * item.estimated_cost_gb;
                          return (
                            <SpaceBetween direction="horizontal" size="xs">
                              <Box>${monthlyCost.toFixed(2)}</Box>
                              <Badge color="grey">${item.estimated_cost_gb.toFixed(3)}/GB</Badge>
                            </SpaceBetween>
                          );
                        }
                      },
                      {
                        id: "actions",
                        header: "Actions",
                        cell: (item: EFSVolume) => (
                          <ButtonDropdown
                            expandToViewport
                            items={[
                              { text: 'Mount to Workspace', id: 'mount', disabled: item.state !== 'available' },
                              { text: 'Unmount', id: 'unmount', disabled: item.state !== 'available' },
                              { text: 'View Details', id: 'details' },
                              { text: 'Delete', id: 'delete', disabled: item.state !== 'available' }
                            ]}
                            onItemClick={({ detail }) => {
                              handleStorageAction(detail.id, item, 'efs');
                            }}
                          >
                            Actions
                          </ButtonDropdown>
                        )
                      }
                    ]}
                    items={state.efsVolumes}
                    loadingText="Loading shared storage volumes from AWS"
                    loading={state.loading}
                    trackBy="name"
                    empty={
                      <Box textAlign="center" color="inherit" padding={{ vertical: 'xl' }}>
                        <SpaceBetween size="m">
                          <Box variant="strong" textAlign="center" color="inherit">
                            No shared storage volumes found
                          </Box>
                          <Box variant="p" color="text-body-secondary">
                            Create shared storage (EFS) for collaborative projects and data that needs to be accessed by multiple workspaces.
                          </Box>
                          <Box textAlign="center">
                            <Button variant="primary">Create Shared Storage</Button>
                          </Box>
                        </SpaceBetween>
                      </Box>
                    }
                    sortingDisabled={false}
                  />
                </Container>
              )
            },
            {
              id: 'private',
              label: `Private (EBS) - ${state.ebsVolumes.length}`,
              content: (
                <Container
                  header={
                    <Header
                      variant="h2"
                      description="Elastic Block Store volumes for high-performance workspace-specific data"
                      counter={`(${state.ebsVolumes.length} volumes)`}
                      actions={
                        <Button variant="primary">
                          Create Private Storage
                        </Button>
                      }
                      info={
                        <Link variant="info" onFollow={() => {}}>
                          Learn more about EBS
                        </Link>
                      }
                    >
                      Private Storage Volumes
                    </Header>
                  }
                >
                  <Table
                    columnDefinitions={[
                      {
                        id: "name",
                        header: "Volume Name",
                        cell: (item: EBSVolume) => <Link fontSize="body-m">{item.name}</Link>,
                        sortingField: "name"
                      },
                      {
                        id: "volume_id",
                        header: "Volume ID",
                        cell: (item: EBSVolume) => item.volume_id
                      },
                      {
                        id: "status",
                        header: "Status",
                        cell: (item: EBSVolume) => (
                          <StatusIndicator
                            type={
                              item.state === 'available' ? 'success' :
                              item.state === 'in-use' ? 'success' :
                              item.state === 'creating' ? 'in-progress' :
                              item.state === 'deleting' ? 'warning' : 'error'
                            }
                            ariaLabel={getStatusLabel('volume', item.state)}
                          >
                            {item.state}
                          </StatusIndicator>
                        )
                      },
                      {
                        id: "type",
                        header: "Type",
                        cell: (item: EBSVolume) => (
                          <SpaceBetween direction="horizontal" size="xs">
                            <Box>{item.volume_type.toUpperCase()}</Box>
                            {item.volume_type.startsWith('gp') && (
                              <Badge color="blue">General Purpose</Badge>
                            )}
                            {item.volume_type.startsWith('io') && (
                              <Badge color="green">High Performance</Badge>
                            )}
                          </SpaceBetween>
                        )
                      },
                      {
                        id: "size",
                        header: "Size",
                        cell: (item: EBSVolume) => `${item.size_gb} GB`
                      },
                      {
                        id: "attached_to",
                        header: "Attached To",
                        cell: (item: EBSVolume) => {
                          if (item.attached_to) {
                            return (
                              <SpaceBetween direction="horizontal" size="xs">
                                <StatusIndicator type="success">
                                  {item.attached_to}
                                </StatusIndicator>
                              </SpaceBetween>
                            );
                          }
                          return <Box color="text-body-secondary">Not attached</Box>;
                        }
                      },
                      {
                        id: "cost",
                        header: "Est. Cost/Month",
                        cell: (item: EBSVolume) => {
                          const monthlyCost = item.size_gb * item.estimated_cost_gb;
                          return (
                            <SpaceBetween direction="horizontal" size="xs">
                              <Box>${monthlyCost.toFixed(2)}</Box>
                              <Badge color="grey">${item.estimated_cost_gb.toFixed(3)}/GB</Badge>
                            </SpaceBetween>
                          );
                        }
                      },
                      {
                        id: "actions",
                        header: "Actions",
                        cell: (item: EBSVolume) => (
                          <ButtonDropdown
                            expandToViewport
                            items={[
                              { text: 'Attach to Workspace', id: 'attach', disabled: item.state !== 'available' },
                              { text: 'Detach', id: 'detach', disabled: item.state !== 'in-use' },
                              { text: 'View Details', id: 'details' },
                              { text: 'Create Snapshot', id: 'snapshot', disabled: item.state !== 'available' && item.state !== 'in-use' },
                              { text: 'Delete', id: 'delete', disabled: item.state === 'in-use' }
                            ]}
                            onItemClick={({ detail }) => {
                              handleStorageAction(detail.id, item, 'ebs');
                            }}
                          >
                            Actions
                          </ButtonDropdown>
                        )
                      }
                    ]}
                    items={state.ebsVolumes}
                    loadingText="Loading private storage volumes from AWS"
                    loading={state.loading}
                    trackBy="name"
                    empty={
                      <Box textAlign="center" color="inherit" padding={{ vertical: 'xl' }}>
                        <SpaceBetween size="m">
                          <Box variant="strong" textAlign="center" color="inherit">
                            No private storage volumes found
                          </Box>
                          <Box variant="p" color="text-body-secondary">
                            Create private storage (EBS) for workspace-specific data and high-performance applications.
                          </Box>
                          <Box textAlign="center">
                            <Button variant="primary">Create Private Storage</Button>
                          </Box>
                        </SpaceBetween>
                      </Box>
                    }
                    sortingDisabled={false}
                  />
                </Container>
              )
            }
          ]}
        />

        {/* Storage Statistics */}
        <Container
          header={
            <Header variant="h2" description="Overview of your storage usage and costs">
              Storage Summary
            </Header>
          }
        >
          <ColumnLayout columns={4} variant="text-grid">
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Total Shared (EFS)</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
                {state.efsVolumes.reduce((sum, v) => sum + Math.round(v.size_bytes / (1024 * 1024 * 1024)), 0)} GB
              </Box>
              <Box color="text-body-secondary">
                Across {state.efsVolumes.length} volume{state.efsVolumes.length !== 1 ? 's' : ''}
              </Box>
            </SpaceBetween>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Total Private (EBS)</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
                {state.ebsVolumes.reduce((sum, v) => sum + v.size_gb, 0)} GB
              </Box>
              <Box color="text-body-secondary">
                Across {state.ebsVolumes.length} volume{state.ebsVolumes.length !== 1 ? 's' : ''}
              </Box>
            </SpaceBetween>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Est. Monthly Cost (EFS)</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-body-secondary">
                ${state.efsVolumes.reduce((sum, v) => {
                  const sizeGB = Math.round(v.size_bytes / (1024 * 1024 * 1024));
                  return sum + (sizeGB * v.estimated_cost_gb);
                }, 0).toFixed(2)}
              </Box>
              <Box color="text-body-secondary">
                ~$0.30/GB/month average
              </Box>
            </SpaceBetween>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Est. Monthly Cost (EBS)</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-warning">
                ${state.ebsVolumes.reduce((sum, v) => sum + (v.size_gb * v.estimated_cost_gb), 0).toFixed(2)}
              </Box>
              <Box color="text-body-secondary">
                ~$0.10/GB/month average
              </Box>
            </SpaceBetween>
          </ColumnLayout>
        </Container>
      </SpaceBetween>
    );
  };

  // Placeholder views for other sections
  // Project Management View
  const ProjectManagementView = () => (
    <SpaceBetween size="l">
      <Header
        variant="h1"
        description="Manage research projects, budgets, and collaboration"
        counter={`(${state.projects.length} projects)`}
        actions={
          <SpaceBetween direction="horizontal" size="xs">
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? <Spinner /> : 'Refresh'}
            </Button>
            <Button variant="primary">
              Create Project
            </Button>
          </SpaceBetween>
        }
      >
        Project Management
      </Header>

      {/* Project Overview Stats */}
      <ColumnLayout columns={4} variant="text-grid">
        <Container header={<Header variant="h3">Total Projects</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
            {state.projects.length}
          </Box>
        </Container>
        <Container header={<Header variant="h3">Active Projects</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
            {state.projects.filter(p => p.status === 'active').length}
          </Box>
        </Container>
        <Container header={<Header variant="h3">Total Budget</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-body-secondary">
            ${state.projects.reduce((sum, p) => sum + (p.budget_limit || 0), 0).toFixed(2)}
          </Box>
        </Container>
        <Container header={<Header variant="h3">Current Spend</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-status-warning">
            ${state.projects.reduce((sum, p) => sum + (p.current_spend || 0), 0).toFixed(2)}
          </Box>
        </Container>
      </ColumnLayout>

      {/* Projects Table */}
      <Container
        header={
          <Header
            variant="h2"
            description="Research projects with budget tracking and member management"
            counter={`(${state.projects.length})`}
            actions={
              <SpaceBetween direction="horizontal" size="xs">
                <Button>Export Data</Button>
                <Button variant="primary">Create Project</Button>
              </SpaceBetween>
            }
          >
            Projects
          </Header>
        }
      >
        <Table
          columnDefinitions={[
            {
              id: "name",
              header: "Project Name",
              cell: (item: Project) => (
                <Link
                  fontSize="body-m"
                  onFollow={() => {
                    setState(prev => ({
                      ...prev,
                      selectedProject: item,
                      activeView: 'project-detail'
                    }));
                  }}
                >
                  {item.name}
                </Link>
              ),
              sortingField: "name"
            },
            {
              id: "description",
              header: "Description",
              cell: (item: Project) => item.description || 'No description',
              sortingField: "description"
            },
            {
              id: "owner",
              header: "Owner",
              cell: (item: Project) => item.owner_email || 'Unknown',
              sortingField: "owner_email"
            },
            {
              id: "budget",
              header: "Budget",
              cell: (item: Project) => `$${(item.budget_limit || 0).toFixed(2)}`,
              sortingField: "budget_limit"
            },
            {
              id: "spend",
              header: "Current Spend",
              cell: (item: Project) => {
                const spend = item.current_spend || 0;
                const limit = item.budget_limit || 0;
                const percentage = limit > 0 ? (spend / limit) * 100 : 0;
                const colorType = percentage > 80 ? 'error' : percentage > 60 ? 'warning' : 'success';

                return (
                  <SpaceBetween direction="horizontal" size="xs">
                    <StatusIndicator
                      type={colorType}
                      ariaLabel={getStatusLabel('budget', colorType === 'error' ? 'critical' : colorType === 'warning' ? 'warning' : 'ok', `$${spend.toFixed(2)}`)}
                    >
                      ${spend.toFixed(2)}
                    </StatusIndicator>
                    {limit > 0 && (
                      <Badge color={colorType === 'error' ? 'red' : colorType === 'warning' ? 'blue' : 'green'}>
                        {percentage.toFixed(1)}%
                      </Badge>
                    )}
                  </SpaceBetween>
                );
              }
            },
            {
              id: "members",
              header: "Members",
              cell: (item: Project) => item.member_count || 1,
              sortingField: "member_count"
            },
            {
              id: "status",
              header: "Status",
              cell: (item: Project) => (
                <StatusIndicator
                  type={
                    item.status === 'active' ? 'success' :
                    item.status === 'suspended' ? 'warning' : 'error'
                  }
                  ariaLabel={getStatusLabel('project', item.status || 'active')}
                >
                  {item.status || 'active'}
                </StatusIndicator>
              ),
              sortingField: "status"
            },
            {
              id: "created",
              header: "Created",
              cell: (item: Project) => new Date(item.created_at).toLocaleDateString(),
              sortingField: "created_at"
            },
            {
              id: "actions",
              header: "Actions",
              cell: (item: Project) => (
                <ButtonDropdown
                  expandToViewport
                  items={[
                    { text: "View Details", id: "view" },
                    { text: "Manage Members", id: "members" },
                    { text: "Budget Analysis", id: "budget" },
                    { text: "Cost Report", id: "costs" },
                    { text: "Usage Statistics", id: "usage" },
                    { text: "Edit Project", id: "edit" },
                    { text: "Suspend", id: "suspend", disabled: item.status === 'suspended' },
                    { text: "Delete", id: "delete" }
                  ]}
                  onItemClick={(detail) => {
                    setState(prev => ({
                      ...prev,
                      notifications: [
                        {
                          type: 'info',
                          header: 'Project Action',
                          content: `${detail.detail.text} for project "${item.name}" - Feature coming soon!`,
                          dismissible: true,
                          id: Date.now().toString()
                        },
                        ...prev.notifications
                      ]
                    }));
                  }}
                >
                  Actions
                </ButtonDropdown>
              )
            }
          ]}
          items={state.projects}
          loadingText="Loading projects..."
          empty={
            <Box textAlign="center" color="text-body-secondary">
              <Box variant="strong" textAlign="center" color="text-body-secondary">
                No projects found
              </Box>
              <Box variant="p" padding={{ bottom: 's' }} color="text-body-secondary">
                Create your first research project to get started.
              </Box>
              <Button variant="primary">Create Project</Button>
            </Box>
          }
          header={
            <Header
              counter={`(${state.projects.length})`}
              description="Research projects with comprehensive budget and collaboration management"
            >
              All Projects
            </Header>
          }
          pagination={<Box>Showing all {state.projects.length} projects</Box>}
        />
      </Container>

      {/* Quick Stats and Analytics */}
      <Container
        header={
          <Header
            variant="h2"
            description="Project analytics and budget insights"
          >
            Project Analytics
          </Header>
        }
      >
        <ColumnLayout columns={2}>
          <SpaceBetween size="m">
            <Header variant="h3">Budget Utilization</Header>
            {state.projects.length > 0 ? (
              state.projects.map((project) => {
                const spend = project.current_spend || 0;
                const limit = project.budget_limit || 0;
                const percentage = limit > 0 ? (spend / limit) * 100 : 0;

                return (
                  <Box key={project.id}>
                    <SpaceBetween direction="horizontal" size="s">
                      <Box fontWeight="bold">{project.name}:</Box>
                      <StatusIndicator
                        type={percentage > 80 ? 'error' : percentage > 60 ? 'warning' : 'success'}
                        ariaLabel={getStatusLabel('budget', percentage > 80 ? 'critical' : percentage > 60 ? 'warning' : 'ok', `${percentage.toFixed(1)}% used`)}
                      >
                        ${spend.toFixed(2)} / ${limit.toFixed(2)} ({percentage.toFixed(1)}%)
                      </StatusIndicator>
                    </SpaceBetween>
                  </Box>
                );
              })
            ) : (
              <Box color="text-body-secondary">No projects to display</Box>
            )}
          </SpaceBetween>

          <SpaceBetween size="m">
            <Header variant="h3">Recent Activity</Header>
            <Box color="text-body-secondary">
              Project activity and cost tracking metrics will be displayed here.
              Connect projects to instances and storage for detailed analytics.
            </Box>
          </SpaceBetween>
        </ColumnLayout>
      </Container>
    </SpaceBetween>
  );

  // User Management View
  const UserManagementView = () => (
    <SpaceBetween size="l">
      <Header
        variant="h1"
        description="Manage research users with persistent identity across Prism workspaces"
        counter={`(${state.users.length} users)`}
        actions={
          <SpaceBetween direction="horizontal" size="xs">
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? <Spinner /> : 'Refresh'}
            </Button>
            <Button variant="primary">
              Create User
            </Button>
          </SpaceBetween>
        }
      >
        User Management
      </Header>

      {/* User Overview Stats */}
      <ColumnLayout columns={4} variant="text-grid">
        <Container header={<Header variant="h3">Total Users</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
            {state.users.length}
          </Box>
        </Container>
        <Container header={<Header variant="h3">Active Users</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
            {state.users.filter(u => u.status !== 'inactive').length}
          </Box>
        </Container>
        <Container header={<Header variant="h3">SSH Keys Generated</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-body-secondary">
            {state.users.reduce((sum, u) => sum + (u.ssh_keys || 0), 0)}
          </Box>
        </Container>
        <Container header={<Header variant="h3">Provisioned Workspaces</Header>}>
          <Box fontSize="display-l" fontWeight="bold" color="text-status-warning">
            {state.users.reduce((sum, u) => sum + (u.provisioned_instances?.length || 0), 0)}
          </Box>
        </Container>
      </ColumnLayout>

      {/* Users Table */}
      <Container
        header={
          <Header
            variant="h2"
            description="Research users with persistent identity and SSH key management"
            counter={`(${state.users.length})`}
            actions={
              <SpaceBetween direction="horizontal" size="xs">
                <Button>Export Users</Button>
                <Button variant="primary">Create User</Button>
              </SpaceBetween>
            }
          >
            Research Users
          </Header>
        }
      >
        <Table
          columnDefinitions={[
            {
              id: "username",
              header: "Username",
              cell: (item: User) => <Link fontSize="body-m">{item.username}</Link>,
              sortingField: "username"
            },
            {
              id: "uid",
              header: "UID",
              cell: (item: User) => item.uid.toString(),
              sortingField: "uid"
            },
            {
              id: "full_name",
              header: "Full Name",
              cell: (item: User) => item.full_name || 'Not provided',
              sortingField: "full_name"
            },
            {
              id: "email",
              header: "Email",
              cell: (item: User) => item.email || 'Not provided',
              sortingField: "email"
            },
            {
              id: "ssh_keys",
              header: "SSH Keys",
              cell: (item: User) => {
                const keyCount = item.ssh_keys || 0;
                return (
                  <SpaceBetween direction="horizontal" size="xs">
                    <StatusIndicator
                      type={keyCount > 0 ? 'success' : 'warning'}
                      ariaLabel={keyCount > 0 ? `User has ${keyCount} SSH keys` : 'User has no SSH keys'}
                    >
                      {keyCount}
                    </StatusIndicator>
                    {keyCount === 0 && (
                      <Badge color="grey">No keys</Badge>
                    )}
                  </SpaceBetween>
                );
              }
            },
            {
              id: "instances",
              header: "Workspaces",
              cell: (item: User) => {
                const count = item.provisioned_instances?.length || 0;
                return count > 0 ? count.toString() : 'None';
              }
            },
            {
              id: "status",
              header: "Status",
              cell: (item: User) => (
                <StatusIndicator
                  type={
                    item.status === 'active' ? 'success' :
                    item.status === 'inactive' ? 'warning' : 'success'
                  }
                  ariaLabel={getStatusLabel('user', item.status || 'active')}
                >
                  {item.status || 'active'}
                </StatusIndicator>
              ),
              sortingField: "status"
            },
            {
              id: "created",
              header: "Created",
              cell: (item: User) => new Date(item.created_at).toLocaleDateString(),
              sortingField: "created_at"
            },
            {
              id: "actions",
              header: "Actions",
              cell: (item: User) => (
                <ButtonDropdown
                  expandToViewport
                  items={[
                    { text: "View Details", id: "view" },
                    { text: "Generate SSH Key", id: "ssh-key", disabled: (item.ssh_keys || 0) > 0 },
                    { text: "Provision on Workspace", id: "provision" },
                    { text: "User Status", id: "status" },
                    { text: "Edit User", id: "edit" },
                    { text: "Delete User", id: "delete" }
                  ]}
                  onItemClick={(detail) => {
                    setState(prev => ({
                      ...prev,
                      notifications: [
                        {
                          type: 'info',
                          header: 'User Action',
                          content: `${detail.detail.text} for user "${item.username}" - Feature coming soon!`,
                          dismissible: true,
                          id: Date.now().toString()
                        },
                        ...prev.notifications
                      ]
                    }));
                  }}
                >
                  Actions
                </ButtonDropdown>
              )
            }
          ]}
          items={state.users}
          loadingText="Loading users..."
          empty={
            <Box textAlign="center" color="text-body-secondary">
              <Box variant="strong" textAlign="center" color="text-body-secondary">
                No users found
              </Box>
              <Box variant="p" padding={{ bottom: 's' }} color="text-body-secondary">
                Create your first research user to enable persistent identity across workspaces.
              </Box>
              <Button variant="primary">Create User</Button>
            </Box>
          }
          header={
            <Header
              counter={`(${state.users.length})`}
              description="Research users with persistent UID/GID mapping and SSH key management"
            >
              All Users
            </Header>
          }
          pagination={<Box>Showing all {state.users.length} users</Box>}
        />
      </Container>

      {/* User Analytics and SSH Key Management */}
      <Container
        header={
          <Header
            variant="h2"
            description="User analytics and SSH key management"
          >
            User Analytics
          </Header>
        }
      >
        <ColumnLayout columns={2}>
          <SpaceBetween size="m">
            <Header variant="h3">SSH Key Status</Header>
            {state.users.length > 0 ? (
              state.users.map((user) => {
                const keyCount = user.ssh_keys || 0;
                return (
                  <Box key={user.username}>
                    <SpaceBetween direction="horizontal" size="s">
                      <Box fontWeight="bold">{user.username}:</Box>
                      <StatusIndicator
                        type={keyCount > 0 ? 'success' : 'warning'}
                        ariaLabel={getStatusLabel('auth', keyCount > 0 ? 'authenticated' : 'warning', `${keyCount} SSH keys`)}
                      >
                        {keyCount > 0 ? `${keyCount} SSH keys` : 'No SSH keys'}
                      </StatusIndicator>
                      {keyCount === 0 && (
                        <Button size="small" variant="link">Generate Key</Button>
                      )}
                    </SpaceBetween>
                  </Box>
                );
              })
            ) : (
              <Box color="text-body-secondary">No users to display</Box>
            )}
          </SpaceBetween>

          <SpaceBetween size="m">
            <Header variant="h3">Workspace Provisioning</Header>
            <Box color="text-body-secondary">
              User provisioning across workspaces and EFS home directory management.
              Persistent identity ensures same UID/GID mapping across all environments.
            </Box>
            {state.users.length > 0 && (
              <SpaceBetween size="s">
                <Box variant="strong">Available for Provisioning:</Box>
                {state.instances.length > 0 ? (
                  state.instances.filter(i => i.state === 'running').map(instance => (
                    <Box key={instance.id}>
                      <StatusIndicator
                        type="success"
                        ariaLabel={getStatusLabel('instance', 'running', instance.name)}
                      >
                        {instance.name}
                      </StatusIndicator>
                    </Box>
                  ))
                ) : (
                  <Box color="text-body-secondary">No running workspaces available</Box>
                )}
              </SpaceBetween>
            )}
          </SpaceBetween>
        </ColumnLayout>
      </Container>
    </SpaceBetween>
  );

  // Project Detail View with Integrated Budget
  const ProjectDetailView = () => {
    if (!state.selectedProject) {
      // Fallback if no project selected
      return (
        <SpaceBetween size="l">
          <Alert type="warning" header="No project selected">
            Please select a project from the Projects page to view details.
          </Alert>
          <Button onClick={() => setState(prev => ({ ...prev, activeView: 'projects' }))}>
            Back to Projects
          </Button>
        </SpaceBetween>
      );
    }

    const project = state.selectedProject;
    const [activeTabId, setActiveTabId] = React.useState('overview');

    // Get budget data for this project
    const projectBudget = state.budgets.find(b => b.project_id === project.id);

    // Get workspaces for this project
    const projectWorkspaces = state.instances.filter(i =>
      i.project === project.name || i.project === project.id
    );

    return (
      <SpaceBetween size="l">
        {/* Project Header */}
        <Header
          variant="h1"
          description={project.description || 'No description provided'}
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button onClick={() => setState(prev => ({ ...prev, activeView: 'projects', selectedProject: null }))}>
                Back to Projects
              </Button>
              <Button>Edit Project</Button>
              <Button variant="primary">Configure Budget</Button>
            </SpaceBetween>
          }
        >
          {project.name}
        </Header>

        {/* Quick Stats */}
        <ColumnLayout columns={4} variant="text-grid">
          <Container>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Status</Box>
              <StatusIndicator
                type={project.status === 'active' ? 'success' : project.status === 'suspended' ? 'warning' : 'error'}
              >
                {project.status || 'active'}
              </StatusIndicator>
            </SpaceBetween>
          </Container>
          <Container>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Workspaces</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
                {projectWorkspaces.length}
              </Box>
            </SpaceBetween>
          </Container>
          <Container>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Members</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
                {project.member_count || 1}
              </Box>
            </SpaceBetween>
          </Container>
          <Container>
            <SpaceBetween size="s">
              <Box variant="awsui-key-label">Owner</Box>
              <Box fontSize="heading-m">{project.owner_email || 'Unknown'}</Box>
            </SpaceBetween>
          </Container>
        </ColumnLayout>

        {/* Tabbed Interface */}
        <Tabs
          activeTabId={activeTabId}
          onChange={({ detail }) => setActiveTabId(detail.activeTabId)}
          tabs={[
            {
              id: 'overview',
              label: 'Overview',
              content: (
                <SpaceBetween size="l">
                  {/* Project Details */}
                  <Container header={<Header variant="h2">Project Details</Header>}>
                    <ColumnLayout columns={2} variant="text-grid">
                      <SpaceBetween size="m">
                        <div>
                          <Box variant="awsui-key-label">Project ID</Box>
                          <Box>{project.id}</Box>
                        </div>
                        <div>
                          <Box variant="awsui-key-label">Created</Box>
                          <Box>{new Date(project.created_at).toLocaleString()}</Box>
                        </div>
                        <div>
                          <Box variant="awsui-key-label">Last Updated</Box>
                          <Box>{new Date(project.updated_at).toLocaleString()}</Box>
                        </div>
                      </SpaceBetween>
                      <SpaceBetween size="m">
                        <div>
                          <Box variant="awsui-key-label">Owner</Box>
                          <Box>{project.owner_email || 'Unknown'}</Box>
                        </div>
                        <div>
                          <Box variant="awsui-key-label">Members</Box>
                          <Box>{project.member_count || 1} member{(project.member_count || 1) !== 1 ? 's' : ''}</Box>
                        </div>
                        <div>
                          <Box variant="awsui-key-label">Status</Box>
                          <StatusIndicator type={project.status === 'active' ? 'success' : 'warning'}>
                            {project.status || 'active'}
                          </StatusIndicator>
                        </div>
                      </SpaceBetween>
                    </ColumnLayout>
                  </Container>

                  {/* Project Workspaces */}
                  <Container
                    header={
                      <Header
                        variant="h2"
                        counter={`(${projectWorkspaces.length})`}
                        description="Workspaces associated with this project"
                      >
                        Project Workspaces
                      </Header>
                    }
                  >
                    {projectWorkspaces.length > 0 ? (
                      <Table
                        columnDefinitions={[
                          {
                            id: 'name',
                            header: 'Workspace Name',
                            cell: (item: Instance) => item.name
                          },
                          {
                            id: 'template',
                            header: 'Template',
                            cell: (item: Instance) => item.template
                          },
                          {
                            id: 'state',
                            header: 'State',
                            cell: (item: Instance) => (
                              <StatusIndicator
                                type={item.state === 'running' ? 'success' : 'stopped'}
                              >
                                {item.state}
                              </StatusIndicator>
                            )
                          },
                          {
                            id: 'type',
                            header: 'Type',
                            cell: (item: Instance) => item.instance_type || 'Unknown'
                          }
                        ]}
                        items={projectWorkspaces}
                        empty={
                          <Box textAlign="center" color="inherit">
                            <Box variant="p">No workspaces in this project</Box>
                          </Box>
                        }
                      />
                    ) : (
                      <Box textAlign="center" padding={{ vertical: 'xl' }}>
                        <SpaceBetween size="m">
                          <Box variant="strong">No workspaces yet</Box>
                          <Box color="text-body-secondary">
                            Launch workspaces and assign them to this project
                          </Box>
                          <Button variant="primary">Launch Workspace</Button>
                        </SpaceBetween>
                      </Box>
                    )}
                  </Container>
                </SpaceBetween>
              )
            },
            {
              id: 'budget',
              label: 'Budget & Costs',
              content: (
                <SpaceBetween size="l">
                  {/* Budget Overview */}
                  <ColumnLayout columns={4} variant="text-grid">
                    <Container>
                      <SpaceBetween size="s">
                        <Box variant="awsui-key-label">Budget Limit</Box>
                        <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
                          ${(project.budget_limit || 0).toFixed(2)}
                        </Box>
                      </SpaceBetween>
                    </Container>
                    <Container>
                      <SpaceBetween size="s">
                        <Box variant="awsui-key-label">Current Spend</Box>
                        <Box
                          fontSize="display-l"
                          fontWeight="bold"
                          color={
                            (project.budget_limit || 0) > 0 &&
                            ((project.current_spend || 0) / project.budget_limit) > 0.8
                              ? 'text-status-error'
                              : 'text-status-success'
                          }
                        >
                          ${(project.current_spend || 0).toFixed(2)}
                        </Box>
                      </SpaceBetween>
                    </Container>
                    <Container>
                      <SpaceBetween size="s">
                        <Box variant="awsui-key-label">Remaining</Box>
                        <Box fontSize="display-l" fontWeight="bold" color="text-status-warning">
                          ${Math.max(0, (project.budget_limit || 0) - (project.current_spend || 0)).toFixed(2)}
                        </Box>
                      </SpaceBetween>
                    </Container>
                    <Container>
                      <SpaceBetween size="s">
                        <Box variant="awsui-key-label">% Used</Box>
                        <Box fontSize="display-l" fontWeight="bold">
                          {project.budget_limit && project.budget_limit > 0
                            ? ((project.current_spend || 0) / project.budget_limit * 100).toFixed(1)
                            : 0}%
                        </Box>
                      </SpaceBetween>
                    </Container>
                  </ColumnLayout>

                  {/* Budget Status Alert */}
                  {project.budget_limit && project.budget_limit > 0 && (
                    (() => {
                      const percentage = ((project.current_spend || 0) / project.budget_limit) * 100;
                      if (percentage >= 95) {
                        return (
                          <Alert type="error" header="Budget Critical">
                            This project has used {percentage.toFixed(1)}% of its budget. Consider hibernating workspaces or increasing the budget limit.
                          </Alert>
                        );
                      } else if (percentage >= 80) {
                        return (
                          <Alert type="warning" header="Budget Warning">
                            This project has used {percentage.toFixed(1)}% of its budget. Monitor spending closely.
                          </Alert>
                        );
                      }
                      return null;
                    })()
                  )}

                  {/* Budget Configuration */}
                  <Container
                    header={
                      <Header
                        variant="h2"
                        description="Configure budget limits and alerts for this project"
                        actions={<Button variant="primary">Edit Budget</Button>}
                      >
                        Budget Configuration
                      </Header>
                    }
                  >
                    <SpaceBetween size="m">
                      {project.budget_limit && project.budget_limit > 0 ? (
                        <>
                          <ColumnLayout columns={2}>
                            <div>
                              <Box variant="awsui-key-label">Monthly Budget Limit</Box>
                              <Box fontSize="heading-l">${project.budget_limit.toFixed(2)}</Box>
                            </div>
                            <div>
                              <Box variant="awsui-key-label">Budget Period</Box>
                              <Box fontSize="heading-m">Monthly (resets 1st of month)</Box>
                            </div>
                          </ColumnLayout>
                          <Box variant="h4">Alert Thresholds</Box>
                          <ColumnLayout columns={3}>
                            <div>
                              <Box color="text-body-secondary">50% Warning</Box>
                              <Box>${(project.budget_limit * 0.5).toFixed(2)}</Box>
                            </div>
                            <div>
                              <Box color="text-body-secondary">80% Alert</Box>
                              <Box>${(project.budget_limit * 0.8).toFixed(2)}</Box>
                            </div>
                            <div>
                              <Box color="text-body-secondary">100% Critical</Box>
                              <Box>${project.budget_limit.toFixed(2)}</Box>
                            </div>
                          </ColumnLayout>
                        </>
                      ) : (
                        <Box textAlign="center" padding={{ vertical: 'xl' }}>
                          <SpaceBetween size="m">
                            <Box variant="strong">No budget configured</Box>
                            <Box color="text-body-secondary">
                              Set a budget limit to track spending and receive alerts
                            </Box>
                            <Button variant="primary">Configure Budget</Button>
                          </SpaceBetween>
                        </Box>
                      )}
                    </SpaceBetween>
                  </Container>

                  {/* Per-Workspace Cost Breakdown */}
                  <Container
                    header={
                      <Header
                        variant="h2"
                        description="Cost breakdown by workspace"
                        counter={`(${projectWorkspaces.length} workspaces)`}
                      >
                        Workspace Costs
                      </Header>
                    }
                  >
                    {projectWorkspaces.length > 0 ? (
                      <Table
                        columnDefinitions={[
                          {
                            id: 'name',
                            header: 'Workspace',
                            cell: (item: Instance) => item.name
                          },
                          {
                            id: 'state',
                            header: 'State',
                            cell: (item: Instance) => (
                              <StatusIndicator type={item.state === 'running' ? 'success' : 'stopped'}>
                                {item.state}
                              </StatusIndicator>
                            )
                          },
                          {
                            id: 'cost',
                            header: 'Accumulated Cost',
                            cell: (item: Instance) => {
                              // Mock cost calculation - in real implementation, fetch from API
                              const mockCost = Math.random() * 50;
                              return `$${mockCost.toFixed(2)}`;
                            }
                          },
                          {
                            id: 'rate',
                            header: 'Hourly Rate',
                            cell: (item: Instance) => {
                              // Mock rate - in real implementation, fetch from API
                              return '$0.85/hr';
                            }
                          },
                          {
                            id: 'runtime',
                            header: 'Runtime',
                            cell: (item: Instance) => {
                              if (item.launch_time) {
                                const hours = Math.floor(
                                  (Date.now() - new Date(item.launch_time).getTime()) / (1000 * 60 * 60)
                                );
                                return `${hours}h`;
                              }
                              return 'N/A';
                            }
                          }
                        ]}
                        items={projectWorkspaces}
                      />
                    ) : (
                      <Box textAlign="center" padding={{ vertical: 'xl' }} color="inherit">
                        <Box variant="p">No workspace costs to display</Box>
                      </Box>
                    )}
                  </Container>

                  {/* Cost Optimization Recommendations */}
                  {projectWorkspaces.some(i => i.state === 'running') && (
                    <Alert type="info" header="💡 Cost Optimization Tips">
                      <ul style={{ marginTop: '8px', paddingLeft: '20px' }}>
                        <li>Hibernate idle workspaces to reduce costs while preserving state</li>
                        <li>Use spot workspaces for non-critical workloads (up to 90% savings)</li>
                        <li>Configure auto-hibernation policies for unused workspaces</li>
                        <li>Right-size workspaces based on actual usage patterns</li>
                      </ul>
                    </Alert>
                  )}
                </SpaceBetween>
              )
            },
            {
              id: 'members',
              label: `Members (${project.member_count || 1})`,
              content: (
                <Container
                  header={
                    <Header
                      variant="h2"
                      description="Manage project members and their permissions"
                      counter={`(${project.member_count || 1} members)`}
                      actions={<Button variant="primary">Add Member</Button>}
                    >
                      Project Members
                    </Header>
                  }
                >
                  <Box textAlign="center" padding={{ vertical: 'xl' }}>
                    <SpaceBetween size="m">
                      <Box variant="strong">Member management coming soon</Box>
                      <Box color="text-body-secondary">
                        View and manage project members, assign roles, and configure permissions
                      </Box>
                    </SpaceBetween>
                  </Box>
                </Container>
              )
            }
          ]}
        />
      </SpaceBetween>
    );
  };

  // Settings View
  const SettingsView = () => (
    <SpaceBetween size="l">
      <Header
        variant="h1"
        description="Configure Prism preferences and system settings"
        actions={
          <SpaceBetween direction="horizontal" size="xs">
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? <Spinner /> : 'Refresh'}
            </Button>
            <Button variant="primary">
              Save Settings
            </Button>
          </SpaceBetween>
        }
      >
        Settings
      </Header>

      {/* System Status Section */}
      <Container
        header={
          <Header
            variant="h2"
            description="System status and daemon configuration"
          >
            System Status
          </Header>
        }
      >
        <ColumnLayout columns={3} variant="text-grid">
          <SpaceBetween size="m">
            <Box variant="awsui-key-label">Daemon Status</Box>
            <StatusIndicator
              type={state.connected ? 'success' : 'error'}
              ariaLabel={getStatusLabel('connection', state.connected ? 'success' : 'error')}
            >
              {state.connected ? 'Connected' : 'Disconnected'}
            </StatusIndicator>
            <Box color="text-body-secondary">
              Prism daemon on port 8947
            </Box>
          </SpaceBetween>
          <SpaceBetween size="m">
            <Box variant="awsui-key-label">API Version</Box>
            <Box fontSize="heading-m">v0.5.1</Box>
            <Box color="text-body-secondary">
              Current Prism version
            </Box>
          </SpaceBetween>
          <SpaceBetween size="m">
            <Box variant="awsui-key-label">Active Resources</Box>
            <Box fontSize="heading-m">{state.instances.length + state.efsVolumes.length + state.ebsVolumes.length}</Box>
            <Box color="text-body-secondary">
              Workspaces, EFS and EBS volumes
            </Box>
          </SpaceBetween>
        </ColumnLayout>
      </Container>

      {/* Configuration Section */}
      <Container
        header={
          <Header
            variant="h2"
            description="Prism configuration and preferences"
          >
            Configuration
          </Header>
        }
      >
        <SpaceBetween size="l">
          <FormField
            label="Auto-refresh interval"
            description="How often the GUI should refresh data from the daemon"
          >
            <Select
              selectedOption={{ label: "30 seconds", value: "30" }}
              onChange={() => {}}
              options={[
                { label: "15 seconds", value: "15" },
                { label: "30 seconds", value: "30" },
                { label: "1 minute", value: "60" },
                { label: "2 minutes", value: "120" }
              ]}
            />
          </FormField>

          <FormField
            label="Default workspace size"
            description="Default size for new workspaces when launching templates"
          >
            <Select
              selectedOption={{ label: "Medium (M)", value: "M" }}
              onChange={() => {}}
              options={[
                { label: "Small (S)", value: "S" },
                { label: "Medium (M)", value: "M" },
                { label: "Large (L)", value: "L" },
                { label: "Extra Large (XL)", value: "XL" }
              ]}
            />
          </FormField>

          <FormField
            label="Show advanced features"
            description="Display advanced management options like hibernation policies and cost tracking"
          >
            <Select
              selectedOption={{ label: "Enabled", value: "enabled" }}
              onChange={() => {}}
              options={[
                { label: "Enabled", value: "enabled" },
                { label: "Disabled", value: "disabled" }
              ]}
            />
          </FormField>
        </SpaceBetween>
      </Container>

      {/* Profile and Authentication Section */}
      <Container
        header={
          <Header
            variant="h2"
            description="AWS profile and authentication settings"
          >
            AWS Configuration
          </Header>
        }
      >
        <ColumnLayout columns={2}>
          <SpaceBetween size="m">
            <FormField
              label="AWS Profile"
              description="Current AWS profile for authentication"
            >
              <Input
                value="aws"
                readOnly
                placeholder="AWS profile name"
              />
            </FormField>
            <FormField
              label="AWS Region"
              description="Current AWS region for resources"
            >
              <Input
                value="us-west-2"
                readOnly
                placeholder="AWS region"
              />
            </FormField>
          </SpaceBetween>
          <SpaceBetween size="m">
            <Box variant="strong">Authentication Status</Box>
            <StatusIndicator
              type="success"
              ariaLabel={getStatusLabel('auth', 'authenticated')}
            >
              Authenticated via AWS profile
            </StatusIndicator>
            <Box color="text-body-secondary">
              Using credentials from AWS profile "aws" in region us-west-2.
              Prism automatically manages authentication for all API calls.
            </Box>
          </SpaceBetween>
        </ColumnLayout>
      </Container>

      {/* Feature Management */}
      <Container
        header={
          <Header
            variant="h2"
            description="Enable or disable Prism features"
          >
            Feature Management
          </Header>
        }
      >
        <SpaceBetween size="m">
          {[
            { name: "Workspace Management", status: "enabled", description: "Launch, manage, and connect to cloud workspaces" },
            { name: "Storage Management", status: "enabled", description: "EFS and EBS volume operations" },
            { name: "Project Management", status: "enabled", description: "Multi-user collaboration and budget tracking" },
            { name: "User Management", status: "enabled", description: "Research users with persistent identity" },
            { name: "Hibernation Policies", status: "enabled", description: "Automated cost optimization through hibernation" },
            { name: "Cost Tracking", status: "partial", description: "Budget monitoring and expense analysis" },
            { name: "Template Marketplace", status: "partial", description: "Community template sharing and discovery" },
            { name: "Scaling Predictions", status: "partial", description: "Resource optimization recommendations" }
          ].map((feature) => (
            <Box key={feature.name}>
              <SpaceBetween direction="horizontal" size="s">
                <Box fontWeight="bold" style={{ minWidth: '200px' }}>{feature.name}:</Box>
                <StatusIndicator
                  type={
                    feature.status === 'enabled' ? 'success' :
                    feature.status === 'partial' ? 'warning' : 'error'
                  }
                  ariaLabel={getStatusLabel('policy', feature.status, feature.name)}
                >
                  {feature.status}
                </StatusIndicator>
                <Box color="text-body-secondary">{feature.description}</Box>
              </SpaceBetween>
            </Box>
          ))}
        </SpaceBetween>
      </Container>

      {/* Debug and Troubleshooting */}
      <Container
        header={
          <Header
            variant="h2"
            description="Debug information and troubleshooting tools"
          >
            Debug & Troubleshooting
          </Header>
        }
      >
        <SpaceBetween size="m">
          <Alert type="info">
            <Box variant="strong">Debug Mode</Box>
            <Box variant="p">
              Console logging is enabled. Check browser developer tools for detailed API interactions and error messages.
            </Box>
          </Alert>

          <ColumnLayout columns={2}>
            <SpaceBetween size="s">
              <Box variant="strong">Quick Actions</Box>
              <Button>Test API Connection</Button>
              <Button>Refresh All Data</Button>
              <Button>Clear Notifications</Button>
              <Button>Export Configuration</Button>
            </SpaceBetween>
            <SpaceBetween size="s">
              <Box variant="strong">Troubleshooting</Box>
              <Button variant="link" external>View Documentation</Button>
              <Button variant="link" external>GitHub Issues</Button>
              <Button variant="link" external>Troubleshooting Guide</Button>
            </SpaceBetween>
          </ColumnLayout>
        </SpaceBetween>
      </Container>
    </SpaceBetween>
  );

  // Budget Management View
  const BudgetManagementView = () => {
    const [selectedTab, setSelectedTab] = useState<number>(0);
    const [selectedBudget, setSelectedBudget] = useState<BudgetData | null>(null);
    const [costBreakdown, setCostBreakdown] = useState<CostBreakdown | null>(null);

    // Load cost breakdown when a budget is selected
    useEffect(() => {
      if (selectedBudget && selectedTab === 1) {
        api.getCostBreakdown(selectedBudget.project_id).then(setCostBreakdown);
      }
    }, [selectedBudget, selectedTab]);

    // Calculate aggregate statistics
    const totalBudget = state.budgets.reduce((sum, b) => sum + b.total_budget, 0);
    const totalSpent = state.budgets.reduce((sum, b) => sum + b.spent_amount, 0);
    const totalRemaining = totalBudget - totalSpent;
    const overallPercent = totalBudget > 0 ? (totalSpent / totalBudget) * 100 : 0;
    const criticalCount = state.budgets.filter(b => b.status === 'critical').length;
    const warningCount = state.budgets.filter(b => b.status === 'warning').length;

    return (
      <SpaceBetween size="l">
        <Header
          variant="h1"
          description="Monitor budgets, analyze costs, and optimize spending across research projects"
          counter={`(${state.budgets.length} budgets)`}
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button onClick={loadApplicationData} disabled={state.loading}>
                {state.loading ? <Spinner /> : 'Refresh'}
              </Button>
              <Button variant="primary">
                Configure Budget
              </Button>
            </SpaceBetween>
          }
        >
          Budget Management
        </Header>

        {/* Budget Overview Stats */}
        <ColumnLayout columns={4} variant="text-grid">
          <Container header={<Header variant="h3">Total Budget</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
              ${totalBudget.toFixed(2)}
            </Box>
          </Container>
          <Container header={<Header variant="h3">Total Spent</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color={overallPercent > 80 ? 'text-status-error' : 'text-status-success'}>
              ${totalSpent.toFixed(2)}
            </Box>
            <Box variant="small" color="text-body-secondary">
              {overallPercent.toFixed(1)}% of budget
            </Box>
          </Container>
          <Container header={<Header variant="h3">Remaining</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-warning">
              ${totalRemaining.toFixed(2)}
            </Box>
          </Container>
          <Container header={<Header variant="h3">Alerts</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color={criticalCount > 0 ? 'text-status-error' : 'text-body-secondary'}>
              {criticalCount} Critical
            </Box>
            <Box variant="small" color="text-body-secondary">
              {warningCount} warnings
            </Box>
          </Container>
        </ColumnLayout>

        {/* Budget Table - Overview Tab */}
        <Container
          header={
            <Header
              variant="h2"
              description="Project budgets with spending tracking and alert monitoring"
              counter={`(${state.budgets.length})`}
              actions={
                <SpaceBetween direction="horizontal" size="xs">
                  <Button>Export Report</Button>
                  <Button variant="primary">Set Budget</Button>
                </SpaceBetween>
              }
            >
              Project Budgets
            </Header>
          }
        >
          <Table
            columnDefinitions={[
              {
                id: "project",
                header: "Project",
                cell: (item: BudgetData) => <Link fontSize="body-m" onFollow={() => setSelectedBudget(item)}>{item.project_name}</Link>,
                sortingField: "project_name"
              },
              {
                id: "budget",
                header: "Budget",
                cell: (item: BudgetData) => `$${item.total_budget.toFixed(2)}`,
                sortingField: "total_budget"
              },
              {
                id: "spent",
                header: "Spent",
                cell: (item: BudgetData) => `$${item.spent_amount.toFixed(2)}`,
                sortingField: "spent_amount"
              },
              {
                id: "remaining",
                header: "Remaining",
                cell: (item: BudgetData) => `$${item.remaining.toFixed(2)}`,
                sortingField: "remaining"
              },
              {
                id: "percentage",
                header: "% Used",
                cell: (item: BudgetData) => {
                  const percent = item.spent_percentage * 100;
                  return (
                    <SpaceBetween direction="horizontal" size="xs">
                      <StatusIndicator
                        type={
                          percent >= 95 ? 'error' :
                          percent >= 80 ? 'warning' : 'success'
                        }
                        ariaLabel={getStatusLabel('budget',
                          percent >= 95 ? 'critical' : percent >= 80 ? 'warning' : 'ok',
                          `${percent.toFixed(1)}%`)}
                      >
                        {percent.toFixed(1)}%
                      </StatusIndicator>
                    </SpaceBetween>
                  );
                }
              },
              {
                id: "status",
                header: "Status",
                cell: (item: BudgetData) => (
                  <StatusIndicator
                    type={
                      item.status === 'critical' ? 'error' :
                      item.status === 'warning' ? 'warning' : 'success'
                    }
                    ariaLabel={getStatusLabel('budget', item.status)}
                  >
                    {item.status === 'ok' ? 'OK' : item.status.toUpperCase()}
                  </StatusIndicator>
                )
              },
              {
                id: "alerts",
                header: "Alerts",
                cell: (item: BudgetData) => {
                  if (item.alert_count > 0) {
                    return (
                      <Badge color="red">{item.alert_count} active</Badge>
                    );
                  }
                  return <Box color="text-body-secondary">None</Box>;
                }
              },
              {
                id: "actions",
                header: "Actions",
                cell: (item: BudgetData) => (
                  <ButtonDropdown
                    expandToViewport
                    items={[
                      { text: "View Breakdown", id: "breakdown" },
                      { text: "View Forecast", id: "forecast" },
                      { text: "Cost Analysis", id: "costs" },
                      { text: "Configure Alerts", id: "alerts" },
                      { text: "Edit Budget", id: "edit" },
                    ]}
                    onItemClick={({ detail }) => {
                      setSelectedBudget(item);
                      if (detail.id === 'breakdown') {
                        setSelectedTab(1);
                      } else if (detail.id === 'forecast') {
                        setSelectedTab(2);
                      }
                    }}
                  >
                    Actions
                  </ButtonDropdown>
                )
              }
            ]}
            items={state.budgets}
            loadingText="Loading budgets..."
            loading={state.loading}
            trackBy="project_id"
            empty={
              <Box textAlign="center" color="text-body-secondary">
                <Box variant="strong" textAlign="center" color="text-body-secondary">
                  No budgets configured
                </Box>
                <Box variant="p" padding={{ bottom: 's' }} color="text-body-secondary">
                  Configure budgets for your projects to track spending and set alerts.
                </Box>
                <Button variant="primary">Configure Budget</Button>
              </Box>
            }
            sortingDisabled={false}
          />
        </Container>

        {/* Cost Breakdown View - when budget is selected */}
        {selectedBudget && selectedTab === 1 && (
          <Container
            header={
              <Header
                variant="h2"
                description={`Detailed cost breakdown for ${selectedBudget.project_name}`}
                actions={
                  <Button onClick={() => { setSelectedBudget(null); setSelectedTab(0); }}>
                    Back to Overview
                  </Button>
                }
              >
                Cost Breakdown
              </Header>
            }
          >
            <SpaceBetween size="m">
              <ColumnLayout columns={3} variant="text-grid">
                <Box>
                  <Box variant="awsui-key-label">Total Spent</Box>
                  <Box fontSize="heading-l" fontWeight="bold">
                    ${selectedBudget.spent_amount.toFixed(2)}
                  </Box>
                </Box>
                <Box>
                  <Box variant="awsui-key-label">Total Budget</Box>
                  <Box fontSize="heading-l" fontWeight="bold">
                    ${selectedBudget.total_budget.toFixed(2)}
                  </Box>
                </Box>
                <Box>
                  <Box variant="awsui-key-label">Remaining</Box>
                  <Box fontSize="heading-l" fontWeight="bold" color="text-status-warning">
                    ${selectedBudget.remaining.toFixed(2)}
                  </Box>
                </Box>
              </ColumnLayout>

              {costBreakdown ? (
                <>
                  <Header variant="h3">Cost by Service</Header>
                  <ColumnLayout columns={2}>
                    <SpaceBetween size="s">
                      <Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold" style={{ minWidth: '150px' }}>EC2 Compute:</Box>
                          <Box>${costBreakdown.ec2_compute.toFixed(2)}</Box>
                        </SpaceBetween>
                      </Box>
                      <Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold" style={{ minWidth: '150px' }}>EBS Storage:</Box>
                          <Box>${costBreakdown.ebs_storage.toFixed(2)}</Box>
                        </SpaceBetween>
                      </Box>
                      <Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold" style={{ minWidth: '150px' }}>EFS Storage:</Box>
                          <Box>${costBreakdown.efs_storage.toFixed(2)}</Box>
                        </SpaceBetween>
                      </Box>
                    </SpaceBetween>
                    <SpaceBetween size="s">
                      <Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold" style={{ minWidth: '150px' }}>Data Transfer:</Box>
                          <Box>${costBreakdown.data_transfer.toFixed(2)}</Box>
                        </SpaceBetween>
                      </Box>
                      <Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold" style={{ minWidth: '150px' }}>Other:</Box>
                          <Box>${costBreakdown.other.toFixed(2)}</Box>
                        </SpaceBetween>
                      </Box>
                      <Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold" style={{ minWidth: '150px' }}>Total:</Box>
                          <Box fontSize="heading-m" fontWeight="bold">${costBreakdown.total.toFixed(2)}</Box>
                        </SpaceBetween>
                      </Box>
                    </SpaceBetween>
                  </ColumnLayout>
                </>
              ) : (
                <Box textAlign="center" padding="l">
                  <Spinner size="large" />
                  <Box variant="p" color="text-body-secondary">Loading cost breakdown...</Box>
                </Box>
              )}
            </SpaceBetween>
          </Container>
        )}

        {/* Forecast View - when budget is selected */}
        {selectedBudget && selectedTab === 2 && (
          <Container
            header={
              <Header
                variant="h2"
                description={`Spending forecast and projections for ${selectedBudget.project_name}`}
                actions={
                  <Button onClick={() => { setSelectedBudget(null); setSelectedTab(0); }}>
                    Back to Overview
                  </Button>
                }
              >
                Spending Forecast
              </Header>
            }
          >
            <SpaceBetween size="m">
              <ColumnLayout columns={3} variant="text-grid">
                <Box>
                  <Box variant="awsui-key-label">Current Spending</Box>
                  <Box fontSize="heading-l" fontWeight="bold">
                    ${selectedBudget.spent_amount.toFixed(2)}
                  </Box>
                  <Box variant="small" color="text-body-secondary">
                    {(selectedBudget.spent_percentage * 100).toFixed(1)}% of budget
                  </Box>
                </Box>
                {selectedBudget.projected_monthly_spend && (
                  <Box>
                    <Box variant="awsui-key-label">Projected Monthly</Box>
                    <Box fontSize="heading-l" fontWeight="bold" color="text-status-warning">
                      ${selectedBudget.projected_monthly_spend.toFixed(2)}
                    </Box>
                  </Box>
                )}
                {selectedBudget.days_until_exhausted && (
                  <Box>
                    <Box variant="awsui-key-label">Budget Exhaustion</Box>
                    <Box fontSize="heading-l" fontWeight="bold" color="text-status-error">
                      {selectedBudget.days_until_exhausted} days
                    </Box>
                  </Box>
                )}
              </ColumnLayout>

              {selectedBudget.projected_monthly_spend && selectedBudget.days_until_exhausted && (
                <Alert type="warning">
                  <Box variant="strong">Budget Alert</Box>
                  <Box>
                    At current spending rate (${selectedBudget.projected_monthly_spend.toFixed(2)}/month),
                    your budget will be exhausted in approximately {selectedBudget.days_until_exhausted} days.
                    Consider implementing cost optimization measures or adjusting your budget.
                  </Box>
                </Alert>
              )}
            </SpaceBetween>
          </Container>
        )}

        {/* Active Alerts */}
        {state.budgets.some(b => b.alert_count > 0) && (
          <Container
            header={
              <Header
                variant="h2"
                description="Active budget alerts requiring attention"
              >
                Active Alerts
              </Header>
            }
          >
            <SpaceBetween size="m">
              {state.budgets.filter(b => b.alert_count > 0).map(budget => (
                <Alert key={budget.project_id} type="warning">
                  <Box variant="strong">{budget.project_name}</Box>
                  <Box>
                    Budget usage: {(budget.spent_percentage * 100).toFixed(1)}%
                    (${budget.spent_amount.toFixed(2)} of ${budget.total_budget.toFixed(2)})
                  </Box>
                  {budget.active_alerts && budget.active_alerts.length > 0 && (
                    <Box variant="small" color="text-body-secondary">
                      {budget.active_alerts.length} active alert(s)
                    </Box>
                  )}
                </Alert>
              ))}
            </SpaceBetween>
          </Container>
        )}
      </SpaceBetween>
    );
  };

  // AMI Management View
  const AMIManagementView = () => {
    const [selectedTab, setSelectedTab] = useState<'amis' | 'builds' | 'regions'>('amis');
    const [selectedAMI, setSelectedAMI] = useState<AMI | null>(null);
    const [deleteModalVisible, setDeleteModalVisible] = useState(false);
    const [buildModalVisible, setBuildModalVisible] = useState(false);

    const totalSize = state.amis.reduce((sum, ami) => sum + ami.size_gb, 0);
    const monthlyCost = totalSize * 0.05; // $0.05 per GB-month

    const handleDeleteAMI = async () => {
      if (!selectedAMI) return;

      try {
        await api.deleteAMI(selectedAMI.id);
        setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'success', content: `AMI ${selectedAMI.id} deleted successfully` }] }));
        setDeleteModalVisible(false);
        setSelectedAMI(null);
        await loadApplicationData();
      } catch (error) {
        setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'error', content: `Failed to delete AMI: ${error}` }] }));
      }
    };

    return (
      <SpaceBetween size="l">
        <Header
          variant="h1"
          description="Manage AMIs for fast workspace launching (30 seconds vs 5-8 minutes)"
          counter={`(${state.amis.length} AMIs)`}
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button onClick={loadApplicationData} disabled={state.loading}>
                {state.loading ? <Spinner /> : 'Refresh'}
              </Button>
              <Button variant="primary" onClick={() => setBuildModalVisible(true)}>
                Build AMI
              </Button>
            </SpaceBetween>
          }
        >
          AMI Management
        </Header>

        {/* Stats Overview */}
        <ColumnLayout columns={4} variant="text-grid">
          <Container header={<Header variant="h3">Total AMIs</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
              {state.amis.length}
            </Box>
          </Container>
          <Container header={<Header variant="h3">Total Size</Header>}>
            <Box fontSize="display-l" fontWeight="bold">
              {totalSize.toFixed(1)} GB
            </Box>
          </Container>
          <Container header={<Header variant="h3">Monthly Cost</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-warning">
              ${monthlyCost.toFixed(2)}
            </Box>
            <Box variant="small" color="text-body-secondary">
              Snapshot storage
            </Box>
          </Container>
          <Container header={<Header variant="h3">Regions</Header>}>
            <Box fontSize="display-l" fontWeight="bold">
              {state.amiRegions.length}
            </Box>
          </Container>
        </ColumnLayout>

        {/* Tabs */}
        <Tabs
          activeTabId={selectedTab}
          onChange={({ detail }) => setSelectedTab(detail.activeTabId as 'amis' | 'builds' | 'regions')}
          tabs={[
            {
              id: 'amis',
              label: 'AMIs',
              content: (
                <Container>
                  <Table
                    columnDefinitions={[
                      {
                        id: 'id',
                        header: 'AMI ID',
                        cell: (item: AMI) => <Link fontSize="body-m" onFollow={() => setSelectedAMI(item)}>{item.id}</Link>,
                        sortingField: 'id'
                      },
                      {
                        id: 'template',
                        header: 'Template',
                        cell: (item: AMI) => item.template_name,
                        sortingField: 'template_name'
                      },
                      {
                        id: 'region',
                        header: 'Region',
                        cell: (item: AMI) => <Badge>{item.region}</Badge>,
                        sortingField: 'region'
                      },
                      {
                        id: 'state',
                        header: 'State',
                        cell: (item: AMI) => (
                          <StatusIndicator
                            type={item.state === 'available' ? 'success' : 'pending'}
                            ariaLabel={getStatusLabel('ami', item.state)}
                          >
                            {item.state}
                          </StatusIndicator>
                        )
                      },
                      {
                        id: 'architecture',
                        header: 'Architecture',
                        cell: (item: AMI) => item.architecture
                      },
                      {
                        id: 'size',
                        header: 'Size',
                        cell: (item: AMI) => `${item.size_gb.toFixed(1)} GB`,
                        sortingField: 'size_gb'
                      },
                      {
                        id: 'created',
                        header: 'Created',
                        cell: (item: AMI) => new Date(item.created_at).toLocaleDateString()
                      },
                      {
                        id: 'actions',
                        header: 'Actions',
                        cell: (item: AMI) => (
                          <ButtonDropdown
                            expandToViewport
                            items={[
                              { text: 'View Details', id: 'details' },
                              { text: 'Copy to Region', id: 'copy', disabled: true },
                              { text: 'Delete AMI', id: 'delete' }
                            ]}
                            onItemClick={({ detail }) => {
                              setSelectedAMI(item);
                              if (detail.id === 'delete') {
                                setDeleteModalVisible(true);
                              }
                            }}
                          >
                            Actions
                          </ButtonDropdown>
                        )
                      }
                    ]}
                    items={state.amis}
                    loadingText="Loading AMIs..."
                    loading={state.loading}
                    trackBy="id"
                    empty={
                      <Box textAlign="center" color="text-body-secondary">
                        <Box variant="strong" textAlign="center" color="text-body-secondary">
                          No AMIs available
                        </Box>
                        <Box variant="p" padding={{ bottom: 's' }} color="text-body-secondary">
                          Build an AMI to enable fast workspace launching (30 seconds vs 5-8 minutes).
                        </Box>
                        <Button variant="primary" onClick={() => setBuildModalVisible(true)}>Build AMI</Button>
                      </Box>
                    }
                    sortingDisabled={false}
                  />
                </Container>
              )
            },
            {
              id: 'builds',
              label: 'Build Status',
              content: (
                <Container>
                  {state.amiBuilds.length === 0 ? (
                    <Box textAlign="center" padding="xl">
                      <Box variant="strong">No active builds</Box>
                      <Box variant="p" color="text-body-secondary">
                        AMI builds typically take 10-15 minutes to complete.
                      </Box>
                    </Box>
                  ) : (
                    <Table
                      columnDefinitions={[
                        { id: 'id', header: 'Build ID', cell: (item: AMIBuild) => item.id },
                        { id: 'template', header: 'Template', cell: (item: AMIBuild) => item.template_name },
                        {
                          id: 'status',
                          header: 'Status',
                          cell: (item: AMIBuild) => (
                            <StatusIndicator
                              type={
                                item.status === 'completed' ? 'success' :
                                item.status === 'failed' ? 'error' : 'in-progress'
                              }
                              ariaLabel={getStatusLabel('build', item.status)}
                            >
                              {item.status}
                            </StatusIndicator>
                          )
                        },
                        { id: 'progress', header: 'Progress', cell: (item: AMIBuild) => `${item.progress}%` },
                        { id: 'step', header: 'Current Step', cell: (item: AMIBuild) => item.current_step || '-' }
                      ]}
                      items={state.amiBuilds}
                      trackBy="id"
                    />
                  )}
                </Container>
              )
            },
            {
              id: 'regions',
              label: 'Regional Coverage',
              content: (
                <Container>
                  <Table
                    columnDefinitions={[
                      {
                        id: 'region',
                        header: 'Region',
                        cell: (item: AMIRegion) => <Badge color={item.ami_count > 0 ? 'green' : 'grey'}>{item.name}</Badge>,
                        sortingField: 'name'
                      },
                      {
                        id: 'count',
                        header: 'AMI Count',
                        cell: (item: AMIRegion) => item.ami_count,
                        sortingField: 'ami_count'
                      },
                      {
                        id: 'size',
                        header: 'Total Size',
                        cell: (item: AMIRegion) => `${item.total_size_gb.toFixed(1)} GB`,
                        sortingField: 'total_size_gb'
                      },
                      {
                        id: 'cost',
                        header: 'Monthly Cost',
                        cell: (item: AMIRegion) => `$${item.monthly_cost.toFixed(2)}`,
                        sortingField: 'monthly_cost'
                      }
                    ]}
                    items={state.amiRegions}
                    trackBy="name"
                    sortingDisabled={false}
                    empty={
                      <Box textAlign="center" padding="xl">
                        <Box variant="strong">No regional data available</Box>
                      </Box>
                    }
                  />
                </Container>
              )
            }
          ]}
        />

        {/* Delete Modal */}
        <Modal
          visible={deleteModalVisible}
          onDismiss={() => setDeleteModalVisible(false)}
          header="Delete AMI"
          footer={
            <Box float="right">
              <SpaceBetween direction="horizontal" size="xs">
                <Button variant="link" onClick={() => setDeleteModalVisible(false)}>Cancel</Button>
                <Button variant="primary" onClick={handleDeleteAMI}>Delete</Button>
              </SpaceBetween>
            </Box>
          }
        >
          <SpaceBetween size="m">
            <Alert type="warning">
              This will permanently delete the AMI and associated snapshots. This action cannot be undone.
            </Alert>
            {selectedAMI && (
              <Box>
                <Box variant="strong">AMI ID:</Box> {selectedAMI.id}
                <br />
                <Box variant="strong">Template:</Box> {selectedAMI.template_name}
                <br />
                <Box variant="strong">Size:</Box> {selectedAMI.size_gb.toFixed(1)} GB
              </Box>
            )}
          </SpaceBetween>
        </Modal>
      </SpaceBetween>
    );
  };


  // Marketplace View
  const MarketplaceView = () => {
    const [searchQuery, setSearchQuery] = useState('');
    const [selectedCategory, setSelectedCategory] = useState<string>('');
    const [selectedTemplate, setSelectedTemplate] = useState<MarketplaceTemplate | null>(null);
    const [installModalVisible, setInstallModalVisible] = useState(false);
    const [filteredTemplates, setFilteredTemplates] = useState<MarketplaceTemplate[]>(state.marketplaceTemplates);

    // Update filtered templates when search or category changes
    useEffect(() => {
      let filtered = state.marketplaceTemplates;

      if (searchQuery) {
        const query = searchQuery.toLowerCase();
        filtered = filtered.filter(t =>
          t.name.toLowerCase().includes(query) ||
          t.display_name.toLowerCase().includes(query) ||
          t.description.toLowerCase().includes(query) ||
          (t.tags && t.tags.some(tag => tag.toLowerCase().includes(query)))
        );
      }

      if (selectedCategory) {
        filtered = filtered.filter(t => t.category === selectedCategory);
      }

      setFilteredTemplates(filtered);
    }, [searchQuery, selectedCategory, state.marketplaceTemplates]);

    const handleInstallTemplate = async () => {
      if (!selectedTemplate) return;

      try {
        await api.installMarketplaceTemplate(selectedTemplate.id);
        setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'success', content: `Installing template: ${selectedTemplate.display_name}` }] }));
        setInstallModalVisible(false);
        setSelectedTemplate(null);
        await loadApplicationData();
      } catch (error) {
        setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'error', content: `Failed to install template: ${error}` }] }));
      }
    };

    const renderRatingStars = (rating: number) => {
      const stars = [];
      for (let i = 1; i <= 5; i++) {
        stars.push(i <= rating ? '★' : '☆');
      }
      return stars.join('');
    };

    return (
      <SpaceBetween size="l">
        <Header
          variant="h1"
          description="Discover and install community-contributed research templates"
          counter={`(${filteredTemplates.length} templates)`}
          actions={
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? <Spinner /> : 'Refresh'}
            </Button>
          }
        >
          Template Marketplace
        </Header>

        {/* Search and Filters */}
        <Container>
          <SpaceBetween size="m">
            <FormField label="Search templates" description="Search by name, description, or tags">
              <Input
                value={searchQuery}
                onChange={({ detail }) => setSearchQuery(detail.value)}
                placeholder="Search templates..."
                clearAriaLabel="Clear search"
                type="search"
              />
            </FormField>
            <FormField label="Category" description="Filter by template category">
              <Select
                selectedOption={selectedCategory ? { label: selectedCategory, value: selectedCategory } : null}
                onChange={({ detail }) => setSelectedCategory(detail.selectedOption?.value || '')}
                options={[
                  { label: 'All Categories', value: '' },
                  ...state.marketplaceCategories.map(c => ({ label: `${c.name} (${c.count})`, value: c.id }))
                ]}
                placeholder="All Categories"
                selectedAriaLabel="Selected"
              />
            </FormField>
          </SpaceBetween>
        </Container>

        {/* Template Cards Grid */}
        <Cards
          cardDefinition={{
            header: (item: MarketplaceTemplate) => (
              <SpaceBetween direction="horizontal" size="xs">
                <Link fontSize="heading-m" onFollow={() => setSelectedTemplate(item)}>
                  {item.display_name || item.name}
                </Link>
                {item.verified && <Badge color="blue">Verified</Badge>}
                {item.featured && <Badge color="green">Featured</Badge>}
              </SpaceBetween>
            ),
            sections: [
              {
                id: 'description',
                content: (item: MarketplaceTemplate) => (
                  <Box>
                    <Box variant="p" color="text-body-secondary">
                      {item.description}
                    </Box>
                  </Box>
                )
              },
              {
                id: 'metadata',
                content: (item: MarketplaceTemplate) => (
                  <ColumnLayout columns={2} variant="text-grid">
                    <div>
                      <Box variant="awsui-key-label">Publisher</Box>
                      <Box>{item.publisher || item.author}</Box>
                    </div>
                    <div>
                      <Box variant="awsui-key-label">Category</Box>
                      <Badge>{item.category}</Badge>
                    </div>
                    <div>
                      <Box variant="awsui-key-label">Rating</Box>
                      <Box color={item.rating >= 4 ? 'text-status-success' : 'inherit'}>
                        {renderRatingStars(item.rating)} ({item.rating.toFixed(1)})
                      </Box>
                    </div>
                    <div>
                      <Box variant="awsui-key-label">Downloads</Box>
                      <Box>{item.downloads.toLocaleString()}</Box>
                    </div>
                  </ColumnLayout>
                )
              },
              {
                id: 'tags',
                content: (item: MarketplaceTemplate) =>
                  item.tags && item.tags.length > 0 ? (
                    <SpaceBetween direction="horizontal" size="xs">
                      {item.tags.slice(0, 5).map(tag => (
                        <Badge key={tag} color="grey">{tag}</Badge>
                      ))}
                    </SpaceBetween>
                  ) : null
              },
              {
                id: 'actions',
                content: (item: MarketplaceTemplate) => (
                  <SpaceBetween direction="horizontal" size="xs">
                    <Button
                      onClick={() => {
                        setSelectedTemplate(item);
                        setInstallModalVisible(true);
                      }}
                    >
                      Install
                    </Button>
                    <Button onClick={() => setSelectedTemplate(item)}>
                      View Details
                    </Button>
                  </SpaceBetween>
                )
              }
            ]
          }}
          items={filteredTemplates}
          cardsPerRow={[{ cards: 1 }, { minWidth: 500, cards: 2 }]}
          loading={state.loading}
          loadingText="Loading marketplace templates..."
          empty={
            <Box textAlign="center" padding="xl">
              <Box variant="strong">No templates found</Box>
              <Box variant="p" color="text-body-secondary">
                {searchQuery || selectedCategory
                  ? 'Try adjusting your search or filter criteria.'
                  : 'No marketplace templates available.'}
              </Box>
            </Box>
          }
        />

        {/* Template Details Modal */}
        {selectedTemplate && !installModalVisible && (
          <Container
            header={
              <Header
                variant="h2"
                actions={<Button onClick={() => setSelectedTemplate(null)}>Close</Button>}
              >
                {selectedTemplate.display_name || selectedTemplate.name}
              </Header>
            }
          >
            <SpaceBetween size="l">
              <ColumnLayout columns={2}>
                <SpaceBetween size="m">
                  <div>
                    <Box variant="awsui-key-label">Publisher</Box>
                    <Box>{selectedTemplate.publisher || selectedTemplate.author}</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Category</Box>
                    <Badge>{selectedTemplate.category}</Badge>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Version</Box>
                    <Box>{selectedTemplate.version}</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Verified</Box>
                    {selectedTemplate.verified ? (
                      <StatusIndicator type="success" ariaLabel={getStatusLabel('marketplace', 'verified')}>Verified Publisher</StatusIndicator>
                    ) : (
                      <StatusIndicator type="pending" ariaLabel={getStatusLabel('marketplace', 'community')}>Community</StatusIndicator>
                    )}
                  </div>
                </SpaceBetween>
                <SpaceBetween size="m">
                  <div>
                    <Box variant="awsui-key-label">Rating</Box>
                    <Box color={selectedTemplate.rating >= 4 ? 'text-status-success' : 'inherit'}>
                      {renderRatingStars(selectedTemplate.rating)} ({selectedTemplate.rating.toFixed(1)})
                    </Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Downloads</Box>
                    <Box>{selectedTemplate.downloads.toLocaleString()}</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Created</Box>
                    <Box>{new Date(selectedTemplate.created_at).toLocaleDateString()}</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Last Updated</Box>
                    <Box>{new Date(selectedTemplate.updated_at).toLocaleDateString()}</Box>
                  </div>
                </SpaceBetween>
              </ColumnLayout>

              <div>
                <Box variant="awsui-key-label">Description</Box>
                <Box variant="p">{selectedTemplate.description}</Box>
              </div>

              {selectedTemplate.tags && selectedTemplate.tags.length > 0 && (
                <div>
                  <Box variant="awsui-key-label">Tags</Box>
                  <SpaceBetween direction="horizontal" size="xs">
                    {selectedTemplate.tags.map(tag => (
                      <Badge key={tag} color="grey">{tag}</Badge>
                    ))}
                  </SpaceBetween>
                </div>
              )}

              {selectedTemplate.badges && selectedTemplate.badges.length > 0 && (
                <div>
                  <Box variant="awsui-key-label">Badges</Box>
                  <SpaceBetween direction="horizontal" size="xs">
                    {selectedTemplate.badges.map(badge => (
                      <Badge key={badge} color="blue">{badge}</Badge>
                    ))}
                  </SpaceBetween>
                </div>
              )}

              {selectedTemplate.ami_available && (
                <Alert type="info">
                  This template has pre-built AMIs available for faster launches (30 seconds vs 5-8 minutes).
                </Alert>
              )}

              <Button
                variant="primary"
                onClick={() => {
                  setInstallModalVisible(true);
                }}
              >
                Install Template
              </Button>
            </SpaceBetween>
          </Container>
        )}

        {/* Install Confirmation Modal */}
        <Modal
          visible={installModalVisible}
          onDismiss={() => { setInstallModalVisible(false); setSelectedTemplate(null); }}
          header="Install Marketplace Template"
          footer={
            <Box float="right">
              <SpaceBetween direction="horizontal" size="xs">
                <Button variant="link" onClick={() => { setInstallModalVisible(false); setSelectedTemplate(null); }}>Cancel</Button>
                <Button variant="primary" onClick={handleInstallTemplate}>Install</Button>
              </SpaceBetween>
            </Box>
          }
        >
          {selectedTemplate && (
            <SpaceBetween size="m">
              <Alert type="info">
                This will download and install the template to your local templates directory.
              </Alert>
              <div>
                <Box variant="strong">Template:</Box> {selectedTemplate.display_name || selectedTemplate.name}
                <br />
                <Box variant="strong">Publisher:</Box> {selectedTemplate.publisher || selectedTemplate.author}
                <br />
                <Box variant="strong">Version:</Box> {selectedTemplate.version}
                <br />
                {selectedTemplate.verified && (
                  <>
                    <Box variant="strong">Status:</Box> <StatusIndicator type="success" ariaLabel={getStatusLabel('marketplace', 'verified')}>Verified Publisher</StatusIndicator>
                  </>
                )}
              </div>
            </SpaceBetween>
          )}
        </Modal>
      </SpaceBetween>
    );
  };

  // Idle Detection & Hibernation View
  const IdleDetectionView = () => {
    const [selectedTab, setSelectedTab] = useState<'policies' | 'schedules'>('policies');
    const [selectedPolicy, setSelectedPolicy] = useState<IdlePolicy | null>(null);

    const getActionBadgeColor = (action: string) => {
      switch (action) {
        case 'hibernate': return 'green';
        case 'stop': return 'blue';
        case 'notify': return 'grey';
        default: return 'grey';
      }
    };

    return (
      <SpaceBetween size="l">
        <Header
          variant="h1"
          description="Automatic cost optimization through idle detection and hibernation"
          actions={
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? <Spinner /> : 'Refresh'}
            </Button>
          }
        >
          Idle Detection & Hibernation
        </Header>

        {/* Overview Stats */}
        <ColumnLayout columns={4} variant="text-grid">
          <Container header={<Header variant="h3">Active Policies</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-info">
              {state.idlePolicies.filter(p => p.enabled).length}
            </Box>
          </Container>
          <Container header={<Header variant="h3">Total Policies</Header>}>
            <Box fontSize="display-l" fontWeight="bold">
              {state.idlePolicies.length}
            </Box>
          </Container>
          <Container header={<Header variant="h3">Monitored Workspaces</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
              {state.idleSchedules.filter(s => s.enabled).length}
            </Box>
          </Container>
          <Container header={<Header variant="h3">Cost Savings</Header>}>
            <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
              ~40%
            </Box>
            <Box variant="small" color="text-body-secondary">
              Through hibernation
            </Box>
          </Container>
        </ColumnLayout>

        <Tabs
          activeTabId={selectedTab}
          onChange={({ detail }) => setSelectedTab(detail.activeTabId as 'policies' | 'schedules')}
          tabs={[
            {
              id: 'policies',
              label: 'Idle Policies',
              content: (
                <Container>
                  <Table
                    columnDefinitions={[
                      {
                        id: 'name',
                        header: 'Policy Name',
                        cell: (item: IdlePolicy) => <Link onFollow={() => setSelectedPolicy(item)}>{item.name}</Link>,
                        sortingField: 'name'
                      },
                      {
                        id: 'idle_minutes',
                        header: 'Idle Threshold',
                        cell: (item: IdlePolicy) => `${item.idle_minutes} minutes`,
                        sortingField: 'idle_minutes'
                      },
                      {
                        id: 'action',
                        header: 'Action',
                        cell: (item: IdlePolicy) => (
                          <Badge color={getActionBadgeColor(item.action)}>
                            {item.action.toUpperCase()}
                          </Badge>
                        )
                      },
                      {
                        id: 'thresholds',
                        header: 'Thresholds',
                        cell: (item: IdlePolicy) => (
                          <Box variant="small">
                            CPU: {item.cpu_threshold}%, Mem: {item.memory_threshold}%, Net: {item.network_threshold} Mbps
                          </Box>
                        )
                      },
                      {
                        id: 'enabled',
                        header: 'Status',
                        cell: (item: IdlePolicy) => (
                          <StatusIndicator
                            type={item.enabled ? 'success' : 'stopped'}
                            ariaLabel={getStatusLabel('idle', item.enabled ? 'enabled' : 'disabled')}
                          >
                            {item.enabled ? 'Enabled' : 'Disabled'}
                          </StatusIndicator>
                        )
                      }
                    ]}
                    items={state.idlePolicies}
                    loadingText="Loading idle policies..."
                    loading={state.loading}
                    trackBy="id"
                    empty={
                      <Box textAlign="center" padding="xl">
                        <Box variant="strong">No idle policies configured</Box>
                        <Box variant="p" color="text-body-secondary">
                          Idle policies automatically hibernate or stop workspaces when they're not being used.
                        </Box>
                      </Box>
                    }
                    sortingDisabled={false}
                  />
                </Container>
              )
            },
            {
              id: 'schedules',
              label: 'Workspace Schedules',
              content: (
                <Container>
                  <Table
                    columnDefinitions={[
                      {
                        id: 'instance',
                        header: 'Workspace',
                        cell: (item: IdleSchedule) => item.instance_name,
                        sortingField: 'instance_name'
                      },
                      {
                        id: 'policy',
                        header: 'Policy',
                        cell: (item: IdleSchedule) => <Badge>{item.policy_name}</Badge>
                      },
                      {
                        id: 'idle_minutes',
                        header: 'Current Idle Time',
                        cell: (item: IdleSchedule) => `${item.idle_minutes} minutes`,
                        sortingField: 'idle_minutes'
                      },
                      {
                        id: 'status',
                        header: 'Status',
                        cell: (item: IdleSchedule) => item.status || 'Active'
                      },
                      {
                        id: 'last_checked',
                        header: 'Last Checked',
                        cell: (item: IdleSchedule) => item.last_checked ? new Date(item.last_checked).toLocaleString() : 'Never'
                      },
                      {
                        id: 'enabled',
                        header: 'Monitoring',
                        cell: (item: IdleSchedule) => (
                          <StatusIndicator
                            type={item.enabled ? 'success' : 'stopped'}
                            ariaLabel={getStatusLabel('idle', item.enabled ? 'enabled' : 'disabled')}
                          >
                            {item.enabled ? 'Enabled' : 'Disabled'}
                          </StatusIndicator>
                        )
                      }
                    ]}
                    items={state.idleSchedules}
                    loadingText="Loading workspace schedules..."
                    loading={state.loading}
                    trackBy="instance_name"
                    empty={
                      <Box textAlign="center" padding="xl">
                        <Box variant="strong">No workspaces being monitored</Box>
                        <Box variant="p" color="text-body-secondary">
                          Start workspaces with idle detection enabled to see them here.
                        </Box>
                      </Box>
                    }
                    sortingDisabled={false}
                  />
                </Container>
              )
            }
          ]}
        />

        {/* Policy Details */}
        {selectedPolicy && (
          <Container
            header={
              <Header
                variant="h2"
                actions={<Button onClick={() => setSelectedPolicy(null)}>Close</Button>}
              >
                {selectedPolicy.name}
              </Header>
            }
          >
            <SpaceBetween size="l">
              <ColumnLayout columns={2}>
                <SpaceBetween size="m">
                  <div>
                    <Box variant="awsui-key-label">Policy ID</Box>
                    <Box>{selectedPolicy.id}</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Idle Threshold</Box>
                    <Box fontWeight="bold">{selectedPolicy.idle_minutes} minutes</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Action</Box>
                    <Badge color={getActionBadgeColor(selectedPolicy.action)}>
                      {selectedPolicy.action.toUpperCase()}
                    </Badge>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Status</Box>
                    <StatusIndicator
                      type={selectedPolicy.enabled ? 'success' : 'stopped'}
                      ariaLabel={getStatusLabel('idle', selectedPolicy.enabled ? 'enabled' : 'disabled')}
                    >
                      {selectedPolicy.enabled ? 'Enabled' : 'Disabled'}
                    </StatusIndicator>
                  </div>
                </SpaceBetween>
                <SpaceBetween size="m">
                  <div>
                    <Box variant="awsui-key-label">CPU Threshold</Box>
                    <Box>{selectedPolicy.cpu_threshold}%</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Memory Threshold</Box>
                    <Box>{selectedPolicy.memory_threshold}%</Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Network Threshold</Box>
                    <Box>{selectedPolicy.network_threshold} Mbps</Box>
                  </div>
                </SpaceBetween>
              </ColumnLayout>

              {selectedPolicy.description && (
                <div>
                  <Box variant="awsui-key-label">Description</Box>
                  <Box variant="p">{selectedPolicy.description}</Box>
                </div>
              )}

              <Alert type="info">
                <Box variant="strong">How It Works:</Box>
                <Box variant="p">
                  This policy monitors workspace activity. When CPU, memory, and network usage all fall below
                  the specified thresholds for {selectedPolicy.idle_minutes} consecutive minutes, the system will
                  automatically {selectedPolicy.action === 'hibernate' ? 'hibernate (preserve RAM state)' :
                  selectedPolicy.action === 'stop' ? 'stop the workspace' : 'send a notification'}.
                </Box>
              </Alert>

              {selectedPolicy.action === 'hibernate' && (
                <Alert type="success">
                  <Box variant="strong">Cost Savings with Hibernation:</Box>
                  <Box variant="p">
                    Hibernation preserves your RAM state to disk, allowing instant resume while only paying for
                    EBS storage (~$0.10/GB/month). This can save ~40% on compute costs for workspaces that are
                    idle for extended periods.
                  </Box>
                </Alert>
              )}
            </SpaceBetween>
          </Container>
        )}

        {/* Educational Content */}
        <Container header={<Header variant="h2">About Idle Detection</Header>}>
          <SpaceBetween size="m">
            <Box variant="p">
              Idle detection monitors your workspaces and automatically hibernates or stops them when they're not
              being used, saving significant compute costs while preserving your work environment.
            </Box>
            <ColumnLayout columns={3}>
              <div>
                <Box variant="strong">Hibernate</Box>
                <Box variant="small" color="text-body-secondary">
                  Preserves RAM state to disk. Resume in seconds with your session intact. Best for
                  workloads that need quick resumption.
                </Box>
              </div>
              <div>
                <Box variant="strong">Stop</Box>
                <Box variant="small" color="text-body-secondary">
                  Fully stops the workspace. Cheaper than hibernation but requires full restart.
                  Best for workspaces that don't need quick resumption.
                </Box>
              </div>
              <div>
                <Box variant="strong">Notify</Box>
                <Box variant="small" color="text-body-secondary">
                  Sends a notification without taking action. Useful for monitoring patterns
                  before enabling automated actions.
                </Box>
              </div>
            </ColumnLayout>
          </SpaceBetween>
        </Container>
      </SpaceBetween>
    );
  };

  // Logs Viewer
  const LogsView = () => {
    const [selectedInstance, setSelectedInstance] = useState<string>('');
    const [logType, setLogType] = useState<string>('console');
    const [logLines, setLogLines] = useState<string[]>([]);
    const [loadingLogs, setLoadingLogs] = useState(false);

    const logTypes = [
      { label: 'Console Output', value: 'console' },
      { label: 'Cloud-Init Log', value: 'cloud-init' },
      { label: 'System Log', value: 'system' },
      { label: 'Application Log', value: 'application' }
    ];

    const runningInstances = state.instances.filter(i => i.state === 'running' || i.state === 'stopped');

    const fetchLogs = async () => {
      if (!selectedInstance) return;

      setLoadingLogs(true);
      try {
        // Mock log fetching - in real implementation would call API
        // const logs = await api.getInstanceLogs(selectedInstance, logType);

        // Generate mock logs for demonstration
        const mockLogs = [
          `[${new Date().toISOString()}] Workspace ${selectedInstance} logs (${logType})`,
          `[INFO] Workspace started successfully`,
          `[INFO] Loading configuration...`,
          `[INFO] Mounting EFS volumes...`,
          `[INFO] Starting services...`,
          `[INFO] Prism template: ${state.instances.find(i => i.name === selectedInstance)?.template || 'unknown'}`,
          `[INFO] All services running`,
          `[DEBUG] Memory usage: 1.2GB / 8GB`,
          `[DEBUG] CPU usage: 5%`,
          `[INFO] Workspace ready for use`,
          `[INFO] SSH access: ssh ${state.instances.find(i => i.name === selectedInstance)?.public_ip || 'N/A'}`,
          `--- End of ${logType} log ---`
        ];

        setLogLines(mockLogs);
      } catch (error) {
        setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'error', content: `Failed to fetch logs: ${error}` }] }));
        setLogLines([`Error fetching logs: ${error}`]);
      } finally {
        setLoadingLogs(false);
      }
    };

    useEffect(() => {
      if (selectedInstance) {
        fetchLogs();
      }
    }, [selectedInstance, logType]);

    return (
      <SpaceBetween size="l">
        <Header
          variant="h1"
          description="View workspace console output and system logs"
          actions={
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? <Spinner /> : 'Refresh'}
            </Button>
          }
        >
          Workspace Logs Viewer
        </Header>

        {/* Workspace and Log Type Selection */}
        <Container>
          <SpaceBetween size="m">
            <FormField
              label="Workspace"
              description="Select a workspace to view its logs"
            >
              <Select
                selectedOption={selectedInstance ?
                  { label: selectedInstance, value: selectedInstance } : null}
                onChange={({ detail }) => {
                  setSelectedInstance(detail.selectedOption?.value || '');
                  setLogLines([]);
                }}
                options={runningInstances.map(i => ({
                  label: `${i.name} (${i.state})`,
                  value: i.name
                }))}
                placeholder="Choose a workspace"
                selectedAriaLabel="Selected workspace"
                disabled={runningInstances.length === 0}
              />
            </FormField>

            {selectedInstance && (
              <FormField
                label="Log Type"
                description="Select the type of log to view"
              >
                <Select
                  selectedOption={logType ?
                    logTypes.find(t => t.value === logType) : null}
                  onChange={({ detail }) => {
                    setLogType(detail.selectedOption?.value || 'console');
                    setLogLines([]);
                  }}
                  options={logTypes}
                  selectedAriaLabel="Selected log type"
                />
              </FormField>
            )}

            {selectedInstance && (
              <Button
                onClick={fetchLogs}
                loading={loadingLogs}
                disabled={loadingLogs}
              >
                Refresh Logs
              </Button>
            )}
          </SpaceBetween>
        </Container>

        {/* Log Display */}
        {selectedInstance ? (
          <Container
            header={
              <Header
                variant="h2"
                description={`Viewing ${logType} logs for ${selectedInstance}`}
              >
                Log Output
              </Header>
            }
          >
            {loadingLogs ? (
              <Box textAlign="center" padding="xl">
                <Spinner size="large" />
                <Box variant="p">Loading logs...</Box>
              </Box>
            ) : logLines.length > 0 ? (
              <Box
                padding="s"
                variant="code"
              >
                <pre style={{
                  fontFamily: 'monospace',
                  fontSize: '12px',
                  lineHeight: '1.5',
                  margin: 0,
                  padding: '8px',
                  backgroundColor: '#232f3e',
                  color: '#d4d4d4',
                  borderRadius: '4px',
                  maxHeight: '600px',
                  overflow: 'auto',
                  whiteSpace: 'pre-wrap',
                  wordWrap: 'break-word'
                }}>
                  {logLines.join('\n')}
                </pre>
              </Box>
            ) : (
              <Box textAlign="center" padding="xl">
                <Box variant="strong">No logs available</Box>
                <Box variant="p" color="text-body-secondary">
                  Select a log type and click "Refresh Logs" to view output.
                </Box>
              </Box>
            )}

            {logLines.length > 0 && (
              <Box padding={{ top: 'm' }}>
                <SpaceBetween direction="horizontal" size="xs">
                  <Button iconName="copy" onClick={() => {
                    navigator.clipboard.writeText(logLines.join('\n'));
                    setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'success', content: 'Logs copied to clipboard' }] }));
                  }}>
                    Copy to Clipboard
                  </Button>
                  <Button iconName="download" onClick={() => {
                    const blob = new Blob([logLines.join('\n')], { type: 'text/plain' });
                    const url = URL.createObjectURL(blob);
                    const a = document.createElement('a');
                    a.href = url;
                    a.download = `${selectedInstance}-${logType}-${new Date().toISOString().split('T')[0]}.log`;
                    a.click();
                    URL.revokeObjectURL(url);
                    setState(prev => ({ ...prev, notifications: [...prev.notifications, { type: 'success', content: 'Log file downloaded' }] }));
                  }}>
                    Download Log File
                  </Button>
                </SpaceBetween>
              </Box>
            )}
          </Container>
        ) : (
          <Container>
            <Box textAlign="center" padding="xl">
              <Box variant="strong">Select a Workspace</Box>
              <Box variant="p" color="text-body-secondary">
                {runningInstances.length === 0
                  ? 'No running or stopped workspaces available. Start a workspace to view its logs.'
                  : 'Choose a workspace from the dropdown above to view its logs.'}
              </Box>
            </Box>
          </Container>
        )}

        {/* Information */}
        <Container header={<Header variant="h2">About Log Viewing</Header>}>
          <SpaceBetween size="m">
            <Box variant="p">
              View real-time console output and system logs from your Prism workspaces.
              Logs are useful for troubleshooting startup issues, monitoring application output,
              and understanding workspace behavior.
            </Box>
            <ColumnLayout columns={4}>
              <div>
                <Box variant="strong">Console Output</Box>
                <Box variant="small" color="text-body-secondary">
                  System boot messages and console output
                </Box>
              </div>
              <div>
                <Box variant="strong">Cloud-Init</Box>
                <Box variant="small" color="text-body-secondary">
                  Prism provisioning logs
                </Box>
              </div>
              <div>
                <Box variant="strong">System Log</Box>
                <Box variant="small" color="text-body-secondary">
                  Operating system events and services
                </Box>
              </div>
              <div>
                <Box variant="strong">Application Log</Box>
                <Box variant="small" color="text-body-secondary">
                  Application-specific output
                </Box>
              </div>
            </ColumnLayout>
            <Alert type="info">
              <Box variant="strong">Note:</Box> Log viewing is read-only. To interact with your workspace,
              use SSH: <Box fontFamily="monospace" variant="code">
                ssh {selectedInstance && state.instances.find(i => i.name === selectedInstance)?.public_ip || 'instance-ip'}
              </Box>
            </Alert>
          </SpaceBetween>
        </Container>
      </SpaceBetween>
    );
  };

  const RightsizingView = () => (
    <PlaceholderView
      title="Rightsizing Recommendations"
      description="Workspace rightsizing recommendations will help optimize your costs by suggesting better-sized workspaces based on actual usage patterns."
    />
  );

  const PolicyView = () => (
    <PlaceholderView
      title="Policy Management"
      description="Policy management allows you to configure institutional policies, access controls, and governance rules for your Prism deployment."
    />
  );


  const WebViewView = () => {
    const [selectedService, setSelectedService] = React.useState<{instance: string, service: any} | null>(null);
    const instancesWithServices = state.instances.filter(i =>
      i.state === 'running' && i.web_services && i.web_services.length > 0
    );

    if (instancesWithServices.length === 0) {
      return (
        <Container header={<Header variant="h1">Web Services</Header>}>
          <Alert type="info">
            No running instances with web services available. Launch a workspace with Jupyter or RStudio to access web services.
          </Alert>
        </Container>
      );
    }

    const serviceOptions = instancesWithServices.flatMap(instance =>
      (instance.web_services || []).map(service => ({
        label: `${instance.name} - ${service.name} (${service.type})`,
        value: JSON.stringify({ instance: instance.name, service }),
        instanceName: instance.name,
        service: service
      }))
    );

    return (
      <SpaceBetween size="l">
        <Container header={<Header variant="h1">Web Services</Header>}>
          <SpaceBetween size="m">
            <FormField label="Select Web Service">
              <Select
                selectedOption={selectedService ?
                  { label: `${selectedService.instance} - ${selectedService.service.name} (${selectedService.service.type})`,
                    value: JSON.stringify(selectedService) } : null}
                onChange={({ detail }) => {
                  if (detail.selectedOption.value) {
                    const parsed = JSON.parse(detail.selectedOption.value);
                    setSelectedService(parsed);
                  }
                }}
                options={serviceOptions.map(opt => ({ label: opt.label, value: opt.value }))}
                placeholder="Choose a web service"
              />
            </FormField>
            {selectedService && (
              <WebView
                url={selectedService.service.url}
                serviceName={selectedService.service.name}
                instanceName={selectedService.instance}
              />
            )}
          </SpaceBetween>
        </Container>
      </SpaceBetween>
    );
  };

  const PlaceholderView = ({ title, description }: { title: string; description: string }) => (
    <Container header={<Header variant="h1">{title}</Header>}>
      <Box textAlign="center" padding="xl">
        <Box variant="strong">{title}</Box>
        <Box variant="p">{description}</Box>
        <Alert type="info">This feature will be available in a future update.</Alert>
      </Box>
    </Container>
  );

  // Launch Modal
  // Delete Confirmation Modal Component
  const DeleteConfirmationModal = () => {
    const getDeleteMessage = () => {
      switch (deleteModalConfig.type) {
        case 'instance':
          return `You are about to permanently delete the workspace "${deleteModalConfig.name}". This action cannot be undone.`;
        case 'efs-volume':
          return `You are about to permanently delete the EFS volume "${deleteModalConfig.name}". All data on this volume will be lost. This action cannot be undone.`;
        case 'ebs-volume':
          return `You are about to permanently delete the EBS volume "${deleteModalConfig.name}". All data on this volume will be lost. This action cannot be undone.`;
        case 'project':
          return `You are about to permanently delete the project "${deleteModalConfig.name}". This action cannot be undone.`;
        case 'user':
          return `You are about to permanently delete the user "${deleteModalConfig.name}". This action cannot be undone.`;
        default:
          return 'This action cannot be undone.';
      }
    };

    const isConfirmationValid = deleteModalConfig.requireNameConfirmation
      ? deleteConfirmationText === deleteModalConfig.name
      : true;

    return (
      <Modal
        visible={deleteModalVisible}
        onDismiss={() => {
          setDeleteModalVisible(false);
          setDeleteConfirmationText('');
        }}
        header={`Delete ${deleteModalConfig.type?.replace('-', ' ') || 'Resource'}?`}
        size="medium"
        footer={
          <Box float="right">
            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="link"
                onClick={() => {
                  setDeleteModalVisible(false);
                  setDeleteConfirmationText('');
                }}
              >
                Cancel
              </Button>
              <Button
                variant="primary"
                onClick={deleteModalConfig.onConfirm}
                disabled={!isConfirmationValid}
              >
                Delete
              </Button>
            </SpaceBetween>
          </Box>
        }
      >
        <SpaceBetween size="m">
          <Alert type="warning" header="Warning: This action is permanent">
            {getDeleteMessage()}
          </Alert>

          {deleteModalConfig.requireNameConfirmation && (
            <FormField
              label={`Type "${deleteModalConfig.name}" to confirm deletion`}
              description="This extra step helps prevent accidental deletions"
              errorText={
                deleteConfirmationText.length > 0 && deleteConfirmationText !== deleteModalConfig.name
                  ? `Name must match exactly: "${deleteModalConfig.name}"`
                  : ""
              }
            >
              <Input
                value={deleteConfirmationText}
                onChange={({ detail }) => setDeleteConfirmationText(detail.value)}
                placeholder={deleteModalConfig.name}
                ariaRequired
                invalid={deleteConfirmationText.length > 0 && deleteConfirmationText !== deleteModalConfig.name}
              />
            </FormField>
          )}

          <Box variant="p" color="text-body-secondary">
            {deleteModalConfig.requireNameConfirmation
              ? 'Enter the exact name above to enable the delete button.'
              : 'Click Delete to confirm this action.'}
          </Box>
        </SpaceBetween>
      </Modal>
    );
  };

  const LaunchModal = () => (
    <Modal
      onDismiss={handleModalDismiss}
      visible={launchModalVisible}
      header={`Launch ${state.selectedTemplate ? getTemplateName(state.selectedTemplate) : 'Research Environment'}`}
      size="medium"
      footer={
        <Box float="right">
          <SpaceBetween direction="horizontal" size="xs">
            <Button variant="link" onClick={handleModalDismiss}>
              Cancel
            </Button>
            <Button
              variant="primary"
              disabled={!launchConfig.name.trim()}
              onClick={handleLaunchInstance}
            >
              Launch Workspace
            </Button>
          </SpaceBetween>
        </Box>
      }
    >
      <Form>
        <SpaceBetween size="m">
          <FormField
            label="Workspace name"
            description="Choose a descriptive name for your research project"
            errorText={!launchConfig.name.trim() ? "Workspace name is required" : ""}
          >
            <Input
              value={launchConfig.name}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, name: detail.value }))}
              placeholder="my-research-project"
            />
          </FormField>

          <FormField label="Workspace size" description="Choose the right size for your workload">
            <Select
              selectedOption={{ label: "Medium (M) - Recommended", value: "M" }}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, size: detail.selectedOption.value }))}
              options={[
                { label: "Small (S) - Light workloads", value: "S" },
                { label: "Medium (M) - Recommended", value: "M" },
                { label: "Large (L) - Heavy compute", value: "L" },
                { label: "Extra Large (XL) - Maximum performance", value: "XL" }
              ]}
              data-testid="select"
            />
          </FormField>

          {state.selectedTemplate && (
            <Alert type="info">
              <Box>
                <Box variant="strong">Template: {getTemplateName(state.selectedTemplate)}</Box>
                <Box>Description: {getTemplateDescription(state.selectedTemplate)}</Box>
                {state.selectedTemplate.package_manager && (
                  <Box>Package Manager: {state.selectedTemplate.package_manager}</Box>
                )}
                {state.selectedTemplate.complexity && (
                  <Box>Complexity: {state.selectedTemplate.complexity}</Box>
                )}
              </Box>
            </Alert>
          )}
        </SpaceBetween>
      </Form>
    </Modal>
  );

  // Onboarding Wizard Modal
  const OnboardingWizard = () => {
    const totalSteps = 3;

    const handleNext = () => {
      if (onboardingStep < totalSteps - 1) {
        setOnboardingStep(onboardingStep + 1);
      } else {
        // Complete onboarding
        localStorage.setItem('cws_onboarding_complete', 'true');
        setOnboardingComplete(true);
        setOnboardingVisible(false);
        setOnboardingStep(0);
      }
    };

    const handleBack = () => {
      if (onboardingStep > 0) {
        setOnboardingStep(onboardingStep - 1);
      }
    };

    const handleSkip = () => {
      localStorage.setItem('cws_onboarding_complete', 'true');
      setOnboardingComplete(true);
      setOnboardingVisible(false);
      setOnboardingStep(0);
    };

    return (
      <Modal
        visible={onboardingVisible}
        onDismiss={handleSkip}
        header={`Welcome to Prism - Step ${onboardingStep + 1} of ${totalSteps}`}
        size="large"
        footer={
          <Box float="right">
            <SpaceBetween direction="horizontal" size="xs">
              {onboardingStep > 0 && (
                <Button onClick={handleBack}>
                  Back
                </Button>
              )}
              <Button variant="link" onClick={handleSkip}>
                Skip Tour
              </Button>
              <Button variant="primary" onClick={handleNext}>
                {onboardingStep < totalSteps - 1 ? 'Next' : 'Get Started'}
              </Button>
            </SpaceBetween>
          </Box>
        }
      >
        <SpaceBetween size="l">
          {/* Step 1: AWS Profile Setup */}
          {onboardingStep === 0 && (
            <SpaceBetween size="m">
              <Alert type="info" header="AWS Credentials Configured">
                Prism is already connected to your AWS account using the configured profile.
              </Alert>
              <Box variant="h2">Step 1: AWS Configuration</Box>
              <Box>
                Prism manages cloud workstations in your AWS account. Your current AWS configuration:
              </Box>
              <Container>
                <ColumnLayout columns={2} variant="text-grid">
                  <div>
                    <Box variant="awsui-key-label">AWS Profile</Box>
                    <Box fontWeight="bold">aws</Box>
                    <Box variant="small" color="text-body-secondary">
                      Your AWS credentials profile
                    </Box>
                  </div>
                  <div>
                    <Box variant="awsui-key-label">Region</Box>
                    <Box fontWeight="bold">us-west-2</Box>
                    <Box variant="small" color="text-body-secondary">
                      Resources will be created here
                    </Box>
                  </div>
                </ColumnLayout>
              </Container>
              <Box variant="p" color="text-body-secondary">
                Prism uses your AWS credentials to create and manage cloud workstations.
                You maintain full control over your resources and costs.
              </Box>
            </SpaceBetween>
          )}

          {/* Step 2: Template Discovery Tour */}
          {onboardingStep === 1 && (
            <SpaceBetween size="m">
              <Box variant="h2">Step 2: Choose Your Research Environment</Box>
              <Box>
                Prism provides pre-configured templates for different research workflows.
                Each template includes specialized software, libraries, and tools.
              </Box>
              <ColumnLayout columns={2}>
                <Container header={<Header variant="h3">Popular Templates</Header>}>
                  <SpaceBetween size="s">
                    <Box>
                      <Box variant="strong">Python Machine Learning</Box>
                      <Box variant="small" color="text-body-secondary">
                        Python 3, Jupyter, TensorFlow, PyTorch, scikit-learn
                      </Box>
                    </Box>
                    <Box>
                      <Box variant="strong">R Research Environment</Box>
                      <Box variant="small" color="text-body-secondary">
                        R, RStudio Server, tidyverse, statistical packages
                      </Box>
                    </Box>
                    <Box>
                      <Box variant="strong">Collaborative Workspace</Box>
                      <Box variant="small" color="text-body-secondary">
                        Multi-language support with Python, R, Julia
                      </Box>
                    </Box>
                  </SpaceBetween>
                </Container>
                <Container header={<Header variant="h3">What's Included</Header>}>
                  <SpaceBetween size="s">
                    <Box>✓ Pre-installed software and dependencies</Box>
                    <Box>✓ Optimized workspace sizing for your workload</Box>
                    <Box>✓ Persistent storage for your data</Box>
                    <Box>✓ SSH and remote access configured</Box>
                    <Box>✓ Security best practices applied</Box>
                  </SpaceBetween>
                </Container>
              </ColumnLayout>
              <Alert type="info">
                You can browse all available templates in the <strong>Templates</strong> section after completing this tour.
              </Alert>
            </SpaceBetween>
          )}

          {/* Step 3: Launch Your First Workspace */}
          {onboardingStep === 2 && (
            <SpaceBetween size="m">
              <Box variant="h2">Step 3: Launch Your First Workstation</Box>
              <Box>
                Ready to get started? Here's how to launch your first cloud workstation:
              </Box>
              <Container>
                <SpaceBetween size="m">
                  <div>
                    <Box variant="h4">1. Select a Template</Box>
                    <Box>Choose a template that matches your research needs from the Templates page.</Box>
                  </div>
                  <div>
                    <Box variant="h4">2. Configure Workspace</Box>
                    <Box>Give your workstation a name and select the appropriate size (Small, Medium, Large).</Box>
                  </div>
                  <div>
                    <Box variant="h4">3. Launch & Connect</Box>
                    <Box>Prism creates your workspace in minutes. Connect via SSH or web interface when ready.</Box>
                  </div>
                </SpaceBetween>
              </Container>
              <Alert type="success" header="You're All Set!">
                After clicking "Get Started", explore the dashboard to see your system status,
                browse templates, and launch your first cloud workstation.
              </Alert>
              <Box variant="p" color="text-body-secondary">
                💡 <strong>Tip:</strong> Start with a Medium (M) sized workspace for most workloads.
                You can always stop, resize, or terminate workspaces to manage costs.
              </Box>
            </SpaceBetween>
          )}
        </SpaceBetween>
      </Modal>
    );
  };

  // Quick Start Wizard
  const QuickStartWizard = () => {
    const handleWizardNavigate = (event: { detail: { requestedStepIndex: number; reason: string } }) => {
      setQuickStartActiveStepIndex(event.detail.requestedStepIndex);
    };

    const handleWizardCancel = () => {
      setQuickStartWizardVisible(false);
      setQuickStartActiveStepIndex(0);
      setQuickStartConfig({
        selectedTemplate: null,
        workspaceName: '',
        size: 'M',
        launchInProgress: false,
        launchedWorkspaceId: null
      });
    };

    const handleWizardSubmit = async () => {
      if (!quickStartConfig.selectedTemplate) return;

      setQuickStartConfig(prev => ({ ...prev, launchInProgress: true }));
      setQuickStartActiveStepIndex(3); // Move to progress step

      try {
        const result = await api.launchInstance({
          template: getTemplateSlug(quickStartConfig.selectedTemplate),
          name: quickStartConfig.workspaceName,
          size: quickStartConfig.size
        });

        setQuickStartConfig(prev => ({
          ...prev,
          launchInProgress: false,
          launchedWorkspaceId: result?.id || null
        }));

        addNotification({
          type: 'success',
          content: `Workspace "${quickStartConfig.workspaceName}" launched successfully!`,
          dismissible: true
        });

        // Refresh workspace list
        await loadApplicationData();
      } catch (error) {
        setQuickStartConfig(prev => ({ ...prev, launchInProgress: false }));
        addNotification({
          type: 'error',
          content: `Failed to launch workspace: ${error instanceof Error ? error.message : 'Unknown error'}`,
          dismissible: true
        });
      }
    };

    const getSizeDescription = (size: string): string => {
      const descriptions: Record<string, string> = {
        'S': 'Small - 2 vCPU, 4GB RAM (~$0.08/hour)',
        'M': 'Medium - 4 vCPU, 8GB RAM (~$0.16/hour)',
        'L': 'Large - 8 vCPU, 16GB RAM (~$0.32/hour)',
        'XL': 'Extra Large - 16 vCPU, 32GB RAM (~$0.64/hour)'
      };
      return descriptions[size] || descriptions['M'];
    };

    const getCategoryTemplates = (category: string): Template[] => {
      return Object.values(state.templates).filter(t => {
        const name = getTemplateName(t).toLowerCase();
        const desc = getTemplateDescription(t).toLowerCase();
        switch (category) {
          case 'ml':
            return name.includes('machine learning') || name.includes('ml') || name.includes('python') && desc.includes('tensorflow');
          case 'datascience':
            return name.includes('python') || name.includes('jupyter') || name.includes('data');
          case 'r':
            return name.includes('r ') || name.includes('rstudio');
          case 'bio':
            return name.includes('bio') || name.includes('genomics');
          default:
            return true;
        }
      });
    };

    return (
      <Modal
        visible={quickStartWizardVisible}
        onDismiss={handleWizardCancel}
        size="large"
        header="Quick Start - Launch Workspace"
      >
        <Wizard
          i18nStrings={{
            stepNumberLabel: stepNumber => `Step ${stepNumber}`,
            collapsedStepsLabel: (stepNumber, stepsCount) => `Step ${stepNumber} of ${stepsCount}`,
            skipToButtonLabel: (step) => `Skip to ${step.title}`,
            navigationAriaLabel: "Steps",
            cancelButton: "Cancel",
            previousButton: "Previous",
            nextButton: "Next",
            submitButton: "Launch Workspace",
            optional: "optional"
          }}
          onNavigate={handleWizardNavigate}
          onCancel={handleWizardCancel}
          onSubmit={handleWizardSubmit}
          activeStepIndex={quickStartActiveStepIndex}
          isLoadingNextStep={quickStartConfig.launchInProgress}
          steps={[
            {
              title: "Select Template",
              description: "Choose a pre-configured research environment",
              content: (
                <SpaceBetween size="l">
                  <Alert type="info">
                    Select a template that matches your research needs. Each template includes specialized software and tools.
                  </Alert>

                  <Tabs
                    tabs={[
                      {
                        id: "all",
                        label: "All Templates",
                        content: (
                          <Cards
                            cardDefinition={{
                              header: item => (
                                <Box variant="h3">{getTemplateName(item)}</Box>
                              ),
                              sections: [
                                {
                                  id: "description",
                                  content: item => getTemplateDescription(item)
                                },
                                {
                                  id: "tags",
                                  content: item => (
                                    <SpaceBetween direction="horizontal" size="xs">
                                      {getTemplateTags(item).slice(0, 3).map((tag, idx) => (
                                        <Badge key={idx} color="blue">{tag}</Badge>
                                      ))}
                                    </SpaceBetween>
                                  )
                                }
                              ]
                            }}
                            items={Object.values(state.templates)}
                            selectionType="single"
                            selectedItems={quickStartConfig.selectedTemplate ? [quickStartConfig.selectedTemplate] : []}
                            onSelectionChange={({ detail }) => {
                              if (detail.selectedItems.length > 0) {
                                setQuickStartConfig(prev => ({
                                  ...prev,
                                  selectedTemplate: detail.selectedItems[0]
                                }));
                              }
                            }}
                            cardsPerRow={[{ cards: 1 }, { minWidth: 500, cards: 2 }]}
                            empty={
                              <Box textAlign="center" color="inherit">
                                <b>No templates available</b>
                                <Box padding={{ bottom: "s" }} variant="p" color="inherit">
                                  No research templates found.
                                </Box>
                              </Box>
                            }
                          />
                        )
                      },
                      {
                        id: "ml",
                        label: "ML/AI",
                        content: (
                          <Cards
                            cardDefinition={{
                              header: item => <Box variant="h3">{getTemplateName(item)}</Box>,
                              sections: [{ id: "description", content: item => getTemplateDescription(item) }]
                            }}
                            items={getCategoryTemplates('ml')}
                            selectionType="single"
                            selectedItems={quickStartConfig.selectedTemplate ? [quickStartConfig.selectedTemplate] : []}
                            onSelectionChange={({ detail }) => {
                              if (detail.selectedItems.length > 0) {
                                setQuickStartConfig(prev => ({ ...prev, selectedTemplate: detail.selectedItems[0] }));
                              }
                            }}
                            cardsPerRow={[{ cards: 1 }, { minWidth: 500, cards: 2 }]}
                          />
                        )
                      },
                      {
                        id: "datascience",
                        label: "Data Science",
                        content: (
                          <Cards
                            cardDefinition={{
                              header: item => <Box variant="h3">{getTemplateName(item)}</Box>,
                              sections: [{ id: "description", content: item => getTemplateDescription(item) }]
                            }}
                            items={getCategoryTemplates('datascience')}
                            selectionType="single"
                            selectedItems={quickStartConfig.selectedTemplate ? [quickStartConfig.selectedTemplate] : []}
                            onSelectionChange={({ detail }) => {
                              if (detail.selectedItems.length > 0) {
                                setQuickStartConfig(prev => ({ ...prev, selectedTemplate: detail.selectedItems[0] }));
                              }
                            }}
                            cardsPerRow={[{ cards: 1 }, { minWidth: 500, cards: 2 }]}
                          />
                        )
                      }
                    ]}
                  />
                </SpaceBetween>
              ),
              isOptional: false
            },
            {
              title: "Configure Workspace",
              description: "Set workspace name and size",
              content: (
                <SpaceBetween size="l">
                  <FormField
                    label="Workspace Name"
                    description="Choose a unique name for your workspace"
                    constraintText="Use lowercase letters, numbers, and hyphens only"
                  >
                    <Input
                      value={quickStartConfig.workspaceName}
                      onChange={({ detail }) => setQuickStartConfig(prev => ({ ...prev, workspaceName: detail.value }))}
                      placeholder="my-research-workspace"
                    />
                  </FormField>

                  <FormField
                    label="Workspace Size"
                    description="Choose the compute resources for your workspace"
                  >
                    <Select
                      selectedOption={{ label: getSizeDescription(quickStartConfig.size), value: quickStartConfig.size }}
                      onChange={({ detail }) => setQuickStartConfig(prev => ({ ...prev, size: detail.selectedOption.value || 'M' }))}
                      options={[
                        { label: getSizeDescription('S'), value: 'S' },
                        { label: getSizeDescription('M'), value: 'M' },
                        { label: getSizeDescription('L'), value: 'L' },
                        { label: getSizeDescription('XL'), value: 'XL' }
                      ]}
                    />
                  </FormField>

                  <Alert type="info">
                    💡 <strong>Tip:</strong> Start with Medium size for most workloads. You can always stop and resize later.
                  </Alert>
                </SpaceBetween>
              ),
              isOptional: false
            },
            {
              title: "Review & Launch",
              description: "Review your configuration",
              content: (
                <SpaceBetween size="l">
                  <Container header={<Header variant="h3">Configuration Summary</Header>}>
                    <ColumnLayout columns={2} variant="text-grid">
                      <div>
                        <Box variant="awsui-key-label">Template</Box>
                        <Box>{quickStartConfig.selectedTemplate ? getTemplateName(quickStartConfig.selectedTemplate) : 'None'}</Box>
                      </div>
                      <div>
                        <Box variant="awsui-key-label">Workspace Name</Box>
                        <Box>{quickStartConfig.workspaceName || 'Not set'}</Box>
                      </div>
                      <div>
                        <Box variant="awsui-key-label">Size</Box>
                        <Box>{getSizeDescription(quickStartConfig.size)}</Box>
                      </div>
                      <div>
                        <Box variant="awsui-key-label">Estimated Cost</Box>
                        <Box>
                          {quickStartConfig.size === 'S' && '~$0.08/hour (~$58/month)'}
                          {quickStartConfig.size === 'M' && '~$0.16/hour (~$115/month)'}
                          {quickStartConfig.size === 'L' && '~$0.32/hour (~$230/month)'}
                          {quickStartConfig.size === 'XL' && '~$0.64/hour (~$460/month)'}
                        </Box>
                      </div>
                    </ColumnLayout>
                  </Container>

                  <Alert type="warning">
                    <strong>Cost Reminder:</strong> Remember to stop or hibernate your workspace when not in use to save costs.
                  </Alert>

                  {quickStartConfig.selectedTemplate && quickStartConfig.workspaceName && (
                    <Alert type="success">
                      ✅ Ready to launch! Click "Launch Workspace" to proceed.
                    </Alert>
                  )}
                </SpaceBetween>
              ),
              isOptional: false
            },
            {
              title: "Launch Progress",
              description: "Launching your workspace",
              content: (
                <SpaceBetween size="l">
                  {quickStartConfig.launchInProgress && (
                    <Box>
                      <ProgressBar value={50} description="Launching workspace..." />
                      <Box margin={{ top: "m" }} color="text-body-secondary">
                        This typically takes 2-3 minutes. Your workspace is being provisioned with all required software and configurations.
                      </Box>
                    </Box>
                  )}

                  {!quickStartConfig.launchInProgress && quickStartConfig.launchedWorkspaceId && (
                    <Alert type="success" header="Workspace Launched Successfully!">
                      <SpaceBetween size="m">
                        <Box>
                          Your workspace <strong>{quickStartConfig.workspaceName}</strong> is now running and ready to use.
                        </Box>
                        <Box>
                          <strong>Next Steps:</strong>
                          <ul>
                            <li>Connect via SSH or web interface from the Workspaces page</li>
                            <li>Access pre-installed software and tools</li>
                            <li>Remember to stop or hibernate when done to save costs</li>
                          </ul>
                        </Box>
                        <SpaceBetween direction="horizontal" size="s">
                          <Button
                            variant="primary"
                            onClick={() => {
                              setState(prev => ({ ...prev, activeView: 'instances' }));
                              handleWizardCancel();
                            }}
                          >
                            View Workspace
                          </Button>
                          <Button onClick={handleWizardCancel}>
                            Close
                          </Button>
                        </SpaceBetween>
                      </SpaceBetween>
                    </Alert>
                  )}

                  {!quickStartConfig.launchInProgress && !quickStartConfig.launchedWorkspaceId && (
                    <Alert type="info">
                      Click "Launch Workspace" to start the deployment process.
                    </Alert>
                  )}
                </SpaceBetween>
              ),
              isOptional: false
            }
          ]}
        />
      </Modal>
    );
  };

  // Main render
  return (
    <>
      <AppLayout
        navigationOpen={navigationOpen}
        onNavigationChange={({ detail }) => setNavigationOpen(detail.open)}
        navigation={
          <SideNavigation
            activeHref={`/${state.activeView}`}
            header={{ text: "Prism", href: "/" }}
            items={[
              {
                type: "link",
                text: "Dashboard",
                href: "/dashboard"
              },
              { type: "divider" },
              {
                type: "link",
                text: "Research Templates",
                href: "/templates",
                info: Object.keys(state.templates).length > 0 ?
                      <Badge color="blue">{Object.keys(state.templates).length}</Badge> : undefined
              },
              {
                type: "link",
                text: "My Workspaces",
                href: "/instances",
                info: state.instances.length > 0 ?
                      <Badge color={state.instances.some(i => i.state === 'running') ? 'green' : 'grey'}>
                        {state.instances.length}
                      </Badge> : undefined
              },
              {
                type: "link",
                text: "Terminal",
                href: "/terminal",
                info: state.instances.filter(i => i.state === 'running').length > 0 ?
                      <Badge color="green">SSH</Badge> : undefined
              },
              {
                type: "link",
                text: "Web Services",
                href: "/webview",
                info: state.instances.filter(i => i.state === 'running' && i.web_services && i.web_services.length > 0).length > 0 ?
                      <Badge color="blue">Available</Badge> : undefined
              },
              { type: "divider" },
              {
                type: "link",
                text: "Storage",
                href: "/storage"
              },
              {
                type: "link",
                text: "Projects",
                href: "/projects"
              },
              {
                type: "link",
                text: "Users",
                href: "/users"
              },
              { type: "divider" },
              {
                type: "link",
                text: "AMI Management",
                href: "/ami",
                info: <Badge>{state.amis.length} AMIs</Badge>
              },
              {
                type: "link",
                text: "Rightsizing",
                href: "/rightsizing",
                info: state.rightsizingRecommendations.length > 0 ?
                      <Badge color="green">{state.rightsizingRecommendations.length} recommendations</Badge> : undefined
              },
              {
                type: "link",
                text: "Policy Framework",
                href: "/policy",
                info: state.policyStatus?.enabled ?
                      <Badge color="green">Enforced</Badge> :
                      <Badge color="grey">Disabled</Badge>
              },
              {
                type: "link",
                text: "Template Marketplace",
                href: "/marketplace",
                info: state.marketplaceTemplates.length > 0 ?
                      <Badge color="blue">{state.marketplaceTemplates.length} templates</Badge> : undefined
              },
              {
                type: "link",
                text: "Idle Detection",
                href: "/idle",
                info: state.idlePolicies.filter(p => p.enabled).length > 0 ?
                      <Badge color="green">{state.idlePolicies.filter(p => p.enabled).length} active</Badge> : undefined
              },
              {
                type: "link",
                text: "Logs Viewer",
                href: "/logs"
              },
              { type: "divider" },
              {
                type: "link",
                text: "Settings",
                href: "/settings"
              }
            ]}
            onFollow={event => {
              if (!event.detail.external) {
                event.preventDefault();
                const view = event.detail.href.substring(1) as any;
                setState(prev => ({ ...prev, activeView: view || 'dashboard' }));
              }
            }}
          />
        }
        notifications={
          <Flashbar
            items={state.notifications}
            onDismiss={({ detail }) => {
              setState(prev => ({
                ...prev,
                notifications: prev.notifications.filter(item => item.id !== detail.id)
              }));
            }}
          />
        }
        content={
          <div id="main-content" role="main">
            {state.activeView === 'dashboard' && <DashboardView />}
            {state.activeView === 'templates' && <TemplateSelectionView />}
            {state.activeView === 'instances' && <InstanceManagementView />}
            <div style={{ display: state.activeView === 'terminal' ? 'block' : 'none' }}>
              {(() => {
                const runningInstances = state.instances.filter(i => i.state === 'running');

                if (runningInstances.length === 0) {
                  return (
                    <Container header={<Header variant="h1">SSH Terminal</Header>}>
                      <Alert type="info">
                        No running workspaces available. Launch a workspace to access the SSH terminal.
                      </Alert>
                    </Container>
                  );
                }

                return (
                  <SpaceBetween size="l">
                    <Container header={<Header variant="h1">SSH Terminal</Header>}>
                      <SpaceBetween size="m">
                        <FormField label="Select Workspace">
                          <Select
                            selectedOption={state.selectedTerminalInstance ? { label: state.selectedTerminalInstance, value: state.selectedTerminalInstance } : null}
                            onChange={({ detail }) => setState({ ...state, selectedTerminalInstance: detail.selectedOption.value || '' })}
                            options={runningInstances.map(i => ({ label: i.name, value: i.name }))}
                            placeholder="Choose a workspace"
                          />
                        </FormField>
                        {state.selectedTerminalInstance && <Terminal instanceName={state.selectedTerminalInstance} />}
                      </SpaceBetween>
                    </Container>
                  </SpaceBetween>
                );
              })()}
            </div>
            {state.activeView === 'webview' && <WebViewView />}
            {state.activeView === 'storage' && <StorageManagementView />}
            {state.activeView === 'projects' && <ProjectManagementView />}
            {state.activeView === 'project-detail' && <ProjectDetailView />}
            {state.activeView === 'users' && <UserManagementView />}
            {state.activeView === 'ami' && <AMIManagementView />}
            {state.activeView === 'rightsizing' && <RightsizingView />}
            {state.activeView === 'policy' && <PolicyView />}
            {state.activeView === 'marketplace' && <MarketplaceView />}
            {state.activeView === 'idle' && <IdleDetectionView />}
            {state.activeView === 'logs' && <LogsView />}
            {state.activeView === 'settings' && <SettingsView />}
          </div>
        }
        toolsHide
      />
      <LaunchModal />
      <DeleteConfirmationModal />
      <OnboardingWizard />
      <QuickStartWizard />
    </>
  );
}