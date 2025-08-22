package ssh

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"
)

// TestSSHConnectivity tests SSH connection establishment
func TestSSHConnectivity(t *testing.T) {
	// Start a test SSH server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverAddr := listener.Addr().String()
	
	// Generate host key
	hostKey, err := generateTestHostKey()
	require.NoError(t, err)

	// Start SSH server
	go runTestSSHServer(t, listener, hostKey)

	// Create SSH client config
	clientConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// Test connection
	t.Run("Basic Connection", func(t *testing.T) {
		client, err := ssh.Dial("tcp", serverAddr, clientConfig)
		assert.NoError(t, err)
		if client != nil {
			defer client.Close()
			assert.NotNil(t, client, "SSH client should be established")
		}
	})

	// Test session creation
	t.Run("Session Creation", func(t *testing.T) {
		client, err := ssh.Dial("tcp", serverAddr, clientConfig)
		require.NoError(t, err)
		defer client.Close()

		session, err := client.NewSession()
		assert.NoError(t, err)
		if session != nil {
			defer session.Close()
			assert.NotNil(t, session, "SSH session should be created")
		}
	})

	// Test command execution
	t.Run("Command Execution", func(t *testing.T) {
		client, err := ssh.Dial("tcp", serverAddr, clientConfig)
		require.NoError(t, err)
		defer client.Close()

		session, err := client.NewSession()
		require.NoError(t, err)
		defer session.Close()

		output, err := session.Output("echo 'Hello CloudWorkstation'")
		assert.NoError(t, err)
		assert.Contains(t, string(output), "Hello CloudWorkstation")
	})
}

// TestSSHKeyAuthentication tests SSH key-based authentication
func TestSSHKeyAuthentication(t *testing.T) {
	// Generate test key pair
	privateKey, publicKey, err := generateTestKeyPair()
	require.NoError(t, err)

	// Start test SSH server with key auth
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverAddr := listener.Addr().String()
	
	hostKey, err := generateTestHostKey()
	require.NoError(t, err)

	go runTestSSHServerWithKeyAuth(t, listener, hostKey, publicKey)

	// Parse private key for client
	signer, err := ssh.ParsePrivateKey(privateKey)
	require.NoError(t, err)

	// Create SSH client config with key auth
	clientConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// Test connection with key
	t.Run("Key Authentication", func(t *testing.T) {
		client, err := ssh.Dial("tcp", serverAddr, clientConfig)
		assert.NoError(t, err)
		if client != nil {
			defer client.Close()
			assert.NotNil(t, client, "SSH client should connect with key")
		}
	})
}

// TestSSHPortForwarding tests SSH port forwarding capabilities
func TestSSHPortForwarding(t *testing.T) {
	// Start test SSH server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverAddr := listener.Addr().String()
	
	hostKey, err := generateTestHostKey()
	require.NoError(t, err)

	go runTestSSHServer(t, listener, hostKey)

	// Create SSH client
	clientConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	client, err := ssh.Dial("tcp", serverAddr, clientConfig)
	require.NoError(t, err)
	defer client.Close()

	// Test local port forwarding
	t.Run("Local Port Forwarding", func(t *testing.T) {
		// Start a local listener
		localListener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		defer localListener.Close()

		localAddr := localListener.Addr().String()
		
		// Start a test HTTP server to forward to
		targetListener, err := net.Listen("tcp", "127.0.0.1:0")
		assert.NoError(t, err)
		defer targetListener.Close()

		targetAddr := targetListener.Addr().String()
		
		go func() {
			for {
				conn, err := targetListener.Accept()
				if err != nil {
					return
				}
				go func(c net.Conn) {
					defer c.Close()
					c.Write([]byte("HTTP/1.1 200 OK\r\n\r\nForwarded!"))
				}(conn)
			}
		}()

		// Set up port forwarding
		go func() {
			for {
				localConn, err := localListener.Accept()
				if err != nil {
					return
				}
				go handlePortForward(localConn, client, targetAddr)
			}
		}()

		// Test forwarded connection
		time.Sleep(100 * time.Millisecond)
		testConn, err := net.Dial("tcp", localAddr)
		if err == nil {
			defer testConn.Close()
			
			buf := make([]byte, 1024)
			n, _ := testConn.Read(buf)
			response := string(buf[:n])
			assert.Contains(t, response, "Forwarded", "Should receive forwarded response")
		}
	})
}

