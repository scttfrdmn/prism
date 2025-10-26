// Unit tests for instance management functionality
import { describe, test, expect, vi, beforeEach } from 'vitest'
import { screen } from '@testing-library/dom'
import { mockInstances } from '../mocks/daemon-server.js'

// Mock instance management functions
function createInstanceCard(instance) {
  const card = document.createElement('div')
  card.className = 'instance-card'
  card.innerHTML = `
    <div class="instance-header">
      <div class="instance-name">${instance.name}</div>
      <div class="instance-status ${instance.state}">${instance.state}</div>
    </div>
    <div class="instance-details">
      ${instance.public_ip ? `<p><strong>IP:</strong> ${instance.public_ip}</p>` : ''}
      ${instance.hourly_rate ? `<p><strong>Cost:</strong> $${instance.hourly_rate.toFixed(4)}/hour</p>` : ''}
      ${instance.region ? `<p><strong>Region:</strong> ${instance.region}</p>` : ''}
    </div>
    <div class="instance-actions">
      <button class="btn-secondary connect-btn" onclick="connectToInstance('${instance.name}')">
        Connect
      </button>
      ${instance.state === 'running' ? 
        `<button class="btn-secondary stop-btn" onclick="stopInstance('${instance.name}')">Stop</button>` :
        `<button class="btn-secondary start-btn" onclick="startInstance('${instance.name}')">Start</button>`
      }
    </div>
  `
  return card
}

function renderInstances(instances) {
  const grid = document.getElementById('instances-grid')
  if (!grid) return
  
  if (instances.length === 0) {
    grid.innerHTML = `
      <div class="instance-card">
        <div class="text-center">
          <p>No instances running</p>
          <small>Launch your first research environment in Quick Start</small>
        </div>
      </div>
    `
    return
  }
  
  grid.innerHTML = ''
  instances.forEach(instance => {
    const card = createInstanceCard(instance)
    grid.appendChild(card)
  })
}

function connectToInstance(name) {
  const instance = mockInstances.find(i => i.name === name)
  if (!instance) {
    throw new Error(`Instance '${name}' not found`)
  }
  
  // Mock showing connection info
  const modal = document.createElement('div')
  modal.className = 'connection-modal'
  modal.innerHTML = `
    <div class="modal-content">
      <h3>Connection Info for ${name}</h3>
      <p>SSH: ssh ec2-user@${instance.public_ip}</p>
      <button onclick="this.parentElement.parentElement.remove()">Close</button>
    </div>
  `
  document.body.appendChild(modal)
  
  return {
    ssh: `ssh ec2-user@${instance.public_ip}`,
    jupyter: `http://${instance.public_ip}:8888`
  }
}

function stopInstance(name) {
  const instance = mockInstances.find(i => i.name === name)
  if (!instance) {
    throw new Error(`Instance '${name}' not found`)
  }
  
  if (instance.state !== 'running') {
    throw new Error(`Instance '${name}' is not running`)
  }
  
  // Update mock data
  instance.state = 'stopping'
  
  // Re-render instances
  renderInstances(mockInstances)
  
  return {
    name,
    previous_state: 'running',
    new_state: 'stopping'
  }
}

function startInstance(name) {
  const instance = mockInstances.find(i => i.name === name)
  if (!instance) {
    throw new Error(`Instance '${name}' not found`)
  }
  
  if (instance.state !== 'stopped') {
    throw new Error(`Instance '${name}' is not stopped`)
  }
  
  // Update mock data
  instance.state = 'starting'
  
  // Re-render instances
  renderInstances(mockInstances)
  
  return {
    name,
    previous_state: 'stopped',
    new_state: 'starting'
  }
}

function setupDOM() {
  document.body.innerHTML = `
    <div id="app">
      <div id="instances-grid"></div>
    </div>
  `
}

