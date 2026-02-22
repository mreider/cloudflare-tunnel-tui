package tunnel

import (
	"fmt"
	"os"
	"os/exec"
	"sync"
)

type Proxy struct {
	Hostname  string
	LocalPort int
	Cmd       *exec.Cmd
}

type Manager struct {
	mu      sync.Mutex
	binary  string
	proxies map[string]*Proxy // keyed by hostname
}

func NewManager(cloudflaredBin string) *Manager {
	return &Manager{
		binary:  cloudflaredBin,
		proxies: make(map[string]*Proxy),
	}
}

func (m *Manager) StartProxy(hostname string, localPort int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.proxies[hostname]; ok {
		return nil // already running
	}

	cmd := exec.Command(m.binary, "access", "tcp",
		"--hostname", hostname,
		"--url", fmt.Sprintf("localhost:%d", localPort),
	)
	cmd.Stdout = nil
	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("starting proxy for %s: %w", hostname, err)
	}

	m.proxies[hostname] = &Proxy{
		Hostname:  hostname,
		LocalPort: localPort,
		Cmd:       cmd,
	}

	// Reap the process in background
	go func() {
		cmd.Wait()
		m.mu.Lock()
		delete(m.proxies, hostname)
		m.mu.Unlock()
	}()

	return nil
}

func (m *Manager) StopProxy(hostname string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if p, ok := m.proxies[hostname]; ok {
		p.Cmd.Process.Signal(os.Interrupt)
		delete(m.proxies, hostname)
	}
}

func (m *Manager) IsRunning(hostname string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.proxies[hostname]
	return ok
}

func (m *Manager) StopAll() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for hostname, p := range m.proxies {
		p.Cmd.Process.Signal(os.Interrupt)
		delete(m.proxies, hostname)
	}
}
