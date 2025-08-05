//go:build linux
// +build linux

// Package security provides Linux Secret Service integration using D-Bus
package security

import (
	"fmt"
	"os"

	"github.com/godbus/dbus/v5"
)

const (
	secretServiceName      = "org.freedesktop.secrets"
	secretServicePath      = "/org/freedesktop/secrets"
	secretServiceInterface = "org.freedesktop.Secret.Service"
	secretCollectionInterface = "org.freedesktop.Secret.Collection"
	secretItemInterface    = "org.freedesktop.Secret.Item"
)

// LinuxSecretServiceNative implements native Linux Secret Service storage
type LinuxSecretServiceNative struct {
	conn         *dbus.Conn
	service      dbus.BusObject
	collection   dbus.BusObject
	collectionPath dbus.ObjectPath
	sessionPath  dbus.ObjectPath
}

// Secret represents a secret value in the Secret Service
type Secret struct {
	Session     dbus.ObjectPath
	Parameters  []byte
	Value       []byte
	ContentType string
}

// NewLinuxSecretServiceNative creates a new native Linux Secret Service provider
func NewLinuxSecretServiceNative() (*LinuxSecretServiceNative, error) {
	// Connect to session D-Bus
	conn, err := dbus.SessionBus()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to session D-Bus: %w", err)
	}

	// Get Secret Service object
	service := conn.Object(secretServiceName, secretServicePath)

	// Open session
	var sessionPath dbus.ObjectPath
	var sessionResult dbus.Variant
	err = service.Call(secretServiceInterface+".OpenSession", 0, "plain", dbus.MakeVariant("")).Store(&sessionResult, &sessionPath)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to open Secret Service session: %w", err)
	}

	// Get default collection
	var collections []dbus.ObjectPath
	err = service.Call(secretServiceInterface+".ReadAlias", 0, "default").Store(&collections)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to get default collection: %w", err)
	}

	var collectionPath dbus.ObjectPath
	if len(collections) > 0 {
		collectionPath = collections[0]
	} else {
		// Create a new collection if default doesn't exist
		collectionPath, err = createCollection(conn, service)
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("failed to create collection: %w", err)
		}
	}

	collection := conn.Object(secretServiceName, collectionPath)

	return &LinuxSecretServiceNative{
		conn:           conn,
		service:        service,
		collection:     collection,
		collectionPath: collectionPath,
		sessionPath:    sessionPath,
	}, nil
}

// createCollection creates a new collection in the Secret Service
func createCollection(conn *dbus.Conn, service dbus.BusObject) (dbus.ObjectPath, error) {
	// Collection properties
	properties := map[string]dbus.Variant{
		"org.freedesktop.Secret.Collection.Label": dbus.MakeVariant("CloudWorkstation"),
	}

	// Create collection
	var collectionPath dbus.ObjectPath
	var promptPath dbus.ObjectPath
	err := service.Call(secretServiceInterface+".CreateCollection", 0, properties, "").Store(&collectionPath, &promptPath)
	if err != nil {
		return "", fmt.Errorf("failed to create collection: %w", err)
	}

	// Handle prompt if needed
	if promptPath != "/" {
		prompt := conn.Object(secretServiceName, promptPath)
		err = prompt.Call("org.freedesktop.Secret.Prompt.Prompt", 0, "").Err
		if err != nil {
			return "", fmt.Errorf("failed to complete collection creation prompt: %w", err)
		}
	}

	return collectionPath, nil
}

