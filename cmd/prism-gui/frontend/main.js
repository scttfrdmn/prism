// CloudWorkstation GUI - Progressive Disclosure Interface
// Minimal, professional UI for academic researchers

let selectedTemplate = null;
let templates = [];
let instances = [];
let currentTheme = 'core';

// Settings Management
let settings = {
    general: {
        autostartGUI: false,
        minimizeStartup: false,
        autoRefresh: true,
        defaultInstanceSize: 'M'
    },
    aws: {
        profile: 'default',
        region: 'us-west-2',
        costWarnings: true,
        dailyCostLimit: 50
    },
    daemon: {
        url: 'http://localhost:8947',
        timeout: 10,
        autoStart: true
    },
    appearance: {
        theme: 'core',
        animations: true,
        compactMode: false
    },
    advanced: {
        debugMode: false,
        logLevel: 'info',
        usageAnalytics: false
    }
};

let settingsChanged = false;

// Connection Management - DCV and SSH
let connectionManager = null;
let activeDCVSessions = new Map();
let activeSSHSessions = new Map();
let currentSession = null;
let currentSessionType = null; // 'dcv' or 'ssh'
let sessionTimers = new Map();

// Connection Detection
let instanceConnectionTypes = new Map(); // instanceName -> connection type
let connectionTypeCache = new Map(); // Cache for connection type detection

// Initialize application
document.addEventListener('DOMContentLoaded', async () => {
    console.log('CloudWorkstation GUI starting...');
    
    // Initialize UI state
    await initializeApp();
    
    // Initialize connection manager
    connectionManager = new CloudWorkstationConnectionManager();
    
    // Start periodic updates
    setInterval(updateInstances, 30000); // Update every 30 seconds
    setInterval(updateClock, 1000); // Update clock every second
    setInterval(updateConnectionDurations, 1000); // Update connection durations
    
    console.log('CloudWorkstation GUI ready');
});

// Initialize application state
async function initializeApp() {
    try {
        // Load templates
        await loadTemplates();
        
        // Load instances
        await loadInstances();
        
        // Update connection status
        updateConnectionStatus(true);
        
        // Apply saved theme
        const savedTheme = localStorage.getItem('cws-theme') || 'core';
        applyTheme(savedTheme);
        
    } catch (error) {
        console.error('Failed to initialize app:', error);
        updateConnectionStatus(false);
    }
}

// Load and display templates
async function loadTemplates() {
    try {
        templates = await window.wails.CloudWorkstationService.GetTemplates();
        renderTemplates();
    } catch (error) {
        console.error('Failed to load templates:', error);
        renderTemplateError();
    }
}

// Render templates with tile system and complexity indicators
function renderTemplates() {
    const grid = document.getElementById('template-grid');
    
    if (templates.length === 0) {
        grid.innerHTML = `
            <div class="template-tile">
                <div class="text-center">
                    <p>No templates available</p>
                    <small>Please ensure the daemon is running</small>
                </div>
            </div>
        `;
        return;
    }
    
    // Apply current filters and sorting
    const filteredTemplates = applyTemplateFilters(templates);
    
    let html = '';
    filteredTemplates.forEach(template => {
        html += createTemplateTile(template);
    });
    
    grid.innerHTML = html;
    
    // Initialize filter event listeners if not already done
    initializeFilterEventListeners();
}

// Create a template tile HTML structure
function createTemplateTile(template) {
    const complexity = template.Complexity || 'simple';
    const category = template.Category || 'General';
    const domain = template.Domain || 'base';
    const icon = template.Icon || '🖥️';
    const popular = template.Popular || false;
    const estimatedTime = template.EstimatedLaunchTime || 3;
    const estimatedCost = template.EstimatedCostPerHour?.[getArchitecture()] || 0.10;
    
    return `
        <div class="template-tile" data-complexity="${complexity}" data-category="${domain}" onclick="selectTemplate('${template.Name}')">
            <!-- Complexity Badge -->
            <div class="complexity-badge ${complexity}">
                <span class="complexity-icon">${getComplexityIcon(complexity)}</span>
                <span class="complexity-label">${getComplexityBadge(complexity)}</span>
            </div>
            
            <!-- Popular Badge -->
            ${popular ? '<div class="popular-badge">⭐ Popular</div>' : ''}
            
            <!-- Main Content Area -->
            <div class="tile-header">
                <div class="category-icon">${icon}</div>
                <div class="tile-title">${template.Name}</div>
                <div class="category-label">${category}</div>
            </div>
            
            <div class="tile-description">
                ${template.Description || 'Professional research environment ready to launch.'}
            </div>
            
            <!-- Features List -->
            <div class="tile-features">
                ${getTemplateFeatures(template).map(feature => 
                    `<span class="feature-tag">${feature}</span>`
                ).join('')}
            </div>
            
            <!-- Footer with Metadata -->
            <div class="tile-footer">
                <div class="launch-time">⚡ ~${estimatedTime} min launch</div>
                <div class="cost-estimate">💰 $${estimatedCost.toFixed(4)}/hour</div>
            </div>
            
            <!-- Selection State Overlay -->
            <div class="tile-selection-overlay">
                <div class="selection-checkmark">✓</div>
            </div>
        </div>
    `;
}

// Get complexity visual indicators
function getComplexityIcon(complexity) {
    switch (complexity) {
        case 'simple': return '🟢';
        case 'moderate': return '🟡';
        case 'advanced': return '🟠';
        case 'complex': return '🔴';
        default: return '🟢';
    }
}

function getComplexityBadge(complexity) {
    switch (complexity) {
        case 'simple': return 'Ready to Use';
        case 'moderate': return 'Some Options';
        case 'advanced': return 'Many Options';
        case 'complex': return 'Full Control';
        default: return 'Ready to Use';
    }
}

// Get template features based on packages and services
function getTemplateFeatures(template) {
    const features = [];
    
    // Add common features based on template characteristics
    if (template.Name.toLowerCase().includes('jupyter')) {
        features.push('Jupyter');
    }
    if (template.Name.toLowerCase().includes('gpu') || template.Name.toLowerCase().includes('cuda')) {
        features.push('GPU Ready');
    }
    if (template.ValidationStatus === 'validated') {
        features.push('Pre-tested');
    }
    if (template.Popular) {
        features.push('Popular Choice');
    }
    
    // Add features based on domain
    const domain = template.Domain || 'base';
    switch (domain) {
        case 'ml':
            features.push('ML Ready');
            break;
        case 'datascience':
            features.push('R/Python');
            break;
        case 'bio':
            features.push('Bioinformatics');
            break;
        case 'web':
            features.push('Web Dev');
            break;
    }
    
    return features.slice(0, 3); // Limit to 3 features for clean display
}

// Get current system architecture (simplified)
function getArchitecture() {
    // This would typically come from the system or be configurable
    return 'x86_64'; // Default to x86_64
}

// Apply template filters based on current filter state
function applyTemplateFilters(templates) {
    let filtered = [...templates];
    
    // Get current filter state
    const complexityFilter = getActiveFilter('complexity');
    const categoryFilter = getActiveFilter('category');
    const sortOrder = document.getElementById('sort-select')?.value || 'popularity';
    
    // Apply complexity filter
    if (complexityFilter !== 'all') {
        filtered = filtered.filter(template => 
            (template.Complexity || 'simple') === complexityFilter
        );
    }
    
    // Apply category filter
    if (categoryFilter !== 'all') {
        filtered = filtered.filter(template => 
            (template.Domain || 'base') === categoryFilter
        );
    }
    
    // Apply sorting
    filtered = sortTemplates(filtered, sortOrder);
    
    return filtered;
}

// Get active filter value
function getActiveFilter(filterType) {
    const activeBtn = document.querySelector(`[data-${filterType}].filter-btn.active`);
    return activeBtn ? activeBtn.dataset[filterType] : 'all';
}

// Sort templates based on specified criteria
function sortTemplates(templates, sortOrder) {
    switch (sortOrder) {
        case 'complexity':
            return templates.sort((a, b) => {
                const complexityOrder = { simple: 1, moderate: 2, advanced: 3, complex: 4 };
                return (complexityOrder[a.Complexity] || 1) - (complexityOrder[b.Complexity] || 1);
            });
        case 'category':
            return templates.sort((a, b) => (a.Category || '').localeCompare(b.Category || ''));
        case 'cost':
            return templates.sort((a, b) => {
                const costA = a.EstimatedCostPerHour?.[getArchitecture()] || 0;
                const costB = b.EstimatedCostPerHour?.[getArchitecture()] || 0;
                return costA - costB;
            });
        case 'launch-time':
            return templates.sort((a, b) => 
                (a.EstimatedLaunchTime || 3) - (b.EstimatedLaunchTime || 3)
            );
        case 'popularity':
        default:
            return templates.sort((a, b) => {
                // Sort by: Popular first, then by validation status, then by name
                if (a.Popular && !b.Popular) return -1;
                if (!a.Popular && b.Popular) return 1;
                if (a.ValidationStatus === 'validated' && b.ValidationStatus !== 'validated') return -1;
                if (a.ValidationStatus !== 'validated' && b.ValidationStatus === 'validated') return 1;
                return a.Name.localeCompare(b.Name);
            });
    }
}

// Initialize filter event listeners
let filtersInitialized = false;
function initializeFilterEventListeners() {
    if (filtersInitialized) return;
    filtersInitialized = true;
    
    // Complexity filter buttons
    document.querySelectorAll('[data-complexity]').forEach(btn => {
        btn.addEventListener('click', function() {
            // Remove active class from siblings
            this.parentElement.querySelectorAll('.filter-btn').forEach(sibling => 
                sibling.classList.remove('active')
            );
            // Add active class to clicked button
            this.classList.add('active');
            // Re-render templates
            renderTemplates();
        });
    });
    
    // Category filter buttons
    document.querySelectorAll('[data-category]').forEach(btn => {
        btn.addEventListener('click', function() {
            // Remove active class from siblings
            this.parentElement.querySelectorAll('.filter-btn').forEach(sibling => 
                sibling.classList.remove('active')
            );
            // Add active class to clicked button
            this.classList.add('active');
            // Re-render templates
            renderTemplates();
        });
    });
    
    // Sort select
    document.getElementById('sort-select')?.addEventListener('change', function() {
        renderTemplates();
    });
}

// Handle template selection (Progressive Disclosure - Step 1)
function selectTemplate(templateName) {
    selectedTemplate = templates.find(t => t.Name === templateName);
    
    // Update visual selection state for tiles
    document.querySelectorAll('.template-tile').forEach(tile => {
        tile.classList.remove('selected');
    });
    
    // Find and select the clicked tile
    const clickedTile = event?.target?.closest('.template-tile');
    if (clickedTile) {
        clickedTile.classList.add('selected');
    }
    
    // Show launch form (Progressive Disclosure - Step 2)
    showLaunchForm();
}

// Show launch form with smart defaults
function showLaunchForm() {
    const form = document.getElementById('launch-form');
    const templateName = document.getElementById('selected-template-name');
    
    templateName.textContent = selectedTemplate.Name;
    form.classList.remove('hidden');
    form.classList.add('fade-in');
    
    // Auto-suggest instance name
    const nameInput = document.getElementById('instance-name');
    if (!nameInput.value) {
        nameInput.value = generateInstanceName(selectedTemplate.Name);
    }
    
    // Scroll to form
    form.scrollIntoView({ behavior: 'smooth', block: 'nearest' });
}

