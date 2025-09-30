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
  BreadcrumbGroup,
  SplitPanel,
  Tabs,
  ProgressBar,
  Link
} from '@cloudscape-design/components';

// Enhanced type definitions for CloudWorkstation
interface Template {
  Name: string;
  Description?: string;
  Category?: string;
  Domain?: string;
  Complexity?: 'simple' | 'moderate' | 'advanced' | 'complex';
  Icon?: string;
  Popular?: boolean;
  EstimatedLaunchTime?: number;
  EstimatedCostPerHour?: { [key: string]: number };
  ValidationStatus?: string;
  ResearchUser?: {
    AutoCreate?: boolean;
    RequireEFS?: boolean;
    EFSMountPoint?: string;
    InstallSSHKeys?: boolean;
    DefaultShell?: string;
    DefaultGroups?: string[];
    DualUserIntegration?: {
      Strategy?: string;
      PrimaryUser?: string;
      CollaborationEnabled?: boolean;
    };
  };
}

interface Instance {
  id: string;
  name: string;
  template: string;
  status: 'running' | 'stopped' | 'hibernated' | 'pending' | 'stopping';
  public_ip?: string;
  cost_per_hour: number;
  launch_time: string;
  region: string;
}

interface Volume {
  name: string;
  id: string;
  state: 'available' | 'creating' | 'deleting';
  size_gb: number;
  mount_targets: string[];
  cost_per_gb: number;
  creation_time: string;
  region: string;
}

interface ResearchUser {
  username: string;
  full_name: string;
  email: string;
  uid: number;
  gid: number;
  home_directory: string;
  shell: string;
  sudo_access: boolean;
  docker_access: boolean;
  ssh_public_keys: string[];
  created_at: string;
}

// Enhanced connection types for tabbed embedded connections
interface ConnectionConfig {
  id: string;
  type: 'ssh' | 'desktop' | 'web' | 'aws-service';
  instanceName?: string;
  awsService?: string;
  region?: string;
  proxyUrl: string;
  authToken?: string;
  embeddingMode: 'iframe' | 'websocket' | 'api';
  title: string;
  status: 'connecting' | 'connected' | 'disconnected' | 'error';
  metadata?: Record<string, any>;
}

interface ConnectionTab {
  id: string;
  title: string;
  type: 'instance' | 'aws-service';
  category: 'compute' | 'research' | 'analytics' | 'management';
  config: ConnectionConfig;
  active: boolean;
  closeable: boolean;
  status: 'connecting' | 'connected' | 'disconnected' | 'error';
}

interface Notification {
  type: 'success' | 'error' | 'warning' | 'info';
  header: string;
  content: string;
  dismissible?: boolean;
  buttonText?: string;
  onButtonClick?: () => void;
  loading?: boolean;
  id?: string;
}

interface CloudWorkstationState {
  activeView: 'templates' | 'instances' | 'volumes' | 'research-users' | 'connections' | 'settings';
  templates: Template[];
  instances: Instance[];
  volumes: Volume[];
  researchUsers: ResearchUser[];
  selectedTemplate: Template | null;
  selectedVolume: Volume | null;
  selectedResearchUser: ResearchUser | null;
  loading: boolean;
  notifications: Notification[];
  splitPanelOpen: boolean;
  splitPanelContent: 'instance-details' | 'template-details' | 'volume-details' | 'research-user-details' | null;
  showMountDialog: boolean;
  mountingVolume: Volume | null;

  // Enhanced connection state
  connectionTabs: ConnectionTab[];
  activeConnectionTab: string | null;
  showConnectionPanel: boolean;
}

// Enhanced Wails API for TypeScript with embedded connections
declare global {
  interface Window {
    wails: {
      CloudWorkstationService: {
        // Existing API methods
        GetTemplates: () => Promise<Template[]>;
        GetInstances: () => Promise<Instance[]>;
        GetVolumes: () => Promise<Volume[]>;
        LaunchInstance: (name: string, templateName: string, size: string) => Promise<void>;
        MountVolume: (volumeName: string, instanceName: string, mountPoint: string) => Promise<void>;
        UnmountVolume: (volumeName: string, instanceName: string) => Promise<void>;

        // Enhanced embedded connection methods
        OpenEmbeddedTerminal: (instanceName: string) => Promise<ConnectionConfig>;
        OpenEmbeddedDesktop: (instanceName: string) => Promise<ConnectionConfig>;
        OpenEmbeddedWeb: (instanceName: string) => Promise<ConnectionConfig>;

        // AWS service connection methods
        OpenBraketConsole: (region: string) => Promise<ConnectionConfig>;
        OpenSageMakerStudio: (region: string) => Promise<ConnectionConfig>;
        OpenAWSConsole: (service: string, region: string) => Promise<ConnectionConfig>;
        OpenCloudShell: (region: string) => Promise<ConnectionConfig>;
        OpenAWSService: (service: string, region: string) => Promise<ConnectionConfig>;
      };
    };
  }
}

