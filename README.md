# 🌐 Network Stabilizer (Go)

A high-performance, cross-platform TUI application built in Golang to monitor, diagnose, and automatically stabilize your internet connection. Ideal for **Gamers**, **Video Meetings**, and **Heavy Browsing**.

## 🚀 Key Features

- **⚡ Smart Monitoring:** Real-time tracking of Latency (RTT), Jitter (StdDev), and Packet Loss.
- **🖥 Interactive TUI:** A professional centered Terminal User Interface (built with Bubble Tea & Lip Gloss).
- **🎯 Auto-Target Selection:** Automatically scans and pings top DNS providers (Google, Cloudflare, Quad9) to select the fastest monitoring target.
- **🔥 Traffic Visibility:** Integrated "Bandwidth Radar" identifies which applications are consuming the most network resources.
- **🎮 Smart Profiles:** Pre-configured thresholds tailored for specific activities:
  - `Gaming`: Ultra-low latency focus (< 100ms).
  - `Meeting`: Reliability focus for Zoom/Teams/Meet.
  - `Browsing`: Standard background monitoring.
- **🛠 Auto-Recovery:** Automatically triggers corrective actions:
  - **Flush DNS Cache:** Clears DNS issues without dropping connection.
  - **Network Reset:** Release/Renew IP (Windows) or Toggle Wi-Fi (macOS/Linux) when connection hangs.
- **🍎🐧🪟 Cross-Platform:** Supports Windows, macOS, and Linux.

## 🛠 Installation

### Prerequisites
- [Go](https://go.dev/dl/) 1.16 or higher (if building from source).

### Build from Source
```bash
git clone https://github.com/kuyaninja/net-stabilizer.git
cd net-stabilizer
go build -o net-stabilizer
```

## 📖 Usage

### Basic Command
Run the application without any flags to use the **Browsing** profile and **Auto-Scan** for the best target:
```bash
./net-stabilizer
```

### Options & Aliases
| Long Flag | Short Flag | Default | Description |
|-----------|------------|---------|-------------|
| `--profile` | `-p` | `Browsing` | Activity profile (`Gaming`, `Meeting`, `Browsing`). Case-insensitive. |
| `--target` | `-t` | `auto` | Target IP to monitor. `auto` selects the fastest DNS server. |

### Examples
- **For Pro Gamers (Recommended):**
  ```bash
  sudo ./net-stabilizer -p Gaming
  ```
- **For Online Meetings:**
  ```bash
  ./net-stabilizer -p Meeting
  ```

> **Note:** Some recovery actions (like resetting the network interface) require **Administrator/Root** privileges. Use `sudo` on macOS/Linux or run as Administrator on Windows.

## ⚙️ How it Works

1. **Auto-Scan:** On startup, the app pings multiple DNS servers and selects the one with the lowest latency and zero loss.
2. **Monitor:** Sends ICMP packets every 2 seconds.
3. **Analyze:** Calculates health metrics over a sliding window.
4. **Identify:** Scans running processes to find "bandwidth hogs" that might be slowing you down.
5. **Threshold:** If metrics exceed your profile's limit for several consecutive checks:
   - **Level 1:** Flushes DNS cache.
   - **Level 2:** Performs a "Soft Reset" of the network interface.
6. **Cooldown:** After a reset, the app waits 2 minutes before taking another hard action to allow the network to stabilize.

## 🧪 Running Tests
```bash
go test -v
```

## 📄 License
MIT License.
