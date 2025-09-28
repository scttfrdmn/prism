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
  Box
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

interface CloudWorkstationState {
  activeView: 'templates' | 'instances' | 'desktop' | 'settings';
  templates: Template[];
  instances: Instance[];
  selectedTemplate: Template | null;
  loading: boolean;
  notifications: any[];
}

// Declare wails API for TypeScript
declare global {
  interface Window {
    wails: {
      CloudWorkstationService: {
        GetTemplates: () => Promise<Template[]>;
        GetInstances: () => Promise<Instance[]>;
        LaunchInstance: (name: string, templateName: string, size: string) => Promise<void>;
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
    selectedTemplate: null,
    loading: true,
    notifications: []
  });

  const [navigationOpen, setNavigationOpen] = useState(false);
  const [launchModalVisible, setLaunchModalVisible] = useState(false);
  const [instanceName, setInstanceName] = useState('');
  const [instanceSize, setInstanceSize] = useState('M');
  const [templateQuery, setTemplateQuery] = useState({ tokens: [], operation: 'and' as const });
  const [filteredTemplates, setFilteredTemplates] = useState<Template[]>([]);
  const [selectedInstances, setSelectedInstances] = useState<Instance[]>([]);

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

      setState(prev => ({
        ...prev,
        templates,
        instances,
        loading: false
      }));

      // Note: filteredTemplates will be updated by useEffect when state.templates changes
    } catch (error) {
      console.error('Failed to load application data:', error);
      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [{
          type: 'error',
          header: 'Failed to load data',
          content: 'Unable to connect to CloudWorkstation daemon',
          dismissible: true
        }]
      }));
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
    setState(prev => ({ ...prev, selectedTemplate }));

    if (selectedTemplate) {
      setInstanceName(`my-${selectedTemplate.Name.toLowerCase().replace(/[^a-z0-9]/g, '-')}`);
      setLaunchModalVisible(true);
    }
  };

  // Handle instance launch
  const handleLaunchInstance = async () => {
    if (!state.selectedTemplate || !instanceName.trim()) return;

    try {
      if (window.wails?.CloudWorkstationService?.LaunchInstance) {
        await window.wails.CloudWorkstationService.LaunchInstance(
          instanceName.trim(),
          state.selectedTemplate.Name,
          instanceSize
        );

        setState(prev => ({
          ...prev,
          notifications: [{
            type: 'success',
            header: 'Instance launching',
            content: `${instanceName} is being launched with ${state.selectedTemplate.Name} template`,
            dismissible: true
          }]
        }));
      }

      setLaunchModalVisible(false);
      setInstanceName('');
      setState(prev => ({ ...prev, selectedTemplate: null }));

      // Refresh instances
      loadApplicationData();
    } catch (error) {
      setState(prev => ({
        ...prev,
        notifications: [{
          type: 'error',
          header: 'Launch failed',
          content: `Failed to launch ${instanceName}: ${error}`,
          dismissible: true
        }]
      }));
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

  // Handle instance actions
  const handleInstanceAction = async (action: string, instance: Instance) => {
    setState(prev => ({
      ...prev,
      notifications: [{
        type: 'info',
        header: `${action} in progress`,
        content: `${action} operation started for ${instance.name}`,
        dismissible: true
      }]
    }));

    try {
      // TODO: Call actual API when available
      await new Promise(resolve => setTimeout(resolve, 1000)); // Simulate API call

      setState(prev => ({
        ...prev,
        notifications: [{
          type: 'success',
          header: `${action} successful`,
          content: `${instance.name} ${action.toLowerCase()} completed successfully`,
          dismissible: true
        }]
      }));
    } catch (error) {
      setState(prev => ({
        ...prev,
        notifications: [{
          type: 'error',
          header: `${action} failed`,
          content: `Failed to ${action.toLowerCase()} ${instance.name}: ${error}`,
          dismissible: true
        }]
      }));
    }
  };

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
            { type: 'divider' },
            { type: 'link', text: 'Settings', href: '#/settings' }
          ]}
          onFollow={(event) => {
            event.preventDefault();
            const view = event.detail.href.split('/')[1] as 'templates' | 'instances' | 'settings';
            setState(prev => ({ ...prev, activeView: view }));
          }}
        />
      }
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
        </SpaceBetween>
      }
    />
  );
}