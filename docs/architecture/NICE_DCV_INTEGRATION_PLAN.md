# CloudWorkstation NICE DCV Web SDK Integration Plan

## Version: v0.4.3+ Enhancement  
**Feature**: Embedded Remote Desktop via NICE DCV Web SDK  
**Target**: Seamless in-GUI remote access to CloudWorkstation instances

---

## 🎯 **Integration Objectives**

### **Primary Goals**
- **Seamless Access**: Users can connect to instances without leaving the CloudWorkstation GUI
- **Professional Experience**: High-quality remote desktop with minimal latency
- **Progressive Disclosure**: Simple connection for basic users, advanced controls for power users
- **Multi-Instance Support**: Manage multiple remote sessions simultaneously

### **Technical Benefits**
- **NICE DCV Advantages**: AWS-optimized protocol for technical computing workloads
- **Web SDK Integration**: Native browser embedding without plugins or additional software
- **GPU Acceleration**: Optimized for machine learning and visualization workloads
- **Adaptive Streaming**: Automatic quality adjustment based on network conditions

---

## 🏗️ **Architecture Overview**

### **NICE DCV Web SDK Integration**

```
┌─────────────────────────────────────────────────────────────────┐
│                    CloudWorkstation GUI                         │
│                     (Wails 3.x WebView)                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  ┌─────────────────┐    ┌──────────────────────────────────────┐ │
│  │  Template &     │    │        NICE DCV Web SDK              │ │
│  │  Instance       │    │      (Embedded Display)             │ │
│  │  Management     │    │                                      │ │
│  │                 │    │  ┌─────────────────────────────────┐ │ │
│  │  • Templates    │    │  │     Remote Instance Desktop    │ │ │
│  │  • Launch       │    │  │                                 │ │ │
│  │  • Status       │    │  │  Ubuntu/Rocky Linux Desktop    │ │ │
│  │  • Settings     │    │  │  • Jupyter Notebook            │ │ │
│  │                 │    │  │  • RStudio                      │ │ │
│  │                 │    │  │  • Code Editor                  │ │ │
│  │                 │    │  │  • Terminal                     │ │ │
│  │                 │    │  └─────────────────────────────────┘ │ │
│  └─────────────────┘    └──────────────────────────────────────┘ │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│                    Connection Management                        │
│  • Auto-discovery of DCV sessions                              │
│  • Authentication & security                                   │
│  • Quality/bandwidth adaptation                                │
│  • Multi-session management                                    │
└─────────────────────────────────────────────────────────────────┘
```

### **Data Flow**

```
┌──────────────┐    ┌─────────────────┐    ┌──────────────────┐
│ CloudWork    │    │  CloudWork      │    │  EC2 Instance    │
│ GUI          │    │  Daemon         │    │  (DCV Server)    │
│              │    │                 │    │                  │
│ 1. Connect   │───▶│ 2. Get DCV      │───▶│ 3. DCV Session   │
│    Request   │    │    Session URL  │    │    Available     │
│              │    │                 │    │                  │
│ 6. Display   │◀───│ 5. Return       │◀───│ 4. Session       │
│    Remote    │    │    Session      │    │    Details       │
│    Desktop   │    │    Details      │    │                  │
└──────────────┘    └─────────────────┘    └──────────────────┘
           │                                         ▲
           │                                         │
           └─────────────────────────────────────────┘
                    7. Direct DCV Web SDK Connection
```

---

## 🎨 **User Experience Design**

### **Progressive Disclosure Integration**

#### **Level 1: Simple Connection (Default)**
```html
<!-- Connect button on instance card -->
<div class="instance-card">
  <div class="instance-header">
    <div class="instance-name">ml-research-workstation</div>
    <div class="instance-status running">running</div>
  </div>
  
  <div class="instance-actions">
    <!-- Simple one-click connection -->
    <button class="btn-primary connect-btn" onclick="connectToDesktop('ml-research-workstation')">
      <span class="btn-icon">🖥️</span>
      Open Desktop
    </button>
    
    <!-- Advanced options (initially hidden) -->
    <button class="btn-secondary" onclick="showAdvancedConnect()">⚙️ More Options</button>
  </div>
</div>
```

