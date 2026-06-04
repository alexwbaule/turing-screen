// Package volume provides a PulseAudio/PipeWire volume reader
// using the native protocol (no external binaries needed).
package volume

import (
	"fmt"
	"os"
	"path/filepath"

	"mrogalski.eu/go/pulseaudio"
)

type Client struct {
	pa *pulseaudio.Client
}

// NewClient connects to PulseAudio/PipeWire via native socket.
// Handles running as root/sudo by finding the real user's socket.
func NewClient() (*Client, error) {
	socket := findSocket()

	// If running as root, set HOME to the real user's home for cookie auth
	if os.Getuid() == 0 {
		if sudoUser := os.Getenv("SUDO_USER"); sudoUser != "" {
			os.Setenv("HOME", "/home/"+sudoUser)
		} else {
			// Try to find any user with a pulse cookie
			entries, _ := os.ReadDir("/home")
			for _, e := range entries {
				cookie := filepath.Join("/home", e.Name(), ".config", "pulse", "cookie")
				if _, err := os.Stat(cookie); err == nil {
					os.Setenv("HOME", filepath.Join("/home", e.Name()))
					break
				}
			}
		}
	}

	var pa *pulseaudio.Client
	var err error

	if socket != "" {
		pa, err = pulseaudio.NewClient(socket)
	} else {
		pa, err = pulseaudio.NewClient()
	}
	if err != nil {
		return nil, fmt.Errorf("connect to pulseaudio: %w", err)
	}
	return &Client{pa: pa}, nil
}

func findSocket() string {
	// Try XDG_RUNTIME_DIR first
	if dir := os.Getenv("XDG_RUNTIME_DIR"); dir != "" {
		path := filepath.Join(dir, "pulse", "native")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// If running as root (sudo), use SUDO_UID
	if sudoUID := os.Getenv("SUDO_UID"); sudoUID != "" {
		path := filepath.Join("/run/user", sudoUID, "pulse", "native")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Scan /run/user/ for any pulse socket
	entries, _ := os.ReadDir("/run/user")
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		path := filepath.Join("/run/user", e.Name(), "pulse", "native")
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// GetVolume returns the default sink volume as percentage (0-100+).
func (c *Client) GetVolume() (int, error) {
	vol, err := c.pa.Volume()
	if err != nil {
		return 0, err
	}
	return int(vol * 100), nil
}

// GetMuted returns whether the default sink is muted.
func (c *Client) GetMuted() (bool, error) {
	return c.pa.Mute()
}

// Close closes the PulseAudio connection.
func (c *Client) Close() {
	if c.pa != nil {
		c.pa.Close()
	}
}
