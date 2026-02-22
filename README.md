# tunneltui

A terminal UI for managing SSH, VNC, and web connections through [Cloudflare Tunnels](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/).

Point it at an encrypted config bundle, enter your password, and get a dashboard of all your tunnel-connected devices with one-keystroke access to SSH sessions, VNC screen sharing, and web interfaces.

![Go](https://img.shields.io/badge/Go-1.23+-00ADD8?logo=go&logoColor=white)

## Features

- **Dashboard view** with all devices and their available services
- **SSH** — drops you into an SSH session (TUI suspends, resumes on exit)
- **VNC** — starts a `cloudflared access tcp` proxy and opens your VNC client
- **HTTP** — opens web services in your browser
- **Encrypted config** — AES-256-GCM + Argon2id. Config is decrypted in memory only, never written to disk
- **Single binary** — no runtime dependencies beyond `cloudflared` on the client machine
- **Proxy lifecycle management** — background proxies are tracked and cleaned up on exit

## Install

One-liner (macOS / Linux):

```bash
curl -sSL https://raw.githubusercontent.com/mreider/cloudflare-tunnel-tui/main/install.sh | sh
```

This detects your OS and architecture, downloads the latest `tunneltui` and `mkbundle` binaries, and installs them to `/usr/local/bin`.

To install to a different directory:

```bash
curl -sSL https://raw.githubusercontent.com/mreider/cloudflare-tunnel-tui/main/install.sh | INSTALL_DIR=~/.local/bin sh
```

Or download a specific binary from [Releases](../../releases). Available for:
- macOS (Intel / Apple Silicon)
- Linux (amd64 / arm64)

### From source

```bash
go install github.com/mreider/cloudflare-tunnel-tui/cmd/tunneltui@latest
go install github.com/mreider/cloudflare-tunnel-tui/cmd/mkbundle@latest
```

### Prerequisites

You need [`cloudflared`](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/downloads/) installed on the machine where you run `tunneltui`.

## Quick Start

### 1. Create a config file

Create a `config.yaml` describing your devices and services:

```yaml
domain: "example.com"
cloudflared_bin: "cloudflared"  # path to cloudflared binary

devices:
  - name: "Home Server"
    services:
      - name: "SSH"
        hostname: "ssh.example.com"
        type: "ssh"
        user: "myuser"
      - name: "VNC"
        hostname: "vnc.example.com"
        type: "vnc"
        proxy_local_port: 15900

  - name: "NAS"
    services:
      - name: "SSH"
        hostname: "ssh-nas.example.com"
        type: "ssh"
        user: "admin"
      - name: "Web UI"
        hostname: "nas.example.com"
        type: "http"
        url: "https://nas.example.com"
```

### 2. Encrypt it into a bundle

```bash
tunneltui --bundle config.yaml
# Enter a strong password (min 8 chars)
# Creates config.enc
```

Or use `mkbundle` for non-interactive / scripted usage:

```bash
mkbundle config.yaml config.enc "your-password"
```

**Delete the plaintext `config.yaml` after creating the bundle.**

### 3. Run

```bash
tunneltui config.enc
# Enter your password
```

## Config Reference

```yaml
domain: "example.com"           # Your Cloudflare domain
cloudflared_bin: "cloudflared"  # Path to cloudflared binary (default: "cloudflared")

devices:
  - name: "Device Name"        # Display name in TUI
    services:
      - name: "Service Name"   # Display name
        hostname: "x.example.com"  # Cloudflare tunnel hostname
        type: "ssh"            # ssh | vnc | rdp | http
        user: "username"       # SSH user (ssh type only)
        proxy_local_port: 15900  # Local port for VNC/RDP proxy (vnc/rdp type only)
        url: "https://..."     # Override URL for browser (http type only)
```

### Service Types

| Type | What it does |
|------|-------------|
| `ssh` | Runs `ssh -l <user> <hostname>`. Requires `~/.ssh/config` with a `ProxyCommand` for `cloudflared access ssh`. TUI suspends during the session. |
| `vnc` | Starts `cloudflared access tcp` proxy in the background, then opens `vnc://localhost:<port>`. |
| `rdp` | Starts `cloudflared access tcp` proxy in the background, then opens `rdp://localhost:<port>`. Works with GNOME Remote Desktop (Wayland) or any RDP server. |
| `http` | Opens the URL in your default browser. |

## SSH Setup

For SSH services to work through Cloudflare Tunnel, add entries to your `~/.ssh/config`:

```
Host ssh.example.com
    ProxyCommand cloudflared access ssh --hostname %h
    User myuser

Host ssh-nas.example.com
    ProxyCommand cloudflared access ssh --hostname %h
    User admin
```

## Encryption Details

The config bundle uses:

- **AES-256-GCM** for authenticated encryption
- **Argon2id** for key derivation (64 MB memory, 3 iterations, 4 threads)
- Random 16-byte salt and 12-byte nonce per bundle
- Config is decrypted in memory and never written to disk in plaintext

The bundle format is: `salt (16 bytes) || nonce (12 bytes) || AES-GCM ciphertext + tag`

## Keyboard Shortcuts

### Dashboard

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Select device |
| `q` | Quit |

### Device Detail

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `Enter` | Connect to service |
| `b` / `Esc` | Back to dashboard |
| `q` | Quit |

## Server-Side Setup

Each device you want to access needs:

1. **`cloudflared` installed** and running as a tunnel
2. **DNS routes** pointing hostnames to the tunnel
3. **Services** running (SSH server, VNC server, web server, etc.)

Example tunnel config on a server (`/etc/cloudflared/config.yml`):

```yaml
tunnel: <TUNNEL_ID>
credentials-file: /etc/cloudflared/<TUNNEL_ID>.json

ingress:
  - hostname: ssh.example.com
    service: ssh://localhost:22
  - hostname: vnc.example.com
    service: tcp://localhost:5900
  - service: http_status:404
```

## License

MIT
