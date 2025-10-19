# CloudWorkstation Tabbed Embedded Connections Implementation Plan

## 🎯 Vision Statement

Transform CloudWorkstation GUI into a unified research platform with tabbed embedded connections supporting traditional compute instances alongside AWS research services (SageMaker, Braket, Console, CloudShell) within a professional Cloudscape-based interface.

## 🏗️ Architecture Overview

### Current State Analysis
- **Backend**: Connection handlers exist but launch external applications
- **Frontend**: Mock `handleInstanceAction('Connect')` with no real implementation
- **Gap**: Missing Wails API bindings between Go backend and TypeScript frontend
- **Issue**: No embedded components - all connections are external launches

### Target Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                    CloudWorkstation GUI (Cloudscape)                │
├─────────────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                    Cloudscape Tabs Component                    │ │
│  │ ┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐┌────────┐   │ │
│  │ │SSH Tab ││DCV Tab ││Web Tab ││SageMkr ││Braket  ││Console │   │ │
│  │ └────────┘└────────┘└────────┘└────────┘└────────┘└────────┘   │ │
│  └─────────────────────────────────────────────────────────────────┘ │
│  ┌─────────────────────────────────────────────────────────────────┐ │
│  │                       Active Tab Content                        │ │
│  │ ┌─────────────────────────────────────────────────────────────┐ │ │
│  │ │  EmbeddedTerminal | EmbeddedDCV | EmbeddedWeb | EmbeddedAWS │ │ │
│  │ │ ┌─────────────────────────────────────────────────────────┐ │ │ │
│  │ │ │    xterm.js | DCV Client | iFrame | AWS Service UI     │ │ │ │
│  │ │ └─────────────────────────────────────────────────────────┘ │ │ │
│  │ └─────────────────────────────────────────────────────────────┘ │ │
│  └─────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
                                  │
                                  ▼
┌─────────────────────────────────────────────────────────────────────┐
│                         Backend Services                            │
│ ┌─────────────┐┌─────────────┐┌─────────────┐┌─────────────────────┐ │
│ │ SSH Proxy   ││ DCV Proxy   ││ Web Proxy   ││ AWS Service Proxy   │ │
│ │(/ssh-proxy) ││(/dcv-proxy) ││(/proxy)     ││(/aws-proxy)         │ │
│ └─────────────┘└─────────────┘└─────────────┘└─────────────────────┘ │
└─────────────────────────────────────────────────────────────────────┘
```

## 🎯 Service Integration Matrix

### Tier 1: Core Research Services (High Priority)
| Service | Use Case | Embedding Method | Auth Method | Implementation Priority |
|---------|----------|------------------|-------------|------------------------|
| **Amazon Braket** | Quantum computing research | iframe + API | Federation | 🔥 Critical |
| **SageMaker Studio** | ML/AI development | Specialized iframe | Federation | 🔥 Critical |
| **AWS Console** | Resource management | iframe with proxy | Federation | 🔥 Critical |
| **CloudShell** | CLI access with AWS context | iframe | Federation | 🔥 Critical |

### Tier 2: Analytics & Data Services
| Service | Use Case | Embedding Method | Auth Method | Implementation Priority |
|---------|----------|------------------|-------------|------------------------|
| **Athena Query Editor** | SQL data analysis | iframe | Federation | ⭐ High |
| **QuickSight** | Data visualization | iframe | Federation | ⭐ High |
| **EMR Studio** | Big data analytics | iframe | Federation | ⭐ High |
| **Glue DataBrew** | Data preparation | iframe | Federation | ⭐ High |

### Tier 3: Development & Monitoring
| Service | Use Case | Embedding Method | Auth Method | Implementation Priority |
|---------|----------|------------------|-------------|------------------------|
| **CloudWatch** | Monitoring & logs | iframe | Federation | ⚡ Medium |
| **Cost Explorer** | Budget analysis | iframe | Federation | ⚡ Medium |
| **Well-Architected Tool** | Architecture review | iframe | Federation | ⚡ Low |

## 📋 Implementation Phases

### Phase 1: Foundation & Wails API Integration (Week 1-3)

#### Backend Updates (Go)

**1.1 Enhanced Connection Types**
```go
// cmd/cws-gui/service.go - Enhanced connection types
type ConnectionType string
const (
    ConnectionTypeSSH       ConnectionType = "ssh"
    ConnectionTypeDesktop   ConnectionType = "desktop"
    ConnectionTypeWeb       ConnectionType = "web"
    ConnectionTypeAWS       ConnectionType = "aws-service"
)

