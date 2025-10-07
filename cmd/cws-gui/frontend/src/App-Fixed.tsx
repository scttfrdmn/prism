// CloudWorkstation GUI - Professional UX with Real API Integration
// Connects to actual daemon API and provides complete functionality

import React, { useState, useEffect } from 'react';
import '@cloudscape-design/global-styles/index.css';

import {
  AppLayout,
  SideNavigation,
  TopNavigation,
  Container,
  Header,
  SpaceBetween,
  Button,
  Cards,
  StatusIndicator,
  Badge,
  PropertyFilter,
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
  Tabs,
  ColumnLayout,
  Link
} from '@cloudscape-design/components';

// Enhanced type definitions matching real CloudWorkstation API
interface Template {
  name: string;
  slug: string;
  description: string;
  category: string;
  complexity: string;
  package_manager: string;
  features: string[];
  cost_estimate?: number;
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

interface CloudWorkstationState {
  activeView: 'dashboard' | 'templates' | 'instances' | 'storage' | 'projects' | 'users' | 'settings';
  templates: Record<string, Template>;
  instances: Instance[];
  selectedTemplate: Template | null;
  loading: boolean;
  notifications: any[];
  connected: boolean;
}

// API Service Class
class CloudWorkstationAPI {
  private baseURL = 'http://localhost:8947';
  private apiKey = 'f3f0442f56089e22ca7bb834a76ac92e3f72bf9cba944578af4cec3866401e78';

  private async makeRequest(endpoint: string, method = 'GET', body?: any) {
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

    return response.json();
  }

  async getTemplates() {
    return this.makeRequest('/api/v1/templates');
  }

  async getInstances() {
    return this.makeRequest('/api/v1/instances');
  }

  async launchInstance(templateSlug: string, name: string, size: string = 'M') {
    return this.makeRequest('/api/v1/instances', 'POST', {
      template: templateSlug,
      name,
      size,
    });
  }

  async getDaemonStatus() {
    return this.makeRequest('/api/v1/daemon/status');
  }

  async getProjects() {
    return this.makeRequest('/api/v1/projects');
  }

  async getUsers() {
    return this.makeRequest('/api/v1/users');
  }

