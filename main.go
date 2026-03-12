package main

import (
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/getlantern/systray"
	"github.com/guptarohit/asciigraph"
	"kuyaninja/net-stabilizer/trayicon"
)

// --- Styles ---
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1).
			MarginBottom(1)

	boxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(1, 2).
			Width(60).
			Height(10)

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#3C3C3C")).
			Padding(0, 1).
			MarginTop(1)

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ADADAD"))

	alertStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5F87")).
			Bold(true)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00D787"))
)

// --- Model ---
type model struct {
	profile      Profile
	targetIP     string
	metrics      Metrics
	logs         []string
	badChecks    int
	lastReset    time.Time
	windowHeight  int
	windowWidth   int
	topApps       string
	latencyHistory []float64
}

type pingMsg Metrics
type errMsg error
type bwMsg string

func (m model) Init() tea.Cmd {
	return tea.Batch(
		doPing(m.targetIP),
		doBandwidthCheck(),
	)
}

func doBandwidthCheck() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		return bwMsg(GetTopBandwidthHogs())
	})
}

func doPing(target string) tea.Cmd {
	return func() tea.Msg {
		metrics, err := Measure(target, 3, 3*time.Second)
		if err != nil {
			return errMsg(err)
		}
		return pingMsg(metrics)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.windowWidth = msg.Width
		m.windowHeight = msg.Height
	case pingMsg:
		m.metrics = Metrics(msg)
		m.processMetrics()
		// keep history of past 60 pings max
		m.latencyHistory = append(m.latencyHistory, float64(m.metrics.Latency.Milliseconds()))
		if len(m.latencyHistory) > 60 {
			m.latencyHistory = m.latencyHistory[1:]
		}
		var lastLog string
		if len(m.logs) > 0 {
			// Strip the lipgloss color codes from the log string before sending to OS Tray
			lastLog = lipgloss.NewStyle().Render(m.logs[len(m.logs)-1])
		}
		updateTrayDetails(m.metrics, m.topApps, lastLog)
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return doPing(m.targetIP)()
		})
	case bwMsg:
		m.topApps = string(msg)
		var lastLog string
		if len(m.logs) > 0 {
			lastLog = lipgloss.NewStyle().Render(m.logs[len(m.logs)-1])
		}
		updateTrayDetails(m.metrics, m.topApps, lastLog)
		return m, doBandwidthCheck()
	case errMsg:
		m.addLog(alertStyle.Render(fmt.Sprintf("Ping error: %v", msg)))
		return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
			return doPing(m.targetIP)()
		})
	}
	return m, nil
}

func (m *model) addLog(s string) {
	timestamp := time.Now().Format("15:04:05")
	m.logs = append(m.logs, fmt.Sprintf("[%s] %s", timestamp, s))
	if len(m.logs) > 8 {
		m.logs = m.logs[1:]
	}
}

func (m *model) processMetrics() {
	if !m.metrics.IsUp {
		m.addLog(alertStyle.Render("❌ Network is DOWN!"))
		m.badChecks++
	} else {
		latMs := int(m.metrics.Latency.Milliseconds())
		jitMs := int(m.metrics.Jitter.Milliseconds())
		isBad := false

		if latMs > m.profile.MaxLatencyMs {
			m.addLog(alertStyle.Render(fmt.Sprintf("⚠️ High Latency: %dms", latMs)))
			isBad = true
		}
		if jitMs > m.profile.MaxJitterMs {
			m.addLog(alertStyle.Render(fmt.Sprintf("⚠️ High Jitter: %dms", jitMs)))
			isBad = true
		}
		if m.metrics.PacketLoss > m.profile.MaxPacketLoss {
			m.addLog(alertStyle.Render(fmt.Sprintf("⚠️ Packet Loss: %.1f%%", m.metrics.PacketLoss)))
			isBad = true
		}

		if isBad {
			m.badChecks++
		} else if m.badChecks > 0 {
			m.addLog(successStyle.Render("✅ Network recovered."))
			m.badChecks = 0
		}
	}

	if m.badChecks >= m.profile.ActionThreshold {
		if time.Since(m.lastReset) > 2*time.Minute {
			m.executeRecovery()
		}
	}
}