// Enhanced connection configuration
type ConnectionConfig struct {
    ID              string                 `json:"id"`
    Type            ConnectionType         `json:"type"`
    InstanceName    string                 `json:"instance_name,omitempty"`
    AWSService      string                 `json:"aws_service,omitempty"`
    Region          string                 `json:"region,omitempty"`
    ProxyURL        string                 `json:"proxy_url"`
    AuthToken       string                 `json:"auth_token,omitempty"`
    EmbeddingMode   string                 `json:"embedding_mode"` // iframe, api, websocket
    Metadata        map[string]interface{} `json:"metadata,omitempty"`
}
```

**1.2 AWS Service Integration**
```go
// cmd/cws-gui/aws_service_handlers.go - New file
package main

import (
    "context"
    "fmt"
    "github.com/aws/aws-sdk-go-v2/service/sts"
)

// AWS service connection handlers
func (s *CloudWorkstationService) OpenAWSService(ctx context.Context,
    service string, region string) (*ConnectionConfig, error) {

    // Generate federated token for AWS service access
    token, err := s.generateServiceToken(ctx, service, region)
    if err != nil {
        return nil, fmt.Errorf("failed to generate service token: %w", err)
    }

    // Build service-specific configuration
    config := &ConnectionConfig{
        ID:            fmt.Sprintf("aws-%s-%s", service, region),
        Type:          ConnectionTypeAWS,
        AWSService:    service,
        Region:        region,
        ProxyURL:      s.buildAWSServiceURL(service, region),
        AuthToken:     token,
        EmbeddingMode: s.getServiceEmbeddingMode(service),
    }

    return config, nil
}

// Specialized AWS service handlers
func (s *CloudWorkstationService) OpenBraketConsole(ctx context.Context, region string) (*ConnectionConfig, error) {
    return s.OpenAWSService(ctx, "braket", region)
}

func (s *CloudWorkstationService) OpenSageMakerStudio(ctx context.Context, region string) (*ConnectionConfig, error) {
    return s.OpenAWSService(ctx, "sagemaker", region)
}

func (s *CloudWorkstationService) OpenAWSConsole(ctx context.Context, service string, region string) (*ConnectionConfig, error) {
    return s.OpenAWSService(ctx, "console", region)
}

func (s *CloudWorkstationService) OpenCloudShell(ctx context.Context, region string) (*ConnectionConfig, error) {
    return s.OpenAWSService(ctx, "cloudshell", region)
}
```

**1.3 Enhanced Instance Connection Handlers**
```go
// cmd/cws-gui/service.go - Update existing methods
func (s *CloudWorkstationService) OpenEmbeddedTerminal(ctx context.Context, instanceName string) (*ConnectionConfig, error) {
    access, err := s.GetInstanceAccess(ctx, instanceName)
    if err != nil {
        return nil, err
    }

    return &ConnectionConfig{
        ID:            fmt.Sprintf("ssh-%s", instanceName),
        Type:          ConnectionTypeSSH,
        InstanceName:  instanceName,
        ProxyURL:      fmt.Sprintf("%s/ssh-proxy/%s", s.daemonURL, instanceName),
        EmbeddingMode: "websocket",
        Metadata: map[string]interface{}{
            "host": access.PublicIP,
            "port": access.SSHPort,
            "username": access.Username,
        },
    }, nil
}

func (s *CloudWorkstationService) OpenEmbeddedDesktop(ctx context.Context, instanceName string) (*ConnectionConfig, error) {
    access, err := s.GetInstanceAccess(ctx, instanceName)
    if err != nil {
        return nil, err
    }

    return &ConnectionConfig{
        ID:            fmt.Sprintf("desktop-%s", instanceName),
        Type:          ConnectionTypeDesktop,
        InstanceName:  instanceName,
        ProxyURL:      fmt.Sprintf("%s/dcv-proxy/%s", s.daemonURL, instanceName),
        EmbeddingMode: "iframe",
        Metadata: map[string]interface{}{
            "host": access.PublicIP,
            "rdp_port": access.RDPPort,
            "vnc_port": access.VNCPort,
        },
    }, nil
}

