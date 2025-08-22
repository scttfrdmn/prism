package web

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/scttfrdmn/cloudworkstation/pkg/types"
)

// DashboardServer provides a web dashboard for CloudWorkstation
type DashboardServer struct {
	templates     *template.Template
	instanceFunc  func() ([]*types.Instance, error)
	templateFunc  func() (map[string]types.RuntimeTemplate, error)
	proxyManager  *ProxyManager
	staticContent map[string][]byte
}

// NewDashboardServer creates a new dashboard server
func NewDashboardServer(
	instanceFunc func() ([]*types.Instance, error),
	templateFunc func() (map[string]types.RuntimeTemplate, error),
	proxyManager *ProxyManager,
) *DashboardServer {
	return &DashboardServer{
		instanceFunc:  instanceFunc,
		templateFunc:  templateFunc,
		proxyManager:  proxyManager,
		staticContent: generateStaticContent(),
		templates:     parseTemplates(),
	}
}

// ServeHTTP implements http.Handler
func (ds *DashboardServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/", "/dashboard":
		ds.serveDashboard(w, r)
	case "/api/instances":
		ds.serveInstances(w, r)
	case "/api/templates":
		ds.serveTemplates(w, r)
	case "/api/proxy/stats":
		ds.serveProxyStats(w, r)
	case "/static/style.css":
		ds.serveStatic(w, r, "style.css", "text/css")
	case "/static/script.js":
		ds.serveStatic(w, r, "script.js", "application/javascript")
	default:
		if r.URL.Path == "/favicon.ico" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		http.NotFound(w, r)
	}
}