// Generate smart instance name
function generateInstanceName(templateName) {
    const prefix = templateName.toLowerCase()
        .replace(/[^a-z0-9]/g, '-')
        .replace(/-+/g, '-')
        .replace(/^-|-$/g, '')
        .substring(0, 15);
    
    const suffix = new Date().toISOString().slice(5, 10).replace('-', '');
    return `${prefix}-${suffix}`;
}

// Clear template selection
function clearSelection() {
    selectedTemplate = null;
    document.querySelectorAll('.template-tile').forEach(tile => {
        tile.classList.remove('selected');
    });
    document.getElementById('launch-form').classList.add('hidden');
}

// Launch instance
async function launchInstance() {
    const nameInput = document.getElementById('instance-name');
    const sizeSelect = document.getElementById('instance-size');
    const launchBtn = document.getElementById('launch-btn');
    
    const instanceName = nameInput.value.trim();
    if (!instanceName) {
        alert('Please enter an instance name');
        nameInput.focus();
        return;
    }
    
    // Validate instance name
    if (!/^[a-z0-9-]+$/.test(instanceName)) {
        alert('Instance name can only contain lowercase letters, numbers, and hyphens');
        nameInput.focus();
        return;
    }
    
    // Disable form during launch
    launchBtn.disabled = true;
    launchBtn.innerHTML = '<div class="loading-spinner"></div> Launching...';
    
    try {
        await window.wails.CloudWorkstationService.LaunchInstance({
            Template: selectedTemplate.Name,
            Name: instanceName,
            Size: sizeSelect.value
        });
        
        // Success feedback
        showSuccess(`Successfully launched ${instanceName}!`);
        
        // Reset form and switch to instances view
        clearSelection();
        nameInput.value = '';
        showSection('my-instances');
        
        // Refresh instances
        await loadInstances();
        
    } catch (error) {
        console.error('Launch failed:', error);
        showError(`Failed to launch instance: ${error.message}`);
    } finally {
        // Re-enable form
        launchBtn.disabled = false;
        launchBtn.innerHTML = '<span class="btn-icon">🚀</span> Launch Research Environment';
    }
}

// Load and display instances
async function loadInstances() {
    try {
        instances = await window.wails.CloudWorkstationService.GetInstances();
        renderInstances();
    } catch (error) {
        console.error('Failed to load instances:', error);
        renderInstanceError();
    }
}

// Render instances
function renderInstances() {
    const grid = document.getElementById('instances-grid');
    
    if (instances.length === 0) {
        grid.innerHTML = `
            <div class="instance-card">
                <div class="text-center">
                    <p>No instances running</p>
                    <small>Launch your first research environment in Quick Start</small>
                </div>
            </div>
        `;
        return;
    }
    
    let html = '';
    instances.forEach(instance => {
        html += `
            <div class="instance-card">
                <div class="instance-header">
                    <div class="instance-name">${instance.Name}</div>
                    <div class="instance-status ${instance.State}">${instance.State}</div>
                </div>
                <div class="instance-details">
                    ${instance.IP ? `<p><strong>IP:</strong> ${instance.IP}</p>` : ''}
                    ${instance.Cost ? `<p><strong>Cost:</strong> $${instance.Cost.toFixed(4)}/hour</p>` : ''}
                    ${instance.Region ? `<p><strong>Region:</strong> ${instance.Region}</p>` : ''}
                </div>
                <div class="instance-actions">
                    <button class="btn-secondary" onclick="connectToInstance('${instance.Name}')">
                        Connect
                    </button>
                    ${instance.State === 'running' ? 
                        `<button class="btn-secondary" onclick="stopInstance('${instance.Name}')">Stop</button>` :
                        `<button class="btn-secondary" onclick="startInstance('${instance.Name}')">Start</button>`
                    }
                </div>
            </div>
        `;
    });
    
    grid.innerHTML = html;
}

// Section navigation
function showSection(sectionId) {
    // Hide all sections
    document.querySelectorAll('.section').forEach(section => {
        section.classList.remove('active');
    });
    
    // Show target section
    const targetSection = document.getElementById(sectionId);
    if (targetSection) {
        targetSection.classList.add('active');
    }
    
    // Update navigation
    document.querySelectorAll('.nav-item').forEach(nav => {
        nav.classList.remove('active');
    });
    
    // Find and activate the corresponding nav item
    const targetNav = document.querySelector(`.nav-item[onclick*="'${sectionId}'"]`);
    if (targetNav) {
        targetNav.classList.add('active');
    }
}

// Theme management
function toggleTheme() {
    const newTheme = currentTheme === 'core' ? 'dark' : 'core';
    applyTheme(newTheme);
}

function applyTheme(themeName) {
    currentTheme = themeName;
    
    // Update theme link
    const themeLink = document.getElementById('theme-link');
    themeLink.href = `/themes/${themeName}.css`;
    
    // Update document attribute for theme-specific styling
    document.documentElement.setAttribute('data-theme', themeName);
    
    // Update theme icon
    const themeIcon = document.getElementById('theme-icon');
    themeIcon.textContent = themeName === 'dark' ? '☀️' : '🌙';
    
    // Save preference
    localStorage.setItem('cws-theme', themeName);
    
    // Update theme selector if visible
    const selector = document.getElementById('theme-selector');
    if (selector) {
        selector.value = themeName;
    }
}

// Settings management
// showSettings function implemented in settings section below

// hideSettings function implemented in settings section below

// Instance actions - connection handled by main connectToInstance function below

async function stopInstance(name) {
    if (!confirm(`Stop instance "${name}"? This will shut down the instance but preserve all data.`)) {
        return;
    }
    
    try {
        await window.wails.CloudWorkstationService.StopInstance(name);
        showSuccess(`Instance "${name}" is stopping`);
        await loadInstances(); // Refresh
    } catch (error) {
        console.error('Stop failed:', error);
        showError(`Failed to stop instance: ${error.message}`);
    }
}

// Utility functions
function updateConnectionStatus(connected) {
    const status = document.getElementById('connection-status');
    const dot = status.querySelector('.status-dot');
    
    if (connected) {
        dot.classList.remove('connecting', 'disconnected');
        dot.classList.add('connected');
        status.innerHTML = '<span class="status-dot connected"></span> Connected to daemon';
    } else {
        dot.classList.remove('connecting', 'connected');
        dot.classList.add('disconnected');
        status.innerHTML = '<span class="status-dot disconnected"></span> Daemon unavailable';
    }
}

function updateClock() {
    const timeElement = document.getElementById('current-time');
    if (timeElement) {
        timeElement.textContent = new Date().toLocaleTimeString();
    }
}

function groupBy(array, key) {
    return array.reduce((groups, item) => {
        const group = item[key] || 'Other';
        groups[group] = groups[group] || [];
        groups[group].push(item);
        return groups;
    }, {});
}

function showSuccess(message) {
    // Simple success notification (can be enhanced later)
    alert(`✅ ${message}`);
}

function showError(message) {
    // Simple error notification (can be enhanced later)
    alert(`❌ ${message}`);
}

function showConnectionInfo(instanceName, connectionInfo) {
    let message = `Connection information for "${instanceName}":\n\n`;
    
    Object.entries(connectionInfo).forEach(([key, value]) => {
        message += `${key.toUpperCase()}: ${value}\n`;
    });
    
    alert(message);
}

function renderTemplateError() {
    const grid = document.getElementById('template-grid');
    grid.innerHTML = `
        <div class="template-card">
            <div class="text-center">
                <p>Failed to load templates</p>
                <small>Please check if the daemon is running</small>
                <br><br>
                <button class="btn-secondary" onclick="loadTemplates()">Retry</button>
            </div>
        </div>
    `;
}

function renderInstanceError() {
    const grid = document.getElementById('instances-grid');
    grid.innerHTML = `
        <div class="instance-card">
            <div class="text-center">
                <p>Failed to load instances</p>
                <small>Please check if the daemon is running</small>
                <br><br>
                <button class="btn-secondary" onclick="loadInstances()">Retry</button>
            </div>
        </div>
    `;
}

// ============================================================================
// NICE DCV Web SDK Integration
// ============================================================================

/**
 * CloudWorkstation NICE DCV Manager
 * Handles remote desktop connections with security and multi-session support
 */
class CloudWorkstationDCVManager {
    constructor() {
        this.activeSessions = new Map();
        this.dcvClients = new Map();
        this.currentViewMode = 'tabbed';
        this.sessionTimers = new Map();
        this.qualityManager = new DCVQualityManager();
        
        console.log('CloudWorkstation DCV Manager initialized');
    }

    /**
     * Connect to a CloudWorkstation instance via NICE DCV
     * @param {string} instanceName - Name of the instance to connect to
     * @returns {Promise<boolean>} - Success status
     */
    async connectToInstance(instanceName) {
        try {
            console.log(`Initiating DCV connection to instance: ${instanceName}`);
            
            // Update UI to show connecting state
            this.updateConnectionStatus(instanceName, 'connecting');
            
            // 1. Get DCV session details from CloudWorkstation daemon
            const sessionInfo = await this.getDCVSessionInfo(instanceName);
            
            // 2. Validate session info and security
            if (!this.validateSessionSecurity(sessionInfo)) {
                throw new Error('Session security validation failed');
            }
            
            // 3. Create DCV client with secure configuration
            const dcvClient = await this.createSecureDCVClient(sessionInfo);
            
            // 4. Establish connection to remote desktop
            await this.establishDCVConnection(dcvClient, instanceName);
            
            // 5. Setup session management and monitoring
            this.setupSessionManagement(instanceName, dcvClient, sessionInfo);
            
            // 6. Update GUI to show connected state
            this.showDCVSession(instanceName);
            
            console.log(`✅ Successfully connected to DCV session: ${instanceName}`);
            return true;
            
        } catch (error) {
            console.error(`❌ Failed to connect to ${instanceName}:`, error);
            this.handleConnectionError(instanceName, error);
            return false;
        }
    }

    /**
     * Get DCV session information from CloudWorkstation daemon
     * @param {string} instanceName - Instance to get session info for
     * @returns {Promise<Object>} - Session information with auth tokens
     */
    async getDCVSessionInfo(instanceName) {
        try {
            // Call CloudWorkstation daemon API for DCV session details
            const response = await fetch(`http://localhost:8947/api/v1/instances/${instanceName}/dcv`, {
                method: 'GET',
                headers: {
                    'Content-Type': 'application/json',
                    'X-CloudWorkstation-Client': 'GUI'
                }
            });
            
            if (!response.ok) {
                throw new Error(`HTTP ${response.status}: ${response.statusText}`);
            }
            
            const sessionInfo = await response.json();
            
            // Validate required session fields
            if (!sessionInfo.sessionId || !sessionInfo.authToken || !sessionInfo.serverUrl) {
                throw new Error('Incomplete session information received from daemon');
            }
            
            return sessionInfo;
            
        } catch (error) {
            console.error('Failed to get DCV session info:', error);
            throw new Error(`Unable to get remote desktop session: ${error.message}`);
        }
    }

    /**
     * Validate DCV session security
     * @param {Object} sessionInfo - Session information to validate
     * @returns {boolean} - Validation result
     */
    validateSessionSecurity(sessionInfo) {
        // Check required security fields
        if (!sessionInfo.authToken || !sessionInfo.sessionId || !sessionInfo.serverUrl) {
            console.warn('Missing required security fields in session info');
            return false;
        }
        
        // Validate token format (JWT structure check)
        if (!this.isValidJWT(sessionInfo.authToken)) {
            console.warn('Invalid authentication token format');
            return false;
        }
        
        // Validate server URL (HTTPS required for security)
        if (!sessionInfo.serverUrl.startsWith('https://')) {
            console.warn('Insecure server URL - HTTPS required for DCV connections');
            return false;
        }
        
        // Check token expiration if available
        if (sessionInfo.expiresAt) {
            const expiry = new Date(sessionInfo.expiresAt);
            if (expiry < new Date()) {
                console.warn('Authentication token has expired');
                return false;
            }
        }
        
        return true;
    }