func (s *CloudWorkstationService) OpenEmbeddedWeb(ctx context.Context, instanceName string) (*ConnectionConfig, error) {
    access, err := s.GetInstanceAccess(ctx, instanceName)
    if err != nil {
        return nil, err
    }

    return &ConnectionConfig{
        ID:            fmt.Sprintf("web-%s", instanceName),
        Type:          ConnectionTypeWeb,
        InstanceName:  instanceName,
        ProxyURL:      fmt.Sprintf("%s/proxy/%s", s.daemonURL, instanceName),
        EmbeddingMode: "iframe",
        Metadata: map[string]interface{}{
            "web_url": access.WebURL,
            "web_port": access.WebPort,
        },
    }, nil
}
```

#### Backend Proxy Infrastructure

**1.4 SSH Proxy Endpoint**
```go
// In daemon server.go - Add SSH WebSocket proxy
func (s *Server) RegisterSSHProxyRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/ssh-proxy/", s.handleSSHProxy)
}

func (s *Server) handleSSHProxy(w http.ResponseWriter, r *http.Request) {
    // WebSocket upgrade for terminal communication
    // SSH connection multiplexing
    // Terminal session management
}
```

**1.5 AWS Service Proxy Endpoint**
```go
// In daemon server.go - Add AWS service proxy
func (s *Server) RegisterAWSServiceProxyRoutes(mux *http.ServeMux) {
    mux.HandleFunc("/aws-proxy/", s.handleAWSServiceProxy)
}

func (s *Server) handleAWSServiceProxy(w http.ResponseWriter, r *http.Request) {
    // AWS service URL routing
    // Federation token injection
    // CORS header management
}
```

#### Frontend Updates (TypeScript)

**1.6 Enhanced Wails API Interface**
```tsx
// cmd/cws-gui/frontend/src/App.tsx - Enhanced type definitions
interface ConnectionConfig {
  id: string;
  type: 'ssh' | 'desktop' | 'web' | 'aws-service';
  instanceName?: string;
  awsService?: string;
  region?: string;
  proxyUrl: string;
  authToken?: string;
  embeddingMode: 'iframe' | 'websocket' | 'api';
  metadata?: Record<string, any>;
}

// Enhanced Wails API declarations
declare global {
  interface Window {
    wails: {
      CloudWorkstationService: {
        // Instance connections
        OpenEmbeddedTerminal: (instanceName: string) => Promise<ConnectionConfig>;
        OpenEmbeddedDesktop: (instanceName: string) => Promise<ConnectionConfig>;
        OpenEmbeddedWeb: (instanceName: string) => Promise<ConnectionConfig>;

        // AWS service connections
        OpenBraketConsole: (region: string) => Promise<ConnectionConfig>;
        OpenSageMakerStudio: (region: string) => Promise<ConnectionConfig>;
        OpenAWSConsole: (service: string, region: string) => Promise<ConnectionConfig>;
        OpenCloudShell: (region: string) => Promise<ConnectionConfig>;

        // Generic AWS service
        OpenAWSService: (service: string, region: string) => Promise<ConnectionConfig>;
      };
    };
  }
}
```

**1.7 Replace Mock Implementation**
```tsx
// cmd/cws-gui/frontend/src/App.tsx - Real implementation
const handleInstanceAction = async (action: string, instance: Instance) => {
  if (action === 'Connect') {
    try {
      // Determine connection type based on instance capabilities
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
          throw new Error('No supported connection type available');
      }

      // Create new connection tab
      createConnectionTab(config);

    } catch (error) {
      addNotification({
        type: 'error',
        header: 'Connection failed',
        content: `Failed to connect to ${instance.name}: ${error.message}`,
        dismissible: true
      });
    }
  }
  // ... other actions
};
```

### Phase 2: Tab Management System (Week 3-4)

#### Connection Tab State Management

**2.1 Enhanced App State**
```tsx
// cmd/cws-gui/frontend/src/App.tsx - Enhanced state
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

interface CloudWorkstationState {
  activeView: 'templates' | 'instances' | 'volumes' | 'research-users' | 'connections' | 'settings';
  // ... existing state

