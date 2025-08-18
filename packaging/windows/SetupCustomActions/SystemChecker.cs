using System;
using System.IO;
using System.Security.Principal;
using Microsoft.Deployment.WindowsInstaller;
using Microsoft.Win32;

namespace SetupCustomActions
{
    /// <summary>
    /// Performs system requirement checks for CloudWorkstation installation
    /// </summary>
    public class SystemChecker
    {
        private readonly Session _session;
        
        public SystemChecker(Session session)
        {
            _session = session;
        }

        /// <summary>
        /// Checks if the Windows version meets minimum requirements
        /// </summary>
        public bool CheckWindowsVersion()
        {
            try
            {
                _session.Log("Checking Windows version...");
                
                var osVersion = Environment.OSVersion;
                _session.Log($"Current OS: {osVersion.Platform} {osVersion.Version}");
                
                // Windows 10 version 1903 (build 18362) or later
                // Windows 11 (build 22000) or later
                if (osVersion.Platform == PlatformID.Win32NT)
                {
                    var version = osVersion.Version;
                    
                    // Windows 10/11 check
                    if (version.Major >= 10)
                    {
                        // For Windows 10, check build number
                        if (version.Major == 10 && version.Build < 18362)
                        {
                            _session.Log($"Windows 10 build {version.Build} is below minimum requirement (18362)");
                            return false;
                        }
                        
                        _session.Log($"Windows version check passed: {version}");
                        return true;
                    }
                    else
                    {
                        _session.Log($"Windows version {version.Major}.{version.Minor} is below minimum requirement (Windows 10)");
                        return false;
                    }
                }
                else
                {
                    _session.Log($"Unsupported platform: {osVersion.Platform}");
                    return false;
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error checking Windows version: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Checks available disk space for installation
        /// </summary>
        public bool CheckDiskSpace()
        {
            try
            {
                _session.Log("Checking available disk space...");
                
                string installDir = _session["INSTALLFOLDER"];
                if (string.IsNullOrEmpty(installDir))
                {
                    installDir = @"C:\Program Files\CloudWorkstation";
                }
                
                // Get drive info for installation directory
                string driveLetter = Path.GetPathRoot(installDir);
                var driveInfo = new DriveInfo(driveLetter);
                
                long availableSpace = driveInfo.AvailableFreeSpace;
                long requiredSpace = 100 * 1024 * 1024; // 100 MB minimum
                
                _session.Log($"Available space on {driveLetter}: {availableSpace / 1024 / 1024} MB");
                _session.Log($"Required space: {requiredSpace / 1024 / 1024} MB");
                
                if (availableSpace < requiredSpace)
                {
                    _session.Log("Insufficient disk space for installation");
                    return false;
                }
                
                _session.Log("Disk space check passed");
                return true;
            }
            catch (Exception ex)
            {
                _session.Log($"Error checking disk space: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Checks for .NET Framework availability
        /// </summary>
        public bool CheckDotNetFramework()
        {
            try
            {
                _session.Log("Checking .NET Framework...");
                
                // Check for .NET Framework 4.8 or later
                using (var key = Registry.LocalMachine.OpenSubKey(@"SOFTWARE\Microsoft\NET Framework Setup\NDP\v4\Full\"))
                {
                    if (key != null)
                    {
                        var release = key.GetValue("Release");
                        if (release != null && int.TryParse(release.ToString(), out int releaseNumber))
                        {
                            _session.Log($".NET Framework release number: {releaseNumber}");
                            
                            // .NET Framework 4.8 = 528040 or later
                            if (releaseNumber >= 528040)
                            {
                                _session.Log(".NET Framework 4.8+ detected");
                                return true;
                            }
                            else
                            {
                                _session.Log($".NET Framework version is below 4.8 (release {releaseNumber})");
                                return false;
                            }
                        }
                    }
                }
                
                _session.Log(".NET Framework 4.8+ not found");
                return false;
            }
            catch (Exception ex)
            {
                _session.Log($"Error checking .NET Framework: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Checks if running with administrator privileges
        /// </summary>
        public bool CheckAdministratorPrivileges()
        {
            try
            {
                _session.Log("Checking administrator privileges...");
                
                using (var identity = WindowsIdentity.GetCurrent())
                {
                    var principal = new WindowsPrincipal(identity);
                    bool isAdmin = principal.IsInRole(WindowsBuiltInRole.Administrator);
                    
                    _session.Log($"Running as administrator: {isAdmin}");
                    
                    if (!isAdmin)
                    {
                        _session.Log("Administrator privileges required for CloudWorkstation installation");
                        return false;
                    }
                    
                    return true;
                }
            }
            catch (Exception ex)
            {
                _session.Log($"Error checking administrator privileges: {ex.Message}");
                return false;
            }
        }

        /// <summary>
        /// Checks for conflicting software that might interfere with CloudWorkstation
        /// </summary>
        public bool CheckConflictingSoftware()
        {
            try
            {
                _session.Log("Checking for conflicting software...");
                
                var conflicts = new[]
                {
                    // Other services using port 8947
                    new { Name = "Other CloudWorkstation", Process = "cwsd", Port = 8947 },
                    
                    // AWS CLI conflicts (informational)
                    new { Name = "AWS CLI v1", Process = "aws", Port = 0 }
                };

                bool hasConflicts = false;

                foreach (var conflict in conflicts)
                {
                    if (IsProcessRunning(conflict.Process))
                    {
                        _session.Log($"Potential conflict detected: {conflict.Name} (process: {conflict.Process})");
                        hasConflicts = true;
                    }

                    if (conflict.Port > 0 && IsPortInUse(conflict.Port))
                    {
                        _session.Log($"Port conflict detected: {conflict.Name} may be using port {conflict.Port}");
                        hasConflicts = true;
                    }
                }

                if (!hasConflicts)
                {
                    _session.Log("No software conflicts detected");
                }

                // Return true even if conflicts found (just warnings)
                return true;
            }
            catch (Exception ex)
            {
                _session.Log($"Error checking conflicting software: {ex.Message}");
                return true;
            }
        }

        /// <summary>
        /// Checks if a process with the given name is running
        /// </summary>
        private bool IsProcessRunning(string processName)
        {
            try
            {
                var processes = System.Diagnostics.Process.GetProcessesByName(processName);
                return processes.Length > 0;
            }
            catch
            {
                return false;
            }
        }

        /// <summary>
        /// Checks if a port is in use
        /// </summary>
        private bool IsPortInUse(int port)
        {
            try
            {
                using (var tcpListener = new System.Net.Sockets.TcpListener(System.Net.IPAddress.Any, port))
                {
                    tcpListener.Start();
                    tcpListener.Stop();
                    return false; // Port is available
                }
            }
            catch
            {
                return true; // Port is in use
            }
        }

        /// <summary>
        /// Performs additional system checks
        /// </summary>
        public bool CheckAdditionalRequirements()
        {
            try
            {
                _session.Log("Performing additional system checks...");

                // Check if Windows Management Instrumentation service is running
                if (!IsServiceRunning("Winmgmt"))
                {
                    _session.Log("Warning: Windows Management Instrumentation service is not running");
                }

                // Check if TCP/IP protocol is available
                if (!IsServiceRunning("Tcpip"))
                {
                    _session.Log("Warning: TCP/IP protocol service is not running");
                }

                // Check available memory
                var memoryInfo = new Microsoft.VisualBasic.Devices.ComputerInfo();
                ulong availableMemory = memoryInfo.AvailablePhysicalMemory;
                ulong requiredMemory = 512 * 1024 * 1024; // 512 MB minimum

                _session.Log($"Available memory: {availableMemory / 1024 / 1024} MB");
                
                if (availableMemory < requiredMemory)
                {
                    _session.Log("Warning: Low available memory");
                }

                _session.Log("Additional system checks completed");
                return true;
            }
            catch (Exception ex)
            {
                _session.Log($"Error in additional system checks: {ex.Message}");
                return true;
            }
        }

        /// <summary>
        /// Checks if a Windows service is running
        /// </summary>
        private bool IsServiceRunning(string serviceName)
        {
            try
            {
                using (var service = new System.ServiceProcess.ServiceController(serviceName))
                {
                    return service.Status == System.ServiceProcess.ServiceControllerStatus.Running;
                }
            }
            catch
            {
                return false;
            }
        }
    }
}