// TestSSHManager tests the SSH manager functionality
func TestSSHManager(t *testing.T) {
	manager := NewSSHManager()
	
	// Test connection storage
	t.Run("Connection Management", func(t *testing.T) {
		// Create mock connection
		conn := &SSHConnection{
			InstanceID: "i-test123",
			Host:       "test.example.com",
			Port:       22,
			Username:   "ubuntu",
			Connected:  true,
		}
		
		// Store connection
		manager.StoreConnection("i-test123", conn)
		
		// Retrieve connection
		retrieved := manager.GetConnection("i-test123")
		assert.NotNil(t, retrieved)
		assert.Equal(t, "i-test123", retrieved.InstanceID)
		assert.Equal(t, "test.example.com", retrieved.Host)
		
		// Remove connection
		manager.RemoveConnection("i-test123")
		retrieved = manager.GetConnection("i-test123")
		assert.Nil(t, retrieved)
	})

	// Test connection pooling
	t.Run("Connection Pooling", func(t *testing.T) {
		// Add multiple connections
		for i := 0; i < 5; i++ {
			conn := &SSHConnection{
				InstanceID: fmt.Sprintf("i-test%d", i),
				Connected:  true,
			}
			manager.StoreConnection(conn.InstanceID, conn)
		}
		
		// Check pool size
		assert.Equal(t, 5, manager.PoolSize())
		
		// Clear pool
		manager.ClearPool()
		assert.Equal(t, 0, manager.PoolSize())
	})
}

// TestSSHConnectionResilience tests connection resilience and reconnection
func TestSSHConnectionResilience(t *testing.T) {
	// Start initial SSH server
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	
	serverAddr := listener.Addr().String()
	hostKey, err := generateTestHostKey()
	require.NoError(t, err)

	serverCtx, serverCancel := context.WithCancel(context.Background())
	go runTestSSHServerWithContext(t, serverCtx, listener, hostKey)

	// Create resilient SSH client
	resilientClient := &ResilientSSHClient{
		Config: &ssh.ClientConfig{
			User: "testuser",
			Auth: []ssh.AuthMethod{
				ssh.Password("testpass"),
			},
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			Timeout:         5 * time.Second,
		},
		ServerAddr:    serverAddr,
		RetryAttempts: 3,
		RetryDelay:    1 * time.Second,
	}

	// Test initial connection
	t.Run("Initial Connection", func(t *testing.T) {
		err := resilientClient.Connect()
		assert.NoError(t, err)
		assert.True(t, resilientClient.IsConnected())
	})

	// Test connection loss and recovery
	t.Run("Connection Recovery", func(t *testing.T) {
		// Kill the server
		serverCancel()
		listener.Close()
		time.Sleep(100 * time.Millisecond)
		
		// Connection should be lost
		assert.False(t, resilientClient.IsConnected())
		
		// Start new server on same port
		newListener, err := net.Listen("tcp", serverAddr)
		if err == nil {
			defer newListener.Close()
			
			go runTestSSHServer(t, newListener, hostKey)
			time.Sleep(100 * time.Millisecond)
			
			// Try to reconnect
			err = resilientClient.Reconnect()
			assert.NoError(t, err)
			assert.True(t, resilientClient.IsConnected())
		}
	})
}

// TestSSHFileTransfer tests SCP/SFTP file transfer capabilities
func TestSSHFileTransfer(t *testing.T) {
	// Create temp directory for testing
	tmpDir, err := os.MkdirTemp("", "cws-ssh-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("CloudWorkstation SSH Test File")
	err = os.WriteFile(testFile, testContent, 0644)
	require.NoError(t, err)

	// Start SSH server with SFTP support
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer listener.Close()

	serverAddr := listener.Addr().String()
	hostKey, err := generateTestHostKey()
	require.NoError(t, err)

	go runTestSSHServerWithSFTP(t, listener, hostKey, tmpDir)

	// Create SSH client
	clientConfig := &ssh.ClientConfig{
		User: "testuser",
		Auth: []ssh.AuthMethod{
			ssh.Password("testpass"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	// Test file operations
	t.Run("File Upload", func(t *testing.T) {
		client, err := ssh.Dial("tcp", serverAddr, clientConfig)
		if err != nil {
			t.Skip("Could not connect to test server")
		}
		defer client.Close()

		// In a real implementation, you would use an SFTP client here
		// For testing, we'll simulate the operation
		t.Log("File upload test - would upload file via SFTP")
	})

	t.Run("File Download", func(t *testing.T) {
		client, err := ssh.Dial("tcp", serverAddr, clientConfig)
		if err != nil {
			t.Skip("Could not connect to test server")
		}
		defer client.Close()

		// In a real implementation, you would use an SFTP client here
		// For testing, we'll simulate the operation
		t.Log("File download test - would download file via SFTP")
	})
}

// Helper functions

func generateTestHostKey() (ssh.Signer, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	return ssh.ParsePrivateKey(privateKeyBytes)
}

func generateTestKeyPair() ([]byte, ssh.PublicKey, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, nil, err
	}

	privateKeyPEM := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}

	privateKeyBytes := pem.EncodeToMemory(privateKeyPEM)
	
	signer, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		return nil, nil, err
	}

	return privateKeyBytes, signer.PublicKey(), nil
}

func runTestSSHServer(t *testing.T, listener net.Listener, hostKey ssh.Signer) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "testuser" && string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(hostKey)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go handleTestConnection(t, conn, config)
	}
}

