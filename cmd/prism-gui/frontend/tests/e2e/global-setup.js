// Global setup for Playwright tests
import { startDaemon, stopDaemon, isDaemonRunning } from './setup-daemon.js'
import { exec } from 'child_process'
import { promisify } from 'util'

const execAsync = promisify(exec)

let daemonPid

async function globalSetup() {
  // ALWAYS kill existing daemons to ensure clean test environment
  // This prevents tests from connecting to a production daemon without PRISM_TEST_MODE
  try {
    console.log('Killing any existing daemon processes...')
    await execAsync('pkill -9 prismd || pkill -9 cwsd || true')
    // Wait for processes to fully terminate
    await new Promise(resolve => setTimeout(resolve, 1000))
  } catch (error) {
    // Ignore errors - process might not exist
  }

  // Start a fresh daemon with test mode enabled
  console.log('Starting daemon for tests with PRISM_TEST_MODE...')
  daemonPid = await startDaemon()
  
  // Return teardown function
  return async () => {
    if (daemonPid) {
      console.log('Stopping daemon after tests...')
      await stopDaemon(daemonPid)
    }
  }
}

export default globalSetup