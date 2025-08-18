using System;
using Microsoft.Deployment.WindowsInstaller;

namespace SetupCustomActions
{
    /// <summary>
    /// Main entry point for CloudWorkstation installer custom actions
    /// </summary>
    public class CustomActions
    {
        /// <summary>
        /// Checks system requirements before installation
        /// </summary>
        [CustomAction]
        public static ActionResult CheckSystemRequirements(Session session)
        {
            session.Log("Begin CheckSystemRequirements");

            try
            {
                var checker = new SystemChecker(session);
                
                // Check Windows version
                if (!checker.CheckWindowsVersion())
                {
                    session.Log("Windows version check failed");
                    return ActionResult.Failure;
                }
                
                // Check available disk space
                if (!checker.CheckDiskSpace())
                {
                    session.Log("Disk space check failed");
                    return ActionResult.Failure;
                }
                
                // Check for required .NET Framework
                if (!checker.CheckDotNetFramework())
                {
                    session.Log("Warning: .NET Framework check failed - continuing with installation");
                    // Don't fail installation for .NET Framework, just log warning
                }
                
                // Check if running as administrator
                if (!checker.CheckAdministratorPrivileges())
                {
                    session.Log("Administrator privileges check failed");
                    return ActionResult.Failure;
                }
                
                // Check for conflicting software
                if (!checker.CheckConflictingSoftware())
                {
                    session.Log("Warning: Conflicting software detected - continuing with installation");
                    // Don't fail installation, just log warning
                }

                session.Log("System requirements check completed successfully");
                return ActionResult.Success;
            }
            catch (Exception ex)
            {
                session.Log($"Error in CheckSystemRequirements: {ex.Message}");
                return ActionResult.Failure;
            }
        }

        /// <summary>
        /// Configures Windows service after installation
        /// </summary>
        [CustomAction]
        public static ActionResult ConfigureWindowsService(Session session)
        {
            session.Log("Begin ConfigureWindowsService");

            try
            {
                var serviceManager = new ServiceManager(session);
                
                // Get installation directory from properties
                string installDir = session["INSTALLFOLDER"];
                if (string.IsNullOrEmpty(installDir))
                {
                    session.Log("Error: INSTALLFOLDER property not set");
                    return ActionResult.Failure;
                }

                string servicePath = System.IO.Path.Combine(installDir, "bin", "cwsd-service.exe");
                
                // Verify service executable exists
                if (!System.IO.File.Exists(servicePath))
                {
                    session.Log($"Error: Service executable not found at {servicePath}");
                    return ActionResult.Failure;
                }

                // Configure service recovery options
                if (!serviceManager.ConfigureServiceRecovery("CloudWorkstationDaemon"))
                {
                    session.Log("Warning: Failed to configure service recovery options");
                    // Continue with installation even if recovery configuration fails
                }

                // Set service description with more details
                if (!serviceManager.SetServiceDescription("CloudWorkstationDaemon", 
                    "Enterprise research management platform daemon for launching cloud research environments. " +
                    "This service manages AWS infrastructure, templates, and provides API access for CLI, TUI, and GUI clients."))
                {
                    session.Log("Warning: Failed to set detailed service description");
                    // Continue with installation
                }

                // Configure service to start automatically (already done by WiX, but verify)
                if (!serviceManager.VerifyServiceConfiguration("CloudWorkstationDaemon"))
                {
                    session.Log("Warning: Service configuration verification failed");
                    // Continue with installation
                }

                session.Log("Windows service configuration completed successfully");
                return ActionResult.Success;
            }
            catch (Exception ex)
            {
                session.Log($"Error in ConfigureWindowsService: {ex.Message}");
                // Don't fail installation for service configuration issues
                return ActionResult.Success;
            }
        }

        /// <summary>
        /// Verifies daemon startup after service installation
        /// </summary>
        [CustomAction]
        public static ActionResult VerifyDaemonStartup(Session session)
        {
            session.Log("Begin VerifyDaemonStartup");

            try
            {
                var serviceManager = new ServiceManager(session);
                
                // Give the service a moment to start
                System.Threading.Thread.Sleep(3000);
                
                // Check if service is running
                if (!serviceManager.IsServiceRunning("CloudWorkstationDaemon"))
                {
                    session.Log("Warning: CloudWorkstation service is not running");
                    
                    // Try to start the service
                    if (serviceManager.StartService("CloudWorkstationDaemon"))
                    {
                        session.Log("Successfully started CloudWorkstation service");
                        
                        // Wait a moment and verify it's still running
                        System.Threading.Thread.Sleep(2000);
                        
                        if (!serviceManager.IsServiceRunning("CloudWorkstationDaemon"))
                        {
                            session.Log("Warning: Service started but stopped immediately");
                        }
                        else
                        {
                            session.Log("Service startup verification successful");
                        }
                    }
                    else
                    {
                        session.Log("Warning: Failed to start CloudWorkstation service");
                    }
                }
                else
                {
                    session.Log("CloudWorkstation service is running successfully");
                }

                // Test daemon connectivity (optional)
                if (serviceManager.TestDaemonConnectivity())
                {
                    session.Log("Daemon connectivity test passed");
                }
                else
                {
                    session.Log("Warning: Daemon connectivity test failed");
                }

                return ActionResult.Success;
            }
            catch (Exception ex)
            {
                session.Log($"Error in VerifyDaemonStartup: {ex.Message}");
                // Don't fail installation for verification issues
                return ActionResult.Success;
            }
        }

