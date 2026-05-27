package main

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"aura-go/internal/reader"
	"aura-go/internal/state"
)

// TestStripHTMLTags verifies tag stripping and proper HTML entity unescaping
func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"<div>Hello World</div>", " Hello World "},
		{"Hello &amp; World", "Hello & World"},
		{"&ldquo;Double Quotes&rdquo;", "“Double Quotes”"},
		{"&eacute;lan", "élan"},
		{"3 &#62; 2", "3 > 2"},
		{"&mdash; &hellip;", "— …"},
		{"&#x2014; &#x2026;", "— …"},
	}

	for _, tt := range tests {
		actual := reader.StripHTMLTags(tt.input)
		// Clean up spacing differences from tag stripping if needed, but here we just match exactly
		if strings.TrimSpace(actual) != strings.TrimSpace(tt.expected) {
			t.Errorf("reader.StripHTMLTags(%q) = %q; want %q", tt.input, actual, tt.expected)
		}
	}
}

// TestGetEPUBOrderedPaths verifies that container.xml and spine are correctly parsed from zip in-memory
func TestGetEPUBOrderedPaths(t *testing.T) {
	// Create an in-memory zip file representing a mock EPUB
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// 1. Write META-INF/container.xml
	containerXML := `<?xml version="1.0"?>
<container version="1.0" xmlns="urn:oasis:names:tc:opendocument:xmlns:container">
  <rootfiles>
    <rootfile full-path="OEBPS/content.opf" media-type="application/oebps-package+xml"/>
  </rootfiles>
</container>`
	w, err := zw.Create("META-INF/container.xml")
	if err != nil {
		t.Fatalf("Failed to create container.xml in zip: %v", err)
	}
	_, _ = w.Write([]byte(containerXML))

	// 2. Write OEBPS/content.opf
	opfXML := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="uuid_id" version="2.0">
  <metadata xmlns:dc="http://purl.org/dc/elements/1.1/"></metadata>
  <manifest>
    <item id="titlepage" href="titlepage.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter2" href="text/ch02.xhtml" media-type="application/xhtml+xml"/>
    <item id="chapter1" href="text/ch01.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine toc="ncx">
    <itemref idref="titlepage"/>
    <itemref idref="chapter1"/>
    <itemref idref="chapter2"/>
  </spine>
</package>`
	w, err = zw.Create("OEBPS/content.opf")
	if err != nil {
		t.Fatalf("Failed to create content.opf in zip: %v", err)
	}
	_, _ = w.Write([]byte(opfXML))

	// 3. Write dummy files for manifest items
	files := []string{"OEBPS/titlepage.xhtml", "OEBPS/text/ch01.xhtml", "OEBPS/text/ch02.xhtml"}
	for _, f := range files {
		w, err = zw.Create(f)
		if err != nil {
			t.Fatalf("Failed to create %s in zip: %v", f, err)
		}
		_, _ = w.Write([]byte("<html><body>Mock content</body></html>"))
	}

	err = zw.Close()
	if err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	// Since GetEPUBOrderedPaths takes *zip.ReadCloser (which is for file paths), 
	// let's temporarily write this buf to a temp file so we can open it with zip.OpenReader!
	tempFile, err := os.CreateTemp("", "mock_epub_*.epub")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, _ = tempFile.Write(buf.Bytes())
	_ = tempFile.Close()

	r, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to open zip reader on temp file: %v", err)
	}
	defer r.Close()

	// 4. Test ordered path extraction
	ordered, err := reader.GetEPUBOrderedPaths(r)
	if err != nil {
		t.Fatalf("reader.GetEPUBOrderedPaths failed: %v", err)
	}

	expected := []string{"OEBPS/titlepage.xhtml", "OEBPS/text/ch01.xhtml", "OEBPS/text/ch02.xhtml"}
	if len(ordered) != len(expected) {
		t.Fatalf("Expected %d ordered paths, got %d", len(expected), len(ordered))
	}

	for i, path := range ordered {
		// Clean and standardize paths for comparison
		actualClean := filepath.ToSlash(path)
		expectedClean := filepath.ToSlash(expected[i])
		if actualClean != expectedClean {
			t.Errorf("At index %d: got %s, want %s", i, actualClean, expectedClean)
		}
	}
}

// TestGetEPUBOrderedPathsFallback checks that GetEPUBOrderedPaths handles lack of container.xml by finding .opf
func TestGetEPUBOrderedPathsFallback(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	// Direct content.opf inside the zip, NO container.xml
	opfXML := `<?xml version="1.0" encoding="utf-8"?>
<package xmlns="http://www.idpf.org/2007/opf" unique-identifier="uuid_id" version="2.0">
  <manifest>
    <item id="ch1" href="ch1.xhtml" media-type="application/xhtml+xml"/>
  </manifest>
  <spine>
    <itemref idref="ch1"/>
  </spine>
</package>`
	w, err := zw.Create("content.opf")
	if err != nil {
		t.Fatalf("Failed to create content.opf in zip: %v", err)
	}
	_, _ = w.Write([]byte(opfXML))

	err = zw.Close()
	if err != nil {
		t.Fatalf("Failed to close zip writer: %v", err)
	}

	tempFile, err := os.CreateTemp("", "mock_fallback_*.epub")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, _ = tempFile.Write(buf.Bytes())
	_ = tempFile.Close()

	r, err := zip.OpenReader(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to open zip reader: %v", err)
	}
	defer r.Close()

	ordered, err := reader.GetEPUBOrderedPaths(r)
	if err != nil {
		t.Fatalf("reader.GetEPUBOrderedPaths fallback failed: %v", err)
	}

	if len(ordered) != 1 || ordered[0] != "ch1.xhtml" {
		t.Errorf("Expected fallback to find ['ch1.xhtml'], got %v", ordered)
	}
}

// TestStateBackupAndAtomicSave verifies copy, state load/save and backups
func TestStateBackupAndAtomicSave(t *testing.T) {
	tempStateDir, err := os.MkdirTemp("", "aura_state_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempStateDir)

	stateFilePath := filepath.Join(tempStateDir, "library_state.json")

	// 1. Initial load should return empty clean state without error
	stateData, err := state.LoadState(stateFilePath)
	if err != nil {
		t.Fatalf("Initial loadState failed: %v", err)
	}
	if len(stateData.Statuses) != 0 {
		t.Errorf("Initial statuses should be empty, got %v", stateData.Statuses)
	}

	// 2. Save state
	stateData.Statuses["test-sha"] = "Reading"
	err = state.SaveState(stateFilePath, stateData)
	if err != nil {
		t.Fatalf("state.SaveState failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(stateFilePath); err != nil {
		t.Fatalf("state file was not created: %v", err)
	}

	// 3. Save again - should create a backup
	stateData.Statuses["test-sha"] = "Read"
	err = state.SaveState(stateFilePath, stateData)
	if err != nil {
		t.Fatalf("state.SaveState 2nd failed: %v", err)
	}

	backupPath := stateFilePath + ".bak"
	if _, err := os.Stat(backupPath); err != nil {
		t.Errorf("Backup file %s was not created: %v", backupPath, err)
	}

	// Read backup and verify it has old status ("Reading")
	backupState, err := state.LoadState(backupPath)
	if err != nil {
		t.Fatalf("Failed to load backup: %v", err)
	}
	if backupState.Statuses["test-sha"] != "Reading" {
		t.Errorf("Backup status should be 'Reading', got %s", backupState.Statuses["test-sha"])
	}

	// 4. Corrupt state file and load it - should warn and backup
	err = os.WriteFile(stateFilePath, []byte("{invalid-json"), 0644)
	if err != nil {
		t.Fatalf("Failed to corrupt state file: %v", err)
	}

	_, err = state.LoadState(stateFilePath)
	if err == nil {
		t.Errorf("Expected loadState on corrupted JSON to return an error, got nil")
	} else if !strings.Contains(err.Error(), "library state corrupted") {
		t.Errorf("Expected corruption warning in error, got %v", err)
	}

	corruptPath := stateFilePath + ".corrupt"
	if _, err := os.Stat(corruptPath); err != nil {
		t.Errorf("Corrupt backup file %s was not created: %v", corruptPath, err)
	}
}