// Store implements KeychainProvider.Store for Linux using Secret Service
func (l *LinuxSecretServiceNative) Store(key string, data []byte) error {
	// Item properties
	properties := map[string]dbus.Variant{
		"org.freedesktop.Secret.Item.Label": dbus.MakeVariant("CloudWorkstation: " + key),
		"org.freedesktop.Secret.Item.Attributes": dbus.MakeVariant(map[string]string{
			"application": "cloudworkstation",
			"account":     key,
		}),
	}

	// Create secret
	secret := Secret{
		Session:     l.sessionPath,
		Parameters:  []byte{},
		Value:       data,
		ContentType: "application/octet-stream",
	}

	// Create item
	var itemPath dbus.ObjectPath
	var promptPath dbus.ObjectPath
	err := l.collection.Call(secretCollectionInterface+".CreateItem", 0, properties, secret, true).Store(&itemPath, &promptPath)
	if err != nil {
		return fmt.Errorf("failed to create secret item: %w", err)
	}

	// Handle prompt if needed
	if promptPath != "/" {
		prompt := l.conn.Object(secretServiceName, promptPath)
		err = prompt.Call("org.freedesktop.Secret.Prompt.Prompt", 0, "").Err
		if err != nil {
			return fmt.Errorf("failed to complete item creation prompt: %w", err)
		}
	}

	return nil
}

// Retrieve implements KeychainProvider.Retrieve for Linux using Secret Service
func (l *LinuxSecretServiceNative) Retrieve(key string) ([]byte, error) {
	// Search for items
	attributes := map[string]string{
		"application": "cloudworkstation",
		"account":     key,
	}

	var items []dbus.ObjectPath
	err := l.collection.Call(secretCollectionInterface+".SearchItems", 0, attributes).Store(&items)
	if err != nil {
		return nil, fmt.Errorf("failed to search for secret items: %w", err)
	}

	if len(items) == 0 {
		return nil, ErrKeychainNotFound
	}

	// Get secret from first matching item
	item := l.conn.Object(secretServiceName, items[0])

	var secret Secret
	err = item.Call(secretItemInterface+".GetSecret", 0, l.sessionPath).Store(&secret)
	if err != nil {
		return nil, fmt.Errorf("failed to get secret: %w", err)
	}

	return secret.Value, nil
}

// Exists implements KeychainProvider.Exists for Linux using Secret Service
func (l *LinuxSecretServiceNative) Exists(key string) bool {
	// Search for items
	attributes := map[string]string{
		"application": "cloudworkstation",
		"account":     key,
	}

	var items []dbus.ObjectPath
	err := l.collection.Call(secretCollectionInterface+".SearchItems", 0, attributes).Store(&items)
	if err != nil {
		return false
	}

	return len(items) > 0
}

// Delete implements KeychainProvider.Delete for Linux using Secret Service
func (l *LinuxSecretServiceNative) Delete(key string) error {
	// Search for items
	attributes := map[string]string{
		"application": "cloudworkstation",
		"account":     key,
	}

	var items []dbus.ObjectPath
	err := l.collection.Call(secretCollectionInterface+".SearchItems", 0, attributes).Store(&items)
	if err != nil {
		return fmt.Errorf("failed to search for secret items: %w", err)
	}

	if len(items) == 0 {
		// Item doesn't exist, which is fine for delete operation
		return nil
	}

	// Delete all matching items
	for _, itemPath := range items {
		item := l.conn.Object(secretServiceName, itemPath)
		
		var promptPath dbus.ObjectPath
		err = item.Call(secretItemInterface+".Delete", 0).Store(&promptPath)
		if err != nil {
			return fmt.Errorf("failed to delete secret item: %w", err)
		}

		// Handle prompt if needed
		if promptPath != "/" {
			prompt := l.conn.Object(secretServiceName, promptPath)
			err = prompt.Call("org.freedesktop.Secret.Prompt.Prompt", 0, "").Err
			if err != nil {
				return fmt.Errorf("failed to complete item deletion prompt: %w", err)
			}
		}
	}

	return nil
}

// Close closes the D-Bus connection
func (l *LinuxSecretServiceNative) Close() error {
	if l.conn != nil {
		return l.conn.Close()
	}
	return nil
}

// GetKeychainInfo returns information about the Linux Secret Service integration
func (l *LinuxSecretServiceNative) GetKeychainInfo() map[string]interface{} {
	return map[string]interface{}{
		"provider":        "Linux Secret Service (Native)",
		"service":         secretServiceName,
		"collection_path": string(l.collectionPath),
		"session_path":    string(l.sessionPath),
		"protocol":        "D-Bus",
		"security_level":  "Desktop environment keyring (GNOME Keyring, KDE Wallet)",
	}
}