// serveDashboard serves the main dashboard HTML
func (ds *DashboardServer) serveDashboard(w http.ResponseWriter, r *http.Request) {
	instances, err := ds.instanceFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title     string
		Instances []*types.Instance
		Timestamp time.Time
	}{
		Title:     "CloudWorkstation Dashboard",
		Instances: instances,
		Timestamp: time.Now(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ds.templates.ExecuteTemplate(w, "dashboard", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// serveInstances serves instance data as JSON
func (ds *DashboardServer) serveInstances(w http.ResponseWriter, r *http.Request) {
	instances, err := ds.instanceFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instances)
}

// serveTemplates serves template data as JSON
func (ds *DashboardServer) serveTemplates(w http.ResponseWriter, r *http.Request) {
	templates, err := ds.templateFunc()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// serveProxyStats serves proxy statistics as JSON
func (ds *DashboardServer) serveProxyStats(w http.ResponseWriter, r *http.Request) {
	stats := ds.proxyManager.GetProxyStats()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// serveStatic serves static content
func (ds *DashboardServer) serveStatic(w http.ResponseWriter, r *http.Request, name, contentType string) {
	content, exists := ds.staticContent[name]
	if !exists {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Write(content)
}

// parseTemplates creates the HTML templates
func parseTemplates() *template.Template {
	tmpl := template.New("dashboard")
	template.Must(tmpl.Parse(dashboardHTML))
	return tmpl
}

// generateStaticContent generates static CSS and JavaScript
func generateStaticContent() map[string][]byte {
	return map[string][]byte{
		"style.css": []byte(dashboardCSS),
		"script.js": []byte(dashboardJS),
	}
}

// HTML template for the dashboard
const dashboardHTML = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/static/style.css">
</head>
<body>
    <div class="container">
        <header>
            <h1>☁️ CloudWorkstation Dashboard</h1>
            <div class="status">
                <span class="timestamp">Last updated: {{.Timestamp.Format "15:04:05"}}</span>
                <button id="refresh-btn" onclick="refreshData()">🔄 Refresh</button>
            </div>
        </header>

        <nav class="tabs">
            <button class="tab-btn active" onclick="showTab('instances')">Instances</button>
            <button class="tab-btn" onclick="showTab('templates')">Templates</button>
            <button class="tab-btn" onclick="showTab('proxy')">Web Proxy</button>
            <button class="tab-btn" onclick="showTab('costs')">Cost Analysis</button>
        </nav>

        <main>
            <div id="instances-tab" class="tab-content active">
                <h2>Active Instances</h2>
                <div class="instances-grid">
                    {{range .Instances}}
                    <div class="instance-card {{.State}}">
                        <div class="instance-header">
                            <h3>{{.Name}}</h3>
                            <span class="instance-state">{{.State}}</span>
                        </div>
                        <div class="instance-details">
                            <p><strong>Type:</strong> {{.InstanceType}}</p>
                            <p><strong>IP:</strong> {{.PublicIP}}</p>
                            <p><strong>Template:</strong> {{.Template}}</p>
                            <p><strong>Launch Time:</strong> {{.LaunchTime.Format "Jan 2, 15:04"}}</p>
                            <p><strong>Cost:</strong> ${{printf "%.2f" .HourlyRate}}/hour</p>
                            {{if .HasWebInterface}}
                            <p><strong>Web Port:</strong> {{.WebPort}}</p>
                            {{end}}
                        </div>
                        <div class="instance-actions">
                            {{if .HasWebInterface}}
                            <a href="/proxy/{{.Name}}/" target="_blank" class="btn btn-primary">Open Web Interface</a>
                            {{end}}
                            <button class="btn btn-secondary" onclick="connectSSH('{{.Name}}')">SSH Connect</button>
                            {{if eq .State "running"}}
                            <button class="btn btn-warning" onclick="stopInstance('{{.ID}}')">Stop</button>
                            {{else if eq .State "stopped"}}
                            <button class="btn btn-success" onclick="startInstance('{{.ID}}')">Start</button>
                            {{end}}
                        </div>
                    </div>
                    {{else}}
                    <div class="no-instances">
                        <p>No instances running</p>
                        <p>Launch an instance using: <code>cws launch template-name my-instance</code></p>
                    </div>
                    {{end}}
                </div>
            </div>

            <div id="templates-tab" class="tab-content">
                <h2>Available Templates</h2>
                <div id="templates-container">Loading templates...</div>
            </div>

            <div id="proxy-tab" class="tab-content">
                <h2>Web Proxy Status</h2>
                <div id="proxy-container">Loading proxy statistics...</div>
            </div>

            <div id="costs-tab" class="tab-content">
                <h2>Cost Analysis</h2>
                <div id="costs-container">
                    <div class="cost-summary">
                        <h3>Current Costs</h3>
                        <div id="cost-metrics">Calculating...</div>
                    </div>
                    <div class="cost-chart">
                        <canvas id="cost-chart"></canvas>
                    </div>
                </div>
            </div>
        </main>

        <footer>
            <p>CloudWorkstation v0.4.5 | <a href="https://github.com/scttfrdmn/cloudworkstation">GitHub</a></p>
        </footer>
    </div>

    <script src="/static/script.js"></script>
</body>
</html>`

// CSS styles for the dashboard
const dashboardCSS = `
:root {
    --primary-color: #2563eb;
    --secondary-color: #64748b;
    --success-color: #10b981;
    --warning-color: #f59e0b;
    --danger-color: #ef4444;
    --bg-color: #f8fafc;
    --card-bg: #ffffff;
    --text-color: #1e293b;
    --border-color: #e2e8f0;
}

* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
}

body {
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, 'Helvetica Neue', Arial, sans-serif;
    background: var(--bg-color);
    color: var(--text-color);
    line-height: 1.6;
}

.container {
    max-width: 1400px;
    margin: 0 auto;
    padding: 20px;
}

header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 20px 0;
    border-bottom: 2px solid var(--border-color);
    margin-bottom: 30px;
}

h1 {
    font-size: 2rem;
    color: var(--primary-color);
}

.status {
    display: flex;
    align-items: center;
    gap: 15px;
}

.timestamp {
    color: var(--secondary-color);
    font-size: 0.9rem;
}

button {
    background: var(--primary-color);
    color: white;
    border: none;
    padding: 8px 16px;
    border-radius: 6px;
    cursor: pointer;
    font-size: 0.9rem;
    transition: background 0.2s;
}

button:hover {
    opacity: 0.9;
}

.tabs {
    display: flex;
    gap: 10px;
    margin-bottom: 30px;
    border-bottom: 2px solid var(--border-color);
}

.tab-btn {
    background: transparent;
    color: var(--secondary-color);
    padding: 10px 20px;
    border-radius: 6px 6px 0 0;
    border: none;
    cursor: pointer;
    transition: all 0.2s;
}

.tab-btn.active {
    background: var(--card-bg);
    color: var(--primary-color);
    border-bottom: 2px solid var(--primary-color);
    margin-bottom: -2px;
}

.tab-content {
    display: none;
}

.tab-content.active {
    display: block;
}

.instances-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
    gap: 20px;
}

.instance-card {
    background: var(--card-bg);
    border-radius: 8px;
    padding: 20px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
    border: 1px solid var(--border-color);
}

.instance-card.running {
    border-left: 4px solid var(--success-color);
}

.instance-card.stopped {
    border-left: 4px solid var(--warning-color);
}

.instance-card.terminated {
    border-left: 4px solid var(--danger-color);
}

.instance-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
}

.instance-header h3 {
    color: var(--primary-color);
}

.instance-state {
    background: var(--success-color);
    color: white;
    padding: 4px 8px;
    border-radius: 4px;
    font-size: 0.8rem;
    text-transform: uppercase;
}

.instance-details p {
    margin: 8px 0;
    font-size: 0.9rem;
}

.instance-actions {
    display: flex;
    gap: 10px;
    margin-top: 15px;
}

.btn {
    padding: 8px 12px;
    border-radius: 4px;
    text-decoration: none;
    font-size: 0.85rem;
    cursor: pointer;
    border: none;
    transition: opacity 0.2s;
}

.btn-primary {
    background: var(--primary-color);
    color: white;
}

.btn-secondary {
    background: var(--secondary-color);
    color: white;
}

.btn-success {
    background: var(--success-color);
    color: white;
}

.btn-warning {
    background: var(--warning-color);
    color: white;
}

.no-instances {
    grid-column: 1 / -1;
    text-align: center;
    padding: 60px 20px;
    background: var(--card-bg);
    border-radius: 8px;
    color: var(--secondary-color);
}

.no-instances code {
    background: var(--bg-color);
    padding: 4px 8px;
    border-radius: 4px;
    font-family: 'Courier New', monospace;
}

footer {
    margin-top: 60px;
    padding-top: 20px;
    border-top: 1px solid var(--border-color);
    text-align: center;
    color: var(--secondary-color);
}

footer a {
    color: var(--primary-color);
    text-decoration: none;
}

footer a:hover {
    text-decoration: underline;
}`

// JavaScript for the dashboard
const dashboardJS = `
// Tab switching
function showTab(tabName) {
    // Hide all tabs
    document.querySelectorAll('.tab-content').forEach(tab => {
        tab.classList.remove('active');
    });
    document.querySelectorAll('.tab-btn').forEach(btn => {
        btn.classList.remove('active');
    });
    
    // Show selected tab
    document.getElementById(tabName + '-tab').classList.add('active');
    event.target.classList.add('active');
    
    // Load tab-specific data
    switch(tabName) {
        case 'templates':
            loadTemplates();
            break;
        case 'proxy':
            loadProxyStats();
            break;
        case 'costs':
            loadCostAnalysis();
            break;
    }
}

// Refresh data
function refreshData() {
    location.reload();
}

// Load templates
async function loadTemplates() {
    try {
        const response = await fetch('/api/templates');
        const templates = await response.json();
        
        const container = document.getElementById('templates-container');
        container.innerHTML = '<div class="templates-grid">' +
            Object.entries(templates).map(([name, template]) => ` + "`" + `
                <div class="template-card">
                    <h3>${name}</h3>
                    <p>${template.description}</p>
                    <p><strong>Instance Type:</strong> ${template.instance_type}</p>
                    <p><strong>Cost:</strong> $${template.estimated_cost_per_hour}/hour</p>
                    <div class="template-ports">
                        ${template.ports.map(port => ` + "`" + `<span class="port">${port}</span>` + "`" + `).join('')}
                    </div>
                </div>
            ` + "`" + `).join('') +
            '</div>';
    } catch (error) {
        console.error('Failed to load templates:', error);
    }
}

// Load proxy statistics
async function loadProxyStats() {
    try {
        const response = await fetch('/api/proxy/stats');
        const stats = await response.json();
        
        const container = document.getElementById('proxy-container');
        if (Object.keys(stats).length === 0) {
            container.innerHTML = '<p>No active proxy connections</p>';
            return;
        }
        
        container.innerHTML = '<div class="proxy-stats">' +
            Object.values(stats).map(stat => ` + "`" + `
                <div class="proxy-card">
                    <h3>${stat.instance_name}</h3>
                    <p><strong>Target:</strong> ${stat.target_url}</p>
                    <p><strong>Access Count:</strong> ${stat.access_count}</p>
                    <p><strong>Last Access:</strong> ${new Date(stat.last_accessed).toLocaleString()}</p>
                </div>
            ` + "`" + `).join('') +
            '</div>';
    } catch (error) {
        console.error('Failed to load proxy stats:', error);
    }
}

// Load cost analysis
async function loadCostAnalysis() {
    try {
        const response = await fetch('/api/instances');
        const instances = await response.json();
        
        let totalHourlyCost = 0;
        let runningCount = 0;
        
        instances.forEach(instance => {
            if (instance.state === 'running') {
                totalHourlyCost += instance.hourly_rate;
                runningCount++;
            }
        });
        
        const dailyCost = totalHourlyCost * 24;
        const monthlyCost = dailyCost * 30;
        
        document.getElementById('cost-metrics').innerHTML = ` + "`" + `
            <div class="cost-metric">
                <span class="metric-label">Running Instances:</span>
                <span class="metric-value">${runningCount}</span>
            </div>
            <div class="cost-metric">
                <span class="metric-label">Hourly Cost:</span>
                <span class="metric-value">$${totalHourlyCost.toFixed(2)}</span>
            </div>
            <div class="cost-metric">
                <span class="metric-label">Daily Cost:</span>
                <span class="metric-value">$${dailyCost.toFixed(2)}</span>
            </div>
            <div class="cost-metric">
                <span class="metric-label">Monthly Estimate:</span>
                <span class="metric-value">$${monthlyCost.toFixed(2)}</span>
            </div>
        ` + "`" + `;
    } catch (error) {
        console.error('Failed to load cost analysis:', error);
    }
}

// Instance actions
function connectSSH(instanceName) {
    alert('SSH connection command:\\n\\ncws connect ' + instanceName);
}

async function stopInstance(instanceId) {
    if (confirm('Are you sure you want to stop this instance?')) {
        alert('Stop command:\\n\\ncws stop ' + instanceId);
    }
}

async function startInstance(instanceId) {
    alert('Start command:\\n\\ncws start ' + instanceId);
}

// Add additional styles for new elements
const style = document.createElement('style');
style.textContent = ` + "`" + `
.templates-grid, .proxy-stats {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 20px;
}

.template-card, .proxy-card {
    background: white;
    padding: 20px;
    border-radius: 8px;
    box-shadow: 0 2px 4px rgba(0,0,0,0.1);
}

.template-ports {
    display: flex;
    gap: 8px;
    margin-top: 10px;
}

.port {
    background: #e2e8f0;
    padding: 2px 8px;
    border-radius: 4px;
    font-size: 0.85rem;
}

.cost-metric {
    display: flex;
    justify-content: space-between;
    padding: 10px;
    background: #f8fafc;
    margin: 5px 0;
    border-radius: 4px;
}

.metric-value {
    font-weight: bold;
    color: #2563eb;
}
` + "`" + `;
document.head.appendChild(style);
`