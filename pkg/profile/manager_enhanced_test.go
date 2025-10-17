package profile

// MockCredentialProvider implements CredentialProvider for testing
type MockCredentialProvider struct {
	storedCredentials map[string]*Credentials
}

func NewMockCredentialProvider() *MockCredentialProvider {
	return &MockCredentialProvider{
		storedCredentials: make(map[string]*Credentials),
	}
}

func (m *MockCredentialProvider) StoreCredentials(profileID string, creds *Credentials) error {
	m.storedCredentials[profileID] = creds
	return nil
}

func (m *MockCredentialProvider) GetCredentials(profileID string) (*Credentials, error) {
	creds, exists := m.storedCredentials[profileID]
	if !exists {
		return nil, ErrCredentialsNotFound
	}
	return creds, nil
}

func (m *MockCredentialProvider) ClearCredentials(profileID string) error {
	delete(m.storedCredentials, profileID)
	return nil
}
