// Setup script to start the actual daemon for E2E testing
import { exec, spawn } from 'child_process'
import { promisify } from 'util'
import path from 'path'
import fs from 'fs'

const execAsync = promisify(exec)

// Function to check if daemon is running
async function isDaemonRunning() {
  try {
    const response = await fetch('http://localhost:8947/api/v1/ping')
    return response.ok
  } catch {
    return false
  }
}

// Function to start the daemon
async function startDaemon() {
  const daemonPath = path.join(process.cwd(), '..', '..', '..', 'bin', 'cwsd')
  
  // Check if daemon binary exists
  if (!fs.existsSync(daemonPath)) {
    console.error(`Daemon binary not found at ${daemonPath}`)
    console.log('Building daemon...')
    
    // Build the daemon
    const buildCmd = 'cd ../../.. && go build -o bin/cwsd ./cmd/cwsd'
    await execAsync(buildCmd)
    
    if (!fs.existsSync(daemonPath)) {
      throw new Error('Failed to build daemon')
    }
  }
  
  // Start daemon in background
  console.log('Starting CloudWorkstation daemon for testing...')
  const daemon = spawn(daemonPath, [], {
    detached: true,
    stdio: ['ignore', 'pipe', 'pipe'],
    env: {
      ...process.env,
      CLOUDWORKSTATION_TEST_MODE: 'true'
    }
  })
  
  // Log daemon output for debugging
  daemon.stdout.on('data', (data) => {
    console.log(`[Daemon] ${data.toString()}`)
  })
  
  daemon.stderr.on('data', (data) => {
    console.error(`[Daemon Error] ${data.toString()}`)
  })
  
  daemon.unref()
  
  // Wait for daemon to be ready
  let attempts = 0
  while (attempts < 30) {
    if (await isDaemonRunning()) {
      console.log('Daemon is ready!')
      return daemon.pid
    }
    await new Promise(resolve => setTimeout(resolve, 1000))
    attempts++
  }
  
  throw new Error('Daemon failed to start within 30 seconds')
}

// Function to stop the daemon
async function stopDaemon(pid) {
  if (pid) {
    try {
      process.kill(pid, 'SIGTERM')
      console.log(`Stopped daemon with PID ${pid}`)
    } catch (error) {
      console.error(`Failed to stop daemon: ${error.message}`)
    }
  }
}

export { startDaemon, stopDaemon, isDaemonRunning }