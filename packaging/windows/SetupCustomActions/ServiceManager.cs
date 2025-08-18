using System;
using System.Management;
using System.ServiceProcess;
using System.Threading;
using Microsoft.Deployment.WindowsInstaller;

namespace SetupCustomActions
{
    /// <summary>
    /// Manages Windows service configuration for CloudWorkstation
    /// </summary>
    public class ServiceManager
    {
        private readonly Session _session;
        
        public ServiceManager(Session session)
        {
            _session = session;
        }

        /// <summary>
        /// Configures service recovery options
        /// </summary>
        public bool ConfigureServiceRecovery(string serviceName)
        {
            try
            {
                _session.Log($"Configuring recovery options for service: {serviceName}");

                // Use WMI to configure service recovery
                using (var searcher = new ManagementObjectSearcher($"SELECT * FROM Win32_Service WHERE Name='{serviceName}'"))
                {
                    foreach (ManagementObject service in searcher.Get())
                    {
                        using (service)
                        {
                            // Set recovery actions: restart service on first 3 failures
                            var recoveryParams = new object[]
                            {
                                30, // Reset failure count after 30 seconds
                                "", // Reboot message (empty)
                                "", // Command to run (empty)
                                3,  // Number of actions
                                new int[] { 1, 5000, 1, 5000, 1, 5000 } // Actions: 1=restart, delay in ms
                            };

                            try
                            {
                                service.InvokeMethod("SetServiceRecoveryOptions", recoveryParams);
                                _session.Log($"Recovery options configured for {serviceName}");
                                return true;
                            }
                            catch (Exception ex)
                            {
                                _session.Log($"Failed to set recovery options via WMI: {ex.Message}");
                                
                                // Fallback to sc.exe command
                                return ConfigureServiceRecoveryFallback(serviceName);
                            }
                        }
                    }
                }

                _session.Log($"Service {serviceName} not found");
                return false;
            }
            catch (Exception ex)
            {
                _session.Log($"Error configuring service recovery: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Fallback method to configure service recovery using sc.exe
        /// </summary>
        private bool ConfigureServiceRecoveryFallback(string serviceName)
        {
            try
            {
                _session.Log($"Using sc.exe fallback for service recovery configuration");

                var startInfo = new System.Diagnostics.ProcessStartInfo
                {
                    FileName = "sc.exe",
                    Arguments = $"failure {serviceName} reset= 30 actions= restart/5000/restart/5000/restart/5000",
                    UseShellExecute = false,
                    RedirectStandardOutput = true,
                    RedirectStandardError = true,
                    CreateNoWindow = true
                };

                using (var process = System.Diagnostics.Process.Start(startInfo))
                {
                    process.WaitForExit(10000); // 10 second timeout

                    if (process.ExitCode == 0)
                    {
                        _session.Log($"Service recovery configured successfully using sc.exe");
                        return true;
                    }
                    else
                    {
                        string output = process.StandardOutput.ReadToEnd();
                        string error = process.StandardError.ReadToEnd();
                        _session.Log($"sc.exe failed with exit code {process.ExitCode}");
                        _session.Log($"Output: {output}");
                        _session.Log($"Error: {error}");
                        return false;
                    }
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error in sc.exe fallback: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Sets detailed service description
        /// </summary>
        public bool SetServiceDescription(string serviceName, string description)
        {
            try
            {
                _session.Log($"Setting description for service: {serviceName}");

                using (var searcher = new ManagementObjectSearcher($"SELECT * FROM Win32_Service WHERE Name='{serviceName}'"))
                {
                    foreach (ManagementObject service in searcher.Get())
                    {
                        using (service)
                        {
                            service["Description"] = description;
                            service.Put();
                            
                            _session.Log($"Service description set successfully");
                            return true;
                        }
                    }
                }

                _session.Log($"Service {serviceName} not found for description update");
                return false;
            }
            catch (Exception ex)
            {
                _session.Log($"Error setting service description: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Verifies service configuration
        /// </summary>
        public bool VerifyServiceConfiguration(string serviceName)
        {
            try
            {
                _session.Log($"Verifying configuration for service: {serviceName}");

                using (var service = new ServiceController(serviceName))
                {
                    _session.Log($"Service Name: {service.ServiceName}");
                    _session.Log($"Display Name: {service.DisplayName}");
                    _session.Log($"Status: {service.Status}");
                    _session.Log($"Startup Type: {service.StartType}");

                    // Verify automatic startup
                    if (service.StartType != ServiceStartMode.Automatic)
                    {
                        _session.Log($"Warning: Service startup type is {service.StartType}, expected Automatic");
                        return false;
                    }

                    _session.Log("Service configuration verification passed");
                    return true;
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error verifying service configuration: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Checks if service is running
        /// </summary>
        public bool IsServiceRunning(string serviceName)
        {
            try
            {
                using (var service = new ServiceController(serviceName))
                {
                    return service.Status == ServiceControllerStatus.Running;
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error checking service status: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Starts a Windows service
        /// </summary>
        public bool StartService(string serviceName)
        {
            try
            {
                _session.Log($"Starting service: {serviceName}");

                using (var service = new ServiceController(serviceName))
                {
                    if (service.Status == ServiceControllerStatus.Running)
                    {
                        _session.Log($"Service {serviceName} is already running");
                        return true;
                    }

                    if (service.Status == ServiceControllerStatus.Stopped)
                    {
                        service.Start();
                        
                        // Wait for service to start (with timeout)
                        service.WaitForStatus(ServiceControllerStatus.Running, TimeSpan.FromSeconds(30));
                        
                        if (service.Status == ServiceControllerStatus.Running)
                        {
                            _session.Log($"Service {serviceName} started successfully");
                            return true;
                        }
                        else
                        {
                            _session.Log($"Service {serviceName} failed to start (status: {service.Status})");
                            return false;
                        }
                    }
                    else
                    {
                        _session.Log($"Service {serviceName} is in unexpected state: {service.Status}");
                        return false;
                    }
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error starting service: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Tests connectivity to the daemon
        /// </summary>
        public bool TestDaemonConnectivity()
        {
            try
            {
                _session.Log("Testing daemon connectivity...");

                // Test HTTP connectivity to daemon port 8947
                using (var client = new System.Net.WebClient())
                {
                    client.Timeout = 5000; // 5 second timeout
                    
                    try
                    {
                        string response = client.DownloadString("http://localhost:8947/api/v1/health");
                        _session.Log($"Daemon health check response: {response}");
                        return true;
                    }
                    catch (Exception ex)
                    {
                        _session.Log($"Daemon connectivity test failed: {ex.Message}");
                        
                        // Try alternative connectivity test - check if port is listening
                        return TestPortConnectivity(8947);
                    }
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error testing daemon connectivity: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Tests if a specific port is listening
        /// </summary>
        private bool TestPortConnectivity(int port)
        {
            try
            {
                using (var tcpClient = new System.Net.Sockets.TcpClient())
                {
                    var result = tcpClient.BeginConnect("127.0.0.1", port, null, null);
                    var success = result.AsyncWaitHandle.WaitOne(TimeSpan.FromSeconds(3));
                    
                    if (success)
                    {
                        tcpClient.EndConnect(result);
                        _session.Log($"Port {port} is listening");
                        return true;
                    }
                    else
                    {
                        _session.Log($"Port {port} is not listening");
                        return false;
                    }
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error testing port connectivity: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Configures service for optimal performance
        /// </summary>
        public bool OptimizeServiceConfiguration(string serviceName)
        {
            try
            {
                _session.Log($"Optimizing configuration for service: {serviceName}");

                // Set service priority to normal (not high) for system stability
                // Set service to interact with desktop if needed
                // Configure service timeout values

                using (var searcher = new ManagementObjectSearcher($"SELECT * FROM Win32_Service WHERE Name='{serviceName}'"))
                {
                    foreach (ManagementObject service in searcher.Get())
                    {
                        using (service)
                        {
                            // Configure service parameters for optimal performance
                            try
                            {
                                // Set process priority to normal
                                var processId = service["ProcessId"];
                                if (processId != null && (uint)processId != 0)
                                {
                                    var process = System.Diagnostics.Process.GetProcessById((int)(uint)processId);
                                    process.PriorityClass = System.Diagnostics.ProcessPriorityClass.Normal;
                                    _session.Log("Service process priority set to Normal");
                                }
                            }
                            catch (Exception ex)
                            {
                                _session.Log($"Could not set process priority: {ex.Message}");
                            }

                            _session.Log("Service optimization completed");
                            return true;
                        }
                    }
                }

                return false;
            }
            catch (Exception ex)
            {
                _session.Log($"Error optimizing service configuration: {ex.Message}");
                return false;
            }
        }
    }
}