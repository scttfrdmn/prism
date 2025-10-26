// Global setup for Playwright tests
import { startDaemon, stopDaemon, isDaemonRunning } from './setup-daemon.js'

let daemonPid

async function globalSetup() {
  // Check if daemon is already running
  if (await isDaemonRunning()) {
    console.log('Daemon is already running, skipping startup')
    return
  }
  
  // Start the daemon for testing
  console.log('Starting daemon for tests...')
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