  // New connection state
  connectionTabs: ConnectionTab[];
  activeConnectionTab: string | null;
  showConnectionPanel: boolean;
}
```

**2.2 Tab Management Functions**
```tsx
const createConnectionTab = (config: ConnectionConfig) => {
  const tab: ConnectionTab = {
    id: config.id,
    title: generateTabTitle(config),
    type: config.instanceName ? 'instance' : 'aws-service',
    category: determineCategory(config),
    config,
    active: true,
    closeable: true,
    status: 'connecting'
  };

  setState(prev => ({
    ...prev,
    connectionTabs: [...prev.connectionTabs, tab],
    activeConnectionTab: tab.id,
    showConnectionPanel: true,
    activeView: 'connections'
  }));
};

const closeConnectionTab = (tabId: string) => {
  setState(prev => {
    const tabs = prev.connectionTabs.filter(tab => tab.id !== tabId);
    const activeTab = tabs.length > 0 ? tabs[tabs.length - 1].id : null;

    return {
      ...prev,
      connectionTabs: tabs,
      activeConnectionTab: activeTab,
      showConnectionPanel: tabs.length > 0
    };
  });
};
```

#### AWS Service Quick Launch

**2.3 AWS Service Launcher Component**
```tsx
const AWSServiceLauncher = () => {
  const [selectedRegion, setSelectedRegion] = useState('us-west-2');

  const serviceCategories = {
    research: [
      {
        name: 'Amazon Braket',
        service: 'braket',
        icon: '⚛️',
        description: 'Quantum computing research and experimentation',
        handler: () => window.wails.CloudWorkstationService.OpenBraketConsole(selectedRegion)
      },
      {
        name: 'SageMaker Studio',
        service: 'sagemaker',
        icon: '🤖',
        description: 'ML/AI development and training',
        handler: () => window.wails.CloudWorkstationService.OpenSageMakerStudio(selectedRegion)
      },
    ],
    compute: [
      {
        name: 'CloudShell',
        service: 'cloudshell',
        icon: '🖥️',
        description: 'Browser-based terminal with AWS CLI',
        handler: () => window.wails.CloudWorkstationService.OpenCloudShell(selectedRegion)
      }
    ],
    management: [
      {
        name: 'AWS Console',
        service: 'console',
        icon: '🎛️',
        description: 'AWS resource management',
        handler: () => window.wails.CloudWorkstationService.OpenAWSConsole('ec2', selectedRegion)
      }
    ]
  };

  return (
    <Container header={<Header variant="h2">AWS Research Services</Header>}>
      <SpaceBetween direction="vertical" size="l">
        <FormField label="Region">
          <Select
            selectedOption={{ label: selectedRegion, value: selectedRegion }}
            onChange={({ detail }) => setSelectedRegion(detail.selectedOption.value)}
            options={[
              { label: 'US West (Oregon)', value: 'us-west-2' },
              { label: 'US East (N. Virginia)', value: 'us-east-1' },
              { label: 'EU (Ireland)', value: 'eu-west-1' }
            ]}
          />
        </FormField>

        {Object.entries(serviceCategories).map(([category, services]) => (
          <Cards
            key={category}
            cardDefinition={{
              header: (item) => (
                <Header variant="h3">
                  <SpaceBetween direction="horizontal" size="xs">
                    <span>{item.icon}</span>
                    <span>{item.name}</span>
                  </SpaceBetween>
                </Header>
              ),
              sections: [
                {
                  id: 'description',
                  content: (item) => item.description
                }
              ]
            }}
            items={services}
            cardsPerRow={[{ cards: 1 }, { minWidth: 500, cards: 2 }]}
            onSelectionChange={({ detail }) => {
              const service = detail.selectedItems[0];
              if (service) {
                service.handler().then(createConnectionTab);
              }
            }}
          />
        ))}
      </SpaceBetween>
    </Container>
  );
};
```

### Phase 3: Embedded Components (Week 4-6)

#### SSH Terminal Component

**3.1 Embedded Terminal**
```tsx
// cmd/cws-gui/frontend/src/components/EmbeddedTerminal.tsx
import React, { useEffect, useRef } from 'react';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import { WebLinksAddon } from 'xterm-addon-web-links';

