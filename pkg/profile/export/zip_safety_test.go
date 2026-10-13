package export

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeZip(t *testing.T, entries map[string][]byte) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "in.zip")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	zw := zip.NewWriter(f)
	for name, data := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write(data); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return p
}

// Profile import extracts a user-supplied archive; an entry named ../x used to be written outside
// the extraction directory (zip slip).
func TestExtractZipArchive_RejectsZipSlip(t *testing.T) {
	for _, name := range []string{"../escape.txt", "credentials/../../escape.txt", "a/../../../escape.txt"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			out := filepath.Join(root, "extract")
			zipPath := writeZip(t, map[string][]byte{name: []byte("pwned")})

			err := extractZipArchive(zipPath, out, "")
			if err == nil || !strings.Contains(err.Error(), "invalid file path") {
				t.Fatalf("err = %v, want invalid file path", err)
			}
			if _, statErr := os.Stat(filepath.Join(root, "escape.txt")); statErr == nil {
				t.Fatal("entry was written outside the extraction directory")
			}
		})
	}
}

func TestExtractZipArchive_ExtractsNormalEntries(t *testing.T) {
	out := filepath.Join(t.TempDir(), "extract")
	zipPath := writeZip(t, map[string][]byte{"profiles.json": []byte(`{}`), "credentials/p1.json": []byte(`{"k":"v"}`)})
	if err := extractZipArchive(zipPath, out, ""); err != nil {
		t.Fatalf("extract: %v", err)
	}
	info, err := os.Stat(filepath.Join(out, "credentials", "p1.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("extracted credential file mode = %o, want 600", info.Mode().Perm())
	}
}

// A highly compressible entry larger than the cap is refused (decompression bomb).
func TestExtractZipArchive_RejectsOversizedEntry(t *testing.T) {
	out := filepath.Join(t.TempDir(), "extract")
	zipPath := writeZip(t, map[string][]byte{"profiles.json": bytes.Repeat([]byte{0}, maxExtractedFileSize+1)})
	err := extractZipArchive(zipPath, out, "")
	if err == nil || !strings.Contains(err.Error(), "exceeds maximum size") {
		t.Fatalf("err = %v, want exceeds maximum size", err)
	}
}
