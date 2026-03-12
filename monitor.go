// Package main provides network monitoring and recovery tools.
package main

import (
	"time"

	"github.com/go-ping/ping"
)

// Metrics holds the calculated network health statistics.
type Metrics struct {
	Latency    time.Duration // Average Round Trip Time
	Jitter     time.Duration // Standard Deviation of RTT as a proxy for jitter
	PacketLoss float64       // Percentage of packets lost
	IsUp       bool          // True if at least one packet was received
}

// Measure performs a series of pings to the target IP and returns calculated metrics.
// targetIP: The destination to ping (e.g., "8.8.8.8").
// count: Number of packets to send.
// timeout: Total time to wait for responses.
func Measure(targetIP string, count int, timeout time.Duration) (Metrics, error) {
	pinger, err := ping.NewPinger(targetIP)
	if err != nil {
		return Metrics{}, err
	}

	pinger.Count = count
	pinger.Timeout = timeout
	
	// SetPrivileged(false) uses UDP pings which work without root on many systems.
	pinger.SetPrivileged(false)

	err = pinger.Run()
	if err != nil {
		return Metrics{IsUp: false}, err
	}

	stats := pinger.Statistics()

	if stats.PacketsRecv == 0 {
		return Metrics{IsUp: false, PacketLoss: 100.0}, nil
	}

	return Metrics{
		Latency:    stats.AvgRtt,
		Jitter:     stats.StdDevRtt,
		PacketLoss: stats.PacketLoss,
		IsUp:       true,
	}, nil
}