interface EmbeddedTerminalProps {
  config: ConnectionConfig;
  onStatusChange: (status: 'connecting' | 'connected' | 'disconnected') => void;
}

export const EmbeddedTerminal: React.FC<EmbeddedTerminalProps> = ({
  config,
  onStatusChange
}) => {
  const terminalRef = useRef<HTMLDivElement>(null);
  const terminal = useRef<Terminal>();
  const websocket = useRef<WebSocket>();

  useEffect(() => {
    if (!terminalRef.current) return;

    // Initialize xterm.js terminal
    terminal.current = new Terminal({
      theme: {
        background: '#1e1e1e',
        foreground: '#ffffff',
        cursor: '#00ff00'
      },
      fontSize: 14,
      fontFamily: 'Monaco, "Lucida Console", monospace'
    });

    const fitAddon = new FitAddon();
    terminal.current.loadAddon(fitAddon);
    terminal.current.loadAddon(new WebLinksAddon());

    terminal.current.open(terminalRef.current);
    fitAddon.fit();

    // Connect WebSocket for SSH proxy
    onStatusChange('connecting');
    websocket.current = new WebSocket(config.proxyUrl.replace('http', 'ws'));

    websocket.current.onopen = () => {
      onStatusChange('connected');
      terminal.current?.writeln('Connected to ' + config.instanceName);
    };

    websocket.current.onmessage = (event) => {
      terminal.current?.write(event.data);
    };

    websocket.current.onclose = () => {
      onStatusChange('disconnected');
      terminal.current?.writeln('\r\nConnection closed');
    };

    // Handle terminal input
    terminal.current.onData((data) => {
      websocket.current?.send(data);
    });

    return () => {
      websocket.current?.close();
      terminal.current?.dispose();
    };
  }, [config, onStatusChange]);

  return (
    <div
      ref={terminalRef}
      style={{
        width: '100%',
        height: '100%',
        backgroundColor: '#1e1e1e'
      }}
    />
  );
};
```

#### AWS Service Components

**3.2 Embedded AWS Service**
```tsx
// cmd/cws-gui/frontend/src/components/EmbeddedAWSService.tsx
import React, { useState, useRef } from 'react';
import { Container, Header, Button, Spinner } from '@cloudscape-design/components';

interface EmbeddedAWSServiceProps {
  config: ConnectionConfig;
  onStatusChange: (status: 'connecting' | 'connected' | 'disconnected') => void;
}

