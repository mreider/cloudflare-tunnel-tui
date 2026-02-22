package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mreider/cloudflare-tunnel-tui/internal/config"
	"github.com/mreider/cloudflare-tunnel-tui/internal/tunnel"
)

// Styles
var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FF8C00")).
			Padding(0, 1)

	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#AAAAAA")).
			Padding(0, 1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFFFFF")).
			Background(lipgloss.Color("#FF8C00")).
			Padding(0, 1)

	normalStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#DDDDDD")).
			Padding(0, 1)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Padding(0, 1)

	statusOK = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			SetString("●")

	statusProxy = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFD700")).
			SetString("◆")

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666")).
			Padding(1, 1)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Padding(0, 1)

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#00FF00")).
			Padding(0, 1)

	credStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFD700")).
			Padding(0, 1)
)

type view int

const (
	viewDashboard view = iota
	viewDevice
)

type statusMsg struct {
	message string
	isError bool
}

type clearStatusMsg struct{}
type tickMsg time.Time
type sshExitMsg struct{ err error }

type Model struct {
	cfg       *config.Config
	mgr       *tunnel.Manager
	view      view
	cursor    int
	dcursor   int // device detail cursor
	width     int
	height    int
	status    string
	isError   bool
	showCreds bool
}

func New(cfg *config.Config, mgr *tunnel.Manager) Model {
	return Model{
		cfg: cfg,
		mgr: mgr,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		tea.WindowSize(),
		tickCmd(),
	)
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tea.KeyMsg:
		return m.handleKey(msg)

	case statusMsg:
		m.status = msg.message
		m.isError = msg.isError
		return m, tea.Tick(4*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})

	case clearStatusMsg:
		m.status = ""
		m.isError = false

	case tickMsg:
		return m, tickCmd()

	case sshExitMsg:
		if msg.err != nil {
			m.status = fmt.Sprintf("SSH error: %v", msg.err)
			m.isError = true
		} else {
			m.status = "SSH session ended"
			m.isError = false
		}
		return m, tea.Tick(8*time.Second, func(t time.Time) tea.Msg {
			return clearStatusMsg{}
		})
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, key.NewBinding(key.WithKeys("q", "ctrl+c"))):
		m.mgr.StopAll()
		return m, tea.Quit

	case key.Matches(msg, key.NewBinding(key.WithKeys("up", "k"))):
		if m.view == viewDashboard {
			if m.cursor > 0 {
				m.cursor--
			}
		} else {
			if m.dcursor > 0 {
				m.dcursor--
			}
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("down", "j"))):
		if m.view == viewDashboard {
			if m.cursor < len(m.cfg.Devices)-1 {
				m.cursor++
			}
		} else {
			dev := m.cfg.Devices[m.cursor]
			if m.dcursor < len(dev.Services)-1 {
				m.dcursor++
			}
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("enter"))):
		if m.view == viewDashboard {
			m.view = viewDevice
			m.dcursor = 0
		} else {
			return m.connectService()
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("esc", "b"))):
		if m.view == viewDevice {
			m.view = viewDashboard
			m.showCreds = false
		}

	case key.Matches(msg, key.NewBinding(key.WithKeys("c"))):
		if m.view == viewDevice {
			m.showCreds = !m.showCreds
		}
	}

	return m, nil
}

func (m Model) connectService() (tea.Model, tea.Cmd) {
	dev := m.cfg.Devices[m.cursor]
	svc := dev.Services[m.dcursor]

	switch svc.Type {
	case "ssh":
		args := []string{svc.Hostname}
		if svc.User != "" {
			args = []string{"-l", svc.User, svc.Hostname}
		}
		c := exec.Command("ssh", args...)
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		return m, tea.ExecProcess(c, func(err error) tea.Msg {
			return sshExitMsg{err: err}
		})

	case "vnc":
		if svc.ProxyLocalPort == 0 {
			m.status = "No proxy port configured for VNC"
			m.isError = true
			return m, nil
		}

		if m.mgr.IsRunning(svc.Hostname) {
			return m, func() tea.Msg {
				if err := openVNCClient(svc.ProxyLocalPort); err != nil {
					return statusMsg{message: fmt.Sprintf("VNC client failed: %v", err), isError: true}
				}
				return statusMsg{message: fmt.Sprintf("VNC opened on localhost:%d", svc.ProxyLocalPort)}
			}
		}

		return m, func() tea.Msg {
			if err := m.mgr.StartProxy(svc.Hostname, svc.ProxyLocalPort); err != nil {
				return statusMsg{message: fmt.Sprintf("Proxy failed: %v", err), isError: true}
			}
			time.Sleep(2 * time.Second)
			if err := openVNCClient(svc.ProxyLocalPort); err != nil {
				return statusMsg{message: fmt.Sprintf("VNC client failed: %v", err), isError: true}
			}
			return statusMsg{
				message: fmt.Sprintf("VNC proxy started on localhost:%d", svc.ProxyLocalPort),
			}
		}

	case "rdp":
		if svc.ProxyLocalPort == 0 {
			m.status = "No proxy port configured for RDP"
			m.isError = true
			return m, nil
		}

		if err := checkRDPClient(); err != nil {
			m.status = err.Error()
			m.isError = true
			return m, nil
		}

		if m.mgr.IsRunning(svc.Hostname) {
			return m, func() tea.Msg {
				if err := openRDPClient(svc.ProxyLocalPort); err != nil {
					return statusMsg{message: fmt.Sprintf("RDP client failed: %v", err), isError: true}
				}
				return statusMsg{message: fmt.Sprintf("RDP opened on localhost:%d", svc.ProxyLocalPort)}
			}
		}

		return m, func() tea.Msg {
			if err := m.mgr.StartProxy(svc.Hostname, svc.ProxyLocalPort); err != nil {
				return statusMsg{message: fmt.Sprintf("Proxy failed: %v", err), isError: true}
			}
			time.Sleep(2 * time.Second)
			if err := openRDPClient(svc.ProxyLocalPort); err != nil {
				return statusMsg{message: fmt.Sprintf("RDP client failed: %v", err), isError: true}
			}
			return statusMsg{
				message: fmt.Sprintf("RDP proxy started on localhost:%d", svc.ProxyLocalPort),
			}
		}

	case "novnc":
		url := svc.URL
		if url == "" {
			url = fmt.Sprintf("https://%s", svc.Hostname)
		}
		return m, func() tea.Msg {
			if err := openURL(url); err != nil {
				return statusMsg{message: fmt.Sprintf("Failed to open browser: %v", err), isError: true}
			}
			return statusMsg{message: fmt.Sprintf("Opened noVNC: %s", url)}
		}

	case "http":
		url := svc.URL
		if url == "" {
			url = fmt.Sprintf("https://%s", svc.Hostname)
		}
		return m, func() tea.Msg {
			if err := openURL(url); err != nil {
				return statusMsg{message: fmt.Sprintf("Failed to open browser: %v", err), isError: true}
			}
			return statusMsg{message: fmt.Sprintf("Opened %s", url)}
		}
	}

	return m, nil
}