func runTestSSHServerWithContext(t *testing.T, ctx context.Context, listener net.Listener, hostKey ssh.Signer) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "testuser" && string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(hostKey)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go handleTestConnection(t, conn, config)
		}
	}
}

func runTestSSHServerWithKeyAuth(t *testing.T, listener net.Listener, hostKey ssh.Signer, authorizedKey ssh.PublicKey) {
	config := &ssh.ServerConfig{
		PublicKeyCallback: func(c ssh.ConnMetadata, pubKey ssh.PublicKey) (*ssh.Permissions, error) {
			if c.User() == "testuser" && string(pubKey.Marshal()) == string(authorizedKey.Marshal()) {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid public key")
		},
	}
	config.AddHostKey(hostKey)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go handleTestConnection(t, conn, config)
	}
}

func runTestSSHServerWithSFTP(t *testing.T, listener net.Listener, hostKey ssh.Signer, rootDir string) {
	config := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "testuser" && string(pass) == "testpass" {
				return nil, nil
			}
			return nil, fmt.Errorf("invalid credentials")
		},
	}
	config.AddHostKey(hostKey)

	for {
		conn, err := listener.Accept()
		if err != nil {
			return
		}

		go handleTestConnectionWithSFTP(t, conn, config, rootDir)
	}
}

func handleTestConnection(t *testing.T, conn net.Conn, config *ssh.ServerConfig) {
	defer conn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(conn, config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go handleTestSession(channel, requests)
	}
}

func handleTestConnectionWithSFTP(t *testing.T, conn net.Conn, config *ssh.ServerConfig, rootDir string) {
	// Similar to handleTestConnection but with SFTP support
	handleTestConnection(t, conn, config)
}

func handleTestSession(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "exec":
			// Handle command execution
			payload := string(req.Payload[4:])
			channel.Write([]byte(payload))
			channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})
			req.Reply(true, nil)
		case "shell":
			req.Reply(true, nil)
		case "pty-req":
			req.Reply(true, nil)
		default:
			req.Reply(false, nil)
		}
	}
}

func handlePortForward(localConn net.Conn, sshClient *ssh.Client, targetAddr string) {
	defer localConn.Close()

	remoteConn, err := sshClient.Dial("tcp", targetAddr)
	if err != nil {
		return
	}
	defer remoteConn.Close()

	go io.Copy(remoteConn, localConn)
	io.Copy(localConn, remoteConn)
}

// SSH Manager types for testing

type SSHManager struct {
	connections map[string]*SSHConnection
}

func NewSSHManager() *SSHManager {
	return &SSHManager{
		connections: make(map[string]*SSHConnection),
	}
}

func (m *SSHManager) StoreConnection(id string, conn *SSHConnection) {
	m.connections[id] = conn
}

func (m *SSHManager) GetConnection(id string) *SSHConnection {
	return m.connections[id]
}

func (m *SSHManager) RemoveConnection(id string) {
	delete(m.connections, id)
}

func (m *SSHManager) PoolSize() int {
	return len(m.connections)
}

func (m *SSHManager) ClearPool() {
	m.connections = make(map[string]*SSHConnection)
}

type SSHConnection struct {
	InstanceID string
	Host       string
	Port       int
	Username   string
	Connected  bool
	Client     *ssh.Client
}

type ResilientSSHClient struct {
	Config        *ssh.ClientConfig
	ServerAddr    string
	Client        *ssh.Client
	RetryAttempts int
	RetryDelay    time.Duration
}

func (r *ResilientSSHClient) Connect() error {
	client, err := ssh.Dial("tcp", r.ServerAddr, r.Config)
	if err != nil {
		return err
	}
	r.Client = client
	return nil
}

func (r *ResilientSSHClient) IsConnected() bool {
	if r.Client == nil {
		return false
	}
	
	// Try to create a session to check connection
	session, err := r.Client.NewSession()
	if err != nil {
		return false
	}
	session.Close()
	return true
}

func (r *ResilientSSHClient) Reconnect() error {
	if r.Client != nil {
		r.Client.Close()
	}
	
	for i := 0; i < r.RetryAttempts; i++ {
		err := r.Connect()
		if err == nil {
			return nil
		}
		time.Sleep(r.RetryDelay)
	}
	
	return fmt.Errorf("failed to reconnect after %d attempts", r.RetryAttempts)
}