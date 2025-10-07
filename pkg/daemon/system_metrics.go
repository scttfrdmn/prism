package daemon

import (
	"bufio"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// getCPUUsagePercent returns current CPU usage as a percentage
// This is platform-specific and uses different methods per OS
func getCPUUsagePercent() float64 {
	switch runtime.GOOS {
	case "linux":
		return getCPUUsageLinux()
	case "darwin":
		return getCPUUsageDarwin()
	case "windows":
		return getCPUUsageWindows()
	default:
		return 0.0
	}
}

// getLoadAverage returns the system load average (1 minute)
// This is platform-specific
func getLoadAverage() float64 {
	switch runtime.GOOS {
	case "linux", "darwin":
		return getLoadAverageUnix()
	case "windows":
		// Windows doesn't have load average, return CPU usage as proxy
		return getCPUUsageWindows()
	default:
		return 0.0
	}
}

// getCPUUsageLinux reads CPU usage from /proc/stat
func getCPUUsageLinux() float64 {
	// Read /proc/stat for CPU times
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0.0
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0.0
	}

	// First line is aggregate CPU: cpu  user nice system idle iowait irq softirq ...
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return 0.0
	}

	// Parse fields
	user, _ := strconv.ParseFloat(fields[1], 64)
	nice, _ := strconv.ParseFloat(fields[2], 64)
	system, _ := strconv.ParseFloat(fields[3], 64)
	idle, _ := strconv.ParseFloat(fields[4], 64)

	total := user + nice + system + idle
	usage := (total - idle) / total * 100.0

	if usage < 0 || usage > 100 {
		return 0.0
	}

	return usage
}

// getCPUUsageDarwin uses sysctl to get CPU usage on macOS
func getCPUUsageDarwin() float64 {
	// Use iostat for quick CPU sample
	cmd := exec.Command("iostat", "-c", "2", "-w", "1")
	output, err := cmd.Output()
	if err != nil {
		return 0.0
	}

	// Parse iostat output (last line has percentages)
	lines := strings.Split(string(output), "\n")
	if len(lines) < 3 {
		return 0.0
	}

	// Last data line has format: us sy id
	fields := strings.Fields(lines[len(lines)-2])
	if len(fields) < 3 {
		return 0.0
	}

	// Get user + system (first two columns)
	user, _ := strconv.ParseFloat(fields[0], 64)
	system, _ := strconv.ParseFloat(fields[1], 64)

	usage := user + system
	if usage < 0 || usage > 100 {
		return 0.0
	}

	return usage
}

// getCPUUsageWindows uses wmic on Windows
func getCPUUsageWindows() float64 {
	cmd := exec.Command("wmic", "cpu", "get", "loadpercentage")
	output, err := cmd.Output()
	if err != nil {
		return 0.0
	}

	// Parse output (second line has the percentage)
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return 0.0
	}

	usage, err := strconv.ParseFloat(strings.TrimSpace(lines[1]), 64)
	if err != nil || usage < 0 || usage > 100 {
		return 0.0
	}

	return usage
}

// getLoadAverageUnix reads load average from /proc/loadavg (Linux) or sysctl (macOS)
func getLoadAverageUnix() float64 {
	if runtime.GOOS == "linux" {
		// Read /proc/loadavg
		data, err := os.ReadFile("/proc/loadavg")
		if err != nil {
			return 0.0
		}

		// Format: 0.05 0.03 0.01 1/123 12345
		fields := strings.Fields(string(data))
		if len(fields) < 1 {
			return 0.0
		}

		load, err := strconv.ParseFloat(fields[0], 64)
		if err != nil || load < 0 {
			return 0.0
		}

		return load
	}

	// macOS: use sysctl
	cmd := exec.Command("sysctl", "-n", "vm.loadavg")
	output, err := cmd.Output()
	if err != nil {
		return 0.0
	}

	// Format: { 1.23 1.45 1.67 }
	str := strings.TrimSpace(string(output))
	str = strings.Trim(str, "{}")
	fields := strings.Fields(str)
	if len(fields) < 1 {
		return 0.0
	}

	load, err := strconv.ParseFloat(fields[0], 64)
	if err != nil || load < 0 {
		return 0.0
	}

	return load
}

// cpuUsageCache caches CPU usage to avoid excessive system calls
type cpuUsageCache struct {
	value     float64
	timestamp time.Time
	ttl       time.Duration
}

var (
	cpuCache  = &cpuUsageCache{ttl: 5 * time.Second}
	loadCache = &cpuUsageCache{ttl: 5 * time.Second}
)

// getCachedCPUUsage returns cached CPU usage if available
func getCachedCPUUsage() float64 {
	if time.Since(cpuCache.timestamp) < cpuCache.ttl && cpuCache.value > 0 {
		return cpuCache.value
	}

	cpuCache.value = getCPUUsagePercent()
	cpuCache.timestamp = time.Now()
	return cpuCache.value
}

// getCachedLoadAverage returns cached load average if available
func getCachedLoadAverage() float64 {
	if time.Since(loadCache.timestamp) < loadCache.ttl && loadCache.value > 0 {
		return loadCache.value
	}

	loadCache.value = getLoadAverage()
	loadCache.timestamp = time.Now()
	return loadCache.value
}
