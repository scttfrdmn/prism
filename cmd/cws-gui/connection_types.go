package main

// Enhanced connection types for tabbed embedded connections
type ConnectionType string

const (
	ConnectionTypeSSH     ConnectionType = "ssh"
	ConnectionTypeDesktop ConnectionType = "desktop"
	ConnectionTypeWeb     ConnectionType = "web"
	ConnectionTypeAWS     ConnectionType = "aws-service"
)

// ConnectionConfig represents configuration for embedded connections
type ConnectionConfig struct {
	ID            string                 `json:"id"`
	Type          ConnectionType         `json:"type"`
	InstanceName  string                 `json:"instance_name,omitempty"`
	AWSService    string                 `json:"aws_service,omitempty"`
	Region        string                 `json:"region,omitempty"`
	ProxyURL      string                 `json:"proxy_url"`
	AuthToken     string                 `json:"auth_token,omitempty"`
	EmbeddingMode string                 `json:"embedding_mode"` // iframe, websocket, api
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Title         string                 `json:"title"`
	Status        string                 `json:"status"` // connecting, connected, disconnected, error
}

// AWSServiceConfig represents AWS service-specific configuration
type AWSServiceConfig struct {
	Service       string            `json:"service"`
	Region        string            `json:"region"`
	AssumeRole    string            `json:"assume_role,omitempty"`
	Parameters    map[string]string `json:"parameters,omitempty"`
	EmbeddingMode string            `json:"embedding_mode"`
	AuthMethod    string            `json:"auth_method"`
}

// ConnectionStatus represents the status of a connection
type ConnectionStatus struct {
	ID        string `json:"id"`
	Status    string `json:"status"`
	Message   string `json:"message,omitempty"`
	Timestamp string `json:"timestamp"`
}
