package profile

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zalando/go-keyring"
	"golang.org/x/crypto/nacl/secretbox"
)

func testProvider(t *testing.T) *SecureCredentialProvider {
	t.Helper()
	return &SecureCredentialProvider{service: "Prism-test", configPath: t.TempDir()}
}

var testCreds = &Credentials{AccessKeyID: "AKIAEXAMPLE", SecretAccessKey: "secret/value", SessionToken: "tok"}

func readCredFile(t *testing.T, p *SecureCredentialProvider, name string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(p.configPath, name+".creds"))
	if err != nil {
		t.Fatal(err)
	}
	return data
}

// With a keychain available, the file is encrypted under a random key held in the keychain.
func TestCredentialFile_KeychainKeyRoundTrip(t *testing.T) {
	keyring.MockInit()
	t.Setenv(credentialsPassEnv, "")
	p := testProvider(t)

	if err := p.storeInFile("prof", testCreds); err != nil {
		t.Fatalf("store: %v", err)
	}
	data := readCredFile(t, p, "prof")
	if !bytes.HasPrefix(data, []byte(credFileMagic+string(keyModeKeychain))) {
		t.Fatalf("file not in keychain-key format: %q", data[:12])
	}
	got, err := p.getFromFile("prof")
	if err != nil || *got != *testCreds {
		t.Fatalf("round trip: %+v, %v", got, err)
	}
}

// The core property of #738: the file plus everything public (home dir, service name) is not
// enough. With the keychain key gone, the file can't be decrypted -- the legacy derivation, which
// needed only public inputs, doesn't open it either.
func TestCredentialFile_KeyIsNotDerivableFromPublicData(t *testing.T) {
	keyring.MockInit()
	t.Setenv(credentialsPassEnv, "")
	p := testProvider(t)
	if err := p.storeInFile("prof", testCreds); err != nil {
		t.Fatal(err)
	}
	data := readCredFile(t, p, "prof")

	legacyKey, err := p.legacyEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}
	body := data[len(credFileMagic)+1:]
	if _, err := openSealed(body, &legacyKey); err == nil {
		t.Fatal("file decrypts with the publicly derivable legacy key")
	}

	keyring.MockInit() // fresh keychain: the random key is gone
	if _, err := p.getFromFile("prof"); err == nil {
		t.Fatal("file decrypted without its keychain-held key")
	}
}

// Without a keychain, storing to the file needs a passphrase; it never falls back to a derivable key.
func TestCredentialFile_NoKeySourceRefusesToStore(t *testing.T) {
	keyring.MockInitWithError(errors.New("no secret service"))
	t.Setenv(credentialsPassEnv, "")
	p := testProvider(t)

	err := p.storeInFile("prof", testCreds)
	if err == nil || !strings.Contains(err.Error(), credentialsPassEnv) {
		t.Fatalf("err = %v, want guidance naming %s", err, credentialsPassEnv)
	}
	if _, statErr := os.Stat(filepath.Join(p.configPath, "prof.creds")); statErr == nil {
		t.Fatal("a credentials file was written anyway")
	}
}

func TestCredentialFile_PassphraseMode(t *testing.T) {
	keyring.MockInitWithError(errors.New("no secret service"))
	t.Setenv(credentialsPassEnv, "correct horse battery staple")
	p := testProvider(t)

	if err := p.storeInFile("prof", testCreds); err != nil {
		t.Fatalf("store: %v", err)
	}
	if data := readCredFile(t, p, "prof"); !bytes.HasPrefix(data, []byte(credFileMagic+string(keyModePassphrase))) {
		t.Fatal("file not in passphrase format")
	}
	if got, err := p.getFromFile("prof"); err != nil || *got != *testCreds {
		t.Fatalf("round trip: %+v, %v", got, err)
	}

	t.Setenv(credentialsPassEnv, "wrong")
	if _, err := p.getFromFile("prof"); err == nil {
		t.Fatal("decrypted with the wrong passphrase")
	}
	t.Setenv(credentialsPassEnv, "")
	if _, err := p.getFromFile("prof"); err == nil || !strings.Contains(err.Error(), credentialsPassEnv) {
		t.Fatalf("err = %v, want it to ask for %s", err, credentialsPassEnv)
	}
}

// writeLegacy writes a .creds file the way the pre-#738 code did.
func writeLegacy(t *testing.T, p *SecureCredentialProvider, name string, creds *Credentials) {
	t.Helper()
	key, err := p.legacyEncryptionKey()
	if err != nil {
		t.Fatal(err)
	}
	plaintext := []byte(`{"access_key_id":"` + creds.AccessKeyID + `","secret_access_key":"` + creds.SecretAccessKey + `","session_token":"` + creds.SessionToken + `"}`)
	var nonce [nonceSize]byte
	if _, err := io.ReadFull(rand.Reader, nonce[:]); err != nil {
		t.Fatal(err)
	}
	data := secretbox.Seal(append([]byte{}, nonce[:]...), plaintext, &nonce, &key)
	if err := os.WriteFile(filepath.Join(p.configPath, name+".creds"), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestCredentialFile_LegacyFileIsReadAndMigrated(t *testing.T) {
	keyring.MockInit()
	t.Setenv(credentialsPassEnv, "")
	p := testProvider(t)
	writeLegacy(t, p, "prof", testCreds)

	got, err := p.getFromFile("prof")
	if err != nil || got.AccessKeyID != testCreds.AccessKeyID || got.SecretAccessKey != testCreds.SecretAccessKey {
		t.Fatalf("legacy read: %+v, %v", got, err)
	}
	if data := readCredFile(t, p, "prof"); !bytes.HasPrefix(data, []byte(credFileMagic)) {
		t.Fatal("legacy file was not re-encrypted in the new format")
	}
	if got, err := p.getFromFile("prof"); err != nil || got.SecretAccessKey != testCreds.SecretAccessKey {
		t.Fatalf("read after migration: %+v, %v", got, err)
	}
}

// With no key source to migrate to, a legacy file stays readable (and unchanged) rather than lost.
func TestCredentialFile_LegacyFileWithoutKeySourceStaysReadable(t *testing.T) {
	keyring.MockInitWithError(errors.New("no secret service"))
	t.Setenv(credentialsPassEnv, "")
	p := testProvider(t)
	writeLegacy(t, p, "prof", testCreds)
	before := readCredFile(t, p, "prof")

	if got, err := p.getFromFile("prof"); err != nil || got.AccessKeyID != testCreds.AccessKeyID {
		t.Fatalf("legacy read: %+v, %v", got, err)
	}
	if after := readCredFile(t, p, "prof"); !bytes.Equal(before, after) {
		t.Fatal("legacy file changed although it could not be migrated")
	}
}