#### **Level 2: Embedded Display**
```html
<!-- DCV session embedded in GUI -->
<div class="dcv-session-container">
  <div class="dcv-session-header">
    <div class="session-info">
      <span class="instance-name">ml-research-workstation</span>
      <span class="session-quality">🟢 Excellent (1080p)</span>
      <span class="session-latency">⚡ 15ms</span>
    </div>
    
    <div class="session-controls">
      <button onclick="toggleFullscreen()">⛶ Fullscreen</button>
      <button onclick="showKeyboardShortcuts()">⌨️ Shortcuts</button>
      <button onclick="adjustQuality()">🎛️ Quality</button>
      <button onclick="disconnectSession()" class="disconnect-btn">✕ Disconnect</button>
    </div>
  </div>
  
  <!-- NICE DCV Web SDK embed area -->
  <div id="dcv-display" class="dcv-display-area"></div>
  
  <div class="dcv-session-footer">
    <div class="bandwidth-usage">📊 2.1 MB/s</div>
    <div class="session-duration">⏱️ Connected: 1h 23m</div>
  </div>
</div>
```

### **Window Management Modes**

#### **Mode 1: Tabbed Interface**
```
┌─────────────────────────────────────────────────────────────┐
│ CloudWorkstation                                  🌙 ⚙️ ✕   │
├─────────────────────────────────────────────────────────────┤
│ 📋 Templates │ 💻 Instances │ 🖥️ ml-research │ 🖥️ data-viz  │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│           Remote Desktop Session (ml-research)             │
│  ┌───────────────────────────────────────────────────────┐  │
│  │  Ubuntu Desktop - Jupyter Lab - RStudio - Terminal   │  │
│  │                                                       │  │
│  │  [Remote instance desktop content displayed here]    │  │
│  │                                                       │  │
│  └───────────────────────────────────────────────────────┘  │
│                                                             │
├─────────────────────────────────────────────────────────────┤
│ 🟢 Connected • 1080p • 15ms • 2.1 MB/s • 1h 23m            │
└─────────────────────────────────────────────────────────────┘
```

#### **Mode 2: Split View**
```
┌─────────────────────────────────────────────────────────────┐
│ CloudWorkstation                                  🌙 ⚙️ ✕   │
├─────────────────────────────────────────────────────────────┤
│ Management │                Remote Desktop                  │
├─────────────────────────────────────────────────────────────┤
│              │                                              │
│ 📋 Templates │  ┌─────────────────────────────────────────┐ │
│              │  │    ml-research-workstation              │ │
│ 💻 Instances │  │                                         │ │
│ • ml-research│  │  [Remote desktop content]               │ │
│ • data-viz   │  │                                         │ │
│              │  │                                         │ │
│ 🔧 Settings  │  └─────────────────────────────────────────┘ │
│              │                                              │
│ 📊 Costs     │  🟢 1080p • 15ms • Connected               │
│              │                                              │
├─────────────────────────────────────────────────────────────┤
│ Status: 3 instances • 2 connected sessions • $2.45/hour    │
└─────────────────────────────────────────────────────────────┘
```

#### **Mode 3: Fullscreen**
```
┌─────────────────────────────────────────────────────────────┐
│  ┌─ ml-research-workstation ──────────────────────── ⛶ ✕ ┐  │
│                                                             │
│              Full Remote Desktop Experience                 │
│                                                             │
│     [Entire screen showing remote instance desktop]        │
│                                                             │
│                                                             │
│  └─ Hover for controls ─ 🟢 Connected ─ 15ms ─ 1080p ─────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## 🔧 **Technical Implementation**

### **NICE DCV Web SDK Integration**

#### **JavaScript Implementation**
```javascript
// DCV Web SDK initialization
import { DcvClient } from '@nice-dcv/web-sdk'

class CloudWorkstationDCVManager {
  constructor() {
    this.activeSessions = new Map()
    this.dcvClients = new Map()
  }
  