func (m *model) executeRecovery() {
	if m.badChecks < m.profile.ActionThreshold+3 {
		m.addLog(lipgloss.NewStyle().Foreground(lipgloss.Color("#5FAFFF")).Render("🔄 Action: Flushing DNS..."))
		if err := FlushDNS(); err != nil {
			m.addLog(alertStyle.Render(fmt.Sprintf("❌ DNS Flush failed: %v", err)))
		} else {
			m.addLog(successStyle.Render("✔️ DNS Flushed."))
		}
	} else {
		m.addLog(lipgloss.NewStyle().Foreground(lipgloss.Color("#5FAFFF")).Render("🔄 Action: Soft Reset Interface..."))
		if err := SoftResetNetwork(); err != nil {
			m.addLog(alertStyle.Render(fmt.Sprintf("❌ Reset failed: %v", err)))
		} else {
			m.addLog(successStyle.Render("✔️ Interface Reset."))
		}
		m.badChecks = 0
	}
	m.lastReset = time.Now()
}

func (m model) View() string {
	var s strings.Builder

	// Header
	header := titleStyle.Render(fmt.Sprintf(" 🌐 NETWORK STABILIZER [%s] ", strings.ToUpper(m.profile.Name)))
	s.WriteString(lipgloss.PlaceHorizontal(m.windowWidth, lipgloss.Center, header))
	s.WriteString("\n")

	// Log Box
	logContent := strings.Join(m.logs, "\n")
	if logContent == "" {
		logContent = "🟢"
	}
	box := boxStyle.Render(logContent)
	s.WriteString(lipgloss.PlaceHorizontal(m.windowWidth, lipgloss.Center, box))
	s.WriteString("\n")

	// Status Bar
	status := fmt.Sprintf(" TARGET: %s | LAT: %v | JIT: %v | LOSS: %.1f%% ",
		m.targetIP,
		m.metrics.Latency.Truncate(time.Millisecond),
		m.metrics.Jitter.Truncate(time.Millisecond),
		m.metrics.PacketLoss,
	)
	
	footer := statusStyle.Render(status) + " " + logStyle.Render("press 'q' to quit")
	s.WriteString(lipgloss.PlaceHorizontal(m.windowWidth, lipgloss.Center, footer))
	s.WriteString("\n")

	bwText := m.topApps
	if bwText == "" {
		bwText = "Calculating Bandwidth..."
	}
	s.WriteString(lipgloss.PlaceHorizontal(m.windowWidth, lipgloss.Center, logStyle.Render(bwText)))
	s.WriteString("\n\n")

	// Chart
	if len(m.latencyHistory) > 0 {
		graph := asciigraph.Plot(m.latencyHistory, asciigraph.Height(8), asciigraph.Caption("Latency History (ms)"))
		// Align left to normalize line lengths so Center doesn't stagger them
		graphBlock := lipgloss.NewStyle().Align(lipgloss.Left).Render(graph)
		s.WriteString(lipgloss.PlaceHorizontal(m.windowWidth, lipgloss.Center, graphBlock))
	} else {
		s.WriteString(lipgloss.PlaceHorizontal(m.windowWidth, lipgloss.Center, logStyle.Render("Waiting for ping data to draw chart...")))
	}

	return lipgloss.PlaceVertical(m.windowHeight, lipgloss.Center, s.String())
}

// --- Data Types ---
type Profile struct {
	Name            string
	MaxLatencyMs    int
	MaxJitterMs     int
	MaxPacketLoss   float64
	ActionThreshold int
}

var Profiles = map[string]Profile{
	"Gaming":   {Name: "Gaming", MaxLatencyMs: 100, MaxJitterMs: 30, MaxPacketLoss: 1.0, ActionThreshold: 3},
	"Meeting":  {Name: "Meeting", MaxLatencyMs: 250, MaxJitterMs: 100, MaxPacketLoss: 5.0, ActionThreshold: 5},
	"Browsing": {Name: "Browsing", MaxLatencyMs: 800, MaxJitterMs: 300, MaxPacketLoss: 10.0, ActionThreshold: 8},
}