    /**
     * Check if token is valid JWT format
     * @param {string} token - Token to validate
     * @returns {boolean} - Validation result
     */
    isValidJWT(token) {
        if (!token || typeof token !== 'string') return false;
        
        const parts = token.split('.');
        if (parts.length !== 3) return false;
        
        try {
            // Try to decode header and payload (signature validation is server-side)
            const header = JSON.parse(atob(parts[0]));
            const payload = JSON.parse(atob(parts[1]));
            
            return header.typ === 'JWT' && payload.exp && payload.aud;
        } catch (error) {
            return false;
        }
    }

    /**
     * Create secure DCV client instance
     * @param {Object} sessionInfo - Session configuration
     * @returns {Promise<Object>} - Configured DCV client
     */
    async createSecureDCVClient(sessionInfo) {
        // AWS NICE DCV Web SDK Integration
        // Production deployment requires: npm install nice-dcv-web-viewer
        // Then: import { DCVViewer } from 'nice-dcv-web-viewer';
        // const viewer = new DCVViewer(dcvViewerOptions);
        // viewer.connect(sessionInfo.serverUrl, sessionInfo.authToken);
        //
        // For development/demo, using compatible interface simulation

        const dcvClient = {
            // Mock DCV client configuration
            sessionId: sessionInfo.sessionId,
            authToken: sessionInfo.authToken,
            serverUrl: sessionInfo.serverUrl,
            
            // Security configuration
            security: {
                enforceHTTPS: true,
                validateCertificate: true,
                enableTLS13: true
            },
            
            // Quality and performance settings
            quality: sessionInfo.quality || 'auto',
            resizeMode: 'stretch',
            enableAudio: true,
            enableClipboard: true,
            enableFullscreen: true,
            
            // Mock methods for simulation
            connect: async (container) => {
                console.log('DCV Client: Connecting to remote desktop...');
                // Simulate connection delay
                await new Promise(resolve => setTimeout(resolve, 2000));
                console.log('DCV Client: Connected successfully');
                return true;
            },
            
            disconnect: () => {
                console.log('DCV Client: Disconnecting from remote desktop...');
                return true;
            },
            
            getConnectionStats: () => ({
                latency: Math.floor(Math.random() * 50) + 10, // 10-60ms
                bandwidth: Math.floor(Math.random() * 5000) + 1000, // 1-6 MB/s in KB
                quality: 'excellent',
                packetsLost: 0,
                framesPerSecond: 30
            }),
            
            setQuality: (quality) => {
                console.log(`DCV Client: Setting quality to ${quality}`);
            }
        };
        
        return dcvClient;
    }

    /**
     * Establish DCV connection with proper error handling
     * @param {Object} dcvClient - Configured DCV client
     * @param {string} instanceName - Instance name for connection
     */
    async establishDCVConnection(dcvClient, instanceName) {
        const displayContainer = document.getElementById('dcv-display');
        
        if (!displayContainer) {
            throw new Error('DCV display container not found in DOM');
        }
        
        // Clear any existing content
        displayContainer.innerHTML = '<div class="dcv-connecting">Connecting to remote desktop...</div>';
        
        // Attempt connection
        const connected = await dcvClient.connect(displayContainer);
        
        if (!connected) {
            throw new Error('Failed to establish DCV connection to remote instance');
        }
        
        // Store client reference
        this.dcvClients.set(instanceName, dcvClient);
        
        console.log(`DCV connection established for ${instanceName}`);
    }

    /**
     * Setup session management and monitoring
     * @param {string} instanceName - Instance name
     * @param {Object} dcvClient - DCV client instance
     * @param {Object} sessionInfo - Session configuration
     */
    setupSessionManagement(instanceName, dcvClient, sessionInfo) {
        // Create session object
        const session = {
            instanceName,
            sessionId: sessionInfo.sessionId,
            connected: true,
            connectTime: Date.now(),
            lastActivity: Date.now(),
            client: dcvClient,
            quality: sessionInfo.quality || 'auto',
            stats: {
                bytesTransferred: 0,
                packetsLost: 0,
                averageLatency: 0
            }
        };
        
        // Store active session
        this.activeSessions.set(instanceName, session);
        
        // Setup periodic monitoring
        this.startSessionMonitoring(instanceName);
        
        // Setup event listeners for the session
        this.setupSessionEventListeners(instanceName, dcvClient);
        
        console.log(`Session management setup complete for ${instanceName}`);
    }

    /**
     * Start monitoring session performance and connection health
     * @param {string} instanceName - Instance to monitor
     */
    startSessionMonitoring(instanceName) {
        if (this.sessionTimers.has(instanceName)) {
            clearInterval(this.sessionTimers.get(instanceName));
        }
        
        const timer = setInterval(() => {
            this.updateSessionMetrics(instanceName);
        }, 5000); // Update every 5 seconds
        
        this.sessionTimers.set(instanceName, timer);
        
        console.log(`Started monitoring for DCV session: ${instanceName}`);
    }

    /**
     * Update session metrics and UI
     * @param {string} instanceName - Instance to update metrics for
     */
    updateSessionMetrics(instanceName) {
        const session = this.activeSessions.get(instanceName);
        const client = this.dcvClients.get(instanceName);
        
        if (!session || !client) return;
        
        // Get current connection statistics
        const stats = client.getConnectionStats();
        
        // Update session stats
        session.stats.averageLatency = stats.latency;
        session.lastActivity = Date.now();
        
        // Update UI elements
        this.updateSessionUI(instanceName, stats);
        
        // Check connection health
        this.qualityManager.checkConnectionHealth(instanceName, stats);
    }

    /**
     * Update session UI with current statistics
     * @param {string} instanceName - Instance name
     * @param {Object} stats - Connection statistics
     */
    updateSessionUI(instanceName, stats) {
        // Update session quality indicator
        const qualityElement = document.getElementById('dcv-session-quality');
        if (qualityElement && currentDCVSession === instanceName) {
            const qualityIcon = this.getQualityIcon(stats.quality);
            const qualityText = this.getQualityText(stats.quality);
            qualityElement.innerHTML = `${qualityIcon} ${qualityText}`;
            qualityElement.className = `session-quality quality-${stats.quality}`;
        }
        
        // Update latency
        const latencyElement = document.getElementById('dcv-session-latency');
        if (latencyElement && currentDCVSession === instanceName) {
            latencyElement.textContent = `⚡ ${stats.latency}ms`;
        }
        
        // Update bandwidth
        const bandwidthElement = document.getElementById('dcv-bandwidth');
        if (bandwidthElement && currentDCVSession === instanceName) {
            const mbps = (stats.bandwidth / 1024).toFixed(1);
            bandwidthElement.textContent = `${mbps} MB/s`;
        }
        
        // Update connection duration
        const durationElement = document.getElementById('dcv-duration');
        if (durationElement && currentDCVSession === instanceName) {
            const session = this.activeSessions.get(instanceName);
            if (session) {
                const duration = this.formatDuration(Date.now() - session.connectTime);
                durationElement.textContent = duration;
            }
        }
        
        // Update connection info
        const connectionInfoElement = document.getElementById('dcv-connection-info');
        if (connectionInfoElement && currentDCVSession === instanceName) {
            connectionInfoElement.textContent = `Connected via NICE DCV`;
        }
    }

    /**
     * Setup event listeners for DCV session
     * @param {string} instanceName - Instance name
     * @param {Object} dcvClient - DCV client instance
     */
    setupSessionEventListeners(instanceName, dcvClient) {
        // DCV Web SDK event listener structure
        // Production code: dcvClient.on('connect', () => this.handleDCVConnect(instanceName));
        // Production code: dcvClient.on('disconnect', () => this.handleDCVDisconnect(instanceName));
        // Production code: dcvClient.on('error', (err) => this.handleDCVError(instanceName, err));

        console.log(`Setting up event listeners for DCV session: ${instanceName}`);

        // Simulate connection events for demo/development
        setTimeout(() => {
            this.handleDCVConnect(instanceName);
        }, 2000);
    }

    /**
     * Handle successful DCV connection
     * @param {string} instanceName - Connected instance name
     */
    handleDCVConnect(instanceName) {
        console.log(`✅ DCV session connected: ${instanceName}`);
        this.updateConnectionStatus(instanceName, 'connected');
        
        // Show success notification
        this.showNotification(`Connected to ${instanceName}`, 'success');
    }

    /**
     * Handle DCV disconnection
     * @param {string} instanceName - Disconnected instance name
     */
    handleDCVDisconnect(instanceName) {
        console.log(`🔌 DCV session disconnected: ${instanceName}`);
        this.updateConnectionStatus(instanceName, 'disconnected');
        this.cleanupSession(instanceName);
        
        // Show disconnection notification
        this.showNotification(`Disconnected from ${instanceName}`, 'info');
    }

    /**
     * Handle DCV connection errors
     * @param {string} instanceName - Instance with error
     * @param {Error} error - Error details
     */
    handleConnectionError(instanceName, error) {
        console.error(`❌ DCV connection error for ${instanceName}:`, error);
        this.updateConnectionStatus(instanceName, 'error');
        this.cleanupSession(instanceName);
        
        // Show error notification
        this.showNotification(`Connection failed: ${error.message}`, 'error');
    }

    /**
     * Update connection status in UI
     * @param {string} instanceName - Instance name
     * @param {string} status - Connection status
     */
    updateConnectionStatus(instanceName, status) {
        // Update session list
        this.renderDCVSessions();
        
        // Update header if this is the current session
        if (currentDCVSession === instanceName) {
            const nameElement = document.getElementById('dcv-instance-name');
            if (nameElement) {
                nameElement.textContent = instanceName;
            }
            
            const qualityElement = document.getElementById('dcv-session-quality');
            if (qualityElement) {
                const statusConfig = {
                    connecting: { icon: '🟡', text: 'Connecting...', class: 'connecting' },
                    connected: { icon: '🟢', text: 'Connected', class: 'connected' },
                    disconnected: { icon: '⚪', text: 'Disconnected', class: 'disconnected' },
                    error: { icon: '🔴', text: 'Connection Error', class: 'error' }
                };
                
                const config = statusConfig[status] || statusConfig.disconnected;
                qualityElement.innerHTML = `${config.icon} ${config.text}`;
                qualityElement.className = `session-quality ${config.class}`;
            }
        }
    }

    /**
     * Show DCV session in main display area
     * @param {string} instanceName - Instance to display
     */
    showDCVSession(instanceName) {
        // Set as current session
        currentDCVSession = instanceName;
        
        // Switch to remote desktop section
        showSection('remote-desktop');
        
        // Update display container
        const displayContainer = document.getElementById('dcv-display');
        if (displayContainer) {
            // Remove placeholder content
            const placeholder = displayContainer.querySelector('.dcv-placeholder');
            if (placeholder) {
                placeholder.style.display = 'none';
            }
            
            // Add connected session content
            displayContainer.innerHTML = `
                <div class="dcv-session-active">
                    <div class="session-content">
                        <h3>🖥️ ${instanceName}</h3>
                        <p>Remote desktop session is active</p>
                        <div class="session-placeholder">
                            <p>NICE DCV Web SDK would render the remote desktop here</p>
                            <p>This is a simulation of the embedded remote desktop display</p>
                        </div>
                    </div>
                </div>
            `;
        }
        
        // Render active sessions list
        this.renderDCVSessions();
        
        console.log(`Showing DCV session for ${instanceName}`);
    }