  async connectToInstance(instanceName) {
    try {
      // 1. Get DCV session details from daemon
      const sessionInfo = await this.getDCVSessionInfo(instanceName)
      
      // 2. Create DCV client
      const dcvClient = new DcvClient({
        sessionId: sessionInfo.sessionId,
        authToken: sessionInfo.authToken,
        serverUrl: sessionInfo.serverUrl,
        
        // Quality and performance settings
        quality: 'auto',
        resizeMode: 'stretch', 
        enableAudio: true,
        enableClipboard: true,
        
        // Event handlers
        onConnect: () => this.onDCVConnect(instanceName),
        onDisconnect: () => this.onDCVDisconnect(instanceName),
        onError: (error) => this.onDCVError(instanceName, error),
        onQualityChange: (quality) => this.onQualityChange(instanceName, quality)
      })
      
      // 3. Connect to DCV session
      await dcvClient.connect(document.getElementById('dcv-display'))
      
      // 4. Store session for management
      this.dcvClients.set(instanceName, dcvClient)
      this.activeSessions.set(instanceName, {
        instanceName,
        connected: true,
        quality: 'auto',
        startTime: Date.now()
      })
      
      // 5. Update UI
      this.showDCVSession(instanceName)
      
    } catch (error) {
      console.error(`Failed to connect to ${instanceName}:`, error)
      this.showConnectionError(instanceName, error)
    }
  }
  
  async getDCVSessionInfo(instanceName) {
    // Call CloudWorkstation daemon to get DCV session details
    const response = await fetch(`http://localhost:8947/api/v1/instances/${instanceName}/dcv`)
    
    if (!response.ok) {
      throw new Error(`Failed to get DCV session info: ${response.statusText}`)
    }
    
    return await response.json()
  }
  
  disconnect(instanceName) {
    const dcvClient = this.dcvClients.get(instanceName)
    if (dcvClient) {
      dcvClient.disconnect()
      this.dcvClients.delete(instanceName)
      this.activeSessions.delete(instanceName)
    }
  }
  
  // Event handlers
  onDCVConnect(instanceName) {
    console.log(`DCV session connected: ${instanceName}`)
    this.updateSessionStatus(instanceName, 'connected')
  }
  
  onDCVDisconnect(instanceName) {
    console.log(`DCV session disconnected: ${instanceName}`)
    this.updateSessionStatus(instanceName, 'disconnected')
    this.showConnectionLost(instanceName)
  }
  
  onDCVError(instanceName, error) {
    console.error(`DCV session error for ${instanceName}:`, error)
    this.showConnectionError(instanceName, error)
  }
  
  // Quality and performance management
  adjustQuality(instanceName, quality) {
    const dcvClient = this.dcvClients.get(instanceName)
    if (dcvClient) {
      dcvClient.setQuality(quality)
    }
  }
  
  toggleFullscreen(instanceName) {
    const dcvContainer = document.getElementById('dcv-display')
    if (dcvContainer.requestFullscreen) {
      dcvContainer.requestFullscreen()
    }
  }
}

// Initialize DCV manager
const dcvManager = new CloudWorkstationDCVManager()
```

#### **Backend Daemon Integration**
```go
// pkg/daemon/dcv_handlers.go
type DCVSessionInfo struct {
    SessionID   string `json:"sessionId"`
    AuthToken   string `json:"authToken"`
    ServerURL   string `json:"serverUrl"`
    InstanceID  string `json:"instanceId"`
    Quality     string `json:"quality"`
    Resolution  string `json:"resolution"`
}

func (d *Daemon) handleDCVConnect(w http.ResponseWriter, r *http.Request) {
    instanceName := mux.Vars(r)["name"]
    
    // 1. Get instance details
    instance, err := d.awsManager.GetInstance(instanceName)
    if err != nil {
        http.Error(w, "Instance not found", http.StatusNotFound)
        return
    }
    
    // 2. Check if DCV is enabled on instance
    if !d.isDCVEnabled(instance) {
        http.Error(w, "DCV not available on instance", http.StatusBadRequest)
        return
    }
    
    // 3. Create or get existing DCV session
    sessionInfo, err := d.getDCVSessionInfo(instance)
    if err != nil {
        http.Error(w, "Failed to get DCV session", http.StatusInternalServerError)
        return
    }
    
    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(sessionInfo)
}

