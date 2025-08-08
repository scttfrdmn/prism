package idle

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

// MetricsCollector collects system metrics from instances via SSH
type MetricsCollector struct {
	sshConfig *ssh.ClientConfig
	timeout   time.Duration
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(keyPath string, username string, timeout time.Duration) (*MetricsCollector, error) {
	// TODO: Load SSH private key from keyPath
	// For now, we'll implement the basic structure
	
	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Use proper host key verification
		Timeout:         timeout,
		// Auth methods will be added when we implement key loading
	}

	return &MetricsCollector{
		sshConfig: config,
		timeout:   timeout,
	}, nil
}

// CollectMetrics collects comprehensive system metrics from a running instance
func (mc *MetricsCollector) CollectMetrics(instanceIP string) (*UsageMetrics, error) {
	// Establish SSH connection
	conn, err := ssh.Dial("tcp", instanceIP+":22", mc.sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", instanceIP, err)
	}
	defer conn.Close()

	// Collect all metrics concurrently for efficiency
	metrics := &UsageMetrics{
		Timestamp: time.Now(),
	}

	// CPU Usage (cross-platform)
	if cpu, err := mc.getCPUUsage(conn); err == nil {
		metrics.CPU = cpu
	}

	// Memory Usage (cross-platform)
	if memory, err := mc.getMemoryUsage(conn); err == nil {
		metrics.Memory = memory
	}

	// Network Activity (cross-platform)
	if network, err := mc.getNetworkActivity(conn); err == nil {
		metrics.Network = network
	}

	// Disk I/O Activity (cross-platform)
	if disk, err := mc.getDiskActivity(conn); err == nil {
		metrics.Disk = disk
	}

	// GPU Usage (if available)
	if gpu, err := mc.getGPUUsage(conn); err == nil && gpu != nil {
		metrics.GPU = gpu
	}

	// User Activity Detection (comprehensive)
	if hasActivity, err := mc.detectUserActivity(conn); err == nil {
		metrics.HasActivity = hasActivity
	}

	return metrics, nil
}

// getCPUUsage gets CPU utilization percentage (works on x86_64 and ARM64)
func (mc *MetricsCollector) getCPUUsage(conn *ssh.Client) (float64, error) {
	// Use /proc/stat for cross-platform CPU monitoring
	cmd := `awk '/^cpu /{u=$2+$4; t=$2+$3+$4+$5; print (u/t*100)}' /proc/stat`
	
	session, err := conn.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		// Fallback to top command
		return mc.getCPUUsageFromTop(conn)
	}

	cpuStr := strings.TrimSpace(string(output))
	return strconv.ParseFloat(cpuStr, 64)
}

// getCPUUsageFromTop fallback CPU monitoring using top
func (mc *MetricsCollector) getCPUUsageFromTop(conn *ssh.Client) (float64, error) {
	// Cross-platform top command (works on Ubuntu, Amazon Linux, Rocky Linux)
	cmd := `top -bn1 | grep "Cpu(s)" | awk '{print $2}' | awk -F'%' '{print $1}'`
	
	session, err := conn.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return 0, err
	}

	cpuStr := strings.TrimSpace(string(output))
	return strconv.ParseFloat(cpuStr, 64)
}

// getMemoryUsage gets memory utilization percentage
func (mc *MetricsCollector) getMemoryUsage(conn *ssh.Client) (float64, error) {
	// Cross-platform memory monitoring using /proc/meminfo
	cmd := `awk '/^MemTotal:/{t=$2} /^MemAvailable:/{a=$2} END{printf "%.2f", (t-a)/t*100}' /proc/meminfo`
	
	session, err := conn.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return 0, err
	}

	memStr := strings.TrimSpace(string(output))
	return strconv.ParseFloat(memStr, 64)
}

// getNetworkActivity gets network activity in KBps
func (mc *MetricsCollector) getNetworkActivity(conn *ssh.Client) (float64, error) {
	// Monitor all network interfaces for activity
	cmd := `cat /proc/net/dev | grep -E ': ' | awk '{rx+=$2; tx+=$10} END{print (rx+tx)/1024}'`
	
	session, err := conn.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return 0, err
	}

	// This gives total bytes, we need rate - would need to store previous values
	// For now, return current total as a proxy for activity
	netStr := strings.TrimSpace(string(output))
	return strconv.ParseFloat(netStr, 64)
}

