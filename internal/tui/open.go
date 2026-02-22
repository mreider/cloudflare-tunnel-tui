package tui

import (
	"fmt"
	"os/exec"
	"runtime"
)

// openURL opens a URL in the default browser.
func openURL(url string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", url).Run()
	default: // linux and others
		return exec.Command("xdg-open", url).Run()
	}
}

// openVNCClient opens a VNC client connected to localhost:port.
func openVNCClient(port int) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", fmt.Sprintf("vnc://localhost:%d", port)).Run()
	default:
		// Try vncviewer (tigervnc / realvnc) first
		if path, err := exec.LookPath("vncviewer"); err == nil {
			return exec.Command(path, fmt.Sprintf("localhost:%d", port)).Start()
		}
		// Fall back to xdg-open with vnc:// URI
		return exec.Command("xdg-open", fmt.Sprintf("vnc://localhost:%d", port)).Run()
	}
}

// openRDPClient opens an RDP client connected to localhost:port.
func openRDPClient(port int) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", fmt.Sprintf("rdp://localhost:%d", port)).Run()
	default:
		if path, err := exec.LookPath("xfreerdp"); err == nil {
			return exec.Command(path, fmt.Sprintf("/v:localhost:%d", port)).Start()
		}
		if path, err := exec.LookPath("xfreerdp3"); err == nil {
			return exec.Command(path, fmt.Sprintf("/v:localhost:%d", port)).Start()
		}
		if path, err := exec.LookPath("remmina"); err == nil {
			return exec.Command(path, fmt.Sprintf("rdp://localhost:%d", port)).Start()
		}
		return fmt.Errorf("no RDP client found — install xfreerdp or remmina")
	}
}

// checkRDPClient verifies an RDP client is available.
func checkRDPClient() error {
	switch runtime.GOOS {
	case "darwin":
		if exec.Command("open", "-Ra", "Microsoft Remote Desktop").Run() == nil {
			return nil
		}
		if exec.Command("open", "-Ra", "Windows App").Run() == nil {
			return nil
		}
		return fmt.Errorf("no RDP client found — install \"Microsoft Remote Desktop\" from the App Store")
	default:
		if _, err := exec.LookPath("xfreerdp"); err == nil {
			return nil
		}
		if _, err := exec.LookPath("xfreerdp3"); err == nil {
			return nil
		}
		if _, err := exec.LookPath("remmina"); err == nil {
			return nil
		}
		return fmt.Errorf("no RDP client found — install xfreerdp (freerdp2-x11) or remmina")
	}
}
