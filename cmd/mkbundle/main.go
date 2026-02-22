package main

import (
	"fmt"
	"os"

	"github.com/mreider/cloudflare-tunnel-tui/internal/config"
)

func main() {
	if len(os.Args) < 4 {
		fmt.Fprintln(os.Stderr, "Usage: mkbundle <config.yaml> <output.enc> <password>")
		os.Exit(1)
	}

	cfg, err := config.LoadYAML(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	if err := config.SaveEncrypted(cfg, os.Args[2], os.Args[3]); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Verify by decrypting
	cfg2, err := config.LoadEncrypted(os.Args[2], os.Args[3])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Verification failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Bundle created: %s\n", os.Args[2])
	fmt.Printf("Verified: %d devices, domain=%s\n", len(cfg2.Devices), cfg2.Domain)
	for _, d := range cfg2.Devices {
		fmt.Printf("  %s: %d services\n", d.Name, len(d.Services))
	}
}
