// Simple mock daemon server for testing
const http = require('http');
const url = require('url');

const PORT = 3001;

// Mock data
const mockTemplates = [
  {
    name: 'Python Machine Learning (Simplified)',
    description: 'Conda + Jupyter + ML packages',
    category: 'Machine Learning',
    icon: '🐍'
  },
  {
    name: 'R Research Environment (Simplified)', 
    description: 'Conda + RStudio + tidyverse packages',
    category: 'Data Science',
    icon: '📊'
  }
];

const mockInstances = [
  {
    name: 'ml-research-workstation',
    state: 'running',
    ip: '54.123.45.67',
    hourly_rate: 0.0416,
    region: 'us-west-2',
    template: 'Python Machine Learning (Simplified)'
  },
  {
    name: 'data-analysis-r',
    state: 'stopped',
    ip: '54.123.45.68',
    hourly_rate: 0.0832,
    region: 'us-west-2',
    template: 'R Research Environment (Simplified)'
  }
];

// Create server
const server = http.createServer((req, res) => {
  const parsedUrl = url.parse(req.url, true);
  const path = parsedUrl.pathname;
  
  // Set CORS headers
  res.setHeader('Access-Control-Allow-Origin', '*');
  res.setHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS');
  res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
  res.setHeader('Content-Type', 'application/json');
  
  // Handle OPTIONS
  if (req.method === 'OPTIONS') {
    res.writeHead(200);
    res.end();
    return;
  }
  
  console.log(`${req.method} ${path}`);
  
  // Routes
  if (path === '/api/v1/ping') {
    res.writeHead(200);
    res.end(JSON.stringify({ status: 'ok' }));
  } else if (path === '/api/v1/templates') {
    res.writeHead(200);
    res.end(JSON.stringify(mockTemplates));
  } else if (path === '/api/v1/instances') {
    res.writeHead(200);
    res.end(JSON.stringify(mockInstances));
  } else if (path === '/api/v1/status') {
    res.writeHead(200);
    res.end(JSON.stringify({ 
      version: '0.4.5',
      status: 'running',
      start_time: new Date().toISOString()
    }));
  } else if (path.startsWith('/api/v1/instances/') && path.endsWith('/stop')) {
    res.writeHead(200);
    res.end(JSON.stringify({ success: true }));
  } else if (path.startsWith('/api/v1/instances/') && path.endsWith('/start')) {
    res.writeHead(200);
    res.end(JSON.stringify({ success: true }));
  } else {
    res.writeHead(404);
    res.end(JSON.stringify({ error: 'Not found' }));
  }
});

server.listen(PORT, () => {
  console.log(`Mock daemon server running on http://localhost:${PORT}`);
});