func (m Model) View() string {
	if m.width == 0 {
		return "Loading..."
	}

	var b strings.Builder

	switch m.view {
	case viewDashboard:
		m.renderDashboard(&b)
	case viewDevice:
		m.renderDevice(&b)
	}

	// Status bar at bottom
	if m.status != "" {
		b.WriteString("\n")
		if m.isError {
			b.WriteString(errorStyle.Render(m.status))
		} else {
			b.WriteString(successStyle.Render(m.status))
		}
	}

	return b.String()
}

func (m Model) renderDashboard(b *strings.Builder) {
	b.WriteString(titleStyle.Render("⚡ Cloudflare Tunnel Manager"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(m.cfg.Domain))
	b.WriteString("\n\n")

	// Header
	b.WriteString(headerStyle.Render(
		fmt.Sprintf("  %-18s %-12s %s", "DEVICE", "TYPE", "SERVICES"),
	))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width-2, 60))))
	b.WriteString("\n")

	for i, dev := range m.cfg.Devices {
		types := []string{}
		for _, svc := range dev.Services {
			types = append(types, svc.Type)
		}
		typeStr := strings.Join(unique(types), ",")

		svcNames := []string{}
		for _, svc := range dev.Services {
			svcNames = append(svcNames, svc.Name)
		}
		svcStr := strings.Join(svcNames, ", ")
		if len(svcStr) > 25 {
			svcStr = svcStr[:22] + "..."
		}

		line := fmt.Sprintf("%s %-18s %-12s %s",
			statusOK.String(),
			dev.Name,
			typeStr,
			svcStr,
		)

		if i == m.cursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(normalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • enter select • q quit"))
}

func (m Model) renderDevice(b *strings.Builder) {
	dev := m.cfg.Devices[m.cursor]

	b.WriteString(titleStyle.Render(fmt.Sprintf("← %s", dev.Name)))
	b.WriteString("\n\n")

	b.WriteString(headerStyle.Render(
		fmt.Sprintf("  %-15s %-35s %-8s %s", "SERVICE", "HOSTNAME", "TYPE", "STATUS"),
	))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width-2, 75))))
	b.WriteString("\n")

	for i, svc := range dev.Services {
		proxyStatus := ""
		if (svc.Type == "vnc" || svc.Type == "rdp") && m.mgr.IsRunning(svc.Hostname) {
			proxyStatus = fmt.Sprintf("%s proxy:%d", statusProxy.String(), svc.ProxyLocalPort)
		}

		line := fmt.Sprintf("%s %-15s %-35s %-8s %s",
			statusOK.String(),
			svc.Name,
			svc.Hostname,
			svc.Type,
			proxyStatus,
		)

		if i == m.dcursor {
			b.WriteString(selectedStyle.Render(line))
		} else {
			b.WriteString(normalStyle.Render(line))
		}
		b.WriteString("\n")
	}

	if m.showCreds {
		svc := dev.Services[m.dcursor]
		b.WriteString("\n")
		b.WriteString(dimStyle.Render(strings.Repeat("─", min(m.width-2, 75))))
		b.WriteString("\n")
		b.WriteString(credStyle.Render("Credentials"))
		b.WriteString("\n")
		if svc.User != "" {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  User:     %s", svc.User)))
			b.WriteString("\n")
		}
		if svc.Password != "" {
			b.WriteString(normalStyle.Render(fmt.Sprintf("  Password: %s", svc.Password)))
			b.WriteString("\n")
		}
		if svc.User == "" && svc.Password == "" {
			b.WriteString(dimStyle.Render("  No credentials configured"))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(helpStyle.Render("↑/↓ navigate • enter connect • c creds • b back • q quit"))
}

func unique(ss []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, s := range ss {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
