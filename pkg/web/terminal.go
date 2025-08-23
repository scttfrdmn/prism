package web

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

// TerminalServer provides web-based terminal access to instances
type TerminalServer struct {
	mu        sync.RWMutex
	sessions  map[string]*TerminalSession
	sshConfig *ssh.ClientConfig
}

// TerminalSession represents an active terminal session
type TerminalSession struct {
	ID         string
	InstanceID string
	SSHClient  *ssh.Client
	Session    *ssh.Session
	StdinPipe  io.WriteCloser
	StdoutPipe io.Reader
	StderrPipe io.Reader
	Connected  bool
	mu         sync.Mutex
}

// TerminalMessage represents a message between client and server
type TerminalMessage struct {
	Type string          `json:"type"` // connect, resize, input, output, error, close
	Data json.RawMessage `json:"data"`
}

// ConnectData contains connection parameters
type ConnectData struct {
	InstanceID string `json:"instance_id"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Username   string `json:"username"`
}

// ResizeData contains terminal resize parameters
type ResizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}

// InputData contains terminal input
type InputData struct {
	Data string `json:"data"`
}

// OutputData contains terminal output
type OutputData struct {
	Data string `json:"data"`
}

// NewTerminalServer creates a new terminal server
func NewTerminalServer(sshConfig *ssh.ClientConfig) *TerminalServer {
	return &TerminalServer{
		sessions:  make(map[string]*TerminalSession),
		sshConfig: sshConfig,
	}
}

// ServeHTTP handles terminal WebSocket connections
func (ts *TerminalServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// In a real implementation, you would upgrade to WebSocket here
	// For now, we'll implement a simple REST API for terminal operations

	switch r.Method {
	case http.MethodPost:
		ts.handleConnect(w, r)
	case http.MethodDelete:
		ts.handleDisconnect(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleConnect establishes a new SSH connection
func (ts *TerminalServer) handleConnect(w http.ResponseWriter, r *http.Request) {
	var connectData ConnectData
	if err := json.NewDecoder(r.Body).Decode(&connectData); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Create SSH connection
	addr := fmt.Sprintf("%s:%d", connectData.Host, connectData.Port)
	client, err := ssh.Dial("tcp", addr, ts.sshConfig)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to connect: %v", err), http.StatusInternalServerError)
		return
	}

	// Create session
	session, err := client.NewSession()
	if err != nil {
		client.Close()
		http.Error(w, fmt.Sprintf("Failed to create session: %v", err), http.StatusInternalServerError)
		return
	}

	// Set up pipes
	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		client.Close()
		http.Error(w, fmt.Sprintf("Failed to create stdin pipe: %v", err), http.StatusInternalServerError)
		return
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		client.Close()
		http.Error(w, fmt.Sprintf("Failed to create stdout pipe: %v", err), http.StatusInternalServerError)
		return
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		client.Close()
		http.Error(w, fmt.Sprintf("Failed to create stderr pipe: %v", err), http.StatusInternalServerError)
		return
	}

	// Request PTY
	if err := session.RequestPty("xterm-256color", 24, 80, ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}); err != nil {
		session.Close()
		client.Close()
		http.Error(w, fmt.Sprintf("Failed to request PTY: %v", err), http.StatusInternalServerError)
		return
	}

	// Start shell
	if err := session.Shell(); err != nil {
		session.Close()
		client.Close()
		http.Error(w, fmt.Sprintf("Failed to start shell: %v", err), http.StatusInternalServerError)
		return
	}

	// Create terminal session
	sessionID := generateSessionID()
	termSession := &TerminalSession{
		ID:         sessionID,
		InstanceID: connectData.InstanceID,
		SSHClient:  client,
		Session:    session,
		StdinPipe:  stdin,
		StdoutPipe: stdout,
		StderrPipe: stderr,
		Connected:  true,
	}

	// Store session
	ts.mu.Lock()
	ts.sessions[sessionID] = termSession
	ts.mu.Unlock()

	// Return session ID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"session_id": sessionID,
		"status":     "connected",
	})
}

// handleDisconnect closes an SSH connection
func (ts *TerminalServer) handleDisconnect(w http.ResponseWriter, r *http.Request) {
	sessionID := r.URL.Query().Get("session_id")
	if sessionID == "" {
		http.Error(w, "session_id required", http.StatusBadRequest)
		return
	}

	ts.mu.Lock()
	session, exists := ts.sessions[sessionID]
	if exists {
		delete(ts.sessions, sessionID)
	}
	ts.mu.Unlock()

	if !exists {
		http.Error(w, "Session not found", http.StatusNotFound)
		return
	}

	// Close SSH connection
	session.mu.Lock()
	if session.Session != nil {
		session.Session.Close()
	}
	if session.SSHClient != nil {
		_ = session.SSHClient.Close()
	}
	session.Connected = false
	session.mu.Unlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "disconnected",
	})
}

// LocalTerminal provides local terminal execution (for development)
type LocalTerminal struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
}

// NewLocalTerminal creates a new local terminal
func NewLocalTerminal() (*LocalTerminal, error) {
	cmd := exec.Command("/bin/bash")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	return &LocalTerminal{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
	}, nil
}

// Write sends input to the terminal
func (lt *LocalTerminal) Write(data []byte) (int, error) {
	return lt.stdin.Write(data)
}

// Read reads output from the terminal
func (lt *LocalTerminal) Read(p []byte) (int, error) {
	return lt.stdout.Read(p)
}

// Close terminates the terminal
func (lt *LocalTerminal) Close() error {
	if lt.cmd != nil && lt.cmd.Process != nil {
		return lt.cmd.Process.Kill()
	}
	return nil
}

// generateSessionID creates a unique session ID
func generateSessionID() string {
	return fmt.Sprintf("term-%d", time.Now().UnixNano())
}

// ServeTerminalHTML serves the terminal HTML interface
func ServeTerminalHTML(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(terminalHTML))
}

// HTML for the terminal interface
const terminalHTML = `<!DOCTYPE html>
<html>
<head>
    <title>CloudWorkstation Terminal</title>
    <link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@4.19.0/css/xterm.css">
    <style>
        body {
            margin: 0;
            padding: 20px;
            background: #1e1e1e;
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
        }
        .terminal-container {
            max-width: 1200px;
            margin: 0 auto;
        }
        .terminal-header {
            background: #2d2d2d;
            color: #fff;
            padding: 10px 15px;
            border-radius: 8px 8px 0 0;
            display: flex;
            justify-content: space-between;
            align-items: center;
        }
        .terminal-title {
            font-size: 14px;
            font-weight: 500;
        }
        .terminal-actions {
            display: flex;
            gap: 10px;
        }
        .terminal-btn {
            background: #3b82f6;
            color: white;
            border: none;
            padding: 6px 12px;
            border-radius: 4px;
            cursor: pointer;
            font-size: 12px;
        }
        .terminal-btn:hover {
            background: #2563eb;
        }
        .terminal-btn.danger {
            background: #ef4444;
        }
        .terminal-btn.danger:hover {
            background: #dc2626;
        }
        #terminal {
            background: #1e1e1e;
            padding: 10px;
            border-radius: 0 0 8px 8px;
        }
        .connection-form {
            background: #2d2d2d;
            padding: 20px;
            border-radius: 8px;
            margin-bottom: 20px;
        }
        .form-group {
            margin-bottom: 15px;
        }
        .form-group label {
            display: block;
            color: #9ca3af;
            margin-bottom: 5px;
            font-size: 14px;
        }
        .form-group input {
            width: 100%;
            padding: 8px 12px;
            background: #1e1e1e;
            border: 1px solid #4b5563;
            border-radius: 4px;
            color: white;
            font-size: 14px;
        }
        .status {
            padding: 4px 8px;
            border-radius: 4px;
            font-size: 12px;
            background: #374151;
        }
        .status.connected {
            background: #10b981;
        }
        .status.disconnected {
            background: #ef4444;
        }
    </style>