        /// <summary>
        /// Launches first-run setup wizard
        /// </summary>
        [CustomAction]
        public static ActionResult LaunchFirstRunWizard(Session session)
        {
            session.Log("Begin LaunchFirstRunWizard");

            try
            {
                // Check if we should skip first-run setup (silent installation)
                string uiLevel = session["UILevel"];
                if (uiLevel == "2") // Silent installation
                {
                    session.Log("Silent installation detected, skipping first-run wizard");
                    return ActionResult.Success;
                }

                var firstRunSetup = new FirstRunSetup(session);
                
                // Get installation directory
                string installDir = session["INSTALLFOLDER"];
                if (string.IsNullOrEmpty(installDir))
                {
                    session.Log("Error: INSTALLFOLDER property not set");
                    return ActionResult.Failure;
                }

                // Launch first-run setup asynchronously (don't block installer)
                if (firstRunSetup.LaunchAsync(installDir))
                {
                    session.Log("First-run setup wizard launched successfully");
                }
                else
                {
                    session.Log("Warning: Failed to launch first-run setup wizard");
                }

                return ActionResult.Success;
            }
            catch (Exception ex)
            {
                session.Log($"Error in LaunchFirstRunWizard: {ex.Message}");
                // Don't fail installation for first-run setup issues
                return ActionResult.Success;
            }
        }

        /// <summary>
        /// Updates system PATH environment variable
        /// </summary>
        [CustomAction]
        public static ActionResult UpdateSystemPath(Session session)
        {
            session.Log("Begin UpdateSystemPath");

            try
            {
                // Get installation directory
                string installDir = session["INSTALLFOLDER"];
                if (string.IsNullOrEmpty(installDir))
                {
                    session.Log("Error: INSTALLFOLDER property not set");
                    return ActionResult.Failure;
                }

                string binPath = System.IO.Path.Combine(installDir, "bin");
                
                // Verify bin directory exists
                if (!System.IO.Directory.Exists(binPath))
                {
                    session.Log($"Warning: Bin directory not found at {binPath}");
                    return ActionResult.Success;
                }

                // Get current system PATH
                string currentPath = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.Machine);
                
                // Check if path is already in PATH
                if (currentPath != null && currentPath.Contains(binPath))
                {
                    session.Log("CloudWorkstation bin path already in system PATH");
                    return ActionResult.Success;
                }

                // Add to system PATH
                string newPath = currentPath + ";" + binPath;
                Environment.SetEnvironmentVariable("PATH", newPath, EnvironmentVariableTarget.Machine);
                
                session.Log($"Added {binPath} to system PATH");
                
                // Notify system of environment variable change
                NotifyEnvironmentChange();
                
                session.Log("System PATH updated successfully");
                return ActionResult.Success;
            }
            catch (Exception ex)
            {
                session.Log($"Error in UpdateSystemPath: {ex.Message}");
                // Don't fail installation for PATH update issues
                return ActionResult.Success;
            }
        }

        /// <summary>
        /// Removes CloudWorkstation from system PATH during uninstall
        /// </summary>
        [CustomAction]
        public static ActionResult RemoveFromSystemPath(Session session)
        {
            session.Log("Begin RemoveFromSystemPath");

            try
            {
                // Get installation directory
                string installDir = session["INSTALLFOLDER"];
                if (string.IsNullOrEmpty(installDir))
                {
                    session.Log("Warning: INSTALLFOLDER property not set during uninstall");
                    return ActionResult.Success;
                }

                string binPath = System.IO.Path.Combine(installDir, "bin");
                
                // Get current system PATH
                string currentPath = Environment.GetEnvironmentVariable("PATH", EnvironmentVariableTarget.Machine);
                
                if (currentPath != null && currentPath.Contains(binPath))
                {
                    // Remove from PATH
                    string newPath = currentPath.Replace(";" + binPath, "").Replace(binPath + ";", "").Replace(binPath, "");
                    Environment.SetEnvironmentVariable("PATH", newPath, EnvironmentVariableTarget.Machine);
                    
                    session.Log($"Removed {binPath} from system PATH");
                    
                    // Notify system of environment variable change
                    NotifyEnvironmentChange();
                }
                else
                {
                    session.Log("CloudWorkstation bin path not found in system PATH");
                }

                return ActionResult.Success;
            }
            catch (Exception ex)
            {
                session.Log($"Error in RemoveFromSystemPath: {ex.Message}");
                return ActionResult.Success;
            }
        }

        /// <summary>
        /// Notifies the system that environment variables have changed
        /// </summary>
        private static void NotifyEnvironmentChange()
        {
            try
            {
                const int HWND_BROADCAST = 0xffff;
                const uint WM_SETTINGCHANGE = 0x001a;
                const int SMTO_ABORTIFHUNG = 0x0002;

                // Use reflection to call SendMessageTimeout to avoid P/Invoke declaration
                var user32 = System.Reflection.Assembly.LoadWithPartialName("user32.dll");
                if (user32 != null)
                {
                    var sendMessageTimeout = user32.GetType().GetMethod("SendMessageTimeout");
                    if (sendMessageTimeout != null)
                    {
                        object[] parameters = { 
                            (IntPtr)HWND_BROADCAST, 
                            WM_SETTINGCHANGE, 
                            IntPtr.Zero, 
                            "Environment", 
                            SMTO_ABORTIFHUNG, 
                            5000, 
                            IntPtr.Zero 
                        };
                        
                        sendMessageTimeout.Invoke(null, parameters);
                    }
                }
            }
            catch
            {
                // Ignore errors in notification
            }
        }
    }
}