export default function CloudWorkstationApp() {
  // Application State
  const [state, setState] = useState<CloudWorkstationState>({
    activeView: 'templates',
    templates: [],
    instances: [],
    volumes: [],
    researchUsers: [],
    selectedTemplate: null,
    selectedVolume: null,
    selectedResearchUser: null,
    loading: true,
    notifications: [],
    splitPanelOpen: false,
    splitPanelContent: null,
    showMountDialog: false,
    mountingVolume: null,

    // Enhanced connection state
    connectionTabs: [],
    activeConnectionTab: null,
    showConnectionPanel: false
  });

  const [navigationOpen, setNavigationOpen] = useState(false);
  const [launchModalVisible, setLaunchModalVisible] = useState(false);
  const [createUserModalVisible, setCreateUserModalVisible] = useState(false);
  const [newUsername, setNewUsername] = useState('');
  const [instanceName, setInstanceName] = useState('');
  const [instanceSize, setInstanceSize] = useState('M');
  const [templateQuery, setTemplateQuery] = useState({ tokens: [], operation: 'and' as const });
  const [filteredTemplates, setFilteredTemplates] = useState<Template[]>([]);
  const [selectedInstances, setSelectedInstances] = useState<Instance[]>([]);
  const [selectedInstance, setSelectedInstance] = useState<Instance | null>(null);
  const [selectedVolumes, setSelectedVolumes] = useState<Volume[]>([]);
  const [mountInstanceName, setMountInstanceName] = useState('');
  const [mountPoint, setMountPoint] = useState('/mnt/shared-volume');

  // Breadcrumb navigation helper
  const getBreadcrumbs = () => {
    const items = [
      { text: 'CloudWorkstation', href: '#/' }
    ];

    switch (state.activeView) {
      case 'templates':
        items.push({ text: 'Research Templates', href: '#/templates' });
        break;
      case 'instances':
        items.push({ text: 'Instances', href: '#/instances' });
        if (selectedInstance) {
          items.push({ text: selectedInstance.name, href: `#/instances/${selectedInstance.id}` });
        }
        break;
      case 'volumes':
        items.push({ text: 'Storage Volumes', href: '#/volumes' });
        break;
      case 'research-users':
        items.push({ text: 'Research Users', href: '#/research-users' });
        break;
      case 'connections':
        items.push({ text: 'Active Connections', href: '#/connections' });
        break;
      case 'settings':
        items.push({ text: 'Settings', href: '#/settings' });
        break;
      default:
        break;
    }

    return items;
  };

  // Enhanced notification helper
  const addNotification = (notification: Omit<Notification, 'id'>) => {
    const id = Math.random().toString(36).substr(2, 9);
    setState(prev => ({
      ...prev,
      notifications: [...prev.notifications, { ...notification, id }]
    }));

    // Auto-dismiss success notifications after 5 seconds
    if (notification.type === 'success' && notification.dismissible !== false) {
      setTimeout(() => {
        setState(prev => ({
          ...prev,
          notifications: prev.notifications.filter(n => n.id !== id)
        }));
      }, 5000);
    }
  };

  // Load data on component mount
  useEffect(() => {
    loadApplicationData();
  }, []);

  // Update filtered templates when query or templates change
  useEffect(() => {
    if (state.templates.length === 0) {
      setFilteredTemplates([]);
      return;
    }

    let filtered = [...state.templates];

    // Apply PropertyFilter query
    if (templateQuery.tokens.length > 0) {
      filtered = filtered.filter(template => {
        return templateQuery.tokens.every(token => {
          const property = token.propertyKey;
          const value = token.value.toLowerCase();
          const operator = token.operator;

          let templateValue = '';
          switch (property) {
            case 'name':
              templateValue = template.Name.toLowerCase();
              break;
            case 'domain':
              templateValue = (template.Domain || 'base').toLowerCase();
              break;
            case 'complexity':
              templateValue = (template.Complexity || 'simple').toLowerCase();
              break;
            default:
              return true;
          }

          switch (operator) {
            case ':':
              return templateValue.includes(value);
            case '!:':
              return !templateValue.includes(value);
            case '=':
              return templateValue === value;
            case '!=':
              return templateValue !== value;
            default:
              return true;
          }
        });
      });
    }

    setFilteredTemplates(filtered);
  }, [state.templates, templateQuery]);

  const loadApplicationData = async () => {
    setState(prev => ({ ...prev, loading: true }));

    try {
      // Load templates from backend via Wails
      let templates: Template[] = [];
      if (window.wails?.CloudWorkstationService?.GetTemplates) {
        templates = await window.wails.CloudWorkstationService.GetTemplates();
      } else {
        // Fallback mock data for development
        templates = [
          {
            Name: 'Python Machine Learning',
            Description: 'Complete ML environment with TensorFlow, PyTorch, and Jupyter',
            Category: 'Machine Learning',
            Domain: 'ml',
            Complexity: 'moderate',
            Icon: '🤖',
            Popular: true,
            EstimatedLaunchTime: 2,
            EstimatedCostPerHour: { 'x86_64': 0.48 },
            ValidationStatus: 'validated'
          },
          {
            Name: 'R Research Environment',
            Description: 'Statistical computing with R, RStudio, and tidyverse packages',
            Category: 'Data Science',
            Domain: 'datascience',
            Complexity: 'simple',
            Icon: '📊',
            Popular: true,
            EstimatedLaunchTime: 3,
            EstimatedCostPerHour: { 'x86_64': 0.24 },
            ValidationStatus: 'validated'
          },
          {
            Name: 'Rocky Linux 9 Base',
            Description: 'Enterprise Linux foundation for custom research environments',
            Category: 'Base System',
            Domain: 'base',
            Complexity: 'simple',
            Icon: '🖥️',
            Popular: false,
            EstimatedLaunchTime: 1,
            EstimatedCostPerHour: { 'x86_64': 0.12 },
            ValidationStatus: 'validated'
          }
        ];
      }

      // Load volumes
      let volumes: Volume[] = [];
      if (window.wails?.CloudWorkstationService?.GetVolumes) {
        volumes = await window.wails.CloudWorkstationService.GetVolumes();
      } else {
        // Enhanced mock data for development
        volumes = [
          {
            name: 'shared-research-data',
            id: 'fs-1234567890abcdef0',
            state: 'available',
            size_gb: 100,
            mount_targets: ['my-ml-research', 'data-analysis-project'],
            cost_per_gb: 0.30,
            creation_time: '2025-09-20T08:00:00Z',
            region: 'us-west-2'
          },
          {
            name: 'backup-storage',
            id: 'fs-0987654321fedcba1',
            state: 'available',
            size_gb: 50,
            mount_targets: [],
            cost_per_gb: 0.30,
            creation_time: '2025-09-25T12:30:00Z',
            region: 'us-west-2'
          },
          {
            name: 'project-archives',
            id: 'fs-abcdef1234567890',
            state: 'creating',
            size_gb: 200,
            mount_targets: [],
            cost_per_gb: 0.30,
            creation_time: '2025-09-28T10:45:00Z',
            region: 'us-east-1'
          }
        ];
      }

      // Load instances
      let instances: Instance[] = [];
      if (window.wails?.CloudWorkstationService?.GetInstances) {
        instances = await window.wails.CloudWorkstationService.GetInstances();
      } else {
        // Enhanced mock data for development
        instances = [
          {
            id: 'i-1234567890abcdef0',
            name: 'my-ml-research',
            template: 'Python Machine Learning',
            status: 'running',
            public_ip: '54.123.45.67',
            cost_per_hour: 0.48,
            launch_time: '2025-09-28T10:30:00Z',
            region: 'us-west-2'
          },
          {
            id: 'i-0987654321fedcba1',
            name: 'data-analysis-project',
            template: 'R Research Environment',
            status: 'hibernated',
            cost_per_hour: 0.24,
            launch_time: '2025-09-27T14:15:00Z',
            region: 'us-west-2'
          },
          {
            id: 'i-abcdef1234567890',
            name: 'web-dev-staging',
            template: 'Basic Ubuntu (APT)',
            status: 'stopped',
            cost_per_hour: 0.12,
            launch_time: '2025-09-26T09:45:00Z',
            region: 'us-east-1'
          },
          {
            id: 'i-fedcba0987654321',
            name: 'gpu-training-cluster',
            template: 'Python Machine Learning',
            status: 'pending',
            cost_per_hour: 1.44,
            launch_time: '2025-09-28T11:00:00Z',
            region: 'us-west-2'
          }
        ];
      }

      // Load research users
      let researchUsers: ResearchUser[] = [];
      if (window.go?.main?.CloudWorkstationService?.GetResearchUsers) {
        researchUsers = await window.go.main.CloudWorkstationService.GetResearchUsers(window.go.context.Context());
      } else {
        // Mock data for development
        researchUsers = [
          {
            username: 'alice-researcher',
            full_name: 'Alice Johnson',
            email: 'alice@university.edu',
            uid: 5001,
            gid: 5001,
            home_directory: '/home/alice-researcher',
            shell: '/bin/bash',
            sudo_access: true,
            docker_access: false,
            ssh_public_keys: ['ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...'],
            created_at: '2025-09-20T10:30:00Z'
          },
          {
            username: 'bob-datascience',
            full_name: 'Bob Smith',
            email: 'bob@research.org',
            uid: 5002,
            gid: 5002,
            home_directory: '/home/bob-datascience',
            shell: '/bin/bash',
            sudo_access: false,
            docker_access: true,
            ssh_public_keys: ['ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACA...', 'ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI...'],
            created_at: '2025-09-25T14:15:00Z'
          }
        ];
      }

      setState(prev => ({
        ...prev,
        templates,
        instances,
        volumes,
        researchUsers,
        loading: false
      }));

      // Note: filteredTemplates will be updated by useEffect when state.templates changes
    } catch (error) {
      console.error('Failed to load application data:', error);
      setState(prev => ({
        ...prev,
        loading: false
      }));

      addNotification({
        type: 'error',
        header: 'Failed to load data',
        content: 'Unable to connect to CloudWorkstation daemon. Please ensure the daemon is running.',
        dismissible: true,
        buttonText: 'Retry',
        onButtonClick: loadApplicationData
      });
    }
  };

  // Template filtering properties for PropertyFilter
  const templateFilteringProperties = [
    {
      key: 'name',
      operators: [':', '!:', '=', '!='],
      propertyLabel: 'Name',
      groupValuesLabel: 'Template names'
    },
    {
      key: 'domain',
      operators: [':', '!:', '=', '!='],
      propertyLabel: 'Domain',
      groupValuesLabel: 'Research domains'
    },
    {
      key: 'complexity',
      operators: [':', '!:', '=', '!='],
      propertyLabel: 'Complexity',
      groupValuesLabel: 'Complexity levels'
    }
  ];

  // Template card definition for Cloudscape Cards component
  const templateCardDefinition = {
    header: (item: Template) => (
      <SpaceBetween direction="horizontal" size="xs">
        <Box fontSize="heading-m">{item.Icon || '🖥️'}</Box>
        <Header variant="h3">{item.Name}</Header>
        {item.Popular && <Badge color="green">Popular</Badge>}
        {item.ResearchUser?.AutoCreate && <Badge color="blue" iconName="user-profile">Research Users</Badge>}
      </SpaceBetween>
    ),
    sections: [
      {
        id: 'description',
        content: (item: Template) => item.Description || 'Professional research environment ready to launch.'
      },
      {
        id: 'features',
        content: (item: Template) => (
          <SpaceBetween direction="horizontal" size="xs">
            <Badge>{item.Category || 'General'}</Badge>
            <Badge color="blue">{item.Complexity || 'simple'}</Badge>
            {item.ValidationStatus === 'validated' && <Badge color="green">Pre-tested</Badge>}
          </SpaceBetween>
        )
      },
      {
        id: 'research-user',
        content: (item: Template) => {
          if (!item.ResearchUser?.AutoCreate) return null;
          return (
            <SpaceBetween direction="vertical" size="xs">
              <Box variant="awsui-key-label">Research User Support</Box>
              <SpaceBetween direction="horizontal" size="xs">
                {item.ResearchUser.AutoCreate && (
                  <Badge color="blue" iconName="user-profile">Auto-creation</Badge>
                )}
                {item.ResearchUser.RequireEFS && (
                  <Badge color="green" iconName="folder">Persistent home</Badge>
                )}
                {item.ResearchUser.InstallSSHKeys && (
                  <Badge color="grey" iconName="key">SSH keys</Badge>
                )}
                {item.ResearchUser.DualUserIntegration?.CollaborationEnabled && (
                  <Badge color="red" iconName="share">Collaboration</Badge>
                )}
              </SpaceBetween>
            </SpaceBetween>
          );
        }
      },
      {
        id: 'metadata',
        content: (item: Template) => (
          <SpaceBetween direction="horizontal" size="l">
            <Box>
              <Box variant="awsui-key-label">Launch Time</Box>
              <Box>~{item.EstimatedLaunchTime || 3} min</Box>
            </Box>
            <Box>
              <Box variant="awsui-key-label">Cost</Box>
              <Box>${(item.EstimatedCostPerHour?.['x86_64'] || 0.12).toFixed(2)}/hour</Box>
            </Box>
          </SpaceBetween>
        )
      }
    ]
  };

  // Handle template selection
  const handleTemplateSelection = ({ detail }: { detail: { selectedItems: Template[] } }) => {
    const selectedTemplate = detail.selectedItems[0] || null;
    setState(prev => ({
      ...prev,
      selectedTemplate,
      splitPanelOpen: selectedTemplate ? true : false,
      splitPanelContent: selectedTemplate ? 'template-details' : null
    }));
  };

  // Handle instance launch
  const handleLaunchInstance = async () => {
    if (!state.selectedTemplate || !instanceName.trim()) return;

    try {
      // Show launching notification with progress
      addNotification({
        type: 'info',
        header: 'Launching instance',
        content: `Starting ${instanceName} with ${state.selectedTemplate.Name} template...`,
        loading: true,
        dismissible: false
      });

      if (window.wails?.CloudWorkstationService?.LaunchInstance) {
        await window.wails.CloudWorkstationService.LaunchInstance(
          instanceName.trim(),
          state.selectedTemplate.Name,
          instanceSize
        );
      }

      // Clear loading notification and show success
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => !n.loading)
      }));

      addNotification({
        type: 'success',
        header: 'Instance launched successfully',
        content: `${instanceName} is now starting up. You'll receive a connection notification when ready.`,
        dismissible: true
      });

      setLaunchModalVisible(false);
      setInstanceName('');
      setState(prev => ({ ...prev, selectedTemplate: null }));

      // Refresh instances
      loadApplicationData();
    } catch (error) {
      // Clear loading notification
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => !n.loading)
      }));

      addNotification({
        type: 'error',
        header: 'Launch failed',
        content: `Failed to launch ${instanceName}: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true,
        buttonText: 'Try Again',
        onButtonClick: () => setLaunchModalVisible(true)
      });
    }
  };

  // Render template selection view
  const renderTemplatesView = () => (
    <Container header={<Header variant="h1">Research Templates</Header>}>
      <SpaceBetween direction="vertical" size="l">
        <PropertyFilter
          filteringProperties={templateFilteringProperties}
          query={templateQuery}
          placeholder="Find templates by name, domain, or complexity..."
          onChange={({ detail }) => setTemplateQuery(detail)}
          countText={`${filteredTemplates.length} of ${state.templates.length} templates`}
          i18nStrings={{
            filteringAriaLabel: "Filter templates",
            dismissAriaLabel: "Dismiss",
            filteringPlaceholder: "Find templates by name, domain, or complexity...",
            groupValuesText: "Values",
            groupPropertiesText: "Properties",
            operatorsText: "Operators",
            operationAndText: "and",
            operationOrText: "or",
            operatorLessText: "Less than",
            operatorLessOrEqualText: "Less than or equal",
            operatorGreaterText: "Greater than",
            operatorGreaterOrEqualText: "Greater than or equal",
            operatorContainsText: "Contains",
            operatorDoesNotContainText: "Does not contain",
            operatorEqualsText: "Equals",
            operatorDoesNotEqualText: "Does not equal",
            editTokenText: "Edit filter",
            propertyText: "Property",
            operatorText: "Operator",
            valueText: "Value",
            cancelActionText: "Cancel",
            applyActionText: "Apply",
            allPropertiesLabel: "All properties",
            tokenLimitShowMore: "Show more",
            tokenLimitShowFewer: "Show fewer",
            clearFiltersText: "Clear filters",
            removeTokenButtonAriaLabel: (token) => `Remove token ${token.propertyKey} ${token.operator} ${token.value}`,
            enteredTextLabel: (text) => `Use: "${text}"`
          }}
        />

        {state.loading ? (
          <Box textAlign="center">
            <Spinner size="large" />
            <Box variant="p">Loading templates...</Box>
          </Box>
        ) : (
          <Cards
            cardDefinition={templateCardDefinition}
            items={filteredTemplates}
            selectionType="single"
            onSelectionChange={handleTemplateSelection}
            cardsPerRow={[
              { cards: 1 },
              { minWidth: 500, cards: 2 },
              { minWidth: 900, cards: 3 }
            ]}
            empty={
              <Box textAlign="center">
                <Box variant="strong">No templates available</Box>
                <Box variant="p">Please ensure the CloudWorkstation daemon is running</Box>
                <Button variant="primary" onClick={loadApplicationData}>Retry</Button>
              </Box>
            }
          />
        )}
      </SpaceBetween>
    </Container>
  );

  // Handle volume selection
  const handleVolumeSelection = ({ detail }: { detail: { selectedItems: Volume[] } }) => {
    const selectedVolume = detail.selectedItems[0] || null;
    setState(prev => ({
      ...prev,
      selectedVolume,
      splitPanelOpen: selectedVolume ? true : false,
      splitPanelContent: selectedVolume ? 'volume-details' : null
    }));
  };

  // Handle volume mount
  const handleMountVolume = async () => {
    const { mountingVolume } = state;
    if (!mountingVolume || !mountInstanceName.trim()) return;

    try {
      addNotification({
        type: 'info',
        header: 'Mounting volume',
        content: `Mounting ${mountingVolume.name} to ${mountInstanceName} at ${mountPoint}...`,
        loading: true,
        dismissible: false
      });

      if (window.wails?.CloudWorkstationService?.MountVolume) {
        await window.wails.CloudWorkstationService.MountVolume(
          mountingVolume.name,
          mountInstanceName.trim(),
          mountPoint
        );
      }

      // Clear loading notification and show success
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => !n.loading),
        showMountDialog: false,
        mountingVolume: null
      }));

      addNotification({
        type: 'success',
        header: 'Volume mounted successfully',
        content: `${mountingVolume.name} is now accessible at ${mountPoint} on ${mountInstanceName}`,
        dismissible: true
      });

      setMountInstanceName('');
      setMountPoint('/mnt/shared-volume');
      loadApplicationData();
    } catch (error) {
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => !n.loading)
      }));

      addNotification({
        type: 'error',
        header: 'Mount failed',
        content: `Failed to mount ${mountingVolume!.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true,
        buttonText: 'Try Again',
        onButtonClick: handleMountVolume
      });
    }
  };

  // Handle volume unmount
  const handleUnmountVolume = async (volume: Volume, instanceName: string) => {
    try {
      addNotification({
        type: 'info',
        header: 'Unmounting volume',
        content: `Unmounting ${volume.name} from ${instanceName}...`,
        loading: true,
        dismissible: false
      });

      if (window.wails?.CloudWorkstationService?.UnmountVolume) {
        await window.wails.CloudWorkstationService.UnmountVolume(
          volume.name,
          instanceName
        );
      }

      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => !n.loading)
      }));

      addNotification({
        type: 'success',
        header: 'Volume unmounted successfully',
        content: `${volume.name} has been unmounted from ${instanceName}`,
        dismissible: true
      });

      loadApplicationData();
    } catch (error) {
      setState(prev => ({
        ...prev,
        notifications: prev.notifications.filter(n => !n.loading)
      }));

      addNotification({
        type: 'error',
        header: 'Unmount failed',
        content: `Failed to unmount ${volume.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true,
        buttonText: 'Try Again',
        onButtonClick: () => handleUnmountVolume(volume, instanceName)
      });
    }
  };

  // Enhanced instance actions with real embedded connection support
  const handleInstanceAction = async (action: string, instance: Instance) => {
    if (action === 'Connect') {
      try {
        // Determine the best connection type for the instance
        const connectionType = determineConnectionType(instance);
        let config: ConnectionConfig;

        switch (connectionType) {
          case 'ssh':
            config = await window.wails.CloudWorkstationService.OpenEmbeddedTerminal(instance.name);
            break;
          case 'desktop':
            config = await window.wails.CloudWorkstationService.OpenEmbeddedDesktop(instance.name);
            break;
          case 'web':
            config = await window.wails.CloudWorkstationService.OpenEmbeddedWeb(instance.name);
            break;
          default:
            // Default to SSH if available
            config = await window.wails.CloudWorkstationService.OpenEmbeddedTerminal(instance.name);
        }

        // Create new connection tab
        createConnectionTab(config);

        addNotification({
          type: 'success',
          header: 'Connection established',
          content: `Connected to ${instance.name} via ${connectionType.toUpperCase()}`,
          dismissible: true
        });

      } catch (error) {
        addNotification({
          type: 'error',
          header: 'Connection failed',
          content: `Failed to connect to ${instance.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
          dismissible: true
        });
      }
    } else {
      // Handle other instance actions (Start, Stop, Hibernate, etc.)
      // Show in-progress notification
      addNotification({
        type: 'info',
        header: `${action} in progress`,
        content: `${action} operation started for ${instance.name}`,
        loading: true,
        dismissible: false
      });

      try {
        // TODO: Call actual API when available for other actions
        await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate API call

        // Clear loading notifications
        setState(prev => ({
          ...prev,
          notifications: prev.notifications.filter(n => !n.loading)
        }));

        // Show success notification
        addNotification({
          type: 'success',
          header: `${action} successful`,
          content: `${instance.name} ${action.toLowerCase()} completed successfully`,
          dismissible: true
        });

        // Refresh data to show updated states
        loadApplicationData();
      } catch (error) {
        // Clear loading notifications
        setState(prev => ({
          ...prev,
          notifications: prev.notifications.filter(n => !n.loading)
        }));

        addNotification({
          type: 'error',
          header: `${action} failed`,
          content: `Failed to ${action.toLowerCase()} ${instance.name}: ${error instanceof Error ? error.message : 'Unknown error'}`,
          dismissible: true,
          buttonText: 'Retry',
          onButtonClick: () => handleInstanceAction(action, instance)
        });
      }
    }
  };

  // Connection management functions
  const determineConnectionType = (instance: Instance): 'ssh' | 'desktop' | 'web' => {
    // Logic to determine the best connection type based on instance template and capabilities
    // For now, default to SSH - this will be enhanced based on template metadata
    const template = state.templates.find(t => t.Name === instance.template);

    if (template?.Category === 'Machine Learning' || template?.Name.includes('Jupyter')) {
      return 'web'; // ML templates likely have Jupyter
    }

    if (template?.Category === 'Desktop' || template?.Name.includes('Desktop')) {
      return 'desktop'; // Desktop templates have GUI
    }

    return 'ssh'; // Default to SSH terminal
  };

  const createConnectionTab = (config: ConnectionConfig) => {
    const tab: ConnectionTab = {
      id: config.id,
      title: config.title,
      type: config.instanceName ? 'instance' : 'aws-service',
      category: determineConnectionCategory(config),
      config,
      active: true,
      closeable: true,
      status: config.status as 'connecting' | 'connected' | 'disconnected' | 'error'
    };

    setState(prev => ({
      ...prev,
      connectionTabs: [...prev.connectionTabs, tab],
      activeConnectionTab: tab.id,
      showConnectionPanel: true,
      activeView: 'connections'
    }));
  };

  const determineConnectionCategory = (config: ConnectionConfig): 'compute' | 'research' | 'analytics' | 'management' => {
    if (config.instanceName) return 'compute';

    switch (config.awsService) {
      case 'braket':
      case 'sagemaker':
        return 'research';
      case 'athena':
      case 'quicksight':
        return 'analytics';
      case 'console':
      case 'cloudshell':
        return 'management';
      default:
        return 'compute';
    }
  };

  const closeConnectionTab = (tabId: string) => {
    setState(prev => {
      const tabs = prev.connectionTabs.filter(tab => tab.id !== tabId);
      const activeTab = tabs.length > 0 ? tabs[tabs.length - 1].id : null;

      return {
        ...prev,
        connectionTabs: tabs,
        activeConnectionTab: activeTab,
        showConnectionPanel: tabs.length > 0,
        activeView: tabs.length > 0 ? 'connections' : 'instances'
      };
    });
  };

  const updateTabStatus = (tabId: string, status: 'connecting' | 'connected' | 'disconnected' | 'error') => {
    setState(prev => ({
      ...prev,
      connectionTabs: prev.connectionTabs.map(tab =>
        tab.id === tabId ? { ...tab, status } : tab
      )
    }));
  };

  // Research Users Handlers
  const handleCreateUser = async () => {
    if (!newUsername.trim()) {
      addNotification({
        type: 'warning',
        header: 'Invalid Input',
        content: 'Username cannot be empty',
        dismissible: true
      });
      return;
    }

    try {
      await window.go.main.CloudWorkstationService.CreateResearchUser(window.go.context.Context(), {
        username: newUsername.trim()
      });

      addNotification({
        type: 'success',
        header: 'User Created',
        content: `Research user "${newUsername}" created successfully`,
        dismissible: true
      });

      setCreateUserModalVisible(false);
      setNewUsername('');
      loadApplicationData(); // Refresh the list
    } catch (error) {
      addNotification({
        type: 'error',
        header: 'Creation Failed',
        content: `Failed to create user: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true
      });
    }
  };

  const handleDeleteUser = async (username: string) => {
    try {
      await window.go.main.CloudWorkstationService.DeleteResearchUser(window.go.context.Context(), username);

      addNotification({
        type: 'success',
        header: 'User Deleted',
        content: `Research user "${username}" deleted successfully`,
        dismissible: true
      });

      loadApplicationData(); // Refresh the list
    } catch (error) {
      addNotification({
        type: 'error',
        header: 'Deletion Failed',
        content: `Failed to delete user: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true
      });
    }
  };

  const handleGenerateSSHKey = async (username: string) => {
    try {
      await window.go.main.CloudWorkstationService.GenerateResearchUserSSHKey(window.go.context.Context(), {
        username: username,
        key_type: 'ed25519'
      });

      addNotification({
        type: 'success',
        header: 'SSH Key Generated',
        content: `SSH key generated for "${username}"`,
        dismissible: true
      });

      loadApplicationData(); // Refresh to show updated key count
    } catch (error) {
      addNotification({
        type: 'error',
        header: 'Key Generation Failed',
        content: `Failed to generate SSH key: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true
      });
    }
  };

  const handleViewUserStatus = async (user: ResearchUser) => {
    try {
      const status = await window.go.main.CloudWorkstationService.GetResearchUserStatus(window.go.context.Context(), user.username);

      setState(prev => ({
        ...prev,
        selectedResearchUser: user,
        splitPanelOpen: true,
        splitPanelContent: 'research-user-details'
      }));

      // Store status in user object for display
      (user as any).status = status;

    } catch (error) {
      addNotification({
        type: 'error',
        header: 'Status Failed',
        content: `Failed to get user status: ${error instanceof Error ? error.message : 'Unknown error'}`,
        dismissible: true
      });
    }
  };

  // Render volumes view with professional Table
  const renderVolumesView = () => (
    <Container
      header={
        <Header
          variant="h1"
          counter={`(${state.volumes.length})`}
          description="Manage EFS storage volumes for multi-instance sharing"
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="normal"
                onClick={loadApplicationData}
                iconName="refresh"
              >
                Refresh
              </Button>
              <Button
                variant="primary"
                disabled={selectedVolumes.length === 0 || selectedVolumes[0].state !== 'available'}
                onClick={() => {
                  setState(prev => ({
                    ...prev,
                    showMountDialog: true,
                    mountingVolume: selectedVolumes[0]
                  }));
                  setMountInstanceName(state.instances.find(i => i.status === 'running')?.name || '');
                }}
              >
                Mount Volume
              </Button>
            </SpaceBetween>
          }
        >
          Storage Volumes
        </Header>
      }
    >
      <SpaceBetween direction="vertical" size="l">
        <Table
          columnDefinitions={[
            {
              id: 'name',
              header: 'Volume Name',
              cell: (item: Volume) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span" fontWeight="bold">{item.name}</Box>
                  <Box variant="small" color="text-body-secondary">{item.id}</Box>
                </SpaceBetween>
              ),
              sortingField: 'name',
              isRowHeader: true
            },
            {
              id: 'status',
              header: 'Status',
              cell: (item: Volume) => (
                <StatusIndicator type={
                  item.state === 'available' ? 'success' :
                  item.state === 'creating' ? 'in-progress' :
                  'stopped'
                }>
                  {item.state === 'available' ? 'Available' :
                   item.state === 'creating' ? 'Creating' :
                   'Deleting'}
                </StatusIndicator>
              ),
              sortingField: 'state'
            },
            {
              id: 'size',
              header: 'Size & Cost',
              cell: (item: Volume) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span">{item.size_gb} GB</Box>
                  <Box variant="small" color="text-body-secondary">
                    ${item.cost_per_gb.toFixed(2)}/GB/month
                  </Box>
                  <Box variant="small" fontWeight="bold">
                    ${(item.size_gb * item.cost_per_gb).toFixed(2)}/month
                  </Box>
                </SpaceBetween>
              ),
              sortingField: 'size_gb'
            },
            {
              id: 'mounts',
              header: 'Mount Status',
              cell: (item: Volume) => (
                <SpaceBetween direction="vertical" size="xs">
                  {item.mount_targets.length > 0 ? (
                    <>
                      <Badge color="green">Mounted ({item.mount_targets.length})</Badge>
                      {item.mount_targets.map(target => (
                        <SpaceBetween key={target} direction="horizontal" size="xs">
                          <Box variant="small">{target}</Box>
                          <Button
                            variant="inline-link"
                            onClick={() => handleUnmountVolume(item, target)}
                          >
                            Unmount
                          </Button>
                        </SpaceBetween>
                      ))}
                    </>
                  ) : (
                    <Badge color="grey">Not mounted</Badge>
                  )}
                </SpaceBetween>
              )
            },
            {
              id: 'metadata',
              header: 'Details',
              cell: (item: Volume) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="small">
                    <Box variant="awsui-key-label">Region</Box>
                    <Box>{item.region}</Box>
                  </Box>
                  <Box variant="small" color="text-body-secondary">
                    Created: {new Date(item.creation_time).toLocaleDateString()}
                  </Box>
                </SpaceBetween>
              )
            }
          ]}
          items={state.volumes}
          selectionType="multi"
          selectedItems={selectedVolumes}
          onSelectionChange={({ detail }) => setSelectedVolumes(detail.selectedItems)}
          onRowClick={({ detail }) => {
            handleVolumeSelection({ detail: { selectedItems: [detail.item] } });
          }}
          sortingDisabled={false}
          variant="borderless"
          stickyHeader={true}
          header={
            selectedVolumes.length > 0 ? (
              <Header
                counter={`(${selectedVolumes.length} selected)`}
                actions={
                  <SpaceBetween direction="horizontal" size="xs">
                    <Button
                      variant="normal"
                      disabled={!selectedVolumes.some(v => v.state === 'available')}
                      onClick={() => {
                        const availableVolume = selectedVolumes.find(v => v.state === 'available');
                        if (availableVolume) {
                          setState(prev => ({
                            ...prev,
                            showMountDialog: true,
                            mountingVolume: availableVolume
                          }));
                          setMountInstanceName(state.instances.find(i => i.status === 'running')?.name || '');
                        }
                      }}
                    >
                      Mount Selected
                    </Button>
                  </SpaceBetween>
                }
              >
                Bulk Actions
              </Header>
            ) : undefined
          }
          empty={
            <Box textAlign="center">
              <SpaceBetween direction="vertical" size="xs">
                <Box variant="strong">No storage volumes found</Box>
                <Box variant="p" color="text-body-secondary">
                  Create EFS volumes using the CLI to enable multi-instance file sharing
                </Box>
                <Box variant="small" color="text-body-secondary">
                  Example: cws volumes create shared-data
                </Box>
              </SpaceBetween>
            </Box>
          }
          loading={state.loading}
        />
      </SpaceBetween>
    </Container>
  );

  // Render instances view with professional Table
  const renderInstancesView = () => (
    <Container
      header={
        <Header
          variant="h1"
          counter={`(${state.instances.length})`}
          description="Manage your running research environments"
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="normal"
                onClick={loadApplicationData}
                iconName="refresh"
              >
                Refresh
              </Button>
              <Button
                variant="primary"
                onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
              >
                Launch Instance
              </Button>
            </SpaceBetween>
          }
        >
          My Instances
        </Header>
      }
    >
      <SpaceBetween direction="vertical" size="l">
        <Table
          columnDefinitions={[
            {
              id: 'name',
              header: 'Instance Name',
              cell: (item: Instance) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span" fontWeight="bold">{item.name}</Box>
                  <Box variant="small" color="text-body-secondary">{item.id}</Box>
                </SpaceBetween>
              ),
              sortingField: 'name',
              isRowHeader: true
            },
            {
              id: 'status',
              header: 'Status',
              cell: (item: Instance) => (
                <StatusIndicator type={
                  item.status === 'running' ? 'success' :
                  item.status === 'stopped' ? 'stopped' :
                  item.status === 'hibernated' ? 'pending' :
                  item.status === 'stopping' ? 'in-progress' :
                  'loading'
                }>
                  {item.status === 'hibernated' ? 'Hibernated' :
                   item.status === 'running' ? 'Running' :
                   item.status === 'stopped' ? 'Stopped' :
                   item.status === 'stopping' ? 'Stopping' :
                   'Pending'}
                </StatusIndicator>
              ),
              sortingField: 'status'
            },
            {
              id: 'template',
              header: 'Template',
              cell: (item: Instance) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span">{item.template}</Box>
                  <Badge color="blue">Region: {item.region}</Badge>
                </SpaceBetween>
              ),
              sortingField: 'template'
            },
            {
              id: 'connection',
              header: 'Connection',
              cell: (item: Instance) => (
                <SpaceBetween direction="vertical" size="xs">
                  {item.public_ip && (
                    <Box variant="small">
                      <Box variant="awsui-key-label">Public IP</Box>
                      <Box fontFamily="monospace">{item.public_ip}</Box>
                    </Box>
                  )}
                  <Box variant="small" color="text-body-secondary">
                    Launched: {new Date(item.launch_time).toLocaleString()}
                  </Box>
                </SpaceBetween>
              )
            },
            {
              id: 'cost',
              header: 'Cost',
              cell: (item: Instance) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span" fontWeight="bold">
                    ${item.cost_per_hour.toFixed(2)}/hour
                  </Box>
                  <Box variant="small" color="text-body-secondary">
                    Est. daily: ${(item.cost_per_hour * 24).toFixed(2)}
                  </Box>
                </SpaceBetween>
              ),
              sortingField: 'cost_per_hour'
            },
            {
              id: 'actions',
              header: 'Actions',
              cell: (item: Instance) => (
                <SpaceBetween direction="horizontal" size="xs">
                  <Button
                    variant="primary"
                    size="small"
                    disabled={item.status !== 'running'}
                    onClick={() => handleInstanceAction('Connect', item)}
                  >
                    Connect
                  </Button>
                  <Button
                    variant="normal"
                    size="small"
                    disabled={item.status === 'pending' || item.status === 'stopping'}
                    onClick={() => handleInstanceAction(
                      item.status === 'running' ? 'Hibernate' :
                      item.status === 'hibernated' ? 'Resume' : 'Start',
                      item
                    )}
                  >
                    {item.status === 'running' ? 'Hibernate' :
                     item.status === 'hibernated' ? 'Resume' :
                     item.status === 'stopped' ? 'Start' : 'Processing...'}
                  </Button>
                </SpaceBetween>
              )
            }
          ]}
          items={state.instances}
          selectionType="multi"
          selectedItems={selectedInstances}
          onSelectionChange={({ detail }) => setSelectedInstances(detail.selectedItems)}
          onRowClick={({ detail }) => {
            setSelectedInstance(detail.item);
            setState(prev => ({
              ...prev,
              splitPanelOpen: true,
              splitPanelContent: 'instance-details'
            }));
          }}
          sortingDisabled={false}
          variant="borderless"
          stickyHeader={true}
          header={
            selectedInstances.length > 0 ? (
              <Header
                counter={`(${selectedInstances.length} selected)`}
                actions={
                  <SpaceBetween direction="horizontal" size="xs">
                    <Button
                      variant="normal"
                      disabled={!selectedInstances.some(i => i.status === 'running')}
                      onClick={() => selectedInstances
                        .filter(i => i.status === 'running')
                        .forEach(i => handleInstanceAction('Hibernate', i))
                      }
                    >
                      Hibernate Selected
                    </Button>
                    <Button
                      variant="normal"
                      disabled={!selectedInstances.some(i => i.status === 'hibernated' || i.status === 'stopped')}
                      onClick={() => selectedInstances
                        .filter(i => i.status === 'hibernated' || i.status === 'stopped')
                        .forEach(i => handleInstanceAction('Resume', i))
                      }
                    >
                      Resume Selected
                    </Button>
                  </SpaceBetween>
                }
              >
                Bulk Actions
              </Header>
            ) : undefined
          }
          empty={
            <Box textAlign="center">
              <SpaceBetween direction="vertical" size="xs">
                <Box variant="strong">No instances running</Box>
                <Box variant="p" color="text-body-secondary">
                  Launch your first research environment to get started
                </Box>
                <Button
                  variant="primary"
                  onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
                >
                  Browse Templates
                </Button>
              </SpaceBetween>
            </Box>
          }
          loading={state.loading}
        />
      </SpaceBetween>
    </Container>
  );

  // Render research users view
  const renderResearchUsersView = () => (
    <Container
      header={
        <Header
          variant="h1"
          counter={`(${state.researchUsers.length})`}
          description="Manage research users with persistent identity across instances"
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="normal"
                onClick={loadApplicationData}
                iconName="refresh"
              >
                Refresh
              </Button>
              <Button
                variant="primary"
                onClick={() => setCreateUserModalVisible(true)}
              >
                Create Research User
              </Button>
            </SpaceBetween>
          }
        >
          Research Users
        </Header>
      }
    >
      <SpaceBetween direction="vertical" size="l">
        <Table
          columnDefinitions={[
            {
              id: 'username',
              header: 'Username',
              cell: (item: ResearchUser) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span" fontWeight="bold">{item.username}</Box>
                  <Box variant="small" color="text-body-secondary">UID: {item.uid}</Box>
                </SpaceBetween>
              ),
              sortingField: 'username',
              isRowHeader: true
            },
            {
              id: 'identity',
              header: 'Identity',
              cell: (item: ResearchUser) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span">{item.full_name}</Box>
                  <Box variant="small" color="text-body-secondary">{item.email}</Box>
                </SpaceBetween>
              ),
              sortingField: 'full_name'
            },
            {
              id: 'access',
              header: 'Access Level',
              cell: (item: ResearchUser) => (
                <SpaceBetween direction="horizontal" size="xs">
                  {item.sudo_access && <Badge color="red">Sudo</Badge>}
                  {item.docker_access && <Badge color="blue">Docker</Badge>}
                  <Badge color="green">SSH ({item.ssh_public_keys.length} keys)</Badge>
                </SpaceBetween>
              )
            },
            {
              id: 'home',
              header: 'Home Directory',
              cell: (item: ResearchUser) => (
                <SpaceBetween direction="vertical" size="xs">
                  <Box variant="span">{item.home_directory}</Box>
                  <Box variant="small" color="text-body-secondary">{item.shell}</Box>
                </SpaceBetween>
              )
            },
            {
              id: 'created',
              header: 'Created',
              cell: (item: ResearchUser) => (
                <Box variant="span">
                  {new Date(item.created_at).toLocaleDateString()}
                </Box>
              ),
              sortingField: 'created_at'
            },
            {
              id: 'actions',
              header: 'Actions',
              cell: (item: ResearchUser) => (
                <SpaceBetween direction="horizontal" size="xs">
                  <Button
                    variant="normal"
                    iconName="key"
                    onClick={() => handleGenerateSSHKey(item.username)}
                  >
                    SSH Key
                  </Button>
                  <Button
                    variant="normal"
                    iconName="status-info"
                    onClick={() => handleViewUserStatus(item)}
                  >
                    Status
                  </Button>
                  <Button
                    variant="normal"
                    iconName="remove"
                    onClick={() => handleDeleteUser(item.username)}
                  >
                    Delete
                  </Button>
                </SpaceBetween>
              )
            }
          ]}
          items={state.researchUsers}
          loading={state.loading}
          loadingText="Loading research users..."
          empty={
            <Box textAlign="center" color="inherit">
              <Box variant="strong" textAlign="center" color="inherit">
                No research users
              </Box>
              <Box variant="p" padding={{ bottom: 's' }} color="inherit">
                Research users provide persistent identity and SSH access across all instances.
              </Box>
              <Button
                variant="primary"
                onClick={() => setCreateUserModalVisible(true)}
              >
                Create your first research user
              </Button>
            </Box>
          }
          onSelectionChange={({ detail }) => {
            const selectedUser = detail.selectedItems[0];
            setState(prev => ({
              ...prev,
              selectedResearchUser: selectedUser || null,
              splitPanelOpen: !!selectedUser,
              splitPanelContent: selectedUser ? 'research-user-details' : null
            }));
          }}
          selectedItems={state.selectedResearchUser ? [state.selectedResearchUser] : []}
          selectionType="single"
        />
      </SpaceBetween>
    </Container>
  );

  // Split panel content renderer
  const renderSplitPanelContent = () => {
    if (!state.splitPanelOpen || !state.splitPanelContent) return null;

    switch (state.splitPanelContent) {
      case 'instance-details':
        if (!selectedInstance) return null;
        return (
          <SpaceBetween direction="vertical" size="l">
            <Header variant="h2">{selectedInstance.name}</Header>
            <SpaceBetween direction="vertical" size="s">
              <Box>
                <Box variant="awsui-key-label">Instance ID</Box>
                <Box>{selectedInstance.id}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Template</Box>
                <Box>{selectedInstance.template}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Status</Box>
                <StatusIndicator type={
                  selectedInstance.status === 'running' ? 'success' :
                  selectedInstance.status === 'stopped' ? 'stopped' :
                  selectedInstance.status === 'hibernated' ? 'pending' :
                  'in-progress'
                }>
                  {selectedInstance.status.charAt(0).toUpperCase() + selectedInstance.status.slice(1)}
                </StatusIndicator>
              </Box>
              {selectedInstance.public_ip && (
                <Box>
                  <Box variant="awsui-key-label">Public IP</Box>
                  <Box>{selectedInstance.public_ip}</Box>
                </Box>
              )}
              <Box>
                <Box variant="awsui-key-label">Cost per Hour</Box>
                <Box>${selectedInstance.cost_per_hour.toFixed(2)}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Launch Time</Box>
                <Box>{new Date(selectedInstance.launch_time).toLocaleDateString()}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Region</Box>
                <Box>{selectedInstance.region}</Box>
              </Box>
            </SpaceBetween>

            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="primary"
                disabled={selectedInstance.status !== 'running'}
                onClick={() => handleInstanceAction('Connect', selectedInstance)}
              >
                Connect
              </Button>
              <Button
                variant="normal"
                onClick={() => {
                  if (selectedInstance.status === 'running') {
                    handleInstanceAction('Hibernate', selectedInstance);
                  } else if (selectedInstance.status === 'hibernated') {
                    handleInstanceAction('Resume', selectedInstance);
                  } else if (selectedInstance.status === 'stopped') {
                    handleInstanceAction('Start', selectedInstance);
                  }
                }}
              >
                {selectedInstance.status === 'running' ? 'Hibernate' :
                 selectedInstance.status === 'hibernated' ? 'Resume' : 'Start'}
              </Button>
            </SpaceBetween>
          </SpaceBetween>
        );

      case 'template-details':
        if (!state.selectedTemplate) return null;
        return (
          <SpaceBetween direction="vertical" size="l">
            <Header variant="h2">
              <SpaceBetween direction="horizontal" size="xs">
                <Box>{state.selectedTemplate.Icon || '🖥️'}</Box>
                <Box>{state.selectedTemplate.Name}</Box>
              </SpaceBetween>
            </Header>
            <Box>{state.selectedTemplate.Description}</Box>
            <SpaceBetween direction="vertical" size="s">
              <Box>
                <Box variant="awsui-key-label">Category</Box>
                <Badge>{state.selectedTemplate.Category}</Badge>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Complexity</Box>
                <Badge color="blue">{state.selectedTemplate.Complexity}</Badge>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Estimated Launch Time</Box>
                <Box>~{state.selectedTemplate.EstimatedLaunchTime || 3} minutes</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Estimated Cost</Box>
                <Box>${(state.selectedTemplate.EstimatedCostPerHour?.['x86_64'] || 0.12).toFixed(2)}/hour</Box>
              </Box>
              {state.selectedTemplate.ResearchUser?.AutoCreate && (
                <Box>
                  <Box variant="awsui-key-label">Research User Integration</Box>
                  <SpaceBetween direction="vertical" size="xs">
                    <SpaceBetween direction="horizontal" size="xs">
                      <Badge color="blue" iconName="user-profile">Auto-creation enabled</Badge>
                      {state.selectedTemplate.ResearchUser.RequireEFS && (
                        <Badge color="green" iconName="folder">Persistent home directory</Badge>
                      )}
                      {state.selectedTemplate.ResearchUser.InstallSSHKeys && (
                        <Badge color="grey" iconName="key">SSH key management</Badge>
                      )}
                      {state.selectedTemplate.ResearchUser.DualUserIntegration?.CollaborationEnabled && (
                        <Badge color="red" iconName="share">Multi-user collaboration</Badge>
                      )}
                    </SpaceBetween>
                    <Box variant="small">
                      Launch with <code>--research-user alice</code> to automatically create and provision research users
                    </Box>
                  </SpaceBetween>
                </Box>
              )}
            </SpaceBetween>
            <Button
              variant="primary"
              onClick={() => {
                setInstanceName(`my-${state.selectedTemplate!.Name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`);
                setLaunchModalVisible(true);
              }}
            >
              Launch Instance
            </Button>
          </SpaceBetween>
        );

      case 'volume-details':
        if (!state.selectedVolume) return null;
        return (
          <SpaceBetween direction="vertical" size="l">
            <Header variant="h2">{state.selectedVolume.name}</Header>
            <SpaceBetween direction="vertical" size="s">
              <Box>
                <Box variant="awsui-key-label">Volume ID</Box>
                <Box>{state.selectedVolume.id}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Status</Box>
                <StatusIndicator type={
                  state.selectedVolume.state === 'available' ? 'success' :
                  state.selectedVolume.state === 'creating' ? 'in-progress' :
                  'stopped'
                }>
                  {state.selectedVolume.state.charAt(0).toUpperCase() + state.selectedVolume.state.slice(1)}
                </StatusIndicator>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Size</Box>
                <Box>{state.selectedVolume.size_gb} GB</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Monthly Cost</Box>
                <Box>${(state.selectedVolume.size_gb * state.selectedVolume.cost_per_gb).toFixed(2)}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Cost per GB</Box>
                <Box>${state.selectedVolume.cost_per_gb.toFixed(2)}/GB/month</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Region</Box>
                <Box>{state.selectedVolume.region}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Created</Box>
                <Box>{new Date(state.selectedVolume.creation_time).toLocaleDateString()}</Box>
              </Box>
              {state.selectedVolume.mount_targets.length > 0 && (
                <Box>
                  <Box variant="awsui-key-label">Mounted To</Box>
                  <SpaceBetween direction="vertical" size="xs">
                    {state.selectedVolume.mount_targets.map(target => (
                      <SpaceBetween key={target} direction="horizontal" size="xs">
                        <Box>{target}</Box>
                        <Button
                          variant="inline-link"
                          onClick={() => handleUnmountVolume(state.selectedVolume!, target)}
                        >
                          Unmount
                        </Button>
                      </SpaceBetween>
                    ))}
                  </SpaceBetween>
                </Box>
              )}
            </SpaceBetween>

            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="primary"
                disabled={state.selectedVolume.state !== 'available'}
                onClick={() => {
                  setState(prev => ({
                    ...prev,
                    showMountDialog: true,
                    mountingVolume: state.selectedVolume
                  }));
                  setMountInstanceName(state.instances.find(i => i.status === 'running')?.name || '');
                }}
              >
                Mount Volume
              </Button>
            </SpaceBetween>
          </SpaceBetween>
        );

      case 'research-user-details':
        if (!state.selectedResearchUser) return null;
        return (
          <SpaceBetween direction="vertical" size="l">
            <Header variant="h2">{state.selectedResearchUser.username}</Header>
            <SpaceBetween direction="vertical" size="s">
              <Box>
                <Box variant="awsui-key-label">Full Name</Box>
                <Box>{state.selectedResearchUser.full_name}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Email</Box>
                <Box>{state.selectedResearchUser.email}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">User ID (UID/GID)</Box>
                <Box>{state.selectedResearchUser.uid}:{state.selectedResearchUser.gid}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Home Directory</Box>
                <Box>{state.selectedResearchUser.home_directory}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Shell</Box>
                <Box>{state.selectedResearchUser.shell}</Box>
              </Box>
              <Box>
                <Box variant="awsui-key-label">Access Permissions</Box>
                <SpaceBetween direction="horizontal" size="xs">
                  {state.selectedResearchUser.sudo_access && (
                    <Badge color="red">Sudo Access</Badge>
                  )}
                  {state.selectedResearchUser.docker_access && (
                    <Badge color="blue">Docker Access</Badge>
                  )}
                  <Badge color="green">SSH Access</Badge>
                </SpaceBetween>
              </Box>
              <Box>
                <Box variant="awsui-key-label">SSH Public Keys</Box>
                <Box>{state.selectedResearchUser.ssh_public_keys.length} key(s) configured</Box>
                {state.selectedResearchUser.ssh_public_keys.map((key, index) => (
                  <Box key={index} variant="small" color="text-body-secondary">
                    {key.substring(0, 50)}...
                  </Box>
                ))}
              </Box>
              <Box>
                <Box variant="awsui-key-label">Created</Box>
                <Box>{new Date(state.selectedResearchUser.created_at).toLocaleString()}</Box>
              </Box>
            </SpaceBetween>
            <SpaceBetween direction="horizontal" size="xs">
              <Button
                variant="primary"
                iconName="key"
                onClick={() => handleGenerateSSHKey(state.selectedResearchUser!.username)}
              >
                Generate SSH Key
              </Button>
              <Button
                variant="normal"
                iconName="remove"
                onClick={() => handleDeleteUser(state.selectedResearchUser!.username)}
              >
                Delete User
              </Button>
            </SpaceBetween>
          </SpaceBetween>
        );

      default:
        return null;
    }
  };

  // Main application layout
  return (
    <AppLayout
      navigationOpen={navigationOpen}
      onNavigationChange={({ detail }) => setNavigationOpen(detail.open)}
      navigation={
        <SideNavigation
          header={{
            href: '#/',
            text: 'CloudWorkstation'
          }}
          items={[
            { type: 'link', text: 'Templates', href: '#/templates' },
            { type: 'link', text: 'Instances', href: '#/instances' },
            { type: 'link', text: 'Active Connections', href: '#/connections' },
            { type: 'link', text: 'Storage Volumes', href: '#/volumes' },
            { type: 'link', text: 'Research Users', href: '#/research-users' },
            { type: 'divider' },
            { type: 'link', text: 'Settings', href: '#/settings' }
          ]}
          onFollow={(event) => {
            event.preventDefault();
            const view = event.detail.href.split('/')[1] as 'templates' | 'instances' | 'volumes' | 'research-users' | 'connections' | 'settings';
            setState(prev => ({ ...prev, activeView: view }));
          }}
        />
      }
      breadcrumbs={
        <BreadcrumbGroup
          items={getBreadcrumbs()}
          ariaLabel="Breadcrumbs"
        />
      }
      splitPanel={
        state.splitPanelOpen ? (
          <SplitPanel
            header={
              state.splitPanelContent === 'instance-details' ? 'Instance Details' :
              state.splitPanelContent === 'template-details' ? 'Template Details' :
              state.splitPanelContent === 'volume-details' ? 'Volume Details' :
              'Details'
            }
          >
            {renderSplitPanelContent()}
          </SplitPanel>
        ) : undefined
      }
      splitPanelOpen={state.splitPanelOpen}
      onSplitPanelToggle={({ detail }) => {
        setState(prev => ({
          ...prev,
          splitPanelOpen: detail.open,
          splitPanelContent: detail.open ? prev.splitPanelContent : null
        }));
      }}
      content={
        <SpaceBetween direction="vertical" size="l">
          {state.notifications.length > 0 && (
            <Flashbar
              items={state.notifications}
              onDismiss={({ detail }) => {
                setState(prev => ({
                  ...prev,
                  notifications: prev.notifications.filter((_, index) => index !== detail.itemIndex)
                }));
              }}
            />
          )}

          {state.activeView === 'templates' && renderTemplatesView()}
          {state.activeView === 'instances' && renderInstancesView()}
          {state.activeView === 'volumes' && renderVolumesView()}
          {state.activeView === 'research-users' && renderResearchUsersView()}
          {state.activeView === 'connections' && (
            <Container
              header={
                <Header
                  variant="h1"
                  counter={`(${state.connectionTabs.length} active)`}
                  actions={
                    <SpaceBetween direction="horizontal" size="xs">
                      <Button
                        variant="normal"
                        onClick={() => handleAWSServiceConnection('braket')}
                      >
                        Launch Braket
                      </Button>
                      <Button
                        variant="normal"
                        onClick={() => handleAWSServiceConnection('sagemaker')}
                      >
                        Launch SageMaker
                      </Button>
                      <Button
                        variant="normal"
                        onClick={() => handleAWSServiceConnection('console')}
                      >
                        AWS Console
                      </Button>
                      <Button
                        variant="primary"
                        onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
                      >
                        Connect Instance
                      </Button>
                    </SpaceBetween>
                  }
                >
                  Active Connections
                </Header>
              }
            >
              <SpaceBetween direction="vertical" size="l">
                {state.connectionTabs.length > 0 ? (
                  <div>
                    <Header variant="h2">Connection Tabs</Header>
                    {state.connectionTabs.map(tab => (
                      <Container key={tab.id}>
                        <SpaceBetween direction="horizontal" size="s">
                          <Box fontWeight="bold">{tab.title}</Box>
                          <Badge color={
                            tab.status === 'connected' ? 'green' :
                            tab.status === 'connecting' ? 'blue' :
                            'red'
                          }>
                            {tab.status}
                          </Badge>
                          <Button
                            variant="link"
                            onClick={() => closeConnectionTab(tab.id)}
                          >
                            Close
                          </Button>
                        </SpaceBetween>
                        <Box variant="small" color="text-body-secondary">
                          Type: {tab.type} | Category: {tab.category}
                        </Box>
                      </Container>
                    ))}
                  </div>
                ) : (
                  <div style={{ textAlign: 'center', padding: '4rem' }}>
                    <SpaceBetween direction="vertical" size="l">
                      <Header variant="h2">No active connections</Header>
                      <Box variant="p">Connect to an instance or launch an AWS service to get started</Box>
                      <SpaceBetween direction="horizontal" size="s">
                        <Button
                          variant="primary"
                          onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
                        >
                          Connect to Instance
                        </Button>
                        <Button
                          variant="normal"
                          onClick={() => handleAWSServiceConnection('braket')}
                        >
                          Launch AWS Service
                        </Button>
                      </SpaceBetween>
                    </SpaceBetween>
                  </div>
                )}
              </SpaceBetween>
            </Container>
          )}
          {state.activeView === 'settings' && (
            <Container header={<Header variant="h1">Settings</Header>}>
              <Box>Settings interface coming soon...</Box>
            </Container>
          )}

          {/* Launch Modal */}
          <Modal
            visible={launchModalVisible}
            onDismiss={() => setLaunchModalVisible(false)}
            header="Launch Research Environment"
            size="medium"
            footer={
              <Box float="right">
                <SpaceBetween direction="horizontal" size="xs">
                  <Button variant="link" onClick={() => setLaunchModalVisible(false)}>
                    Cancel
                  </Button>
                  <Button
                    variant="primary"
                    onClick={handleLaunchInstance}
                    disabled={!instanceName.trim()}
                  >
                    Launch Instance
                  </Button>
                </SpaceBetween>
              </Box>
            }
          >
            {state.selectedTemplate && (
              <Form>
                <SpaceBetween direction="vertical" size="l">
                  <Alert type="info">
                    Launching <strong>{state.selectedTemplate.Name}</strong> template.
                    Estimated launch time: ~{state.selectedTemplate.EstimatedLaunchTime || 3} minutes.
                  </Alert>

                  <FormField
                    label="Instance Name"
                    description="Choose a descriptive name for your research environment"
                  >
                    <Input
                      value={instanceName}
                      onChange={({ detail }) => setInstanceName(detail.value)}
                      placeholder="my-research-project"
                    />
                  </FormField>

                  <FormField
                    label="Instance Size"
                    description="Select the compute resources for your workload"
                  >
                    <Select
                      selectedOption={{ label: instanceSize === 'S' ? 'Small - 2 CPU, 4GB RAM (Best for testing)' :
                        instanceSize === 'M' ? 'Medium - 2 CPU, 8GB RAM (Recommended)' :
                        instanceSize === 'L' ? 'Large - 4 CPU, 16GB RAM (Data analysis)' :
                        'Extra Large - 8 CPU, 32GB RAM (Heavy workloads)', value: instanceSize }}
                      onChange={({ detail }) => setInstanceSize(detail.selectedOption.value || 'M')}
                      options={[
                        { label: 'Small - 2 CPU, 4GB RAM (Best for testing)', value: 'S' },
                        { label: 'Medium - 2 CPU, 8GB RAM (Recommended)', value: 'M' },
                        { label: 'Large - 4 CPU, 16GB RAM (Data analysis)', value: 'L' },
                        { label: 'Extra Large - 8 CPU, 32GB RAM (Heavy workloads)', value: 'XL' }
                      ]}
                    />
                  </FormField>

                  <Box>
                    <Box variant="awsui-key-label">Estimated Cost</Box>
                    <Box variant="h3">
                      ${((state.selectedTemplate.EstimatedCostPerHour?.['x86_64'] || 0.12) *
                      (instanceSize === 'S' ? 1 : instanceSize === 'M' ? 2 : instanceSize === 'L' ? 4 : 8)).toFixed(2)}/hour
                    </Box>
                  </Box>
                </SpaceBetween>
              </Form>
            )}
          </Modal>

          {/* Mount Volume Modal */}
          <Modal
            visible={state.showMountDialog}
            onDismiss={() => {
              setState(prev => ({
                ...prev,
                showMountDialog: false,
                mountingVolume: null
              }));
              setMountInstanceName('');
              setMountPoint('/mnt/shared-volume');
            }}
            header="Mount EFS Volume"
            size="medium"
            footer={
              <Box float="right">
                <SpaceBetween direction="horizontal" size="xs">
                  <Button
                    variant="link"
                    onClick={() => {
                      setState(prev => ({
                        ...prev,
                        showMountDialog: false,
                        mountingVolume: null
                      }));
                      setMountInstanceName('');
                      setMountPoint('/mnt/shared-volume');
                    }}
                  >
                    Cancel
                  </Button>
                  <Button
                    variant="primary"
                    onClick={handleMountVolume}
                    disabled={!mountInstanceName.trim()}
                  >
                    Mount Volume
                  </Button>
                </SpaceBetween>
              </Box>
            }
          >
            {state.mountingVolume && (
              <Form>
                <SpaceBetween direction="vertical" size="l">
                  <Alert type="info">
                    Mounting <strong>{state.mountingVolume.name}</strong> ({state.mountingVolume.size_gb} GB)
                    for multi-instance file sharing.
                  </Alert>

                  <FormField
                    label="Target Instance"
                    description="Select the running instance to mount the volume to"
                  >
                    <Select
                      selectedOption={
                        mountInstanceName ?
                          { label: mountInstanceName, value: mountInstanceName } :
                          { label: 'Select an instance...', value: '' }
                      }
                      onChange={({ detail }) => setMountInstanceName(detail.selectedOption.value || '')}
                      options={[
                        ...state.instances
                          .filter(i => i.status === 'running')
                          .map(i => ({ label: `${i.name} (${i.template})`, value: i.name })),
                        ...state.instances
                          .filter(i => i.status !== 'running')
                          .map(i => ({
                            label: `${i.name} (${i.status}) - Not Available`,
                            value: i.name,
                            disabled: true
                          }))
                      ]}
                      placeholder="Select running instance..."
                    />
                  </FormField>

                  <FormField
                    label="Mount Point"
                    description="Directory path where the volume will be accessible"
                  >
                    <Input
                      value={mountPoint}
                      onChange={({ detail }) => setMountPoint(detail.value)}
                      placeholder="/mnt/shared-volume"
                    />
                  </FormField>

                  {state.instances.filter(i => i.status === 'running').length === 0 && (
                    <Alert type="warning">
                      No running instances found. Start an instance first to mount volumes.
                    </Alert>
                  )}

                  <Box>
                    <Box variant="awsui-key-label">Volume Details</Box>
                    <SpaceBetween direction="horizontal" size="l">
                      <Box>
                        <Box variant="small" color="text-body-secondary">Size</Box>
                        <Box>{state.mountingVolume.size_gb} GB</Box>
                      </Box>
                      <Box>
                        <Box variant="small" color="text-body-secondary">Monthly Cost</Box>
                        <Box>${(state.mountingVolume.size_gb * state.mountingVolume.cost_per_gb).toFixed(2)}</Box>
                      </Box>
                      <Box>
                        <Box variant="small" color="text-body-secondary">Region</Box>
                        <Box>{state.mountingVolume.region}</Box>
                      </Box>
                    </SpaceBetween>
                  </Box>
                </SpaceBetween>
              </Form>
            )}
          </Modal>

          {/* Create Research User Modal */}
          <Modal
            visible={createUserModalVisible}
            onDismiss={() => {
              setCreateUserModalVisible(false);
              setNewUsername('');
            }}
            header="Create Research User"
            size="medium"
            footer={
              <Box float="right">
                <SpaceBetween direction="horizontal" size="xs">
                  <Button
                    variant="link"
                    onClick={() => {
                      setCreateUserModalVisible(false);
                      setNewUsername('');
                    }}
                  >
                    Cancel
                  </Button>
                  <Button
                    variant="primary"
                    onClick={handleCreateUser}
                    disabled={!newUsername.trim()}
                  >
                    Create User
                  </Button>
                </SpaceBetween>
              </Box>
            }
          >
            <Form>
              <SpaceBetween direction="vertical" size="l">
                <Alert type="info">
                  Research users provide persistent identity and SSH access across all CloudWorkstation instances.
                  The user will be automatically configured with EFS home directory and SSH key management.
                </Alert>
                <FormField
                  label="Username"
                  description="Choose a unique username (lowercase letters, numbers, hyphens only)"
                >
                  <Input
                    value={newUsername}
                    onChange={({ detail }) => setNewUsername(detail.value)}
                    placeholder="researcher-name"
                  />
                </FormField>
                <Box>
                  <Box variant="awsui-key-label">What will be created:</Box>
                  <SpaceBetween direction="vertical" size="xs">
                    <Box>• Persistent user account with consistent UID/GID</Box>
                    <Box>• EFS home directory at /home/{newUsername || '{username}'}</Box>
                    <Box>• SSH key generation capability</Box>
                    <Box>• Cross-instance identity preservation</Box>
                  </SpaceBetween>
                </Box>
              </SpaceBetween>
            </Form>
          </Modal>
        </SpaceBetween>
      }
    />
  );
}