    /**
     * Render active DCV sessions list
     */
    renderDCVSessions() {
        const sessionList = document.getElementById('dcv-session-list');
        if (!sessionList) return;
        
        if (this.activeSessions.size === 0) {
            sessionList.innerHTML = `
                <div class="session-item placeholder">
                    <div class="session-info">
                        <div class="session-name">No active sessions</div>
                        <div class="session-status">Connect to an instance to start a remote desktop session</div>
                    </div>
                </div>
            `;
            return;
        }
        
        let html = '';
        this.activeSessions.forEach((session, instanceName) => {
            const isActive = currentDCVSession === instanceName;
            const duration = this.formatDuration(Date.now() - session.connectTime);
            
            html += `
                <div class="session-item ${isActive ? 'active' : ''}" onclick="switchDCVSession('${instanceName}')">
                    <div class="session-info">
                        <div class="session-name">${instanceName}</div>
                        <div class="session-status">
                            <span class="connection-dot ${session.connected ? 'connected' : 'disconnected'}"></span>
                            ${session.connected ? 'Connected' : 'Disconnected'}
                        </div>
                        <div class="session-duration">${duration}</div>
                    </div>
                    <div class="session-actions">
                        <button class="btn-icon" onclick="event.stopPropagation(); focusDCVSession('${instanceName}')" title="Focus">👁️</button>
                        <button class="btn-icon" onclick="event.stopPropagation(); disconnectDCVSession('${instanceName}')" title="Disconnect">✕</button>
                    </div>
                </div>
            `;
        });
        
        sessionList.innerHTML = html;
    }

    /**
     * Format duration in a human-readable format
     * @param {number} milliseconds - Duration in milliseconds
     * @returns {string} - Formatted duration
     */
    formatDuration(milliseconds) {
        const minutes = Math.floor(milliseconds / 60000);
        const hours = Math.floor(minutes / 60);
        
        if (hours > 0) {
            return `${hours}h ${minutes % 60}m`;
        }
        return `${minutes}m`;
    }

    /**
     * Get quality icon based on connection quality
     * @param {string} quality - Quality level
     * @returns {string} - Quality icon
     */
    getQualityIcon(quality) {
        const icons = {
            excellent: '🟢',
            good: '🟢',
            fair: '🟡',
            poor: '🔴'
        };
        return icons[quality] || '⚪';
    }

    /**
     * Get quality text based on connection quality
     * @param {string} quality - Quality level
     * @returns {string} - Quality text
     */
    getQualityText(quality) {
        const texts = {
            excellent: 'Excellent',
            good: 'Good',
            fair: 'Fair',
            poor: 'Poor'
        };
        return texts[quality] || 'Unknown';
    }

    /**
     * Show notification to user
     * @param {string} message - Notification message
     * @param {string} type - Notification type (success, error, info)
     */
    showNotification(message, type) {
        // Real toast notification implementation
        console.log(`${type.toUpperCase()}: ${message}`);

        const icon = {
            success: '✅',
            error: '❌',
            info: 'ℹ️',
            warning: '⚠️'
        }[type] || 'ℹ️';

        // Create toast container if it doesn't exist
        let toastContainer = document.getElementById('toast-container');
        if (!toastContainer) {
            toastContainer = document.createElement('div');
            toastContainer.id = 'toast-container';
            toastContainer.className = 'toast-container';
            document.body.appendChild(toastContainer);
        }

        // Create toast element
        const toast = document.createElement('div');
        toast.className = `toast toast-${type}`;
        toast.innerHTML = `
            <span class="toast-icon">${icon}</span>
            <span class="toast-message">${message}</span>
            <button class="toast-close" onclick="this.parentElement.remove()">×</button>
        `;

        // Add to container
        toastContainer.appendChild(toast);

        // Trigger animation
        setTimeout(() => toast.classList.add('toast-show'), 10);

        // Auto-dismiss after 5 seconds
        setTimeout(() => {
            toast.classList.remove('toast-show');
            setTimeout(() => toast.remove(), 300);
        }, 5000);
    }

    /**
     * Cleanup session resources
     * @param {string} instanceName - Instance to cleanup
     */
    cleanupSession(instanceName) {
        // Stop monitoring timer
        if (this.sessionTimers.has(instanceName)) {
            clearInterval(this.sessionTimers.get(instanceName));
            this.sessionTimers.delete(instanceName);
        }
        
        // Remove client and session
        this.dcvClients.delete(instanceName);
        this.activeSessions.delete(instanceName);
        
        // Clear current session if it was this one
        if (currentDCVSession === instanceName) {
            currentDCVSession = null;
            this.showDCVPlaceholder();
        }
        
        // Re-render sessions list
        this.renderDCVSessions();
        
        console.log(`Cleaned up session resources for ${instanceName}`);
    }

    /**
     * Show DCV placeholder content
     */
    showDCVPlaceholder() {
        const displayContainer = document.getElementById('dcv-display');
        if (displayContainer) {
            displayContainer.innerHTML = `
                <div class="dcv-placeholder">
                    <div class="placeholder-content">
                        <h3>🖥️ Remote Desktop</h3>
                        <p>Select an instance from "My Instances" and click "Connect" to start a remote desktop session.</p>
                        <div class="placeholder-features">
                            <div class="feature-item">
                                <span class="feature-icon">⚡</span>
                                <span>Low-latency streaming</span>
                            </div>
                            <div class="feature-item">
                                <span class="feature-icon">🎮</span>
                                <span>GPU acceleration support</span>
                            </div>
                            <div class="feature-item">
                                <span class="feature-icon">🔒</span>
                                <span>Secure encrypted connection</span>
                            </div>
                            <div class="feature-item">
                                <span class="feature-icon">📋</span>
                                <span>Clipboard synchronization</span>
                            </div>
                        </div>
                    </div>
                </div>
            `;
        }
    }

    /**
     * Disconnect from a specific DCV session
     * @param {string} instanceName - Instance to disconnect
     */
    async disconnect(instanceName) {
        console.log(`Disconnecting from DCV session: ${instanceName}`);
        
        const client = this.dcvClients.get(instanceName);
        if (client) {
            client.disconnect();
        }
        
        this.handleDCVDisconnect(instanceName);
    }

    /**
     * Disconnect all active DCV sessions
     */
    async disconnectAll() {
        console.log('Disconnecting all DCV sessions...');
        
        const instances = Array.from(this.activeSessions.keys());
        for (const instanceName of instances) {
            await this.disconnect(instanceName);
        }
        
        console.log('All DCV sessions disconnected');
    }
}

/**
 * DCV Quality Manager - Handles automatic quality adjustment
 */
class DCVQualityManager {
    constructor() {
        this.qualityProfiles = {
            'auto': { resolution: 'auto', quality: 'auto', frameRate: 'auto' },
            'high': { resolution: '1920x1080', quality: '90', frameRate: '30' },
            'medium': { resolution: '1280x720', quality: '75', frameRate: '24' },
            'low': { resolution: '1024x768', quality: '60', frameRate: '15' },
            'minimal': { resolution: '800x600', quality: '40', frameRate: '10' }
        };
    }

    /**
     * Check connection health and adjust quality if needed
     * @param {string} instanceName - Instance to check
     * @param {Object} stats - Connection statistics
     */
    checkConnectionHealth(instanceName, stats) {
        const client = dcvManager.dcvClients.get(instanceName);
        if (!client) return;

        // Auto-adjust based on performance metrics
        if (stats.latency > 200) {
            console.log(`High latency detected (${stats.latency}ms), reducing quality`);
            client.setQuality('low');
        } else if (stats.bandwidth < 1000) {
            console.log(`Low bandwidth detected (${stats.bandwidth}KB/s), reducing quality`);
            client.setQuality('minimal');
        } else if (stats.latency < 50 && stats.bandwidth > 3000) {
            console.log(`Excellent connection detected, enabling high quality`);
            client.setQuality('high');
        }
    }
}

// ============================================================================
// DCV Integration Functions - Called from GUI
// ============================================================================

/**
 * Initialize DCV manager
 */
function initializeDCVManager() {
    if (!dcvManager) {
        dcvManager = new CloudWorkstationDCVManager();
        console.log('DCV Manager initialized');
    }
}

/**
 * Connect to an instance via DCV - called from instance cards
 * @param {string} instanceName - Instance to connect to
 */
async function connectToInstanceDCV(instanceName) {
    initializeDCVManager();
    
    console.log(`User requested DCV connection to: ${instanceName}`);
    
    const success = await dcvManager.connectToInstance(instanceName);
    if (success) {
        console.log(`Successfully initiated DCV connection to ${instanceName}`);
    } else {
        console.error(`Failed to connect to ${instanceName} via DCV`);
    }
}

/**
 * Switch to a different DCV session
 * @param {string} instanceName - Instance to switch to
 */
function switchDCVSession(instanceName) {
    if (dcvManager && dcvManager.activeSessions.has(instanceName)) {
        dcvManager.showDCVSession(instanceName);
        console.log(`Switched to DCV session: ${instanceName}`);
    }
}

/**
 * Focus on a specific DCV session
 * @param {string} instanceName - Instance to focus
 */
function focusDCVSession(instanceName) {
    switchDCVSession(instanceName);
}

/**
 * Disconnect from current DCV session
 */
async function disconnectDCVSession(instanceName = null) {
    if (!dcvManager) return;
    
    const targetInstance = instanceName || currentDCVSession;
    if (targetInstance) {
        await dcvManager.disconnect(targetInstance);
    }
}

/**
 * Disconnect from all DCV sessions
 */
async function disconnectAllDCVSessions() {
    if (dcvManager) {
        await dcvManager.disconnectAll();
    }
}

/**
 * Refresh DCV sessions list
 */
function refreshDCVSessions() {
    if (dcvManager) {
        dcvManager.renderDCVSessions();
        console.log('DCV sessions refreshed');
    }
}

/**
 * Toggle DCV fullscreen mode
 */
function toggleDCVFullscreen() {
    const container = document.getElementById('dcv-display-container');
    if (container) {
        container.classList.toggle('dcv-fullscreen');
        
        const btn = document.getElementById('dcv-fullscreen-btn');
        if (btn) {
            btn.textContent = container.classList.contains('dcv-fullscreen') ? '⛙' : '⛶';
        }
    }
}

/**
 * Show DCV keyboard shortcuts modal
 */
function showDCVKeyboardShortcuts() {
    alert(`🖥️ Remote Desktop Keyboard Shortcuts:

🔧 Session Control:
• F11 - Toggle fullscreen
• Ctrl+Alt+Shift+D - Disconnect
• Ctrl+Alt+Shift+Q - Adjust quality

📋 Clipboard:
• Ctrl+C / Ctrl+V - Copy/paste (synchronized)
• Ctrl+Shift+V - Paste from local clipboard

🖱️ Mouse & Display:
• Ctrl+Alt+Shift+R - Reset display
• Ctrl+Alt+Shift+M - Release mouse capture

ℹ️ Note: These shortcuts work when the remote desktop has focus.`);
}

/**
 * Adjust DCV quality settings
 */
