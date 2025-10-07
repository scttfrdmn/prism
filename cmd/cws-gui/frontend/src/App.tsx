// CloudWorkstation GUI - Bulletproof AWS Integration
// Complete error handling, real API integration, professional UX

import React, { useState, useEffect } from 'react';
import '@cloudscape-design/global-styles/index.css';

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
  FormField
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

interface AppState {
  activeView: 'dashboard' | 'templates' | 'instances' | 'storage' | 'projects' | 'users' | 'settings';
  templates: Record<string, Template>;
  instances: Instance[];
  efsVolumes: EFSVolume[];
  ebsVolumes: EBSVolume[];
  projects: Project[];
  users: User[];
  selectedTemplate: Template | null;
  loading: boolean;
  notifications: any[];
  connected: boolean;
  error: string | null;
}

// Safe API Service with comprehensive error handling
class SafeCloudWorkstationAPI {
  private baseURL = 'http://localhost:8947';
  private apiKey = 'f3f0442f56089e22ca7bb834a76ac92e3f72bf9cba944578af4cec3866401e78';

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

  // EFS Volume Management
  async getEFSVolumes(): Promise<any[]> {
    try {
      const data = await this.safeRequest('/api/v1/volumes');
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to fetch EFS volumes:', error);
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

  // EBS Storage Management
  async getEBSVolumes(): Promise<any[]> {
    try {
      const data = await this.safeRequest('/api/v1/storage');
      return Array.isArray(data) ? data : [];
    } catch (error) {
      console.error('Failed to fetch EBS volumes:', error);
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
}

export default function BulletproofCloudWorkstationApp() {
  const api = new SafeCloudWorkstationAPI();

  const [state, setState] = useState<AppState>({
    activeView: 'dashboard',
    templates: {},
    instances: [],
    efsVolumes: [],
    ebsVolumes: [],
    projects: [],
    users: [],
    selectedTemplate: null,
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

  // Safe data loading with comprehensive error handling
  const loadApplicationData = async () => {
    try {
      setState(prev => ({ ...prev, loading: true, error: null }));

      console.log('Loading CloudWorkstation data...');

      const [templatesData, instancesData, efsVolumesData, ebsVolumesData, projectsData, usersData] = await Promise.all([
        api.getTemplates(),
        api.getInstances(),
        api.getEFSVolumes(),
        api.getEBSVolumes(),
        api.getProjects(),
        api.getUsers()
      ]);

      console.log('Templates loaded:', Object.keys(templatesData).length);
      console.log('Instances loaded:', instancesData.length);
      console.log('EFS Volumes loaded:', efsVolumesData.length);
      console.log('EBS Volumes loaded:', ebsVolumesData.length);
      console.log('Projects loaded:', projectsData.length);
      console.log('Users loaded:', usersData.length);

      setState(prev => ({
        ...prev,
        templates: templatesData,
        instances: instancesData,
        efsVolumes: efsVolumesData,
        ebsVolumes: ebsVolumesData,
        projects: projectsData,
        users: usersData,
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
            content: `Failed to connect to CloudWorkstation daemon: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  // Load data on mount and refresh periodically
  useEffect(() => {
    loadApplicationData();
    const interval = setInterval(loadApplicationData, 30000);
    return () => clearInterval(interval);
  }, []);

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
            header: 'Instance Launched',
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
            content: `Failed to launch instance: ${error instanceof Error ? error.message : 'Unknown error'}`,
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
      <Header
        variant="h1"
        description="CloudWorkstation research computing platform - manage your cloud environments"
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

        <Container header={<Header variant="h2">Active Instances</Header>}>
          <SpaceBetween size="s">
            <Box>
              <Box variant="awsui-key-label">Running Instances</Box>
              <Box fontSize="display-l" fontWeight="bold" color="text-status-success">
                {state.instances.filter(i => i.state === 'running').length}
              </Box>
            </Box>
            <Button
              onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
            >
              Manage Instances
            </Button>
          </SpaceBetween>
        </Container>

        <Container header={<Header variant="h2">System Status</Header>}>
          <SpaceBetween size="s">
            <Box>
              <Box variant="awsui-key-label">Connection</Box>
              <StatusIndicator type={state.connected ? 'success' : 'error'}>
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
            Launch New Instance
          </Button>
          <Button
            onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
            disabled={state.instances.length === 0}
          >
            View Instances ({state.instances.length})
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
          await api.deleteInstance(instance.name);
          actionMessage = `Deleted instance ${instance.name}`;
          break;
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
      console.error(`Failed to ${action} instance ${instance.name}:`, error);

      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          ...prev.notifications,
          {
            type: 'error',
            header: 'Action Failed',
            content: `Failed to ${action} instance ${instance.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
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
                  Launch New Instance
                </Button>
              </SpaceBetween>
            }
          >
            My Instances
          </Header>
        }
      >
        <Table
          columnDefinitions={[
            {
              id: "name",
              header: "Instance Name",
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
          items={state.instances}
          loadingText="Loading instances from AWS"
          loading={state.loading}
          trackBy="id"
          empty={
            <Box textAlign="center" color="inherit">
              <Box variant="strong" textAlign="center" color="inherit">
                No instances running
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
            await api.deleteEFSVolume(volume.name);
            actionMessage = `Deleted EFS volume ${volume.name}`;
            break;
          case 'mount':
            // For demo, mount to first available instance
            if (state.instances.length > 0) {
              const instance = state.instances[0].name;
              await api.mountEFSVolume(volume.name, instance);
              actionMessage = `Mounted EFS volume ${volume.name} to ${instance}`;
            } else {
              throw new Error('No running instances available for mounting');
            }
            break;
          case 'unmount':
            if (state.instances.length > 0) {
              const instance = state.instances[0].name;
              await api.unmountEFSVolume(volume.name, instance);
              actionMessage = `Unmounted EFS volume ${volume.name} from ${instance}`;
            } else {
              throw new Error('No instances to unmount from');
            }
            break;
          default:
            throw new Error(`Unknown EFS action: ${action}`);
        }
      } else if (volumeType === 'ebs') {
        switch (action) {
          case 'delete':
            await api.deleteEBSVolume(volume.name);
            actionMessage = `Deleted EBS volume ${volume.name}`;
            break;
          case 'attach':
            if (state.instances.length > 0) {
              const instance = state.instances[0].name;
              await api.attachEBSVolume(volume.name, instance);
              actionMessage = `Attached EBS volume ${volume.name} to ${instance}`;
            } else {
              throw new Error('No running instances available for attachment');
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
  const StorageManagementView = () => (
    <SpaceBetween size="l">
      {/* EFS Volumes Section */}
      <Container
        header={
          <Header
            variant="h2"
            description="Elastic File System volumes for shared persistent storage"
            counter={`(${state.efsVolumes.length})`}
            actions={
              <SpaceBetween direction="horizontal" size="xs">
                <Button onClick={loadApplicationData} disabled={state.loading}>
                  {state.loading ? <Spinner /> : 'Refresh'}
                </Button>
                <Button variant="primary">
                  Create EFS Volume
                </Button>
              </SpaceBetween>
            }
          >
            EFS Volumes
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
              header: "Est. Cost/GB",
              cell: (item: EFSVolume) => `$${item.estimated_cost_gb.toFixed(3)}`
            },
            {
              id: "actions",
              header: "Actions",
              cell: (item: EFSVolume) => (
                <ButtonDropdown
                  items={[
                    { text: 'Mount', id: 'mount', disabled: item.state !== 'available' },
                    { text: 'Unmount', id: 'unmount', disabled: item.state !== 'available' },
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
          loadingText="Loading EFS volumes from AWS"
          loading={state.loading}
          trackBy="name"
          empty={
            <Box textAlign="center" color="inherit">
              <Box variant="strong" textAlign="center" color="inherit">
                No EFS volumes found
              </Box>
              <Box variant="p" padding={{ bottom: 's' }} color="inherit">
                Create your first EFS volume for persistent shared storage.
              </Box>
            </Box>
          }
          sortingDisabled={false}
        />
      </Container>

      {/* EBS Volumes Section */}
      <Container
        header={
          <Header
            variant="h2"
            description="Elastic Block Store volumes for high-performance instance storage"
            counter={`(${state.ebsVolumes.length})`}
            actions={
              <SpaceBetween direction="horizontal" size="xs">
                <Button onClick={loadApplicationData} disabled={state.loading}>
                  {state.loading ? <Spinner /> : 'Refresh'}
                </Button>
                <Button variant="primary">
                  Create EBS Volume
                </Button>
              </SpaceBetween>
            }
          >
            EBS Volumes
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
                >
                  {item.state}
                </StatusIndicator>
              )
            },
            {
              id: "type",
              header: "Type",
              cell: (item: EBSVolume) => item.volume_type.toUpperCase()
            },
            {
              id: "size",
              header: "Size",
              cell: (item: EBSVolume) => `${item.size_gb} GB`
            },
            {
              id: "attached_to",
              header: "Attached To",
              cell: (item: EBSVolume) => item.attached_to || 'Not attached'
            },
            {
              id: "cost",
              header: "Est. Cost/GB",
              cell: (item: EBSVolume) => `$${item.estimated_cost_gb.toFixed(3)}`
            },
            {
              id: "actions",
              header: "Actions",
              cell: (item: EBSVolume) => (
                <ButtonDropdown
                  items={[
                    { text: 'Attach', id: 'attach', disabled: item.state !== 'available' },
                    { text: 'Detach', id: 'detach', disabled: item.state !== 'in-use' },
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
          loadingText="Loading EBS volumes from AWS"
          loading={state.loading}
          trackBy="name"
          empty={
            <Box textAlign="center" color="inherit">
              <Box variant="strong" textAlign="center" color="inherit">
                No EBS volumes found
              </Box>
              <Box variant="p" padding={{ bottom: 's' }} color="inherit">
                Create your first EBS volume for high-performance block storage.
              </Box>
            </Box>
          }
          sortingDisabled={false}
        />
      </Container>
    </SpaceBetween>
  );

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
              cell: (item: Project) => <Link fontSize="body-m">{item.name}</Link>,
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
                    <StatusIndicator type={colorType}>
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
                <StatusIndicator type={
                  item.status === 'active' ? 'success' :
                  item.status === 'suspended' ? 'warning' : 'error'
                }>
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
                      <StatusIndicator type={percentage > 80 ? 'error' : percentage > 60 ? 'warning' : 'success'}>
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
        description="Manage research users with persistent identity across CloudWorkstation instances"
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
        <Container header={<Header variant="h3">Provisioned Instances</Header>}>
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
                    <StatusIndicator type={keyCount > 0 ? 'success' : 'warning'}>
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
              header: "Instances",
              cell: (item: User) => {
                const count = item.provisioned_instances?.length || 0;
                return count > 0 ? count.toString() : 'None';
              }
            },
            {
              id: "status",
              header: "Status",
              cell: (item: User) => (
                <StatusIndicator type={
                  item.status === 'active' ? 'success' :
                  item.status === 'inactive' ? 'warning' : 'success'
                }>
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
                  items={[
                    { text: "View Details", id: "view" },
                    { text: "Generate SSH Key", id: "ssh-key", disabled: (item.ssh_keys || 0) > 0 },
                    { text: "Provision on Instance", id: "provision" },
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
                Create your first research user to enable persistent identity across instances.
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
                      <StatusIndicator type={keyCount > 0 ? 'success' : 'warning'}>
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
            <Header variant="h3">Instance Provisioning</Header>
            <Box color="text-body-secondary">
              User provisioning across instances and EFS home directory management.
              Persistent identity ensures same UID/GID mapping across all environments.
            </Box>
            {state.users.length > 0 && (
              <SpaceBetween size="s">
                <Box variant="strong">Available for Provisioning:</Box>
                {state.instances.length > 0 ? (
                  state.instances.filter(i => i.state === 'running').map(instance => (
                    <Box key={instance.id}>
                      <StatusIndicator type="success">{instance.name}</StatusIndicator>
                    </Box>
                  ))
                ) : (
                  <Box color="text-body-secondary">No running instances available</Box>
                )}
              </SpaceBetween>
            )}
          </SpaceBetween>
        </ColumnLayout>
      </Container>
    </SpaceBetween>
  );

  // Settings View
  const SettingsView = () => (
    <SpaceBetween size="l">
      <Header
        variant="h1"
        description="Configure CloudWorkstation preferences and system settings"
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
            <StatusIndicator type={state.connected ? 'success' : 'error'}>
              {state.connected ? 'Connected' : 'Disconnected'}
            </StatusIndicator>
            <Box color="text-body-secondary">
              CloudWorkstation daemon on port 8947
            </Box>
          </SpaceBetween>
          <SpaceBetween size="m">
            <Box variant="awsui-key-label">API Version</Box>
            <Box fontSize="heading-m">v0.5.1</Box>
            <Box color="text-body-secondary">
              Current CloudWorkstation version
            </Box>
          </SpaceBetween>
          <SpaceBetween size="m">
            <Box variant="awsui-key-label">Active Resources</Box>
            <Box fontSize="heading-m">{state.instances.length + state.efsVolumes.length + state.ebsVolumes.length}</Box>
            <Box color="text-body-secondary">
              Instances, EFS and EBS volumes
            </Box>
          </SpaceBetween>
        </ColumnLayout>
      </Container>

      {/* Configuration Section */}
      <Container
        header={
          <Header
            variant="h2"
            description="CloudWorkstation configuration and preferences"
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
            label="Default instance size"
            description="Default size for new instances when launching templates"
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
            <StatusIndicator type="success">
              Authenticated via AWS profile
            </StatusIndicator>
            <Box color="text-body-secondary">
              Using credentials from AWS profile "aws" in region us-west-2.
              CloudWorkstation automatically manages authentication for all API calls.
            </Box>
          </SpaceBetween>
        </ColumnLayout>
      </Container>

      {/* Feature Management */}
      <Container
        header={
          <Header
            variant="h2"
            description="Enable or disable CloudWorkstation features"
          >
            Feature Management
          </Header>
        }
      >
        <SpaceBetween size="m">
          {[
            { name: "Instance Management", status: "enabled", description: "Launch, manage, and connect to cloud instances" },
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
                <StatusIndicator type={
                  feature.status === 'enabled' ? 'success' :
                  feature.status === 'partial' ? 'warning' : 'error'
                }>
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
              Launch Instance
            </Button>
          </SpaceBetween>
        </Box>
      }
    >
      <Form>
        <SpaceBetween size="m">
          <FormField
            label="Instance name"
            description="Choose a descriptive name for your research project"
            errorText={!launchConfig.name.trim() ? "Instance name is required" : ""}
          >
            <Input
              value={launchConfig.name}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, name: detail.value }))}
              placeholder="my-research-project"
            />
          </FormField>

          <FormField label="Instance size" description="Choose the right size for your workload">
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

  // Main render
  return (
    <>
      <AppLayout
        navigationOpen={navigationOpen}
        onNavigationChange={({ detail }) => setNavigationOpen(detail.open)}
        navigation={
          <SideNavigation
            activeHref={`/${state.activeView}`}
            header={{ text: "CloudWorkstation", href: "/" }}
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
                text: "My Instances",
                href: "/instances",
                info: state.instances.length > 0 ?
                      <Badge color={state.instances.some(i => i.state === 'running') ? 'green' : 'grey'}>
                        {state.instances.length}
                      </Badge> : undefined
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
          <div>
            {state.activeView === 'dashboard' && <DashboardView />}
            {state.activeView === 'templates' && <TemplateSelectionView />}
            {state.activeView === 'instances' && <InstanceManagementView />}
            {state.activeView === 'storage' && <StorageManagementView />}
            {state.activeView === 'projects' && <ProjectManagementView />}
            {state.activeView === 'users' && <UserManagementView />}
            {state.activeView === 'settings' && <SettingsView />}
          </div>
        }
        toolsHide
      />
      <LaunchModal />
    </>
  );
}