func (d *Daemon) getDCVSessionInfo(instance *EC2Instance) (*DCVSessionInfo, error) {
    // Connect to instance and get DCV session details
    // This would use AWS Systems Manager or direct connection to query DCV
    
    return &DCVSessionInfo{
        SessionID:  "desktop-session",
        AuthToken:  d.generateDCVAuthToken(instance.ID),
        ServerURL:  fmt.Sprintf("https://%s:8443", instance.PublicIP),
        InstanceID: instance.ID,
        Quality:    "auto",
        Resolution: "1920x1080",
    }, nil
}
```

### **Security Considerations**

#### **Authentication & Authorization**
```javascript
// Secure token management
const DCVSecurity = {
  // Generate time-limited auth tokens
  generateAuthToken(instanceId, userId) {
    return jwt.sign({
      instanceId,
      userId,
      exp: Math.floor(Date.now() / 1000) + (60 * 60), // 1 hour expiry
      aud: 'dcv-session'
    }, process.env.DCV_JWT_SECRET)
  },
  
  // Validate session access
  validateAccess(instanceId, userId) {
    // Check if user has access to this instance
    // Verify instance is owned by user or shared
    return this.checkInstancePermissions(instanceId, userId)
  },
  
  // Secure connection validation
  validateConnection(sessionToken) {
    try {
      const decoded = jwt.verify(sessionToken, process.env.DCV_JWT_SECRET)
      return decoded.instanceId && decoded.userId
    } catch (error) {
      return false
    }
  }
}
```

#### **Network Security**
- **HTTPS Only**: All DCV connections over encrypted HTTPS
- **Token-Based Auth**: Short-lived JWT tokens for session access
- **Instance Isolation**: Each user can only access their own instances
- **Security Groups**: AWS security groups restrict DCV access to authorized users

---

## 🎛️ **User Interface Implementation**

### **Connection Workflow Components**

#### **Instance Card with DCV Support**
```javascript
// Enhanced instance card with DCV connection
function createInstanceCard(instance) {
  const hasDCV = instance.services && instance.services.includes('dcv')
  
  return `
    <div class="instance-card" data-instance="${instance.name}">
      <div class="instance-header">
        <div class="instance-name">${instance.name}</div>
        <div class="instance-status ${instance.state}">${instance.state}</div>
        ${hasDCV ? '<div class="dcv-available">🖥️ DCV</div>' : ''}
      </div>
      
      <div class="instance-details">
        <p><strong>IP:</strong> ${instance.public_ip}</p>
        <p><strong>Cost:</strong> $${instance.hourly_rate}/hour</p>
        ${hasDCV ? '<p><strong>Desktop:</strong> Ready to connect</p>' : ''}
      </div>
      
      <div class="instance-actions">
        ${hasDCV ? 
          `<button class="btn-primary" onclick="connectToDesktop('${instance.name}')">
             <span class="btn-icon">🖥️</span> Open Desktop
           </button>` :
          `<button class="btn-secondary" onclick="enableDCV('${instance.name}')">
             <span class="btn-icon">⚙️</span> Enable Desktop
           </button>`
        }
        <button class="btn-secondary" onclick="showInstanceDetails('${instance.name}')">
          Details
        </button>
      </div>
    </div>
  `
}
```

#### **DCV Session Management Panel**
```html
<!-- Session management sidebar -->
<div class="dcv-sessions-panel">
  <div class="panel-header">
    <h3>Active Sessions</h3>
    <button onclick="disconnectAllSessions()" class="btn-text">Disconnect All</button>
  </div>
  
  <div class="session-list">
    <!-- Active session item -->
    <div class="session-item active" data-instance="ml-research">
      <div class="session-info">
        <div class="session-name">ml-research-workstation</div>
        <div class="session-status">🟢 Connected • 1080p</div>
        <div class="session-duration">⏱️ 1h 23m</div>
      </div>
      <div class="session-actions">
        <button onclick="focusSession('ml-research')" class="btn-icon">👁️</button>
        <button onclick="disconnectSession('ml-research')" class="btn-icon">✕</button>
      </div>
    </div>
    
    <!-- Available instance (not connected) -->
    <div class="session-item available" data-instance="data-viz">
      <div class="session-info">
        <div class="session-name">data-viz-workstation</div>
        <div class="session-status">⚪ Ready to connect</div>
      </div>
      <div class="session-actions">
        <button onclick="connectToDesktop('data-viz')" class="btn-primary-sm">Connect</button>
      </div>
    </div>
  </div>