// getDiskActivity gets disk I/O activity in KBps
func (mc *MetricsCollector) getDiskActivity(conn *ssh.Client) (float64, error) {
	// Cross-platform disk activity monitoring
	cmd := `awk '/^(sd|nvme|xvd)/{rs+=$6; ws+=$10} END{print (rs+ws)*512/1024}' /proc/diskstats`
	
	session, err := conn.NewSession()
	if err != nil {
		return 0, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return 0, err
	}

	diskStr := strings.TrimSpace(string(output))
	return strconv.ParseFloat(diskStr, 64)
}

// getGPUUsage gets GPU utilization if available
func (mc *MetricsCollector) getGPUUsage(conn *ssh.Client) (*float64, error) {
	// Try NVIDIA first
	if gpu, err := mc.getNVIDIAGPU(conn); err == nil {
		return gpu, nil
	}

	// Try AMD GPU
	if gpu, err := mc.getAMDGPU(conn); err == nil {
		return gpu, nil
	}

	// No GPU detected
	return nil, fmt.Errorf("no GPU detected")
}

// getNVIDIAGPU gets NVIDIA GPU utilization
func (mc *MetricsCollector) getNVIDIAGPU(conn *ssh.Client) (*float64, error) {
	cmd := `nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits`
	
	session, err := conn.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return nil, err
	}

	gpuStr := strings.TrimSpace(string(output))
	gpu, err := strconv.ParseFloat(gpuStr, 64)
	if err != nil {
		return nil, err
	}

	return &gpu, nil
}

// getAMDGPU gets AMD GPU utilization (placeholder)
func (mc *MetricsCollector) getAMDGPU(conn *ssh.Client) (*float64, error) {
	// AMD GPU monitoring would go here
	// This is more complex and depends on the specific AMD drivers
	return nil, fmt.Errorf("AMD GPU monitoring not implemented")
}

// detectUserActivity comprehensive user activity detection
func (mc *MetricsCollector) detectUserActivity(conn *ssh.Client) (bool, error) {
	// Check multiple indicators of user activity
	
	// 1. Active user sessions
	if active, err := mc.checkActiveUsers(conn); err == nil && active {
		return true, nil
	}

	// 2. Recent keyboard/mouse activity
	if active, err := mc.checkInputActivity(conn); err == nil && active {
		return true, nil
	}

	// 3. Active user processes
	if active, err := mc.checkUserProcesses(conn); err == nil && active {
		return true, nil
	}

	// 4. Network connections indicating user activity
	if active, err := mc.checkUserConnections(conn); err == nil && active {
		return true, nil
	}

	return false, nil
}

// checkActiveUsers checks for currently logged-in users
func (mc *MetricsCollector) checkActiveUsers(conn *ssh.Client) (bool, error) {
	// Check for active user sessions (excluding system accounts)
	cmd := `who | grep -v "^root" | wc -l`
	
	session, err := conn.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return false, err
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// checkInputActivity checks for recent keyboard/mouse input activity
func (mc *MetricsCollector) checkInputActivity(conn *ssh.Client) (bool, error) {
	// Check for input device interrupts in the last minute
	cmd := `find /dev/input -name "event*" -newer <(date -d '1 minute ago' '+%Y-%m-%d %H:%M:%S') 2>/dev/null | wc -l`
	
	session, err := conn.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		// Fallback: check /proc/interrupts for input activity
		return mc.checkInterruptActivity(conn)
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// checkInterruptActivity checks interrupt counters for input devices
func (mc *MetricsCollector) checkInterruptActivity(conn *ssh.Client) (bool, error) {
	// This is a simplified check - in reality we'd need to compare with previous values
	cmd := `grep -i "input\|keyboard\|mouse" /proc/interrupts | wc -l`
	
	session, err := conn.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return false, err
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// checkUserProcesses checks for active user processes
func (mc *MetricsCollector) checkUserProcesses(conn *ssh.Client) (bool, error) {
	// Check for non-system user processes
	cmd := `ps aux | grep -v -E "^\[|root.*\[" | grep -v -E "(kthread|ksoftirq|migration|rcu_|systemd)" | grep -v "ps aux" | wc -l`
	
	session, err := conn.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return false, err
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false, err
	}

	// Some user processes are normal, but many indicate active use
	return count > 10, nil
}

// checkUserConnections checks for active user network connections
func (mc *MetricsCollector) checkUserConnections(conn *ssh.Client) (bool, error) {
	// Check for user applications with network connections (excluding SSH)
	cmd := `ss -tuln | grep -v ":22 " | grep LISTEN | wc -l`
	
	session, err := conn.NewSession()
	if err != nil {
		return false, err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return false, err
	}

	countStr := strings.TrimSpace(string(output))
	count, err := strconv.Atoi(countStr)
	if err != nil {
		return false, err
	}

	// Active listening services might indicate user applications
	return count > 5, nil
}