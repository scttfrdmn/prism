package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(_ *http.Request) bool {
		// Allow connections from the same origin (GUI frontend)
		return true
	},
}

// TerminalSize represents terminal dimensions
type TerminalSize struct {
	Rows uint32 `json:"rows"`
	Cols uint32 `json:"cols"`
}

// TerminalMessage represents a message from the frontend
type TerminalMessage struct {
	Type string        `json:"type"` // "input", "resize"
	Data string        `json:"data"` // Terminal input data
	Size *TerminalSize `json:"size"` // Terminal size for resize events
}

// TerminalSession manages a single terminal session
type TerminalSession struct {
	ws         *websocket.Conn
	sshClient  *ssh.Client
	sshSession *ssh.Session
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
	closed     bool
}

// HandleTerminalWebSocket handles WebSocket connections for terminal access
func (s *PrismService) HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	instanceName := r.URL.Query().Get("instance")
	if instanceName == "" {
		http.Error(w, "instance parameter required", http.StatusBadRequest)
		return
	}

	// Upgrade HTTP connection to WebSocket
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket: %v", err)
		return
	}

	// Get instance access information
	access, err := s.GetInstanceAccess(context.Background(), instanceName)
	if err != nil {
		_ = ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to get instance access: %v\r\n", err)))
		_ = ws.Close()
		return
	}

	if access.SSHPort == 0 {
		_ = ws.WriteMessage(websocket.TextMessage, []byte("Error: No SSH access available\r\n"))
		_ = ws.Close()
		return
	}

	// Create terminal session
	ctx, cancel := context.WithCancel(context.Background())
	session := &TerminalSession{
		ws:     ws,
		ctx:    ctx,
		cancel: cancel,
	}

	// Start SSH connection in background
	go session.connectSSH(access)
}

// connectSSH establishes SSH connection and sets up bidirectional data flow
func (ts *TerminalSession) connectSSH(access *InstanceAccess) {
	defer ts.cleanup()

	// Get SSH private key from standard location
	// For now, use the default key path - we'll need to enhance this to get the actual key name
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		ts.sendError("Failed to get home directory")
		return
	}

	// Try to find SSH key in ~/.ssh/ - try multiple common key names
	possibleKeys := []string{
		"cws-test-aws-west2-key",
		"cws-west2-key",
		"cws-aws-default-key",
		"cws-test-us-west-2-key.pem",
	}

	var privateKey []byte
	var privateKeyPath string
	for _, keyName := range possibleKeys {
		tryPath := filepath.Join(homeDir, ".ssh", keyName)
		data, err := os.ReadFile(tryPath)
		if err == nil {
			privateKey = data
			privateKeyPath = tryPath
			break
		}
	}

	if privateKey == nil {
		ts.sendError("Failed to find SSH private key. Tried: " + filepath.Join(homeDir, ".ssh", possibleKeys[0]))
		return
	}

	ts.sendMessage(fmt.Sprintf("Using SSH key: %s\r\n", filepath.Base(privateKeyPath)))

	signer, err := ssh.ParsePrivateKey(privateKey)
	if err != nil {
		ts.sendError(fmt.Sprintf("Failed to parse SSH private key: %v", err))
		return
	}

	// Debug: Show public key type
	ts.sendMessage(fmt.Sprintf("Key type: %s\r\n", signer.PublicKey().Type()))

	// SSH client configuration
	config := &ssh.ClientConfig{
		User: access.Username,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Proper host key verification
		Timeout:         10 * time.Second,
		Config: ssh.Config{
			// Support both modern and legacy RSA signature algorithms
			KeyExchanges: []string{
				"curve25519-sha256", "curve25519-sha256@libssh.org",
				"ecdh-sha2-nistp256", "ecdh-sha2-nistp384", "ecdh-sha2-nistp521",
				"diffie-hellman-group14-sha256", "diffie-hellman-group14-sha1",
			},
		},
	}

	// Connect to SSH server
	ts.sendMessage(fmt.Sprintf("Connecting to %s@%s:%d...\r\n", access.Username, access.PublicIP, access.SSHPort))
	addr := fmt.Sprintf("%s:%d", access.PublicIP, access.SSHPort)

	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		ts.sendError(fmt.Sprintf("Failed to connect via SSH: %v", err))
		ts.sendMessage(fmt.Sprintf("\r\nDebug: User=%s, Addr=%s, KeyLoaded=%v\r\n", access.Username, addr, signer != nil))
		return
	}
	ts.sshClient = sshClient
	defer func() { _ = sshClient.Close() }()

	// Create SSH session
	sshSession, err := sshClient.NewSession()
	if err != nil {
		ts.sendError(fmt.Sprintf("Failed to create SSH session: %v", err))
		return
	}
	ts.sshSession = sshSession
	defer func() { _ = sshSession.Close() }()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // Enable echoing
		ssh.TTY_OP_ISPEED: 14400, // Input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // Output speed = 14.4kbaud
	}

	// Request pseudo terminal
	if err := sshSession.RequestPty("xterm-256color", 40, 80, modes); err != nil {
		ts.sendError(fmt.Sprintf("Failed to request PTY: %v", err))
		return
	}

	// Set PATH environment variable before starting shell
	// This ensures commands are available when .bashrc runs
	if err := sshSession.Setenv("PATH", "/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/snap/bin"); err != nil {
		// Some SSH servers don't allow Setenv, but that's okay - we'll try anyway
		ts.sendMessage(fmt.Sprintf("Note: Could not set PATH via SSH (server may not allow it): %v\r\n", err))
	}

	// Set up pipes
	sshStdin, err := sshSession.StdinPipe()
	if err != nil {
		ts.sendError(fmt.Sprintf("Failed to get stdin pipe: %v", err))
		return
	}

	sshStdout, err := sshSession.StdoutPipe()
	if err != nil {
		ts.sendError(fmt.Sprintf("Failed to get stdout pipe: %v", err))
		return
	}

	sshStderr, err := sshSession.StderrPipe()
	if err != nil {
		ts.sendError(fmt.Sprintf("Failed to get stderr pipe: %v", err))
		return
	}

	// Start the user's default shell as a login shell
	// Shell() automatically requests a login shell which sources /etc/profile and ~/.bash_profile
	// Combined with PATH set above, this ensures environment is properly configured
	if err := sshSession.Shell(); err != nil {
		ts.sendError(fmt.Sprintf("Failed to start shell: %v", err))
		return
	}

	ts.sendMessage("Connected!\r\n")

	// Bidirectional data flow
	var wg sync.WaitGroup

	// SSH output -> WebSocket
	wg.Add(2)
	go ts.copyOutput(&wg, sshStdout)
	go ts.copyOutput(&wg, sshStderr)

	// WebSocket input -> SSH
	go ts.readWebSocketInput(sshStdin)

	// Wait for session to end
	wg.Wait()
	_ = sshSession.Wait()
}

