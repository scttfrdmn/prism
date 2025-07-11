# CloudWorkstation Idle Detection System

This document describes the idle detection system introduced in CloudWorkstation 0.3.0, designed to optimize costs while respecting research workflows.

## Overview

The idle detection system monitors resource usage on CloudWorkstation instances and automatically takes action when an instance is determined to be idle. This system is separate from budget management and focuses exclusively on detecting when instances are not actively being used.

## Key Features

- Multi-metric monitoring for comprehensive idle detection
- Domain-specific profiles for different research workloads
- User-configurable thresholds and actions
- Transparent notification system
- Integration with CloudWorkstation's cost management

## Idle Detection Metrics

The system monitors multiple metrics to accurately determine if an instance is idle:

| Metric | Description | Default Threshold |
|--------|-------------|-----------------|
| CPU usage | Percentage of CPU being used | 10% |
| Memory usage | Percentage of memory being used | 30% |
| Network traffic | Network throughput in KBps | 50 KBps |
| Disk I/O | Disk read/write in KBps | 100 KBps |
| GPU usage | Percentage of GPU being used (if available) | 5% |

An instance is considered idle when ALL metrics remain below their thresholds for the specified idle period.

## Configuration Options

Idle detection can be configured globally, per-domain, or per-instance:

### Global Configuration

The global configuration is stored in `~/.cloudworkstation/idle.json`:

```json
{
  "enabled": true,
  "default_profile": "standard",
  "profiles": {
    "standard": {
      "cpu_threshold": 10,
      "memory_threshold": 30,
      "network_threshold": 50,
      "disk_threshold": 100,
      "gpu_threshold": 5,
      "idle_minutes": 30,
      "action": "stop"
    },
    "batch": {
      "cpu_threshold": 5,
      "memory_threshold": 20,
      "idle_minutes": 60,
      "action": "hibernate"
    }
  },
  "domain_mappings": {
    "machine-learning": "gpu",
    "genomics": "batch",
    "data-science": "standard"
  },
  "instance_overrides": {
    "my-gpu-instance": {
      "profile": "gpu",
      "idle_minutes": 15
    }
  }
}
```

### Template Configuration

Templates can specify their own idle detection settings:

```yaml
idle_detection:
  profile: "gpu"
  cpu_threshold: 5
  memory_threshold: 20
  network_threshold: 50
  disk_threshold: 100
  gpu_threshold: 3
  idle_minutes: 15
  action: "stop"
  notification: true
```

### Per-Instance Configuration

When launching an instance, idle detection can be configured:

```bash
cws launch machine-learning my-instance --idle-profile gpu --idle-minutes 15
```

## Idle Detection Profiles

CloudWorkstation provides several pre-configured profiles for different research workloads:

### Standard Profile

Balanced settings for interactive work:
- CPU threshold: 10%
- Memory threshold: 30%
- Network threshold: 50 KBps
- Disk threshold: 100 KBps
- GPU threshold: 5%
- Idle minutes: 30
- Action: stop

### Batch Profile

For batch processing jobs with longer idle periods:
- CPU threshold: 5%
- Memory threshold: 20%
- Network threshold: 25 KBps
- Disk threshold: 50 KBps
- GPU threshold: 3%
- Idle minutes: 60
- Action: hibernate

### GPU Profile

Optimized for GPU workloads:
- CPU threshold: 5%
- Memory threshold: 20%
- Network threshold: 50 KBps
- Disk threshold: 100 KBps
- GPU threshold: 3%
- Idle minutes: 15
- Action: stop

### Data-Intensive Profile

For data processing workloads:
- CPU threshold: 8%
- Memory threshold: 40%
- Network threshold: 100 KBps
- Disk threshold: 200 KBps
- GPU threshold: 5%
- Idle minutes: 45
- Action: stop

## Domain-Specific Settings

CloudWorkstation 0.3.0 includes domain-specific idle detection settings for optimal cost management:

### Machine Learning / AI

- Profile: gpu
- CPU threshold: 5%
- Memory threshold: 20%
- GPU threshold: 3%
- Idle minutes: 15

### Genomics & Bioinformatics

- Profile: batch
- CPU threshold: 5%
- Memory threshold: 20%
- Disk threshold: 50 KBps
- Idle minutes: 60

### Data Science

- Profile: standard
- CPU threshold: 10%
- Memory threshold: 30%
- Idle minutes: 30

### Climate Science

- Profile: batch
- CPU threshold: 5%
- Memory threshold: 20%
- Idle minutes: 45

## Available Actions

When an instance is determined to be idle, one of the following actions can be taken:

| Action | Description |
|--------|-------------|
| `stop` | Stop the instance (can be restarted later) |
| `hibernate` | Hibernate the instance (preserves memory state) |
| `notify` | Send notification only, no action taken |

## Implementation Details

### Monitoring Agent

A lightweight monitoring agent runs on each instance, collecting usage metrics and reporting them to the CloudWorkstation daemon. The agent is deployed automatically during instance launch.

### Daemon Integration

The CloudWorkstation daemon receives metrics from the monitoring agent and determines if an instance is idle based on the configured thresholds and idle period. When an instance is determined to be idle, the daemon takes the configured action.

### CloudSnooze Integration

The idle detection system is inspired by and compatible with the CloudSnooze project. CloudSnooze can be used as an alternative monitoring agent if installed separately.

## Command Line Interface

### Configure Idle Detection

```bash
# Enable/disable idle detection globally
cws idle enable
cws idle disable

# Configure global settings
cws idle config --default-profile standard

# Set profile for a domain
cws idle domain machine-learning --profile gpu

# Override settings for an instance
cws idle instance my-instance --profile custom --cpu-threshold 15 --idle-minutes 45
```

### View Idle Status

```bash
# View global idle detection status
cws idle status

# View idle status for an instance
cws idle status my-instance

# View idle history for an instance
cws idle history my-instance
```

## Best Practices

1. **Match profiles to workloads**: Use appropriate profiles for different research domains.
2. **Start conservative**: Begin with higher thresholds and longer idle periods, then adjust based on usage patterns.
3. **Use notifications**: Start with the `notify` action to understand when instances would be stopped before enabling automatic actions.
4. **Monitor false positives**: Regularly check if instances are incorrectly detected as idle.
5. **Consider workflow patterns**: Set longer idle periods for workloads with natural gaps in activity.
6. **Document settings**: Document your idle detection configuration for reproducibility.

## Troubleshooting

### Common Issues

1. **False positives**: Instance stopped while still in use
   - Increase thresholds or idle period
   - Switch to notification-only mode temporarily
   
2. **Failed to detect idle**: Instance continues running despite being idle
   - Check if any single metric is above threshold
   - Verify monitoring agent is running
   - Check logs for monitoring errors

3. **Metrics not reported**: No metrics visible in idle status
   - Check network connectivity between instance and daemon
   - Verify monitoring agent is running with `ps aux | grep idle-monitor`
   - Check agent logs in `/var/log/cloudworkstation/idle-monitor.log`

### Log Files

- Daemon logs: `~/.cloudworkstation/logs/daemon.log`
- Agent logs: `/var/log/cloudworkstation/idle-monitor.log`
- Action history: `~/.cloudworkstation/logs/idle-actions.log`

## Security Considerations

The idle detection system is designed with security in mind:

- All communication between the agent and daemon is encrypted
- Agent runs with minimal privileges required for monitoring
- No sensitive data is collected or transmitted
- All actions are logged for audit purposes