func findBestTarget(targets []string) string {
	fmt.Println("🔍 Scanning for best monitoring target...")

	var wg sync.WaitGroup
	var mu sync.Mutex
	bestTarget := ""
	bestLatency := time.Hour // arbitrarily large

	for _, target := range targets {
		wg.Add(1)
		go func(ip string) {
			defer wg.Done()
			// 3 pings, 2 second timeout total
			metrics, err := Measure(ip, 3, 2*time.Second)
			if err == nil && metrics.IsUp && metrics.PacketLoss == 0 {
				mu.Lock()
				if metrics.Latency < bestLatency {
					bestLatency = metrics.Latency
					bestTarget = ip
				}
				mu.Unlock()
			}
		}(target)
	}

	wg.Wait()

	if bestTarget != "" {
		fmt.Printf("✅ Selected: %s (Latency: %v)\n", bestTarget, bestLatency.Truncate(time.Millisecond))
		time.Sleep(1 * time.Second) // give user a moment to read
	}

	return bestTarget
}

// Global elements for systray
var mItemLat *systray.MenuItem
var mItemApp *systray.MenuItem
var mItemLog *systray.MenuItem
var program *tea.Program

// toBoldFont converts ascii numbers to Unicode Mathematical Bold Sans-Serif digits
func toBoldFont(s string) string {
	var result strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			// Mathematical sans-serif bold digit 0 is U+1D7EC
			result.WriteRune(rune(0x1D7EC + (r - '0')))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// ansiRegex removes terminal styling codes before parsing output to system tray
var ansiRegex = regexp.MustCompile("\x1b\\[[0-9;]*m")

func stripANSI(str string) string {
	return ansiRegex.ReplaceAllString(str, "")
}

func updateTrayDetails(metrics Metrics, topApps string, lastLog string) {
	if mItemLat != nil {
		status := "DOWN"
		if metrics.IsUp {
			latStr := fmt.Sprintf("L:%dms J:%dms", metrics.Latency.Milliseconds(), metrics.Jitter.Milliseconds())
			status = toBoldFont(latStr)
		}
		systray.SetTitle(status)
		systray.SetTooltip(fmt.Sprintf("Net-Stabilizer: %s", status))
		mItemLat.SetTitle(fmt.Sprintf("Metrics: %s", status))
	}
	if mItemApp != nil && topApps != "" {
		mItemApp.SetTitle(fmt.Sprintf("Top: %s", topApps))
	}
	if mItemLog != nil && lastLog != "" {
		// Attempt to strip ANSI codes added by lipgloss before giving to OS 
		// (A simple approach since OS menus don't render terminal colors)
		cleanLog := stripANSI(lastLog)
		mItemLog.SetTitle(cleanLog)
	}
}

func onTrayReady() {
	systray.SetIcon(trayicon.Data)
	systray.SetTitle("--")
	systray.SetTooltip("Network Stabilizer")
	
	mItemLat = systray.AddMenuItem("Current Latency: --", "Live Latency")
	mItemLat.Disable()
	
	mItemApp = systray.AddMenuItem("Top: --", "Top bandwidth application")
	mItemApp.Disable()

	mItemLog = systray.AddMenuItem("Log: --", "Most recent status event")
	mItemLog.Disable()

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit", "Quit the whole app")

	go func() {
		<-mQuit.ClickedCh
		systray.Quit()
		if program != nil {
			program.Quit()
		}
		os.Exit(0)
	}()
}

func onTrayExit() {
	// cleanup if necessary
}

func main() {
	var profileName string
	var targetIP string
	flag.StringVar(&profileName, "profile", "Browsing", "Activity profile (Gaming, Meeting, Browsing). Default: Browsing")
	flag.StringVar(&profileName, "p", "Browsing", "Activity profile (alias)")
	flag.StringVar(&targetIP, "target", "auto", "Target IP (default 'auto' scans best DNS)")
	flag.StringVar(&targetIP, "t", "auto", "Target IP (short)")
	var backgroundMode bool
	flag.BoolVar(&backgroundMode, "bg", false, "Run in background (system tray only)")
	flag.BoolVar(&backgroundMode, "b", false, "Run in background (alias)")
	
	var isChild bool
	flag.BoolVar(&isChild, "child", false, "Internal flag used for background daemon")

	flag.Parse()

	// Normalize profileName to Title Case (e.g., "gaming" -> "Gaming")
	profileName = strings.Title(strings.ToLower(profileName))

	profile, ok := Profiles[profileName]
	if !ok {
		log.Fatalf("Unknown profile: %s. Valid profiles: Gaming, Meeting, Browsing", profileName)
	}

	if targetIP == "auto" {
		// Popular reliable DNS servers
		defaultTargets := []string{"8.8.8.8", "1.1.1.1", "9.9.9.9", "208.67.222.222"}
		targetIP = findBestTarget(defaultTargets)
		if targetIP == "" {
			fmt.Println("❌ No internet connection detected. All test pings failed. Please check your network and try again.")
			os.Exit(1)
		}
	}

	m := model{
		profile:   profile,
		targetIP:  targetIP,
		logs:      []string{},
		lastReset: time.Now().Add(-5 * time.Minute),
		latencyHistory: []float64{},
	}

	// Disable standard logger to prevent background libraries (like go-ping)
	// from printing errors that corrupt the Bubble Tea UI.
	log.SetOutput(io.Discard)

	if backgroundMode && !isChild {
		// Parent process: spawn child and exit
		args := []string{}
		for _, arg := range os.Args[1:] {
			if arg != "-bg" && arg != "-b" { // strip the bg flags
				args = append(args, arg)
			}
		}
		args = append(args, "-child")
		
		cmd := exec.Command(os.Args[0], args...)
		err := cmd.Start()
		if err != nil {
			fmt.Printf("❌ Failed to start background process: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("🚀 Net-Stabilizer started in background! (PID: %d)\n", cmd.Process.Pid)
		fmt.Println("Check your System Tray for the icon. You can now close this terminal.")
		os.Exit(0)
	}

	if isChild {
		// Run headless daemon
		go func() {
			tickPing := time.NewTicker(2 * time.Second)
			tickBw := time.NewTicker(2 * time.Second)
			defer tickPing.Stop()
			defer tickBw.Stop()

			for {
				select {
				case <-tickPing.C:
					go func() {
						metrics, err := Measure(m.targetIP, 3, 3*time.Second)
						if err == nil {
							m.metrics = metrics
							m.processMetrics()
							m.latencyHistory = append(m.latencyHistory, float64(metrics.Latency.Milliseconds()))
							if len(m.latencyHistory) > 60 {
								m.latencyHistory = m.latencyHistory[1:]
							}
							
							var lastLog string
							if len(m.logs) > 0 {
								lastLog = lipgloss.NewStyle().Render(m.logs[len(m.logs)-1])
							}
							updateTrayDetails(m.metrics, m.topApps, lastLog)
						}
					}()
				case <-tickBw.C:
					go func() {
						m.topApps = GetTopBandwidthHogs()
						var lastLog string
						if len(m.logs) > 0 {
							lastLog = lipgloss.NewStyle().Render(m.logs[len(m.logs)-1])
						}
						updateTrayDetails(m.metrics, m.topApps, lastLog)
					}()
				}
			}
		}()

		// Start systray blocking the main thread
		systray.Run(onTrayReady, onTrayExit)
	} else {
		program = tea.NewProgram(m, tea.WithAltScreen())

		go func() {
			if _, err := program.Run(); err != nil {
				fmt.Printf("Alas, there's been an error: %v\n", err)
				os.Exit(1)
			}
			// When TUI quits, quit systray
			systray.Quit()
			os.Exit(0)
		}()

		// Start systray blocking the main thread 
		systray.Run(onTrayReady, onTrayExit)
	}
}