// copyOutput copies SSH output to WebSocket
func (ts *TerminalSession) copyOutput(wg *sync.WaitGroup, reader io.Reader) {
	defer wg.Done()

	buf := make([]byte, 32*1024) // 32KB buffer
	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			n, err := reader.Read(buf)
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading SSH output: %v", err)
				}
				return
			}

			if n > 0 {
				ts.mu.Lock()
				if !ts.closed {
					if err := ts.ws.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
						log.Printf("Error writing to WebSocket: %v", err)
						ts.mu.Unlock()
						return
					}
				}
				ts.mu.Unlock()
			}
		}
	}
}

// readWebSocketInput reads input from WebSocket and writes to SSH
func (ts *TerminalSession) readWebSocketInput(stdin io.WriteCloser) {
	defer func() { _ = stdin.Close() }()

	for {
		select {
		case <-ts.ctx.Done():
			return
		default:
			_, message, err := ts.ws.ReadMessage()
			if err != nil {
				if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
					log.Printf("Error reading from WebSocket: %v", err)
				}
				ts.cancel()
				return
			}

			// Parse terminal message
			var termMsg TerminalMessage
			if err := json.Unmarshal(message, &termMsg); err != nil {
				// Assume raw input if JSON parsing fails
				if _, err := stdin.Write(message); err != nil {
					log.Printf("Error writing to SSH stdin: %v", err)
					return
				}
				continue
			}

			switch termMsg.Type {
			case "input":
				// Write user input to SSH
				if _, err := stdin.Write([]byte(termMsg.Data)); err != nil {
					log.Printf("Error writing to SSH stdin: %v", err)
					return
				}

			case "resize":
				// Handle terminal resize
				if termMsg.Size != nil && ts.sshSession != nil {
					if err := ts.sshSession.WindowChange(
						int(termMsg.Size.Rows),
						int(termMsg.Size.Cols),
					); err != nil {
						log.Printf("Error resizing terminal: %v", err)
					}
				}
			}
		}
	}
}

// sendMessage sends a text message to the WebSocket
func (ts *TerminalSession) sendMessage(msg string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if !ts.closed {
		_ = ts.ws.WriteMessage(websocket.TextMessage, []byte(msg))
	}
}

// sendError sends an error message to the WebSocket
func (ts *TerminalSession) sendError(msg string) {
	ts.sendMessage(fmt.Sprintf("\r\n\033[31mError:\033[0m %s\r\n", msg))
}

// cleanup closes all resources
func (ts *TerminalSession) cleanup() {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if ts.closed {
		return
	}

	ts.closed = true
	ts.cancel()

	if ts.sshSession != nil {
		_ = ts.sshSession.Close()
	}

	if ts.sshClient != nil {
		_ = ts.sshClient.Close()
	}

	if ts.ws != nil {
		_ = ts.ws.Close()
	}
}
