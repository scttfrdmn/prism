// CloudWorkstation GUI - Progressive Disclosure Interface
// Minimal, professional UI for academic researchers

let selectedTemplate = null;
let templates = [];
let instances = [];
let currentTheme = 'core';

// Initialize application
document.addEventListener('DOMContentLoaded', async () => {
    console.log('CloudWorkstation GUI starting...');
    
    // Initialize UI state
    await initializeApp();
    
    // Start periodic updates
    setInterval(updateInstances, 30000); // Update every 30 seconds
    setInterval(updateClock, 1000); // Update clock every second
    
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

// Render templates with progressive disclosure
function renderTemplates() {
    const grid = document.getElementById('template-grid');
    
    if (templates.length === 0) {
        grid.innerHTML = `
            <div class="template-card">
                <div class="text-center">
                    <p>No templates available</p>
                    <small>Please ensure the daemon is running</small>
                </div>
            </div>
        `;
        return;
    }
    
    // Group templates by category for better organization
    const categories = groupBy(templates, 'Category');
    
    let html = '';
    Object.entries(categories).forEach(([category, categoryTemplates]) => {
        categoryTemplates.forEach(template => {
            html += `
                <div class="template-card" onclick="selectTemplate('${template.Name}')">
                    <div class="template-header">
                        <span class="template-icon">${template.Icon}</span>
                        <div>
                            <div class="template-title">${template.Name}</div>
                            <div class="template-category">${template.Category}</div>
                        </div>
                    </div>
                    <div class="template-description">
                        ${template.Description}
                    </div>
                </div>
            `;
        });
    });
    
    grid.innerHTML = html;
}

// Handle template selection (Progressive Disclosure - Step 1)
function selectTemplate(templateName) {
    selectedTemplate = templates.find(t => t.Name === templateName);
    
    // Update UI to show selection
    document.querySelectorAll('.template-card').forEach(card => {
        card.classList.remove('selected');
    });
    
    event.currentTarget.classList.add('selected');
    
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
    document.querySelectorAll('.template-card').forEach(card => {
        card.classList.remove('selected');
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
    document.getElementById(sectionId).classList.add('active');
    
    // Update navigation
    document.querySelectorAll('.nav-item').forEach(nav => {
        nav.classList.remove('active');
    });
    
    event.currentTarget.classList.add('active');
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
function showSettings() {
    document.getElementById('settings-modal').classList.remove('hidden');
}

function hideSettings() {
    document.getElementById('settings-modal').classList.add('hidden');
}

// Instance actions
async function connectToInstance(name) {
    try {
        const connectionInfo = await window.wails.CloudWorkstationService.ConnectToInstance(name);
        
        // Show connection modal
        showConnectionInfo(name, connectionInfo);
        
    } catch (error) {
        console.error('Connection failed:', error);
        showError(`Failed to get connection info: ${error.message}`);
    }
}

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