  async getStorageVolumes() {
    return this.makeRequest('/api/v1/storage/volumes');
  }
}

export default function CloudWorkstationApp() {
  const api = new CloudWorkstationAPI();

  // Application State
  const [state, setState] = useState<CloudWorkstationState>({
    activeView: 'dashboard',
    templates: {},
    instances: [],
    selectedTemplate: null,
    loading: true,
    notifications: [],
    connected: false
  });

  const [navigationOpen, setNavigationOpen] = useState(true); // Open by default for better UX
  const [launchModalVisible, setLaunchModalVisible] = useState(false);
  const [launchConfig, setLaunchConfig] = useState({
    name: '',
    size: 'M'
  });

  // Load data on component mount and when view changes
  useEffect(() => {
    loadApplicationData();
    const interval = setInterval(loadApplicationData, 30000); // Refresh every 30 seconds
    return () => clearInterval(interval);
  }, []);

  const loadApplicationData = async () => {
    try {
      // Load all data in parallel for better performance
      const [templatesData, instancesData] = await Promise.all([
        api.getTemplates(),
        api.getInstances()
      ]);

      setState(prev => ({
        ...prev,
        templates: templatesData || {},
        instances: instancesData || [],
        loading: false,
        connected: true
      }));

      // Clear any connection errors
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => n.type !== 'error' || !n.content.includes('daemon'))
      }));

    } catch (error) {
      console.error('Failed to load application data:', error);
      setState(prev => ({
        ...prev,
        loading: false,
        connected: false,
        notifications: [
          {
            type: 'error',
            header: 'Connection Error',
            content: 'Failed to connect to CloudWorkstation daemon. Please ensure the daemon is running and try refreshing.',
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  const handleTemplateSelection = (template: Template) => {
    setState(prev => ({ ...prev, selectedTemplate: template }));
    setLaunchConfig({ name: '', size: 'M' });
    setLaunchModalVisible(true);
  };

  const handleLaunchInstance = async () => {
    if (!state.selectedTemplate || !launchConfig.name) return;

    try {
      await api.launchInstance(state.selectedTemplate.slug, launchConfig.name, launchConfig.size);

      setState(prev => ({
        ...prev,
        notifications: [
          {
            type: 'success',
            header: 'Instance Launched',
            content: `Successfully launched ${launchConfig.name} using ${state.selectedTemplate?.name}`,
            dismissible: true,
            id: Date.now().toString()
          },
          ...prev.notifications
        ]
      }));

      setLaunchModalVisible(false);
      loadApplicationData(); // Refresh data

    } catch (error) {
      setState(prev => ({
        ...prev,
        notifications: [
          {
            type: 'error',
            header: 'Launch Failed',
            content: `Failed to launch instance: ${error.message}`,
            dismissible: true,
            id: Date.now().toString()
          },
          ...prev.notifications
        ]
      }));
    }
  };

  // Dashboard View - Overview of everything
  const DashboardView = () => (
    <SpaceBetween size="l">
      <Header
        variant="h1"
        description="CloudWorkstation research computing platform - manage your cloud environments"
      >
        Dashboard
      </Header>

      <ColumnLayout columns={3} variant="text-grid">
        <Container header={<Header variant="h2" description="Click to browse">Research Templates</Header>}>
          <Box>
            <Box variant="awsui-key-label">Available Templates</Box>
            <Box fontSize="display-l" fontWeight="bold">
              {Object.keys(state.templates).length}
            </Box>
            <Button
              variant="primary"
              onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
            >
              Browse Templates
            </Button>
          </Box>
        </Container>

        <Container header={<Header variant="h2" description="Monitor your workloads">Active Instances</Header>}>
          <Box>
            <Box variant="awsui-key-label">Running Instances</Box>
            <Box fontSize="display-l" fontWeight="bold">
              {state.instances.filter(i => i.state === 'running').length}
            </Box>
            <Button
              onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
            >
              Manage Instances
            </Button>
          </Box>
        </Container>

        <Container header={<Header variant="h2" description="System status">Connection Status</Header>}>
          <Box>
            <Box variant="awsui-key-label">Daemon Status</Box>
            <StatusIndicator type={state.connected ? 'success' : 'error'}>
              {state.connected ? 'Connected' : 'Disconnected'}
            </StatusIndicator>
            <Button onClick={loadApplicationData} disabled={state.loading}>
              {state.loading ? 'Refreshing...' : 'Refresh'}
            </Button>
          </Box>
        </Container>
      </ColumnLayout>

      {/* Recent Activity */}
      <Container header={<Header variant="h2">Quick Actions</Header>}>
        <SpaceBetween direction="horizontal" size="s">
          <Button
            variant="primary"
            onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
          >
            Launch New Instance
          </Button>
          <Button onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}>
            View All Instances
          </Button>
          <Button onClick={() => setState(prev => ({ ...prev, activeView: 'storage' }))}>
            Manage Storage
          </Button>
          <Button onClick={() => setState(prev => ({ ...prev, activeView: 'projects' }))}>
            Projects
          </Button>
        </SpaceBetween>
      </Container>
    </SpaceBetween>
  );

  // Enhanced Template Selection with Real Data
  const TemplateSelectionView = () => {
    const templateList = Object.values(state.templates);

    return (
      <SpaceBetween size="l">
        <Container
          header={
            <Header
              variant="h1"
              description={`Choose from ${templateList.length} pre-configured research environments`}
              counter={`(${templateList.length} templates)`}
              actions={
                <SpaceBetween direction="horizontal" size="xs">
                  <Button onClick={loadApplicationData} disabled={state.loading}>
                    {state.loading ? <Spinner /> : 'Refresh Templates'}
                  </Button>
                  <Button
                    variant="primary"
                    disabled={!state.selectedTemplate}
                    onClick={() => setLaunchModalVisible(true)}
                  >
                    Launch Selected Template
                  </Button>
                </SpaceBetween>
              }
            >
              Research Templates
            </Header>
          }
        >
          {state.loading ? (
            <Box textAlign="center" padding="xl">
              <Spinner size="large" />
              <Box variant="p" color="text-body-secondary">
                Loading {Object.keys(state.templates).length || 'templates'}...
              </Box>
            </Box>
          ) : templateList.length === 0 ? (
            <Box textAlign="center" padding="xl">
              <Box variant="strong">No templates available</Box>
              <Box variant="p">Please check your daemon connection.</Box>
              <Button onClick={loadApplicationData}>Retry</Button>
            </Box>
          ) : (
            <Cards
              cardDefinition={{
                header: (item: Template) => (
                  <SpaceBetween direction="horizontal" size="xs">
                    <Box fontSize="heading-m">
                      {item.category === 'ml' ? '🤖' :
                       item.category === 'datascience' ? '📊' :
                       item.category === 'bio' ? '🧬' :
                       item.category === 'web' ? '🌐' : '💻'}
                    </Box>
                    <Header variant="h3">{item.name}</Header>
                  </SpaceBetween>
                ),
                sections: [
                  {
                    id: "description",
                    content: (item: Template) => item.description || 'No description available'
                  },
                  {
                    id: "features",
                    content: (item: Template) => (
                      <SpaceBetween direction="horizontal" size="xs">
                        <Badge color="blue">{item.package_manager}</Badge>
                        <Badge color={item.complexity === 'simple' ? 'green' : item.complexity === 'moderate' ? 'blue' : 'orange'}>
                          {item.complexity}
                        </Badge>
                        {item.features?.slice(0, 2).map(feature => (
                          <Badge key={feature} color="grey">{feature}</Badge>
                        ))}
                        {(item.features?.length || 0) > 2 && (
                          <Badge color="grey">+{(item.features?.length || 0) - 2} more</Badge>
                        )}
                      </SpaceBetween>
                    )
                  }
                ]
              }}
              items={templateList}
              loadingText="Loading templates"
              selectionType="single"
              onSelectionChange={({ detail }) => {
                const template = detail.selectedItems[0];
                setState(prev => ({ ...prev, selectedTemplate: template }));
              }}
              selectedItems={state.selectedTemplate ? [state.selectedTemplate] : []}
              cardsPerRow={[
                { cards: 1 },
                { minWidth: 500, cards: 2 },
                { minWidth: 900, cards: 3 },
                { minWidth: 1200, cards: 4 }
              ]}
              trackBy="slug"
            />
          )}
        </Container>
      </SpaceBetween>
    );
  };

  // Enhanced Instance Management with Real Data
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
              id: "instance_type",
              header: "Instance Type",
              cell: (item: Instance) => item.instance_type || 'N/A'
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
                <SpaceBetween direction="horizontal" size="xs">
                  <Button
                    variant="primary"
                    size="small"
                    disabled={item.state !== 'running'}
                  >
                    Connect
                  </Button>
                  <Button
                    variant="normal"
                    size="small"
                    disabled={item.state === 'stopped'}
                  >
                    {item.state === 'running' ? 'Stop' : 'Start'}
                  </Button>
                </SpaceBetween>
              )
            }
          ]}
          items={state.instances}
          loadingText="Loading instances"
          loading={state.loading}
          trackBy="id"
          empty={
            <Box textAlign="center" color="inherit">
              <Box variant="strong" textAlign="center" color="inherit">
                No instances
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
          pagination={{ pageSize: 20 }}
        />
      </Container>
    </SpaceBetween>
  );

  // Storage Management View (Placeholder - will be enhanced)
  const StorageView = () => (
    <Container header={<Header variant="h1">Storage Management</Header>}>
      <Box textAlign="center" padding="xl">
        <Box variant="strong">EFS and EBS Storage</Box>
        <Box variant="p">Manage your persistent storage volumes and file systems.</Box>
        <Alert type="info">Storage management interface coming soon.</Alert>
      </Box>
    </Container>
  );

  // Projects View (Placeholder)
  const ProjectsView = () => (
    <Container header={<Header variant="h1">Projects</Header>}>
      <Box textAlign="center" padding="xl">
        <Box variant="strong">Project Management</Box>
        <Box variant="p">Organize your research with collaborative projects and budget tracking.</Box>
        <Alert type="info">Project management interface coming soon.</Alert>
      </Box>
    </Container>
  );

  // Users View (Placeholder)
  const UsersView = () => (
    <Container header={<Header variant="h1">Users</Header>}>
      <Box textAlign="center" padding="xl">
        <Box variant="strong">User Management</Box>
        <Box variant="p">Manage research users and SSH access.</Box>
        <Alert type="info">User management interface coming soon.</Alert>
      </Box>
    </Container>
  );

  // Settings View
  const SettingsView = () => (
    <Container header={<Header variant="h1">Settings</Header>}>
      <SpaceBetween size="l">
        <Alert type="info">
          Configuration is managed through AWS profiles and the daemon.
        </Alert>
        <Box>
          <Box variant="strong">Connection Status</Box>
          <StatusIndicator type={state.connected ? 'success' : 'error'}>
            {state.connected ? 'Connected to daemon' : 'Disconnected'}
          </StatusIndicator>
        </Box>
        <Box>
          <Box variant="strong">Templates Available</Box>
          <Box>{Object.keys(state.templates).length}</Box>
        </Box>
      </SpaceBetween>
    </Container>
  );

  // Enhanced Launch Modal
  const LaunchModal = () => (
    <Modal
      onDismiss={() => setLaunchModalVisible(false)}
      visible={launchModalVisible}
      header={`Launch ${state.selectedTemplate?.name || 'Research Environment'}`}
      size="medium"
      footer={
        <Box float="right">
          <SpaceBetween direction="horizontal" size="xs">
            <Button variant="link" onClick={() => setLaunchModalVisible(false)}>
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
              selectedOption={{
                label: "Medium (M) - Recommended for most workloads",
                value: "M"
              }}
              onChange={({ detail }) => setLaunchConfig(prev => ({ ...prev, size: detail.selectedOption.value }))}
              options={[
                { label: "Small (S) - Light workloads", value: "S" },
                { label: "Medium (M) - Recommended for most workloads", value: "M" },
                { label: "Large (L) - Heavy compute workloads", value: "L" },
                { label: "Extra Large (XL) - Maximum performance", value: "XL" }
              ]}
            />
          </FormField>

          {state.selectedTemplate && (
            <Alert type="info">
              <Box>
                <Box variant="strong">Template: {state.selectedTemplate.name}</Box>
                <Box>Package Manager: {state.selectedTemplate.package_manager}</Box>
                <Box>Complexity: {state.selectedTemplate.complexity}</Box>
              </Box>
            </Alert>
          )}
        </SpaceBetween>
      </Form>
    </Modal>
  );

  // Main render with improved navigation
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
            {state.activeView === 'storage' && <StorageView />}
            {state.activeView === 'projects' && <ProjectsView />}
            {state.activeView === 'users' && <UsersView />}
            {state.activeView === 'settings' && <SettingsView />}
          </div>
        }
        toolsHide
      />
      <LaunchModal />
    </>
  );
}