function adjustDCVQuality() {
    if (!currentDCVSession || !dcvManager) return;
    
    const quality = prompt(`Select quality level for ${currentDCVSession}:

1 - Auto (recommended)
2 - High (1920x1080, best quality)
3 - Medium (1280x720, balanced)
4 - Low (1024x768, low bandwidth)
5 - Minimal (800x600, minimal bandwidth)

Enter 1-5:`);
    
    const qualityMap = {
        '1': 'auto',
        '2': 'high', 
        '3': 'medium',
        '4': 'low',
        '5': 'minimal'
    };
    
    if (qualityMap[quality]) {
        const client = dcvManager.dcvClients.get(currentDCVSession);
        if (client) {
            client.setQuality(qualityMap[quality]);
            console.log(`Quality set to ${qualityMap[quality]} for ${currentDCVSession}`);
        }
    }
}

/**
 * Set DCV view mode (tabbed, split, fullscreen)
 * @param {string} mode - View mode to set
 */
function setViewMode(mode) {
    if (!dcvManager) return;
    
    dcvManager.currentViewMode = mode;
    
    // Update UI mode indicators
    document.querySelectorAll('.mode-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    
    const activeBtn = document.querySelector(`[data-mode="${mode}"]`);
    if (activeBtn) {
        activeBtn.classList.add('active');
    }
    
    // Apply view mode styles
    const container = document.getElementById('dcv-display-container');
    if (container) {
        container.className = `dcv-display-container dcv-${mode}-view`;
    }
    
    console.log(`DCV view mode set to: ${mode}`);
}

// Initialize DCV manager when GUI loads
document.addEventListener('DOMContentLoaded', () => {
    // Initialize DCV manager after other components
    setTimeout(initializeDCVManager, 1000);
});

// ============================================================================
// INTELLIGENT CONNECTION DETECTION SYSTEM
// ============================================================================

/**
 * CloudWorkstation Connection Manager
 * Automatically detects whether to use NICE DCV (GUI instances) or SSH Terminal (headless instances)
 */
class CloudWorkstationConnectionManager {
    constructor() {
        this.dcvSessions = new Map();
        this.sshSessions = new Map();
        this.connectionDetectionCache = new Map();
        this.activeConnections = new Map();
        this.connectionTimers = new Map();
        
        console.log('CloudWorkstation Connection Manager initialized');
    }

    /**
     * Intelligent connection to an instance
     * Automatically determines if DCV or SSH should be used
     * @param {string} instanceName - Instance to connect to
     * @returns {Promise<boolean>} - Success status
     */
    async connectToInstance(instanceName) {
        console.log(`Intelligent connection requested for: ${instanceName}`);
        
        try {
            // Detect instance connection type
            const connectionType = await this.detectInstanceConnectionType(instanceName);
            
            // Cache the detection result
            this.connectionDetectionCache.set(instanceName, connectionType);
            instanceConnectionTypes.set(instanceName, connectionType);
            
            console.log(`Instance ${instanceName} detected as: ${connectionType}`);
            
            // Connect using appropriate method
            if (connectionType === 'dcv') {
                return await this.connectDCV(instanceName);
            } else if (connectionType === 'ssh') {
                return await this.connectSSH(instanceName);
            } else if (connectionType === 'web') {
                return await this.connectWeb(instanceName);
            } else if (connectionType === 'both') {
                return await this.promptUserConnectionChoice(instanceName);
            } else if (connectionType === 'all') {
                return await this.promptUserConnectionChoice(instanceName);
            } else {
                console.error(`Unknown connection type: ${connectionType}`);
                return false;
            }
            
        } catch (error) {
            console.error(`Failed to connect to ${instanceName}:`, error);
            this.showConnectionError(instanceName, error.message);
            return false;
        }
    }

    /**
     * Detect the appropriate connection type for an instance
     * @param {string} instanceName - Instance to analyze
     * @returns {Promise<string>} - Connection type ('dcv' or 'ssh')
     */
    async detectInstanceConnectionType(instanceName) {
        // Check cache first
        if (this.connectionDetectionCache.has(instanceName)) {
            return this.connectionDetectionCache.get(instanceName);
        }

        try {
            // Get instance details from backend
            const instanceInfo = await this.getInstanceConnectionInfo(instanceName);
            
            // Detection logic based on instance characteristics
            const connectionType = this.analyzeInstanceForConnectionType(instanceInfo);
            
            console.log(`Connection type analysis for ${instanceName}:`, {
                hasDesktop: instanceInfo.hasDesktop,
                hasDisplay: instanceInfo.hasDisplay,
                templateType: instanceInfo.templateType,
                services: instanceInfo.services,
                recommendedType: connectionType
            });
            
            return connectionType;
            
        } catch (error) {
            console.warn(`Could not detect connection type for ${instanceName}, defaulting to SSH:`, error);
            return 'ssh'; // Default to SSH for safety
        }
    }

    /**
     * Get instance connection information from backend
     * @param {string} instanceName - Instance name
     * @returns {Promise<Object>} - Instance connection info
     */
    async getInstanceConnectionInfo(instanceName) {
        try {
            // Try to get detailed instance info from backend
            const response = await window.wails.CloudWorkstationService.GetInstanceConnectionInfo(instanceName);
            return response;
        } catch (error) {
            // Fallback: analyze based on template information
            console.log('Using fallback connection detection based on template');
            
            const instance = instances.find(inst => inst.Name === instanceName);
            if (!instance) {
                throw new Error(`Instance ${instanceName} not found`);
            }

            // Get template information to make educated guess
            const template = templates.find(tmpl => tmpl.Name === instance.Template);
            
            return {
                instanceName: instanceName,
                hasDesktop: this.templateHasDesktop(template),
                hasDisplay: this.templateHasDisplay(template),
                templateType: template?.Domain || 'unknown',
                services: this.extractTemplateServices(template),
                ports: instance.Ports || [],
                template: template
            };
        }
    }

    /**
     * Analyze instance information to determine connection type
     * @param {Object} instanceInfo - Instance information
     * @returns {string} - Connection type ('dcv' or 'ssh')
     */
    analyzeInstanceForConnectionType(instanceInfo) {
        const { hasDesktop, hasDisplay, templateType, services, ports, template } = instanceInfo;

        // PRIORITY 1: Check for explicit template declaration
        if (template && template.ConnectionType) {
            if (template.ConnectionType === 'dcv') {
                console.log(`Template explicitly declares DCV connection`);
                return 'dcv';
            } else if (template.ConnectionType === 'ssh') {
                console.log(`Template explicitly declares SSH connection`);
                return 'ssh';
            } else if (template.ConnectionType === 'web') {
                console.log(`Template explicitly declares Web interface connection`);
                return 'web';
            } else if (template.ConnectionType === 'both') {
                console.log(`Template supports both DCV and SSH - will prompt user`);
                return 'both';
            } else if (template.ConnectionType === 'all') {
                console.log(`Template supports all connection types - will prompt user`);
                return 'all';
            }
            // If 'auto', continue to detection logic below
        }

        // PRIORITY 2: Explicit desktop environment detection
        if (hasDesktop || hasDisplay) {
            return 'dcv';
        }

        // PRIORITY 3: Check for web-specific services (before GUI services)
        const webServices = ['jupyter', 'rstudio', 'streamlit', 'dash', 'shiny', 'bokeh', 'plotly', 'gradio'];
        if (services && services.some(service => 
            webServices.some(web => service.toLowerCase().includes(web)))) {
            return 'web';
        }

        // PRIORITY 4: Check for GUI-specific services
        const guiServices = ['vnc', 'rdp', 'x11', 'gnome', 'kde', 'xfce', 'mate'];
        if (services && services.some(service => 
            guiServices.some(gui => service.toLowerCase().includes(gui)))) {
            return 'dcv';
        }

        // Check for GUI-specific ports (common VNC/RDP ports)
        const guiPorts = [5900, 5901, 5902, 3389]; // VNC and RDP ports
        if (ports && ports.some(port => guiPorts.includes(port))) {
            return 'dcv';
        }

        // Template-based detection
        if (template) {
            // Check template name/description for GUI indicators
            const templateText = (template.Name + ' ' + template.Description + ' ' + (template.LongDescription || '')).toLowerCase();
            const guiKeywords = ['desktop', 'gui', 'gnome', 'kde', 'xfce', 'mate', 'ubuntu-desktop', 'fedora-workstation', 'workstation'];
            
            if (guiKeywords.some(keyword => templateText.includes(keyword))) {
                return 'dcv';
            }

            // Check packages for GUI components
            if (template.Packages) {
                const allPackages = [
                    ...(template.Packages.System || []),
                    ...(template.Packages.Conda || []),
                    ...(template.Packages.Spack || [])
                ].join(' ').toLowerCase();

                const guiPackages = ['ubuntu-desktop', 'gnome-desktop', 'kde-full', 'xfce4', 'mate-desktop', 'firefox', 'chromium'];
                if (guiPackages.some(pkg => allPackages.includes(pkg))) {
                    return 'dcv';
                }
            }
        }

        // Template type analysis
        const guiDomains = ['viz', 'visualization', 'desktop', 'workstation'];
        if (templateType && guiDomains.includes(templateType.toLowerCase())) {
            return 'dcv';
        }

        // Default to SSH for headless/server instances
        return 'ssh';
    }

    /**
     * Check if template has desktop environment
     * @param {Object} template - Template object
     * @returns {boolean} - Has desktop
     */
    templateHasDesktop(template) {
        if (!template) return false;
        
        const text = (template.Name + ' ' + template.Description).toLowerCase();
        return text.includes('desktop') || text.includes('workstation') || text.includes('gui');
    }

    /**
     * Check if template has display capabilities
     * @param {Object} template - Template object  
     * @returns {boolean} - Has display
     */
    templateHasDisplay(template) {
        if (!template || !template.Services) return false;
        
        return template.Services.some(service => 
            ['vnc', 'x11', 'display'].some(display => 
                service.Name.toLowerCase().includes(display)));
    }

    /**
     * Extract services from template
     * @param {Object} template - Template object
     * @returns {Array} - Service names
     */
    extractTemplateServices(template) {
        if (!template || !template.Services) return [];
        return template.Services.map(service => service.Name);
    }

    /**
     * Prompt user to choose connection type when template supports multiple
     * @param {string} instanceName - Instance to connect
     * @returns {Promise<boolean>} - Success status
     */
    async promptUserConnectionChoice(instanceName) {
        console.log(`Prompting user for connection choice for ${instanceName}`);
        
        // Get the detected connection type to customize the prompt
        const connectionType = this.connectionDetectionCache.get(instanceName);
        
        if (connectionType === 'all') {
            // Ultimate workstation - all options
            const choice = prompt(`${instanceName} supports ALL connection types:\n\n` +
                `1 - 🌐 Web Interface (Jupyter/RStudio in browser)\n` +
                `  • Perfect for data science, research\n` +
                `  • Works on any device with browser\n\n` +
                `2 - 🖥️ Remote Desktop (DCV)\n` +
                `  • Full graphical interface\n` +
                `  • Perfect for GUI applications\n\n` +
                `3 - 💻 SSH Terminal\n` +
                `  • Command-line interface\n` +
                `  • Perfect for scripts, automation\n\n` +
                `Enter 1, 2, or 3:`);
            
            switch(choice) {
                case '1':
                    instanceConnectionTypes.set(instanceName, 'web');
                    return await this.connectWeb(instanceName);
                case '2':
                    instanceConnectionTypes.set(instanceName, 'dcv');
                    return await this.connectDCV(instanceName);
                case '3':
                    instanceConnectionTypes.set(instanceName, 'ssh');
                    return await this.connectSSH(instanceName);
                default:
                    console.log('Invalid choice, defaulting to web interface');
                    instanceConnectionTypes.set(instanceName, 'web');
                    return await this.connectWeb(instanceName);
            }
        } else {
            // Two-option choice (DCV + SSH)
            const choice = confirm(`${instanceName} supports multiple connection types:\n\n` +
                `📊 Click "OK" for Remote Desktop (DCV)\n` +
                `  • Full graphical interface\n` +
                `  • Perfect for data visualization, GUI tools\n` +
                `  • Mouse and keyboard interaction\n\n` +
                `💻 Click "Cancel" for SSH Terminal\n` +
                `  • Command-line interface\n` +
                `  • Perfect for scripts, automation\n` +
                `  • Faster, lower bandwidth\n\n` +
                `Choose Remote Desktop (DCV)?`);
            
            if (choice) {
                console.log(`User chose DCV for ${instanceName}`);
                instanceConnectionTypes.set(instanceName, 'dcv');
                return await this.connectDCV(instanceName);
            } else {
                console.log(`User chose SSH for ${instanceName}`);
                instanceConnectionTypes.set(instanceName, 'ssh');
                return await this.connectSSH(instanceName);
            }
        }
    }

    /**
     * Connect via NICE DCV for GUI instances
     * @param {string} instanceName - Instance to connect
     * @returns {Promise<boolean>} - Success status
     */
    async connectDCV(instanceName) {
        console.log(`Connecting to ${instanceName} via NICE DCV (GUI)`);
        
        try {
            // Initialize DCV manager if needed
            initializeDCVManager();
            
            // Use existing DCV connection functionality
            const success = await dcvManager.connectToInstance(instanceName);
            
            if (success) {
                // Show DCV display area
                this.showDCVDisplay();
                this.activeConnections.set(instanceName, 'dcv');
                this.startConnectionTimer(instanceName);
                this.updateConnectionUI(instanceName, 'dcv');
                return true;
            }
            
            return false;
            
        } catch (error) {
            console.error(`DCV connection failed for ${instanceName}:`, error);
            return false;
        }
    }

    /**
     * Connect via SSH terminal for headless instances
     * @param {string} instanceName - Instance to connect
     * @returns {Promise<boolean>} - Success status
     */
    async connectSSH(instanceName) {
        console.log(`Connecting to ${instanceName} via SSH Terminal (headless)`);
        
        try {
            // Get SSH connection details
            const sshInfo = await this.getSSHConnectionInfo(instanceName);
            
            // Create SSH terminal interface
            const terminal = await this.createSSHTerminal(instanceName, sshInfo);
            
            if (terminal) {
                // Show SSH display area
                this.showSSHDisplay();
                this.activeConnections.set(instanceName, 'ssh');
                this.sshSessions.set(instanceName, terminal);
                this.startConnectionTimer(instanceName);
                this.updateConnectionUI(instanceName, 'ssh');
                return true;
            }
            
            return false;
            
        } catch (error) {
            console.error(`SSH connection failed for ${instanceName}:`, error);
            return false;
        }
    }

    /**
     * Get SSH connection information
     * @param {string} instanceName - Instance name
     * @returns {Promise<Object>} - SSH connection details
     */
    async getSSHConnectionInfo(instanceName) {
        try {
            return await window.wails.CloudWorkstationService.GetSSHConnectionInfo(instanceName);
        } catch (error) {
            // Fallback to instance IP if available
            const instance = instances.find(inst => inst.Name === instanceName);
            if (instance && instance.IP) {
                return {
                    host: instance.IP,
                    port: 22,
                    username: 'ubuntu', // Default username
                    keyPath: null // Will use SSH agent or prompt for password
                };
            }
            throw new Error(`Could not get SSH info for ${instanceName}`);
        }
    }

    /**
     * Create SSH terminal interface
     * @param {string} instanceName - Instance name
     * @param {Object} sshInfo - SSH connection info
     * @returns {Promise<Object>} - Terminal object
     */
    async createSSHTerminal(instanceName, sshInfo) {
        console.log(`Creating SSH terminal for ${instanceName}:`, sshInfo);

        // Create real WebSocket terminal connection to backend
        const terminal = {
            instanceName,
            host: sshInfo.host,
            port: sshInfo.port,
            username: sshInfo.username,
            connected: false,
            ws: null,
            outputBuffer: [],

            connect: async () => {
                console.log(`SSH Terminal: Connecting to ${sshInfo.username}@${sshInfo.host}:${sshInfo.port}`);

                return new Promise((resolve, reject) => {
                    // Connect to CloudWorkstation daemon's WebSocket terminal endpoint
                    const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
                    const wsUrl = `${wsProtocol}//${window.location.host}/api/terminal`;

                    terminal.ws = new WebSocket(wsUrl);

                    terminal.ws.onopen = () => {
                        console.log('WebSocket connection established');

                        // Send SSH connection parameters
                        terminal.ws.send(JSON.stringify({
                            host: sshInfo.host,
                            port: sshInfo.port,
                            username: sshInfo.username,
                            instanceName: instanceName
                        }));
                    };

                    terminal.ws.onmessage = (event) => {
                        // Handle terminal output
                        if (typeof event.data === 'string') {
                            const data = event.data;

                            // Check for connection success message
                            if (data.includes('Connected') || data.includes('Welcome')) {
                                terminal.connected = true;
                                this.renderSSHTerminalContent(instanceName, terminal);
                                resolve(true);
                            }

                            // Add to output buffer and update display
                            terminal.outputBuffer.push(data);
                            this.updateTerminalOutput(instanceName, data);
                        } else {
                            // Binary data (for file transfers, etc.)
                            const reader = new FileReader();
                            reader.onload = () => {
                                const text = new TextDecoder().decode(reader.result);
                                terminal.outputBuffer.push(text);
                                this.updateTerminalOutput(instanceName, text);
                            };
                            reader.readAsArrayBuffer(event.data);
                        }
                    };

                    terminal.ws.onerror = (error) => {
                        console.error('WebSocket error:', error);
                        reject(new Error('Failed to establish terminal connection'));
                    };

                    terminal.ws.onclose = () => {
                        console.log('WebSocket connection closed');
                        terminal.connected = false;
                        this.handleSSHDisconnect(instanceName);
                    };

                    // Timeout after 10 seconds
                    setTimeout(() => {
                        if (!terminal.connected) {
                            reject(new Error('Connection timeout'));
                        }
                    }, 10000);
                });
            },

            disconnect: () => {
                console.log('SSH Terminal: Disconnecting...');
                if (terminal.ws && terminal.ws.readyState === WebSocket.OPEN) {
                    terminal.ws.close();
                }
                terminal.connected = false;
                this.handleSSHDisconnect(instanceName);
            },

            write: (data) => {
                if (terminal.ws && terminal.ws.readyState === WebSocket.OPEN) {
                    terminal.ws.send(data);
                } else {
                    console.warn('Cannot write to disconnected terminal');
                }
            },

            resize: (cols, rows) => {
                // Send resize command to backend
                if (terminal.ws && terminal.ws.readyState === WebSocket.OPEN) {
                    terminal.ws.send(JSON.stringify({
                        type: 'resize',
                        cols: cols,
                        rows: rows
                    }));
                }
            }
        };

        // Start connection
        try {
            const connected = await terminal.connect();
            if (connected) {
                return terminal;
            }
        } catch (error) {
            console.error('Terminal connection failed:', error);
        }

        return null;
    }

    /**
     * Connect via Web interface for browser-based applications
     * @param {string} instanceName - Instance to connect
     * @returns {Promise<boolean>} - Success status
     */
    async connectWeb(instanceName) {
        console.log(`Connecting to ${instanceName} via Web Interface`);
        
        try {
            // Get web interface information
            const webInfo = await this.getWebInterfaceInfo(instanceName);
            
            // Create web interface display
            const webInterface = await this.createWebInterface(instanceName, webInfo);
            
            if (webInterface) {
                // Show web display area
                this.showWebDisplay();
                this.activeConnections.set(instanceName, 'web');
                this.startConnectionTimer(instanceName);
                this.updateConnectionUI(instanceName, 'web');
                return true;
            }
            
            return false;
            
        } catch (error) {
            console.error(`Web interface connection failed for ${instanceName}:`, error);
            return false;
        }
    }

    /**
     * Get web interface information
     * @param {string} instanceName - Instance name
     * @returns {Promise<Object>} - Web interface details
     */
    async getWebInterfaceInfo(instanceName) {
        try {
            return await window.wails.CloudWorkstationService.GetWebInterfaceInfo(instanceName);
        } catch (error) {
            // Fallback: use instance IP and common ports
            const instance = instances.find(inst => inst.Name === instanceName);
            if (instance && instance.IP) {
                return {
                    host: instance.IP,
                    interfaces: [
                        { name: 'Jupyter', port: 8888, path: '/jupyter/', icon: '📊' },
                        { name: 'RStudio', port: 8787, path: '/', icon: '📈' },
                        { name: 'Streamlit', port: 8501, path: '/', icon: '🌊' }
                    ]
                };
            }
            throw new Error(`Could not get web interface info for ${instanceName}`);
        }
    }

    /**
     * Create web interface display
     * @param {string} instanceName - Instance name
     * @param {Object} webInfo - Web interface info
     * @returns {Promise<Object>} - Web interface object
     */
    async createWebInterface(instanceName, webInfo) {
        console.log(`Creating web interface for ${instanceName}:`, webInfo);

        const webInterface = {
            instanceName,
            host: webInfo.host,
            interfaces: webInfo.interfaces,
            activeInterface: webInfo.interfaces[0], // Default to first interface
            connected: false,
            
            // Connect to primary interface
            connect: async () => {
                console.log(`Web Interface: Connecting to ${webInterface.activeInterface.name} at ${webInfo.host}:${webInterface.activeInterface.port}`);
                // Simulate connection delay
                await new Promise(resolve => setTimeout(resolve, 1500));
                webInterface.connected = true;
                console.log('Web Interface: Connected successfully');
                this.renderWebInterfaceContent(instanceName, webInterface);
                return true;
            },
            
            disconnect: () => {
                console.log('Web Interface: Disconnecting...');
                webInterface.connected = false;
                this.handleWebDisconnect(instanceName);
            },
            
            switchInterface: (interfaceName) => {
                const newInterface = webInterface.interfaces.find(iface => iface.name === interfaceName);
                if (newInterface) {
                    webInterface.activeInterface = newInterface;
                    this.renderWebInterfaceContent(instanceName, webInterface);
                }
            }
        };

        // Start connection
        const connected = await webInterface.connect();
        if (connected) {
            return webInterface;
        }
        
        return null;
    }

    /**
     * Show web interface display area
     */
    showWebDisplay() {
        const dcvDisplay = document.getElementById('dcv-display');
        const sshDisplay = document.getElementById('ssh-display');
        const webDisplay = document.getElementById('web-display');
        const placeholder = document.getElementById('connection-placeholder');

        if (dcvDisplay) dcvDisplay.classList.add('hidden');
        if (sshDisplay) sshDisplay.classList.add('hidden');
        if (webDisplay) webDisplay.classList.remove('hidden');
        if (placeholder) placeholder.style.display = 'none';
    }

    /**
     * Render web interface content
     * @param {string} instanceName - Instance name
     * @param {Object} webInterface - Web interface object
     */
    renderWebInterfaceContent(instanceName, webInterface) {
        let webDisplay = document.getElementById('web-display');
        
        if (!webDisplay) {
            // Create web display area if it doesn't exist
            const connectionDisplay = document.getElementById('connection-display');
            if (connectionDisplay) {
                const webDisplayHTML = `
                    <div id="web-display" class="web-display-area hidden">
                        <div class="web-interface-header">
                            <div class="interface-tabs" id="web-interface-tabs">
                                <!-- Interface tabs will be populated here -->
                            </div>
                            <div class="interface-controls">
                                <button class="btn-icon" onclick="refreshWebInterface()" title="Refresh">🔄</button>
                                <button class="btn-icon" onclick="openInNewTab()" title="Open in New Tab">🗗</button>
                                <button class="btn-icon disconnect-btn" onclick="disconnectWebInterface()" title="Disconnect">✕</button>
                            </div>
                        </div>
                        <div class="web-interface-container" id="web-interface-container">
                            <!-- Web interface content will be embedded here -->
                        </div>
                    </div>
                `;
                connectionDisplay.insertAdjacentHTML('beforeend', webDisplayHTML);
                webDisplay = document.getElementById('web-display');
            }
        }

        if (webDisplay) {
            webDisplay.classList.remove('hidden');
            
            // Render interface tabs
            const tabsContainer = document.getElementById('web-interface-tabs');
            if (tabsContainer) {
                let tabsHTML = '';
                webInterface.interfaces.forEach(iface => {
                    const isActive = iface.name === webInterface.activeInterface.name;
                    tabsHTML += `
                        <button class="interface-tab ${isActive ? 'active' : ''}" 
                                onclick="connectionManager.switchWebInterface('${instanceName}', '${iface.name}')">
                            ${iface.icon} ${iface.name}
                        </button>
                    `;
                });
                tabsContainer.innerHTML = tabsHTML;
            }

            // Render interface content
            const container = document.getElementById('web-interface-container');
            if (container) {
                const activeInterface = webInterface.activeInterface;

                // Use proxy URL through CloudWorkstation daemon to handle CORS and authentication
                const proxyUrl = `/proxy/${instanceName}${activeInterface.path}`;

                // Embed web application in iframe
                container.innerHTML = `
                    <div class="web-interface-content">
                        <div class="web-interface-loading" id="web-loading-${instanceName}">
                            <div class="loading-spinner"></div>
                            <p>Loading ${activeInterface.icon} ${activeInterface.name}...</p>
                        </div>
                        <iframe
                            id="web-iframe-${instanceName}"
                            src="${proxyUrl}"
                            class="web-interface-iframe"
                            sandbox="allow-same-origin allow-scripts allow-forms allow-popups allow-modals"
                            allow="clipboard-read; clipboard-write"
                            title="${activeInterface.name} Interface"
                            onload="connectionManager.handleWebInterfaceLoad('${instanceName}')"></iframe>
                        </div>
                    </div>
                `;
            }
        }
    }

    /**
     * Handle web interface disconnect
     * @param {string} instanceName - Instance name
     */
    handleWebDisconnect(instanceName) {
        this.activeConnections.delete(instanceName);
        this.stopConnectionTimer(instanceName);
        this.updateConnectionUI(instanceName, 'disconnected');
        
        // Show placeholder if no active connections
        if (this.activeConnections.size === 0) {
            const placeholder = document.getElementById('connection-placeholder');
            if (placeholder) placeholder.style.display = 'block';
        }
    }

    /**
     * Switch web interface for an instance
     * @param {string} instanceName - Instance name
     * @param {string} interfaceName - Interface to switch to
     */
    switchWebInterface(instanceName, interfaceName) {
        console.log(`Switching to ${interfaceName} for ${instanceName}`);

        // Find the interface configuration
        const connections = this.webInterfaces || new Map();
        const webInterface = connections.get(instanceName);

        if (webInterface) {
            const newInterface = webInterface.interfaces.find(iface => iface.name === interfaceName);

            if (newInterface) {
                webInterface.activeInterface = newInterface;

                // Update iframe src to new interface
                const iframe = document.getElementById(`web-iframe-${instanceName}`);
                const proxyUrl = `/proxy/${instanceName}${newInterface.path}`;

                if (iframe) {
                    // Show loading indicator
                    const loading = document.getElementById(`web-loading-${instanceName}`);
                    if (loading) loading.style.display = 'flex';

                    iframe.src = proxyUrl;
                }

                // Update active tab styling
                const tabs = document.querySelectorAll('.interface-tab');
                tabs.forEach(tab => {
                    if (tab.textContent.includes(interfaceName)) {
                        tab.classList.add('active');
                    } else {
                        tab.classList.remove('active');
                    }
                });
            }
        }
    }

    /**
     * Handle web interface iframe load
     * @param {string} instanceName - Instance name
     */
    handleWebInterfaceLoad(instanceName) {
        console.log(`Web interface loaded for ${instanceName}`);

        // Hide loading indicator
        const loading = document.getElementById(`web-loading-${instanceName}`);
        if (loading) {
            loading.style.display = 'none';
        }

        // Show iframe
        const iframe = document.getElementById(`web-iframe-${instanceName}`);
        if (iframe) {
            iframe.style.display = 'block';
        }
    }

    /**
     * Show DCV display area
     */
    showDCVDisplay() {
        const dcvDisplay = document.getElementById('dcv-display');
        const sshDisplay = document.getElementById('ssh-display');
        const placeholder = document.getElementById('connection-placeholder');

        if (dcvDisplay) dcvDisplay.classList.remove('hidden');
        if (sshDisplay) sshDisplay.classList.add('hidden');
        if (placeholder) placeholder.style.display = 'none';
    }

    /**
     * Show SSH terminal display area
     */
    showSSHDisplay() {
        const dcvDisplay = document.getElementById('dcv-display');
        const sshDisplay = document.getElementById('ssh-display');
        const placeholder = document.getElementById('connection-placeholder');

        if (dcvDisplay) dcvDisplay.classList.add('hidden');
        if (sshDisplay) sshDisplay.classList.remove('hidden');
        if (placeholder) placeholder.style.display = 'none';
    }

    /**
     * Render SSH terminal content
     * @param {string} instanceName - Instance name
     * @param {Object} terminal - Terminal object
     */
    renderSSHTerminalContent(instanceName, terminal) {
        const terminalContent = document.getElementById('ssh-terminal-content');
        const terminalTitle = document.getElementById('ssh-terminal-title');

        if (terminalTitle) {
            terminalTitle.textContent = `${terminal.username}@${instanceName}`;
        }

        if (terminalContent) {
            // Create interactive terminal interface with WebSocket backend
            terminalContent.innerHTML = `
                <div class="terminal-session" id="terminal-${instanceName}">
                    <div class="terminal-header">
                        <span class="terminal-info">Connected to: ${terminal.username}@${terminal.host}:${terminal.port}</span>
                        <button class="terminal-clear-btn" onclick="connectionManager.clearTerminal('${instanceName}')">Clear</button>
                    </div>
                    <div class="terminal-output" id="terminal-output-${instanceName}"></div>
                    <div class="terminal-input-line">
                        <span class="terminal-prompt">${terminal.username}@${instanceName}:~$ </span>
                        <input type="text"
                               class="terminal-input"
                               id="terminal-input-${instanceName}"
                               placeholder="Type command and press Enter..."
                               autocomplete="off"
                               spellcheck="false">
                    </div>
                </div>
            `;

            // Set up keyboard input handling
            const input = document.getElementById(`terminal-input-${instanceName}`);
            if (input) {
                input.focus();

                input.addEventListener('keydown', (e) => {
                    if (e.key === 'Enter') {
                        const command = input.value;
                        if (command) {
                            // Send command to terminal
                            terminal.write(command + '\n');

                            // Echo command in output
                            this.appendTerminalOutput(instanceName,
                                `<span class="terminal-prompt">${terminal.username}@${instanceName}:~$ </span>${command}`
                            );

                            // Clear input
                            input.value = '';
                        }
                    } else if (e.key === 'ArrowUp') {
                        // Command history (future enhancement)
                        e.preventDefault();
                    }
                });

                // Handle Ctrl+C
                input.addEventListener('keydown', (e) => {
                    if (e.ctrlKey && e.key === 'c') {
                        e.preventDefault();
                        terminal.write('\x03'); // Send Ctrl+C signal
                    }
                });
            }
        }
    }

    /**
     * Update terminal output with new data
     * @param {string} instanceName - Instance name
     * @param {string} data - Output data
     */
    updateTerminalOutput(instanceName, data) {
        this.appendTerminalOutput(instanceName, data);
    }

    /**
     * Append text to terminal output
     * @param {string} instanceName - Instance name
     * @param {string} text - Text to append
     */
    appendTerminalOutput(instanceName, text) {
        const outputDiv = document.getElementById(`terminal-output-${instanceName}`);
        if (outputDiv) {
            const line = document.createElement('div');
            line.className = 'terminal-line';
            line.innerHTML = this.escapeTerminalOutput(text);
            outputDiv.appendChild(line);

            // Auto-scroll to bottom
            outputDiv.scrollTop = outputDiv.scrollHeight;
        }
    }

    /**
     * Clear terminal output
     * @param {string} instanceName - Instance name
     */
    clearTerminal(instanceName) {
        const outputDiv = document.getElementById(`terminal-output-${instanceName}`);
        if (outputDiv) {
            outputDiv.innerHTML = '';
        }
    }

    /**
     * Escape terminal output for HTML display
     * @param {string} text - Raw text
     * @returns {string} - HTML-safe text
     */
    escapeTerminalOutput(text) {
        return text
            .replace(/&/g, '&amp;')
            .replace(/</g, '&lt;')
            .replace(/>/g, '&gt;')
            .replace(/\n/g, '<br>')
            .replace(/ {2}/g, '&nbsp;&nbsp;')
            .replace(/\t/g, '&nbsp;&nbsp;&nbsp;&nbsp;');
    }

    /**
     * Handle SSH disconnect
     * @param {string} instanceName - Instance name
     */
    handleSSHDisconnect(instanceName) {
        this.sshSessions.delete(instanceName);
        this.activeConnections.delete(instanceName);
        this.stopConnectionTimer(instanceName);
        this.updateConnectionUI(instanceName, 'disconnected');
        
        // Show placeholder if no active connections
        if (this.activeConnections.size === 0) {
            const placeholder = document.getElementById('connection-placeholder');
            if (placeholder) placeholder.style.display = 'block';
        }
    }

    /**
     * Start connection duration timer
     * @param {string} instanceName - Instance name
     */
    startConnectionTimer(instanceName) {
        const startTime = Date.now();
        this.connectionTimers.set(instanceName, {
            startTime,
            interval: setInterval(() => {
                this.updateConnectionDuration(instanceName, Date.now() - startTime);
            }, 1000)
        });
    }

    /**
     * Stop connection timer
     * @param {string} instanceName - Instance name
     */
    stopConnectionTimer(instanceName) {
        const timer = this.connectionTimers.get(instanceName);
        if (timer) {
            clearInterval(timer.interval);
            this.connectionTimers.delete(instanceName);
        }
    }

    /**
     * Update connection UI elements
     * @param {string} instanceName - Instance name
     * @param {string} connectionType - Connection type
     */
    updateConnectionUI(instanceName, connectionType) {
        currentSession = instanceName;
        currentSessionType = connectionType;

        const connectionTypeInfo = document.getElementById('connection-type-info');
        if (connectionTypeInfo) {
            const typeLabels = {
                'dcv': '🖥️ NICE DCV Remote Desktop',
                'ssh': '💻 SSH Terminal',
                'disconnected': 'Not connected'
            };
            connectionTypeInfo.textContent = typeLabels[connectionType] || 'Unknown';
        }

        // Update session header
        const sessionHeader = document.getElementById('dcv-session-header');
        if (sessionHeader) {
            const instanceNameEl = document.getElementById('dcv-instance-name');
            const sessionQuality = document.getElementById('dcv-session-quality');
            
            if (instanceNameEl) {
                instanceNameEl.textContent = connectionType === 'disconnected' ? 'No Session' : instanceName;
            }
            
            if (sessionQuality) {
                const statusLabels = {
                    'dcv': '🟢 Connected (DCV)',
                    'ssh': '🟢 Connected (SSH)',
                    'disconnected': '⚪ Disconnected'
                };
                sessionQuality.textContent = statusLabels[connectionType] || '⚪ Disconnected';
            }
        }
    }

    /**
     * Update connection duration display
     * @param {string} instanceName - Instance name
     * @param {number} duration - Duration in milliseconds
     */
    updateConnectionDuration(instanceName, duration) {
        const durationEl = document.getElementById('connection-duration');
        if (durationEl && currentSession === instanceName) {
            const minutes = Math.floor(duration / 60000);
            const seconds = Math.floor((duration % 60000) / 1000);
            durationEl.textContent = `${minutes}m ${seconds}s`;
        }
    }

    /**
     * Show connection error
     * @param {string} instanceName - Instance name
     * @param {string} errorMessage - Error message
     */
    showConnectionError(instanceName, errorMessage) {
        console.error(`Connection error for ${instanceName}: ${errorMessage}`);

        // Show user-friendly error dialog
        const dialog = document.createElement('div');
        dialog.className = 'error-dialog-overlay';
        dialog.innerHTML = `
            <div class="error-dialog">
                <div class="error-dialog-header">
                    <span class="error-icon">⚠️</span>
                    <h3>Connection Failed</h3>
                </div>
                <div class="error-dialog-body">
                    <p><strong>Instance:</strong> ${instanceName}</p>
                    <p><strong>Error:</strong> ${errorMessage}</p>
                    <div class="error-suggestions">
                        <p><strong>Suggestions:</strong></p>
                        <ul>
                            <li>Verify the instance is running</li>
                            <li>Check network connectivity</li>
                            <li>Ensure security groups allow access</li>
                            <li>Try reconnecting in a few moments</li>
                        </ul>
                    </div>
                </div>
                <div class="error-dialog-footer">
                    <button class="btn-secondary" onclick="this.closest('.error-dialog-overlay').remove()">Close</button>
                    <button class="btn-primary" onclick="connectionManager.retryConnection('${instanceName}'); this.closest('.error-dialog-overlay').remove();">Retry</button>
                </div>
            </div>
        `;

        document.body.appendChild(dialog);

        // Also show toast notification
        this.showNotification(`Failed to connect to ${instanceName}`, 'error');
    }

    /**
     * Retry connection to an instance
     * @param {string} instanceName - Instance to reconnect
     */
    async retryConnection(instanceName) {
        console.log(`Retrying connection to ${instanceName}`);
        this.showNotification(`Retrying connection to ${instanceName}...`, 'info');

        // Get instance info and attempt to connect
        const instances = await window.wails.CloudWorkstationService.GetInstances();
        const instance = instances.find(i => i.Name === instanceName);

        if (instance) {
            await this.connectToInstance(instance);
        }
    }

    /**
     * Disconnect from any active session for an instance
     * @param {string} instanceName - Instance name
     */
    async disconnectInstance(instanceName) {
        const connectionType = this.activeConnections.get(instanceName);
        
        if (connectionType === 'dcv') {
            await dcvManager?.disconnect(instanceName);
        } else if (connectionType === 'ssh') {
            const terminal = this.sshSessions.get(instanceName);
            if (terminal) {
                terminal.disconnect();
            }
        }
    }

    /**
     * Get connection status for an instance
     * @param {string} instanceName - Instance name
     * @returns {Object} - Connection status
     */
    getConnectionStatus(instanceName) {
        const connectionType = this.activeConnections.get(instanceName);
        const timer = this.connectionTimers.get(instanceName);
        
        return {
            connected: !!connectionType,
            type: connectionType || 'none',
            duration: timer ? Date.now() - timer.startTime : 0,
            detectedType: instanceConnectionTypes.get(instanceName) || 'unknown'
        };
    }
}

// ============================================================================
// CONNECTION INTERFACE FUNCTIONS
// ============================================================================

/**
 * Main connection function - called from instance cards
 * Uses intelligent detection to choose DCV or SSH
 * @param {string} instanceName - Instance to connect to
 */
async function connectToInstance(instanceName) {
    console.log(`Connect requested for instance: ${instanceName}`);
    
    if (!connectionManager) {
        console.error('Connection manager not initialized');
        return;
    }
    
    // Show remote desktop section
    showSection('remote-desktop');
    
    // Connect using intelligent detection
    const success = await connectionManager.connectToInstance(instanceName);
    
    if (success) {
        console.log(`Successfully connected to ${instanceName}`);
    } else {
        console.error(`Failed to connect to ${instanceName}`);
    }
}

/**
 * Disconnect from SSH terminal
 */
function disconnectSSHSession() {
    if (currentSession && currentSessionType === 'ssh' && connectionManager) {
        connectionManager.disconnectInstance(currentSession);
    }
}

/**
 * Toggle SSH terminal fullscreen
 */
function toggleSSHFullscreen() {
    const sshDisplay = document.getElementById('ssh-display');
    if (sshDisplay) {
        sshDisplay.classList.toggle('ssh-fullscreen');
    }
}

/**
 * Update connection duration displays (called periodically)
 */
function updateConnectionDurations() {
    if (connectionManager && currentSession) {
        const status = connectionManager.getConnectionStatus(currentSession);
        if (status.connected) {
            connectionManager.updateConnectionDuration(currentSession, status.duration);
        }
    }
}

// Enhanced instance card rendering with connection type indicators
const originalRenderInstances = renderInstances;
renderInstances = function() {
    const grid = document.getElementById('instances-grid');
    
    if (instances.length === 0) {
        grid.innerHTML = `
            <div class="instance-card">
                <div class="text-center">
                    <p>No instances running</p>
                    <small>Launch your first research environment in Quick Start</small>
                </div>
            </div>
        `;
        return;
    }
    
    let html = '';
    instances.forEach(instance => {
        // Get connection type detection
        const detectedType = instanceConnectionTypes.get(instance.Name) || 'detecting...';
        const connectionIcon = detectedType === 'dcv' ? '🖥️' : detectedType === 'ssh' ? '💻' : '🔍';
        const connectionStatus = connectionManager?.getConnectionStatus(instance.Name);
        
        html += `
            <div class="instance-card">
                <div class="instance-header">
                    <div class="instance-name">${instance.Name}</div>
                    <div class="instance-status ${instance.State}">${instance.State}</div>
                </div>
                <div class="instance-details">
                    ${instance.IP ? `<p><strong>IP:</strong> ${instance.IP}</p>` : ''}
                    ${instance.Cost ? `<p><strong>Cost:</strong> $${instance.Cost.toFixed(4)}/hour</p>` : ''}
                    ${instance.Region ? `<p><strong>Region:</strong> ${instance.Region}</p>` : ''}
                    <p><strong>Connection:</strong> ${connectionIcon} ${detectedType === 'dcv' ? 'Remote Desktop' : detectedType === 'ssh' ? 'SSH Terminal' : 'Detecting...'}</p>
                    ${connectionStatus?.connected ? `<p><strong>Status:</strong> 🟢 Connected (${Math.floor(connectionStatus.duration / 60000)}m)</p>` : ''}
                </div>
                <div class="instance-actions">
                    ${connectionStatus?.connected ? 
                        `<button class="btn-primary" onclick="showSection('remote-desktop')">View Session</button>
                         <button class="btn-secondary" onclick="connectionManager.disconnectInstance('${instance.Name}')">Disconnect</button>` :
                        `<button class="btn-primary" onclick="connectToInstance('${instance.Name}')">Connect</button>`
                    }
                    ${instance.State === 'running' ? 
                        `<button class="btn-secondary" onclick="stopInstance('${instance.Name}')">Stop</button>` :
                        `<button class="btn-secondary" onclick="startInstance('${instance.Name}')">Start</button>`
                    }
                </div>
            </div>
        `;
    });
    
    grid.innerHTML = html;
};

// =============================================================================
// SETTINGS MANAGEMENT
// =============================================================================

// Show/hide settings modal
function showSettings() {
    const modal = document.getElementById("settings-modal");
    modal.classList.remove("hidden");
    loadSettingsIntoForm();
    showSettingsSection("general");
    settingsChanged = false;
}

function hideSettings() {
    const modal = document.getElementById("settings-modal");
    modal.classList.add("hidden");
}

// Settings section navigation
function showSettingsSection(sectionName) {
    document.querySelectorAll(".settings-section").forEach(section => {
        section.classList.remove("active");
    });
    document.querySelectorAll(".settings-nav-btn").forEach(btn => {
        btn.classList.remove("active");
    });
    document.getElementById(`settings-${sectionName}`).classList.add("active");
    document.querySelector(`[onclick="showSettingsSection('${sectionName}')"]`).classList.add("active");
}

// Load settings into form
function loadSettingsIntoForm() {
    const elements = [
        { id: "autostart-gui", key: "general.autostartGUI" },
        { id: "auto-refresh", key: "general.autoRefresh" },
        { id: "theme-selector", key: "appearance.theme" },
        { id: "debug-mode", key: "advanced.debugMode" }
    ];
    
    elements.forEach(({id, key}) => {
        const element = document.getElementById(id);
        if (element) {
            const value = getNestedProperty(settings, key);
            if (element.type === "checkbox") {
                element.checked = value;
            } else {
                element.value = value;
            }
        }
    });
}

// Save settings
function saveSettings() {
    showNotification("Settings saved successfully", "success");
    hideSettings();
    settingsChanged = false;
}

// Auto-start configuration
async function toggleAutoStart(enabled) {
    try {
        settings.general.autostartGUI = enabled;
        localStorage.setItem("cws-settings", JSON.stringify(settings));
        showNotification(enabled ? "Auto-start enabled" : "Auto-start disabled", "success");
        settingsChanged = true;
    } catch (error) {
        showNotification("Failed to configure auto-start", "error");
    }
}

// Test daemon connection
async function testDaemonConnection() {
    try {
        const response = await fetch("http://localhost:8947/api/v1/status");
        if (response.ok) {
            showNotification("✅ Daemon connection successful", "success");
        } else {
            showNotification("❌ Daemon connection failed", "error");
        }
    } catch (error) {
        showNotification("❌ Connection failed: " + error.message, "error");
    }
}

// Utility functions
function showNotification(message, type = "info") {
    console.log(`[${type.toUpperCase()}] ${message}`);
}

function getNestedProperty(obj, path) {
    return path.split(".").reduce((o, p) => o && o[p], obj);
}

function markSettingsChanged() {
    settingsChanged = true;
}

function resetSettings() {
    if (confirm("Reset all settings to defaults?")) {
        localStorage.removeItem("cws-settings");
        location.reload();
    }
}

function exportSettings() {
    const blob = new Blob([JSON.stringify(settings, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a");
    a.href = url;
    a.download = "cloudworkstation-settings.json";
    a.click();
    URL.revokeObjectURL(url);
}

// Initialize settings on page load
document.addEventListener("DOMContentLoaded", () => {
    const saved = localStorage.getItem("cws-settings");
    if (saved) {
        try {
            settings = { ...settings, ...JSON.parse(saved) };
        } catch (e) {
            console.error("Failed to load settings:", e);
        }
    }
});

// =============================================================================
// END SETTINGS MANAGEMENT
// =============================================================================
