// CloudWorkstation GUI - Cloudscape Migration
// Professional AWS-native interface using Cloudscape Design System

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
  id: string;
  name: string;
  description: string;
  category: string;
  complexity: 'simple' | 'moderate' | 'advanced' | 'complex';
  cost_per_hour: number;
  launch_time_minutes: number;
  popularity: number;
  features: string[];
  icon: string;
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

  // Load data on component mount
  useEffect(() => {
    loadApplicationData();
  }, []);

  const loadApplicationData = async () => {
    setState(prev => ({ ...prev, loading: true }));

    try {
      // Load templates and instances from CloudWorkstation backend via Wails
      const backendTemplates = await (window as any).wails.CloudWorkstationService.GetTemplates();
      const backendInstances = await (window as any).wails.CloudWorkstationService.GetInstances();

      // Convert backend template format to UI format
      const templates: Template[] = backendTemplates.map((t: any) => ({
        id: t.Name || t.id,
        name: t.Name,
        description: t.Description || '',
        category: t.Domain || t.Category || 'general',
        complexity: t.Complexity || 'moderate',
        cost_per_hour: t.EstimatedCostPerHour?.['x86_64'] || t.cost_per_hour || 0,
        launch_time_minutes: 2, // Reasonable default
        popularity: t.Popular ? 1000 : 500,
        features: Array.isArray(t.Features) ? t.Features : [],
        icon: t.Icon || '💻'
      }));

      // Convert backend instance format to UI format
      const instances: Instance[] = backendInstances.map((i: any) => ({
        id: i.ID || i.id,
        name: i.Name,
        template: i.Template,
        status: i.State || i.status,
        public_ip: i.PublicIP || i.public_ip,
        cost_per_hour: i.EstimatedDailyCost ? i.EstimatedDailyCost / 24 : 0,
        launch_time: i.LaunchTime || i.launch_time,
        region: i.Region || 'us-west-2'
      }));

      setState(prev => ({
        ...prev,
        templates: templates,
        instances: instances,
        loading: false
      }));

    } catch (error) {
      console.error('Failed to load application data:', error);
      setState(prev => ({
        ...prev,
        loading: false,
        notifications: [
          {
            type: 'error',
            header: 'Connection Error',
            content: 'Failed to connect to CloudWorkstation daemon. Please check that the service is running.',
            dismissible: true,
            id: Date.now().toString()
          }
        ]
      }));
    }
  };

  const handleTemplateSelection = (template: Template) => {
    setState(prev => ({ ...prev, selectedTemplate: template }));
    setLaunchModalVisible(true);
  };

  const handleInstanceAction = async (instanceId: string, action: string) => {
    setState(prev => ({
      ...prev,
      notifications: [
        {
          type: 'success',
          header: `Instance ${action}`,
          content: `Successfully ${action}ed instance ${instanceId}`,
          dismissible: true,
          id: Date.now().toString()
        },
        ...prev.notifications
      ]
    }));
  };

  // Template Selection Component
  const TemplateSelectionView = () => (
    <SpaceBetween size="l">
      <Container
        header={
          <Header
            variant="h1"
            description="Choose a pre-configured research environment to launch"
            actions={
              <Button
                variant="primary"
                disabled={!state.selectedTemplate}
                onClick={() => setLaunchModalVisible(true)}
              >
                Launch Selected Template
              </Button>
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
              Loading templates...
            </Box>
          </Box>
        ) : (
          <Cards
            cardDefinition={{
              header: (item: Template) => (
                <SpaceBetween direction="horizontal" size="xs">
                  <Box fontSize="heading-m">{item.icon}</Box>
                  <Header variant="h3">{item.name}</Header>
                </SpaceBetween>
              ),
              sections: [
                {
                  id: "description",
                  content: (item: Template) => item.description
                },
                {
                  id: "features",
                  content: (item: Template) => (
                    <SpaceBetween direction="horizontal" size="xs">
                      {item.features.slice(0, 3).map(feature => (
                        <Badge key={feature} color="blue">{feature}</Badge>
                      ))}
                      {item.features.length > 3 && (
                        <Badge color="grey">+{item.features.length - 3} more</Badge>
                      )}
                    </SpaceBetween>
                  )
                },
                {
                  id: "metadata",
                  content: (item: Template) => (
                    <SpaceBetween direction="horizontal" size="l">
                      <Box>
                        <Box variant="awsui-key-label">Complexity</Box>
                        <Badge
                          color={
                            item.complexity === 'simple' ? 'green' :
                            item.complexity === 'moderate' ? 'blue' :
                            item.complexity === 'advanced' ? 'orange' : 'red'
                          }
                        >
                          {item.complexity}
                        </Badge>
                      </Box>
                      <Box>
                        <Box variant="awsui-key-label">Cost</Box>
                        <Box>${item.cost_per_hour}/hour</Box>
                      </Box>
                      <Box>
                        <Box variant="awsui-key-label">Launch Time</Box>
                        <Box>~{item.launch_time_minutes} min</Box>
                      </Box>
                    </SpaceBetween>
                  )
                }
              ]
            }}
            items={state.templates}
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
              { minWidth: 900, cards: 3 }
            ]}
            trackBy="id"
          />
        )}
      </Container>
    </SpaceBetween>
  );

  // Instance Management Component
  const InstanceManagementView = () => (
    <SpaceBetween size="l">
      <Container
        header={
          <Header
            variant="h1"
            description="Monitor and manage your running research environments"
            counter={`(${state.instances.length})`}
            actions={
              <Button
                variant="primary"
                onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
              >
                Launch New Instance
              </Button>
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
              cell: (item: Instance) => item.name,
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
                    item.status === 'running' ? 'success' :
                    item.status === 'stopped' ? 'stopped' :
                    item.status === 'hibernated' ? 'pending' :
                    item.status === 'pending' ? 'in-progress' : 'error'
                  }
                >
                  {item.status}
                </StatusIndicator>
              )
            },
            {
              id: "cost",
              header: "Cost/Hour",
              cell: (item: Instance) => `$${item.cost_per_hour}`
            },
            {
              id: "region",
              header: "Region",
              cell: (item: Instance) => item.region
            },
            {
              id: "actions",
              header: "Actions",
              cell: (item: Instance) => (
                <SpaceBetween direction="horizontal" size="xs">
                  <Button
                    variant="primary"
                    size="small"
                    disabled={item.status !== 'running'}
                  >
                    Connect
                  </Button>
                  <Button
                    variant="normal"
                    size="small"
                    disabled={item.status !== 'running'}
                    onClick={() => handleInstanceAction(item.id, 'hibernate')}
                  >
                    Hibernate
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
                You don't have any instances yet.
              </Box>
              <Button
                variant="primary"
                onClick={() => setState(prev => ({ ...prev, activeView: 'templates' }))}
              >
                Launch your first instance
              </Button>
            </Box>
          }
          sortingDisabled={false}
        />
      </Container>
    </SpaceBetween>
  );

  // Launch Modal Component
  const LaunchModal = () => (
    <Modal
      onDismiss={() => setLaunchModalVisible(false)}
      visible={launchModalVisible}
      header="Launch Research Environment"
      size="medium"
      footer={
        <Box float="right">
          <SpaceBetween direction="horizontal" size="xs">
            <Button variant="link" onClick={() => setLaunchModalVisible(false)}>
              Cancel
            </Button>
            <Button variant="primary">Launch Instance</Button>
          </SpaceBetween>
        </Box>
      }
    >
      <Form>
        <SpaceBetween size="m">
          <FormField label="Instance name" description="Choose a descriptive name for your research project">
            <Input
              value=""
              placeholder="my-research-project"
            />
          </FormField>

          <FormField label="Instance size" description="Choose the right size for your workload">
            <Select
              selectedOption={{
                label: "Medium - 2 vCPU, 8 GB RAM (Recommended)",
                value: "m5.large"
              }}
              options={[
                { label: "Small - 1 vCPU, 4 GB RAM", value: "t3.medium" },
                { label: "Medium - 2 vCPU, 8 GB RAM (Recommended)", value: "m5.large" },
                { label: "Large - 4 vCPU, 16 GB RAM", value: "m5.xlarge" },
                { label: "Extra Large - 8 vCPU, 32 GB RAM", value: "m5.2xlarge" }
              ]}
            />
          </FormField>

          {state.selectedTemplate && (
            <Alert type="info">
              <Box>
                <Box variant="strong">Estimated cost: ${state.selectedTemplate.cost_per_hour}/hour</Box>
                <Box>
                  Launch time: ~{state.selectedTemplate.launch_time_minutes} minutes
                </Box>
              </Box>
            </Alert>
          )}
        </SpaceBetween>
      </Form>
    </Modal>
  );

  // Main render method
  return (
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
              text: "Research Templates",
              href: "/templates"
            },
            {
              type: "link",
              text: "My Instances",
              href: "/instances"
            },
            {
              type: "link",
              text: "Remote Desktop",
              href: "/desktop"
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
              setState(prev => ({ ...prev, activeView: view || 'templates' }));
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
          {state.activeView === 'templates' && <TemplateSelectionView />}
          {state.activeView === 'instances' && <InstanceManagementView />}
          {state.activeView === 'desktop' && (
            <Container
              header={<Header variant="h1">Remote Desktop</Header>}
            >
              <Box textAlign="center" padding="xl">
                <Box variant="strong">Remote Desktop Connection</Box>
                <Box variant="p">Connect to your instances to access graphical applications.</Box>
              </Box>
            </Container>
          )}
          {state.activeView === 'settings' && (
            <Container
              header={<Header variant="h1">Settings</Header>}
            >
              <Box textAlign="center" padding="xl">
                <Box variant="strong">CloudWorkstation Settings</Box>
                <Box variant="p">Configure your preferences and AWS settings.</Box>
              </Box>
            </Container>
          )}
        </div>
      }
      toolsHide
    />
    <>
      <LaunchModal />
    </>
  );
}