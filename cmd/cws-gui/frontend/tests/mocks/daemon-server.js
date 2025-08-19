// Mock Service Worker for CloudWorkstation Daemon API
import { http, HttpResponse } from 'msw'
import { setupServer } from 'msw/node'

// Mock data
const mockTemplates = [
  {
    name: 'Python Machine Learning (Simplified)',
    description: 'Conda + Jupyter + ML packages (scikit-learn, pandas, matplotlib)',
    category: 'Machine Learning',
    icon: '🐍',
    validation_status: 'validated'
  },
  {
    name: 'R Research Environment (Simplified)', 
    description: 'Conda + RStudio + tidyverse packages for statistical analysis',
    category: 'Data Science',
    icon: '📊',
    validation_status: 'validated'
  },
  {
    name: 'Rocky Linux 9 Base',
    description: 'Rocky Linux 9 + DNF + system tools + rocky user',
    category: 'Base Systems',
    icon: '🖥️',
    validation_status: 'validated'
  }
]

const mockInstances = [
  {
    name: 'ml-research-workstation',
    id: 'i-1234567890abcdef0',
    state: 'running',
    public_ip: '54.123.45.67',
    instance_type: 't3.medium',
    template: 'Python Machine Learning (Simplified)',
    hourly_rate: 0.0416,
    current_spend: 2.45,
    region: 'us-west-2',
    launch_time: '2024-06-15T10:30:00Z'
  },
  {
    name: 'data-analysis-r',
    id: 'i-0987654321fedcba0', 
    state: 'stopped',
    public_ip: '54.123.45.68',
    instance_type: 't3.large',
    template: 'R Research Environment (Simplified)',
    hourly_rate: 0.0832,
    current_spend: 1.25,
    region: 'us-west-2',
    launch_time: '2024-06-14T15:20:00Z'
  }
]

// API handlers
const handlers = [
  // Templates API
  http.get('http://localhost:8947/api/v1/templates', () => {
    return HttpResponse.json(mockTemplates)
  }),
  
  http.get('http://localhost:8947/api/v1/templates/validate', () => {
    return HttpResponse.json({
      total_templates: mockTemplates.length,
      validated: mockTemplates.length,
      errors: [],
      validation_results: mockTemplates.map(template => ({
        template: template.name,
        status: 'valid',
        checks_passed: 8,
        issues: []
      }))
    })
  }),
  
  // Instances API
  http.get('http://localhost:8947/api/v1/instances', () => {
    return HttpResponse.json(mockInstances)
  }),
  
  http.post('http://localhost:8947/api/v1/instances/launch', async ({ request }) => {
    const body = await request.json()
    const newInstance = {
      name: body.name,
      instance_id: 'i-' + Math.random().toString(36).substring(2, 17),
      state: 'launching',
      template: body.template,
      estimated_ready_time: new Date(Date.now() + 5 * 60000).toISOString(),
      hourly_rate: body.size === 'S' ? 0.0208 : body.size === 'M' ? 0.0416 : body.size === 'L' ? 0.0832 : 0.1664,
      launch_progress: 15
    }
    
    // Add to mock instances list
    mockInstances.push({
      ...newInstance,
      id: newInstance.instance_id,
      state: 'running',
      public_ip: `54.123.45.${Math.floor(Math.random() * 255)}`,
      instance_type: body.size === 'S' ? 't3.small' : body.size === 'M' ? 't3.medium' : body.size === 'L' ? 't3.large' : 't3.xlarge',
      current_spend: 0,
      region: 'us-west-2',
      launch_time: new Date().toISOString()
    })
    
    return HttpResponse.json(newInstance)
  }),
  
  http.post('http://localhost:8947/api/v1/instances/:name/stop', ({ params }) => {
    const instanceName = params.name
    const instance = mockInstances.find(i => i.name === instanceName)
    
    if (!instance) {
      return HttpResponse.json(
        { error: { code: 'INSTANCE_NOT_FOUND', message: `Instance '${instanceName}' not found` } },
        { status: 404 }
      )
    }
    
    instance.state = 'stopping'
    
    return HttpResponse.json({
      name: instanceName,
      previous_state: 'running',
      new_state: 'stopping',
      message: 'Instance stopping - all data preserved'
    })
  }),
  
  http.post('http://localhost:8947/api/v1/instances/:name/start', ({ params }) => {
    const instanceName = params.name
    const instance = mockInstances.find(i => i.name === instanceName)
    
    if (!instance) {
      return HttpResponse.json(
        { error: { code: 'INSTANCE_NOT_FOUND', message: `Instance '${instanceName}' not found` } },
        { status: 404 }
      )
    }
    
    instance.state = 'starting'
    
    return HttpResponse.json({
      name: instanceName,
      previous_state: 'stopped',
      new_state: 'starting',
      estimated_ready_time: new Date(Date.now() + 2 * 60000).toISOString()
    })
  }),
  
  http.get('http://localhost:8947/api/v1/instances/:name/connect', ({ params }) => {
    const instanceName = params.name
    const instance = mockInstances.find(i => i.name === instanceName)
    
    if (!instance) {
      return HttpResponse.json(
        { error: { code: 'INSTANCE_NOT_FOUND', message: `Instance '${instanceName}' not found` } },
        { status: 404 }
      )
    }
    
    return HttpResponse.json({
      ssh: {
        command: `ssh -i ~/.ssh/cloudworkstation.pem ec2-user@${instance.public_ip}`,
        host: instance.public_ip,
        user: 'ec2-user',
        port: 22
      },
      services: {
        jupyter: {
          url: `http://${instance.public_ip}:8888`,
          token: 'mock_token_' + Math.random().toString(36).substring(2, 20),
          local_forward: `ssh -L 8888:localhost:8888 -i ~/.ssh/cloudworkstation.pem ec2-user@${instance.public_ip}`
        }
      }
    })
  }),
  
  // Daemon status
  http.get('http://localhost:8947/api/v1/daemon/status', () => {
    return HttpResponse.json({
      status: 'healthy',
      version: '0.4.3',
      uptime: '2h 45m 30s',
      api_version: 'v1',
      aws_connectivity: 'connected',
      active_profiles: 1,
      total_instances: mockInstances.length,
      active_instances: mockInstances.filter(i => i.state === 'running').length
    })
  }),
  
  // Error simulation handlers
  http.get('http://localhost:8947/api/v1/templates/error', () => {
    return HttpResponse.json(
      { error: { code: 'DAEMON_ERROR', message: 'Simulated daemon error for testing' } },
      { status: 500 }
    )
  })
]

// Create and export the server
export const server = setupServer(...handlers)

// Export mock data for use in tests
export { mockTemplates, mockInstances }