describe('Instance Management', () => {
  beforeEach(() => {
    setupDOM()
    // Reset mock instances to original state
    mockInstances[0].state = 'running'
    mockInstances[1].state = 'stopped'
  })

  test('renders instances correctly', () => {
    renderInstances(mockInstances)
    
    // Check that all instances are rendered
    expect(document.querySelectorAll('.instance-card')).toHaveLength(mockInstances.length)
    
    // Check specific instance content
    expect(screen.getByText('ml-research-workstation')).toBeInTheDocument()
    expect(screen.getByText('data-analysis-r')).toBeInTheDocument()
  })

  test('displays instance metadata correctly', () => {
    renderInstances(mockInstances)
    
    // Check instance names
    expect(screen.getByText('ml-research-workstation')).toBeInTheDocument()
    expect(screen.getByText('data-analysis-r')).toBeInTheDocument()
    
    // Check instance states
    expect(screen.getByText('running')).toBeInTheDocument()
    expect(screen.getByText('stopped')).toBeInTheDocument()
    
    // Check IPs are displayed
    expect(screen.getByText('54.123.45.67')).toBeInTheDocument()
    expect(screen.getByText('54.123.45.68')).toBeInTheDocument()
    
    // Check costs are displayed
    expect(screen.getByText('$0.0416/hour')).toBeInTheDocument()
    expect(screen.getByText('$0.0832/hour')).toBeInTheDocument()
  })

  test('shows correct action buttons based on state', () => {
    renderInstances(mockInstances)
    
    const cards = document.querySelectorAll('.instance-card')
    
    // Running instance should have Stop button
    const runningCard = Array.from(cards).find(card => 
      card.querySelector('.instance-status').textContent === 'running'
    )
    expect(runningCard.querySelector('.stop-btn')).toBeInTheDocument()
    expect(runningCard.querySelector('.stop-btn').textContent.trim()).toBe('Stop')
    
    // Stopped instance should have Start button  
    const stoppedCard = Array.from(cards).find(card =>
      card.querySelector('.instance-status').textContent === 'stopped'
    )
    expect(stoppedCard.querySelector('.start-btn')).toBeInTheDocument()
    expect(stoppedCard.querySelector('.start-btn').textContent.trim()).toBe('Start')
    
    // Both should have Connect button
    cards.forEach(card => {
      expect(card.querySelector('.connect-btn')).toBeInTheDocument()
      expect(card.querySelector('.connect-btn').textContent.trim()).toBe('Connect')
    })
  })

  test('handles instance connection', () => {
    renderInstances(mockInstances)
    
    const connectionInfo = connectToInstance('ml-research-workstation')
    
    expect(connectionInfo).toEqual({
      ssh: 'ssh ec2-user@54.123.45.67',
      jupyter: 'http://54.123.45.67:8888'
    })
    
    // Check that connection modal is shown
    expect(document.querySelector('.connection-modal')).toBeInTheDocument()
    expect(screen.getByText('Connection Info for ml-research-workstation')).toBeInTheDocument()
  })

  test('handles stopping running instance', () => {
    renderInstances(mockInstances)
    
    const result = stopInstance('ml-research-workstation')
    
    expect(result).toEqual({
      name: 'ml-research-workstation',
      previous_state: 'running',
      new_state: 'stopping'
    })
    
    // Check that instance state is updated in UI
    expect(screen.getByText('stopping')).toBeInTheDocument()
  })

  test('handles starting stopped instance', () => {
    renderInstances(mockInstances)
    
    const result = startInstance('data-analysis-r')
    
    expect(result).toEqual({
      name: 'data-analysis-r',
      previous_state: 'stopped', 
      new_state: 'starting'
    })
    
    // Check that instance state is updated in UI
    expect(screen.getByText('starting')).toBeInTheDocument()
  })

  test('handles errors for non-existent instances', () => {
    renderInstances(mockInstances)
    
    expect(() => connectToInstance('non-existent')).toThrow("Instance 'non-existent' not found")
    expect(() => stopInstance('non-existent')).toThrow("Instance 'non-existent' not found")
    expect(() => startInstance('non-existent')).toThrow("Instance 'non-existent' not found")
  })

  test('handles invalid state transitions', () => {
    renderInstances(mockInstances)
    
    // Try to stop an already stopped instance
    expect(() => stopInstance('data-analysis-r')).toThrow("Instance 'data-analysis-r' is not running")
    
    // Try to start an already running instance
    expect(() => startInstance('ml-research-workstation')).toThrow("Instance 'ml-research-workstation' is not stopped")
  })

  test('handles empty instance list', () => {
    renderInstances([])
    
    expect(screen.getByText('No instances running')).toBeInTheDocument()
    expect(screen.getByText('Launch your first research environment in Quick Start')).toBeInTheDocument()
  })

  test('instance cards have correct CSS classes and structure', () => {
    renderInstances(mockInstances)
    
    const cards = document.querySelectorAll('.instance-card')
    cards.forEach(card => {
      expect(card).toHaveClass('instance-card')
      expect(card.querySelector('.instance-header')).toBeInTheDocument()
      expect(card.querySelector('.instance-name')).toBeInTheDocument()
      expect(card.querySelector('.instance-status')).toBeInTheDocument()
      expect(card.querySelector('.instance-details')).toBeInTheDocument()
      expect(card.querySelector('.instance-actions')).toBeInTheDocument()
    })
  })

  test('instance status has correct CSS classes', () => {
    renderInstances(mockInstances)
    
    const runningStatus = document.querySelector('.instance-status.running')
    const stoppedStatus = document.querySelector('.instance-status.stopped')
    
    expect(runningStatus).toBeInTheDocument()
    expect(runningStatus).toHaveClass('instance-status', 'running')
    
    expect(stoppedStatus).toBeInTheDocument()
    expect(stoppedStatus).toHaveClass('instance-status', 'stopped')
  })
})