</head>
<body>
    <div class="terminal-container">
        <div class="connection-form" id="connectionForm">
            <h2 style="color: white; margin-top: 0;">Connect to Instance</h2>
            <div class="form-group">
                <label>Instance Host</label>
                <input type="text" id="host" placeholder="instance.example.com">
            </div>
            <div class="form-group">
                <label>SSH Port</label>
                <input type="number" id="port" value="22">
            </div>
            <div class="form-group">
                <label>Username</label>
                <input type="text" id="username" placeholder="ubuntu">
            </div>
            <button class="terminal-btn" onclick="connect()">Connect</button>
        </div>
        
        <div id="terminalWrapper" style="display: none;">
            <div class="terminal-header">
                <div class="terminal-title">
                    <span id="instanceName">Terminal</span>
                    <span id="connectionStatus" class="status disconnected">Disconnected</span>
                </div>
                <div class="terminal-actions">
                    <button class="terminal-btn" onclick="clearTerminal()">Clear</button>
                    <button class="terminal-btn danger" onclick="disconnect()">Disconnect</button>
                </div>
            </div>
            <div id="terminal"></div>
        </div>
    </div>

    <script src="https://cdn.jsdelivr.net/npm/xterm@4.19.0/lib/xterm.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.5.0/lib/xterm-addon-fit.js"></script>
    <script src="https://cdn.jsdelivr.net/npm/xterm-addon-web-links@0.6.0/lib/xterm-addon-web-links.js"></script>
    <script>
        let term;
        let fitAddon;
        let ws;
        let sessionId;

        function initTerminal() {
            term = new Terminal({
                cursorBlink: true,
                fontSize: 14,
                fontFamily: 'Menlo, Monaco, "Courier New", monospace',
                theme: {
                    background: '#1e1e1e',
                    foreground: '#ffffff',
                    cursor: '#ffffff',
                    selection: '#3b82f6'
                }
            });

            fitAddon = new FitAddon.FitAddon();
            term.loadAddon(fitAddon);
            
            const webLinksAddon = new WebLinksAddon.WebLinksAddon();
            term.loadAddon(webLinksAddon);

            term.open(document.getElementById('terminal'));
            fitAddon.fit();

            window.addEventListener('resize', () => fitAddon.fit());
        }

        function connect() {
            const host = document.getElementById('host').value;
            const port = document.getElementById('port').value;
            const username = document.getElementById('username').value;

            if (!host || !username) {
                alert('Please fill in all required fields');
                return;
            }

            document.getElementById('connectionForm').style.display = 'none';
            document.getElementById('terminalWrapper').style.display = 'block';
            
            initTerminal();
            
            // WebSocket connection would go here
            // For now, just show a message
            term.writeln('Connecting to ' + username + '@' + host + ':' + port + '...');
            term.writeln('\\r\\nNote: WebSocket terminal connection not implemented in this demo');
            term.writeln('Use "cws connect <instance-name>" for SSH access\\r\\n');
            
            document.getElementById('instanceName').textContent = host;
            document.getElementById('connectionStatus').textContent = 'Demo Mode';
            document.getElementById('connectionStatus').className = 'status connected';
            
            // Demo: Echo local input
            term.onData(data => {
                term.write(data);
            });
        }

        function disconnect() {
            if (ws) {
                ws.close();
            }
            if (term) {
                term.dispose();
            }
            document.getElementById('connectionForm').style.display = 'block';
            document.getElementById('terminalWrapper').style.display = 'none';
        }

        function clearTerminal() {
            if (term) {
                term.clear();
            }
        }
    </script>
</body>
</html>`