</div>
```

### **Quality and Performance Controls**

#### **Adaptive Quality Settings**
```javascript
// Automatic quality adjustment based on network conditions
const QualityManager = {
  profiles: {
    'auto': { resolution: 'auto', quality: 'auto', frameRate: 'auto' },
    'high': { resolution: '1920x1080', quality: '90', frameRate: '30' },
    'medium': { resolution: '1280x720', quality: '75', frameRate: '24' },
    'low': { resolution: '1024x768', quality: '60', frameRate: '15' },
    'minimal': { resolution: '800x600', quality: '40', frameRate: '10' }
  },
  
  // Monitor connection quality and adjust automatically
  monitorConnection(instanceName) {
    setInterval(() => {
      const session = dcvManager.activeSessions.get(instanceName)
      if (session) {
        const stats = session.getConnectionStats()
        this.adjustQualityBasedOnStats(instanceName, stats)
      }
    }, 5000) // Check every 5 seconds
  },
  
  adjustQualityBasedOnStats(instanceName, stats) {
    if (stats.latency > 200) {
      // High latency - reduce quality
      dcvManager.adjustQuality(instanceName, 'low')
    } else if (stats.packetLoss > 0.02) {
      // Packet loss - reduce frame rate
      dcvManager.adjustQuality(instanceName, 'medium')
    } else if (stats.bandwidth < 1000000) {
      // Low bandwidth - reduce resolution
      dcvManager.adjustQuality(instanceName, 'minimal')
    }
  }
}
```

---

## 🚀 **Implementation Roadmap**

### **Phase 1: Core DCV Integration (Week 1-2)**
1. **NICE DCV Web SDK Setup**
   - Install and configure NICE DCV Web SDK in frontend
   - Create basic connection manager
   - Implement simple connect/disconnect functionality

2. **Backend API Integration**
   - Add DCV session endpoints to daemon API
   - Implement DCV session discovery and management
   - Add security token generation

3. **Basic UI Components**
   - Add "Open Desktop" buttons to instance cards
   - Create simple DCV display container
   - Implement basic connection status indicators

### **Phase 2: Enhanced UX (Week 3)**
1. **Progressive Disclosure Integration**
   - Integrate DCV sessions with existing GUI navigation
   - Add tabbed interface for multiple sessions
   - Implement window management modes

2. **Quality Management**
   - Add quality adjustment controls
   - Implement automatic quality adaptation
   - Create bandwidth monitoring

3. **Session Management**
   - Multi-session support
   - Session persistence and recovery
   - Connection status monitoring

### **Phase 3: Advanced Features (Week 4)**
1. **Fullscreen and Window Management**
   - Fullscreen mode implementation
   - Split-view layout options
   - Picture-in-picture support

2. **Performance Optimization**
   - Connection pooling
   - Efficient resource usage
   - Memory leak prevention

3. **Security Hardening**
   - Token refresh mechanisms
   - Session timeout management
   - Access control validation

### **Phase 4: Polish and Testing (Week 5)**
1. **Visual Testing**
   - DCV session rendering across all themes
   - Responsive behavior testing
   - Cross-browser compatibility

2. **Performance Testing**
   - Connection latency optimization
   - Bandwidth usage optimization
   - Multi-session performance

3. **User Testing**
   - Researcher workflow validation
   - Accessibility compliance
   - Documentation and help system

---

## 📊 **Success Metrics**

### **Technical Metrics**
- **Connection Success Rate**: >95% successful connections
- **Session Latency**: <50ms average latency
- **Quality Adaptation**: Automatic adjustment within 5 seconds of network changes
- **Memory Usage**: <100MB additional memory per active session

### **User Experience Metrics**
- **Time to Connect**: <10 seconds from click to desktop
- **Session Stability**: >99% uptime for established sessions
- **User Satisfaction**: Positive feedback on desktop experience
- **Feature Adoption**: >60% of users using embedded desktop vs. SSH

---

## 🔒 **Security and Compliance**

### **Data Protection**
- **Encrypted Transport**: All DCV traffic over TLS 1.3
- **Token Security**: Short-lived JWT tokens with regular rotation
- **Session Isolation**: Each user's sessions are completely isolated
- **Audit Logging**: All connection attempts and session activities logged

### **Access Control**
- **User Authentication**: Integration with existing AWS profile system
- **Instance Permissions**: Users can only access their own instances
- **Network Isolation**: Security groups restrict access to authorized users
- **Session Timeouts**: Automatic disconnection after inactivity

---

**Total Implementation Time**: ~5 weeks for complete NICE DCV Web SDK integration with all advanced features.

This integration will transform CloudWorkstation from a simple instance launcher into a complete **integrated research computing platform** where users can launch, manage, and directly access their remote research environments without ever leaving the application.