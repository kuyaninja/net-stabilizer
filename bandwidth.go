// Package main provides network monitoring and recovery tools.
package main

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

// ProcessBandwidth stores the cumulative bytes sent and received by a process.
type ProcessBandwidth struct {
	PID       int32
	Name      string
	BytesSent uint64
	BytesRecv uint64
}

// NetRate stores the calculated data transfer rate for a process.
type NetRate struct {
	Name    string
	RateBps uint64
}

var (
	lastStats   map[int32]ProcessBandwidth
	lastCheck   time.Time
	trackerLock sync.Mutex
)

func init() {
	lastStats = make(map[int32]ProcessBandwidth)
	lastCheck = time.Now()
}

// GetTopBandwidthHogs attempts to find the processes using the most network/IO.
// It returns a formatted string showing the top active application or global network load.
// Note: On macOS and Windows without root, per-process net IO might return empty or be restricted.
func GetTopBandwidthHogs() string {
	trackerLock.Lock()
	defer trackerLock.Unlock()

	procs, err := process.Processes()
	if err != nil {
		return "App Stats: Unavailable (Perm)"
	}

	currentStats := make(map[int32]ProcessBandwidth)
	var rates []NetRate

	now := time.Now()
	duration := now.Sub(lastCheck).Seconds()
	if duration <= 0 {
		duration = 1
	}

	// Try getting per-process I/O stats (Disk/Net combined on some OS, Disk only on others)
	for _, p := range procs {
		ioStat, err := p.IOCounters()
		// If we can't read IO stats for this process, skip
		if err != nil {
			continue
		}

		name, err := p.Name()
		if err != nil {
			name = "unknown"
		}

		sent := ioStat.WriteBytes
		recv := ioStat.ReadBytes

		currentStats[p.Pid] = ProcessBandwidth{
			PID:       p.Pid,
			Name:      name,
			BytesSent: sent,
			BytesRecv: recv,
		}

		if last, ok := lastStats[p.Pid]; ok {
			deltaSent := sent - last.BytesSent
			deltaRecv := recv - last.BytesRecv
			
			// Protect against counter resets
			if sent >= last.BytesSent && recv >= last.BytesRecv {
				totalDelta := deltaSent + deltaRecv
				rate := uint64(float64(totalDelta) / duration)
				if rate > 1024 { // Filter out tiny background noise (< 1KB/s)
					rates = append(rates, NetRate{Name: name, RateBps: rate})
				}
			}
		}
	}

	lastStats = currentStats
	lastCheck = now

	if len(rates) == 0 {
		// Fallback: Check global network rate if per-process fails or is quiet
		globalIO, err := net.IOCounters(false)
		if err == nil && len(globalIO) > 0 {
			return formatGlobalNet(globalIO[0])
		}
		return "App Stats: Idle"
	}

	// Sort by highest rate
	sort.Slice(rates, func(i, j int) bool {
		return rates[i].RateBps > rates[j].RateBps
	})

	// Format top 1
	top := rates[0]
	return fmt.Sprintf("🔥 Top App: %s (%s)", top.Name, formatBytes(top.RateBps))
}

var lastGlobalNet net.IOCountersStat
var lastGlobalTime time.Time

// formatGlobalNet calculates the global network throughput since the last check.
func formatGlobalNet(current net.IOCountersStat) string {
	now := time.Now()
	if lastGlobalTime.IsZero() {
		lastGlobalNet = current
		lastGlobalTime = now
		return "Net Activity: Measuring..."
	}

	duration := now.Sub(lastGlobalTime).Seconds()
	deltaSent := current.BytesSent - lastGlobalNet.BytesSent
	deltaRecv := current.BytesRecv - lastGlobalNet.BytesRecv
	
	lastGlobalNet = current
	lastGlobalTime = now

	rateSent := uint64(float64(deltaSent) / duration)
	rateRecv := uint64(float64(deltaRecv) / duration)

	return fmt.Sprintf("🌐 Net Load: ⬇ %s | ⬆ %s", formatBytes(rateRecv), formatBytes(rateSent))
}

// formatBytes converts bytes per second into a human-readable string.
func formatBytes(bps uint64) string {
	if bps > 1024*1024 {
		return fmt.Sprintf("%.1f MB/s", float64(bps)/(1024*1024))
	} else if bps > 1024 {
		return fmt.Sprintf("%.1f KB/s", float64(bps)/1024)
	}
	return fmt.Sprintf("%d B/s", bps)
}