export const EmbeddedAWSService: React.FC<EmbeddedAWSServiceProps> = ({
  config,
  onStatusChange
}) => {
  const [loading, setLoading] = useState(true);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  const handleLoad = () => {
    setLoading(false);
    onStatusChange('connected');
  };

  const handleError = () => {
    setLoading(false);
    onStatusChange('disconnected');
  };

  return (
    <Container
      header={
        <Header
          variant="h2"
          actions={
            <Button
              variant="normal"
              iconName="refresh"
              onClick={() => {
                if (iframeRef.current) {
                  iframeRef.current.src = iframeRef.current.src;
                  setLoading(true);
                  onStatusChange('connecting');
                }
              }}
            >
              Reload
            </Button>
          }
        >
          {config.awsService?.toUpperCase()} ({config.region})
        </Header>
      }
    >
      {loading && (
        <div style={{ textAlign: 'center', padding: '2rem' }}>
          <Spinner size="large" />
          <p>Loading {config.awsService}...</p>
        </div>
      )}
      <iframe
        ref={iframeRef}
        src={config.proxyUrl}
        style={{
          width: '100%',
          height: '600px',
          border: 'none',
          display: loading ? 'none' : 'block'
        }}
        onLoad={handleLoad}
        onError={handleError}
        title={`${config.awsService} - ${config.region}`}
      />
    </Container>
  );
};
```

**3.3 Connection Tab Renderer**
```tsx
const renderConnectionContent = (tab: ConnectionTab) => {
  switch (tab.config.type) {
    case 'ssh':
      return (
        <EmbeddedTerminal
          config={tab.config}
          onStatusChange={(status) => updateTabStatus(tab.id, status)}
        />
      );
    case 'aws-service':
      return (
        <EmbeddedAWSService
          config={tab.config}
          onStatusChange={(status) => updateTabStatus(tab.id, status)}
        />
      );
    case 'web':
      return (
        <EmbeddedWebView
          config={tab.config}
          onStatusChange={(status) => updateTabStatus(tab.id, status)}
        />
      );
    default:
      return <div>Unsupported connection type</div>;
  }
};
```

### Phase 4: Enhanced UI Integration (Week 6-7)

**4.1 Connection View with Cloudscape Tabs**
```tsx
const renderConnectionsView = () => (
  <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
    <Container
      header={
        <Header
          variant="h1"
          counter={`(${state.connectionTabs.length} active)`}
          actions={
            <SpaceBetween direction="horizontal" size="xs">
              <Button variant="normal" onClick={() => setShowServiceLauncher(true)}>
                Add AWS Service
              </Button>
              <Button variant="primary" onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}>
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
          <Tabs
            tabs={state.connectionTabs.map(tab => ({
              id: tab.id,
              label: (
                <SpaceBetween direction="horizontal" size="xs">
                  <StatusIndicator type={
                    tab.status === 'connected' ? 'success' :
                    tab.status === 'connecting' ? 'in-progress' :
                    'error'
                  }>
                    {tab.title}
                  </StatusIndicator>
                  {tab.closeable && (
                    <Button
                      variant="inline-icon"
                      iconName="close"
                      onClick={(e) => {
                        e.stopPropagation();
                        closeConnectionTab(tab.id);
                      }}
                    />
                  )}
                </SpaceBetween>
              ),
              content: renderConnectionContent(tab)
            }))}
            activeTabId={state.activeConnectionTab}
            onChange={({ detail }) => setState(prev => ({
              ...prev,
              activeConnectionTab: detail.activeTabId
            }))}
          />
        ) : (
          <div style={{ textAlign: 'center', padding: '4rem' }}>
            <SpaceBetween direction="vertical" size="l">
              <Header variant="h2">No active connections</Header>
              <p>Connect to an instance or launch an AWS service to get started</p>
              <SpaceBetween direction="horizontal" size="s">
                <Button
                  variant="primary"
                  onClick={() => setState(prev => ({ ...prev, activeView: 'instances' }))}
                >
                  Connect to Instance
                </Button>
                <Button variant="normal" onClick={() => setShowServiceLauncher(true)}>
                  Launch AWS Service
                </Button>
              </SpaceBetween>
            </SpaceBetween>
          </div>
        )}
      </SpaceBetween>
    </Container>
  </div>
);
```

## 🔧 Technical Implementation Details

### Authentication & Security
- **AWS Federation**: Use STS AssumeRole for service access
- **Token Management**: Automatic refresh and renewal
- **CORS Handling**: Proxy-based approach for iframe embedding
- **Security Headers**: Proper CSP and frame-ancestors configuration

### Performance Considerations
- **Connection Pooling**: Reuse connections where possible
- **Memory Management**: Proper cleanup of WebSocket and iframe resources
- **Tab Limits**: Configurable maximum concurrent connections
- **Resource Monitoring**: Track memory and CPU usage per connection

### Error Handling
- **Connection Failures**: Graceful degradation and retry logic
- **Service Unavailability**: Clear error messages and alternative suggestions
- **Network Issues**: Offline detection and reconnection handling

## 📊 Success Metrics

### Technical Metrics
- **Connection Time**: < 3 seconds for SSH, < 5 seconds for AWS services
- **Session Stability**: > 99% uptime for established connections
- **Resource Usage**: < 200MB RAM per active connection tab
- **Tab Performance**: Support for 10+ simultaneous connections

### User Experience Metrics
- **Zero External Launches**: All connections embedded in GUI
- **Seamless Service Integration**: Single-click AWS service access
- **Connection Recovery**: Automatic reconnection after network issues

## 📝 Next Steps

1. **Document this plan** ✅
2. **Implement Phase 1** - Wails API integration and backend proxy infrastructure
3. **Build basic tab management** - Phase 2 implementation
4. **Create SSH terminal component** - Most critical embedded component
5. **Add AWS service integration** - Starting with Amazon Braket and SageMaker
6. **Iterate and enhance** - Based on user feedback and usage patterns

This plan transforms CloudWorkstation into a comprehensive research platform that seamlessly integrates traditional compute with AWS's powerful research services, all within a professional, tabbed interface.