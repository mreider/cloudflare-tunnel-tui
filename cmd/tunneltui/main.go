package main

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/mreider/cloudflare-tunnel-tui/internal/config"
	"github.com/mreider/cloudflare-tunnel-tui/internal/tui"
	"github.com/mreider/cloudflare-tunnel-tui/internal/tunnel"
	"golang.org/x/term"
)

func usage() {
	fmt.Fprintf(os.Stderr, `Usage:
  tunneltui <config.enc>                  Run TUI with encrypted config bundle
  tunneltui --bundle <config.yaml>        Encrypt a YAML config into a .enc bundle
  tunneltui --generate [file]             Generate a sample config.yaml template
  tunneltui --help                        Show this help
`)
}

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--help", "-h":
		usage()
		return
	case "--generate":
		outFile := "config.yaml"
		if len(os.Args) >= 3 {
			outFile = os.Args[2]
		}
		generateTemplate(outFile)
	case "--bundle":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: tunneltui --bundle <config.yaml>")
			os.Exit(1)
		}
		bundleConfig(os.Args[2])
	default:
		runTUI(os.Args[1])
	}
}

func readPassword(prompt string) string {
	fmt.Fprint(os.Stderr, prompt)
	pw, err := term.ReadPassword(int(syscall.Stdin))
	fmt.Fprintln(os.Stderr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading password: %v\n", err)
		os.Exit(1)
	}
	return string(pw)
}

func generateTemplate(outFile string) {
	if _, err := os.Stat(outFile); err == nil {
		fmt.Fprintf(os.Stderr, "Error: %s already exists (won't overwrite)\n", outFile)
		os.Exit(1)
	}

	template := `# Cloudflare Tunnel TUI - Configuration
#
# This file defines your devices and services for the TUI dashboard.
# After editing, encrypt it into a bundle:
#
#   tunneltui --bundle config.yaml
#
# Then run the TUI with:
#
#   tunneltui config.enc

# Your Cloudflare domain (the domain your tunnels serve traffic on)
domain: "example.com"

# Path to the cloudflared binary (or just "cloudflared" if it's in your PATH)
cloudflared_bin: "cloudflared"

devices:
  # Each device represents a machine reachable through a Cloudflare Tunnel.
  # A device can have multiple services (SSH, VNC, RDP, HTTP).

  - name: "Web Server"
    services:
      # SSH - connects via ProxyCommand in ~/.ssh/config
      # Requires: ssh config entry with "ProxyCommand cloudflared access ssh --hostname %h"
      - name: "SSH"
        hostname: "ssh.example.com"       # Must match your tunnel's public hostname
        type: "ssh"
        user: "deploy"                    # SSH username
        password: "s3cur3-passw0rd"       # Optional - shown in TUI with 'c' key

      # HTTP - opens a URL in your default browser
      - name: "Admin Panel"
        hostname: "admin.example.com"
        type: "http"
        url: "https://admin.example.com"  # Optional - defaults to https://<hostname>

  - name: "Desktop"
    services:
      - name: "SSH"
        hostname: "ssh-desktop.example.com"
        type: "ssh"
        user: "matt"
        password: "my-password"

      # VNC - starts a cloudflared TCP proxy, then opens a VNC client
      # macOS: uses built-in Screen Sharing.app; Linux: uses vncviewer or xdg-open
      - name: "VNC"
        hostname: "vnc-desktop.example.com"
        type: "vnc"
        proxy_local_port: 15900           # Local port for the cloudflared TCP proxy
        user: "matt"                      # Optional - for credential display only
        password: "vnc-password"          # Optional - for credential display only

      # RDP - starts a cloudflared TCP proxy, then opens an RDP client
      # macOS: requires "Microsoft Remote Desktop" or "Windows App" from the App Store
      # Linux: requires xfreerdp (freerdp2-x11) or remmina
      - name: "Remote Desktop"
        hostname: "rdp-desktop.example.com"
        type: "rdp"
        proxy_local_port: 15901           # Must be unique across all services
        user: "matt"
        password: "rdp-password"

      # noVNC - browser-based VNC via Cloudflare Tunnel (no proxy or client needed)
      # Server-side requires: x11vnc + noVNC + websockify, with the tunnel ingress
      # pointing to the noVNC HTTP port (e.g., http://localhost:6080)
      - name: "Browser VNC"
        hostname: "novnc-desktop.example.com"
        type: "novnc"
        user: "matt"
        password: "vnc-password"

  - name: "NAS"
    services:
      - name: "SSH"
        hostname: "ssh-nas.example.com"
        type: "ssh"
        user: "admin"
        password: "nas-admin-pw"

      - name: "Web UI"
        hostname: "nas.example.com"
        type: "http"
`

	if err := os.WriteFile(outFile, []byte(template), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing %s: %v\n", outFile, err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Generated: %s\n", outFile)
	fmt.Fprintf(os.Stderr, "\nNext steps:\n")
	fmt.Fprintf(os.Stderr, "  1. Edit %s with your devices and services\n", outFile)
	fmt.Fprintf(os.Stderr, "  2. tunneltui --bundle %s\n", outFile)
	fmt.Fprintf(os.Stderr, "  3. tunneltui %s\n", strings.TrimSuffix(strings.TrimSuffix(outFile, ".yaml"), ".yml")+".enc")
}

func bundleConfig(yamlPath string) {
	cfg, err := config.LoadYAML(yamlPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %v\n", yamlPath, err)
		os.Exit(1)
	}

	pw := readPassword("Enter password for bundle: ")
	pw2 := readPassword("Confirm password: ")

	if pw != pw2 {
		fmt.Fprintln(os.Stderr, "Passwords do not match")
		os.Exit(1)
	}

	if len(pw) < 8 {
		fmt.Fprintln(os.Stderr, "Password must be at least 8 characters")
		os.Exit(1)
	}

	outPath := strings.TrimSuffix(yamlPath, ".yaml")
	outPath = strings.TrimSuffix(outPath, ".yml")
	outPath += ".enc"

	if err := config.SaveEncrypted(cfg, outPath, pw); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating bundle: %v\n", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stderr, "Bundle created: %s\n", outPath)
	fmt.Fprintf(os.Stderr, "Encryption: AES-256-GCM + Argon2id\n")
	fmt.Fprintf(os.Stderr, "\nTo run: tunneltui %s\n", outPath)
}

func runTUI(configPath string) {
	pw := readPassword("Password: ")

	cfg, err := config.LoadEncrypted(configPath, pw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	bin := cfg.CloudflaredBin
	if bin == "" {
		bin = "cloudflared"
	}

	mgr := tunnel.NewManager(bin)
	defer mgr.StopAll()

	model := tui.New(cfg, mgr)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
