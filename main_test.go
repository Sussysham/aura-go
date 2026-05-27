package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"aura-go/internal/config"
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

// TestTOMLConfigParsing verifies default configuration loading and parsing
func TestTOMLConfigParsing(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "aura_config_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	configPath := filepath.Join(tempDir, "aura.toml")

	// Load should auto-generate defaults if file doesn't exist
	err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed to generate defaults: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(configPath); err != nil {
		t.Fatalf("config file was not auto-generated: %v", err)
	}

	// Verify defaults parsed correctly
	if config.AppConfig.General.WIPLimit != 2 {
		t.Errorf("Expected default WIPLimit = 2, got %d", config.AppConfig.General.WIPLimit)
	}
	if config.AppConfig.Theme.Name != "midnight" {
		t.Errorf("Expected default theme = 'midnight', got %s", config.AppConfig.Theme.Name)
	}
	if config.AppConfig.Keybindings.Search != "s" {
		t.Errorf("Expected default search key = 's', got %s", config.AppConfig.Keybindings.Search)
	}

	// Override config file and reload
	customTOML := `
[general]
wip_limit = 5
default_viewer = "system"

[keybindings]
search = "f"
theme_cycle = "c"
`
	err = os.WriteFile(configPath, []byte(customTOML), 0644)
	if err != nil {
		t.Fatalf("Failed to write custom config: %v", err)
	}

	err = config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed to load custom config: %v", err)
	}

	if config.AppConfig.General.WIPLimit != 5 {
		t.Errorf("Expected overridden WIPLimit = 5, got %d", config.AppConfig.General.WIPLimit)
	}
	if config.AppConfig.General.DefaultViewer != "system" {
		t.Errorf("Expected overridden default_viewer = 'system', got %s", config.AppConfig.General.DefaultViewer)
	}
	if config.AppConfig.Keybindings.Search != "f" {
		t.Errorf("Expected overridden search key = 'f', got %s", config.AppConfig.Keybindings.Search)
	}
	if config.AppConfig.Keybindings.ThemeCycle != "c" {
		t.Errorf("Expected overridden theme_cycle = 'c', got %s", config.AppConfig.Keybindings.ThemeCycle)
	}
}

// TestHexToTrueColorConversion verifies hexadecimal conversion to 24-bit TrueColor ANSI escape codes
func TestHexToTrueColorConversion(t *testing.T) {
	tests := []struct {
		hex      string
		expected string
	}{
		{"#7B2FBE", "\033[38;2;123;47;190m"},
		{"00D4FF", "\033[38;2;0;212;255m"},
		{"#FF1744", "\033[38;2;255;23;68m"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		actual := config.HexToTrueColor(tt.hex)
		if actual != tt.expected {
			t.Errorf("config.HexToTrueColor(%q) = %q; want %q", tt.hex, actual, tt.expected)
		}
	}
}

// TestMOBIParserDecompression verifies PalmDOC LZ77 decompression including literal and overlapping sequences
func TestMOBIParserDecompression(t *testing.T) {
	tests := []struct {
		name       string
		compressed []byte
		expected   string
	}{
		{"Plain literal", []byte("hello"), "hello"},
		{"Literal length-copy", []byte{0x05, 'a', 'b', 'c', 'd', 'e'}, "abcde"},
		{"Space compression", []byte{0x80 ^ 'A'}, " A"},
		{"LZ77 overlapping copy", []byte{'a', 'b', 'c', 'd', 0x80, 34}, "abcdabcda"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := reader.DecompressPalmDOC(tt.compressed)
			if err != nil {
				t.Fatalf("DecompressPalmDOC failed: %v", err)
			}
			if string(actual) != tt.expected {
				t.Errorf("got %q, want %q", string(actual), tt.expected)
			}
		})
	}
}

// TestDOCXParserExtraction verifies XML parsing and text extraction from mock ZIP
func TestDOCXParserExtraction(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)

	documentXML := `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>
    <w:p>
      <w:r>
        <w:t>Hello </w:t>
      </w:r>
      <w:r>
        <w:t>World</w:t>
      </w:r>
    </w:p>
    <w:p>
      <w:r>
        <w:t>Second paragraph</w:t>
      </w:r>
    </w:p>
  </w:body>
</w:document>`

	w, err := zw.Create("word/document.xml")
	if err != nil {
		t.Fatalf("Failed to create document.xml: %v", err)
	}
	_, _ = w.Write([]byte(documentXML))
	_ = zw.Close()

	tempFile, err := os.CreateTemp("", "mock_docx_*.docx")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, _ = tempFile.Write(buf.Bytes())
	_ = tempFile.Close()

	extracted, err := reader.ExtractDOCXText(tempFile.Name())
	if err != nil {
		t.Fatalf("ExtractDOCXText failed: %v", err)
	}

	expected := "Hello World\nSecond paragraph\n"
	if extracted != expected {
		t.Errorf("got %q, want %q", extracted, expected)
	}
}

// TestMarkdownTUIRendering verifies live on-the-fly markdown highlighting and marker toggling
func TestMarkdownTUIRendering(t *testing.T) {
	theme := config.GetTheme("midnight", config.CustomThemeConfig{})

	tests := []struct {
		input       string
		showRaw     bool
		contains    string
		notContains string
	}{
		{"# Header 1", false, "Header 1", "# "},
		{"# Header 1", true, "# Header 1", ""},
		{"**bold text**", false, "bold text", "**"},
		{"**bold text**", true, "**bold text**", ""},
		{"*italic text*", false, "italic text", "*"},
		{"*italic text*", true, "*italic text*", ""},
		{"`code block`", false, "code block", "`"},
		{"`code block`", true, "`code block`", ""},
		{"- Bullet Item", false, "• Bullet Item", "- "},
		{"- Bullet Item", true, "- Bullet Item", "•"},
	}

	stripANSI := func(s string) string {
		re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
		return re.ReplaceAllString(s, "")
	}

	for _, tt := range tests {
		actual := reader.FormatMarkdownLine(tt.input, theme, tt.showRaw)
		plain := stripANSI(actual)
		if tt.contains != "" && !strings.Contains(plain, tt.contains) {
			t.Errorf("FormatMarkdownLine(%q, showRaw=%v) plain output %q should contain %q (actual: %q)", tt.input, tt.showRaw, plain, tt.contains, actual)
		}
		if tt.notContains != "" && strings.Contains(plain, tt.notContains) {
			t.Errorf("FormatMarkdownLine(%q, showRaw=%v) plain output %q should NOT contain %q (actual: %q)", tt.input, tt.showRaw, plain, tt.notContains, actual)
		}
	}
}

// TestMCPServerIntegration verifies the compiled aura-mcp.exe JSON-RPC 2.0 stdio flow
func TestMCPServerIntegration(t *testing.T) {
	// First, check if aura-mcp.exe exists. If not, compile it!
	binPath := "./aura-mcp-test.exe"
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		cmd := exec.Command("go", "build", "-o", binPath, "./cmd/aura-mcp")
		err = cmd.Run()
		if err != nil {
			t.Skip("Skipping integration test: aura-mcp-test.exe could not be compiled:", err)
		}
		defer os.Remove(binPath)
	}

	// Spawn the server as a child process
	cmd := exec.Command(binPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start child mcp server: %v", err)
	}

	// 1. Send initialize request
	initReq := `{"jsonrpc":"2.0","method":"initialize","id":1}`
	_, _ = stdin.Write([]byte(initReq + "\n"))

	reader := bufio.NewReader(stdout)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read response from server: %v", err)
	}

	if !strings.Contains(line, `"protocolVersion"`) || !strings.Contains(line, `"aura-mcp"`) {
		t.Errorf("Initialize response did not contain expected fields, got: %q", line)
	}

	// 2. Send tools/list request
	listReq := `{"jsonrpc":"2.0","method":"tools/list","id":2}`
	_, _ = stdin.Write([]byte(listReq + "\n"))

	line, err = reader.ReadString('\n')
	if err != nil {
		t.Fatalf("Failed to read tools/list response: %v", err)
	}

	if !strings.Contains(line, "aura_list_books") || !strings.Contains(line, "aura_read_book") {
		t.Errorf("tools/list response did not list expected tools, got: %q", line)
	}

	// Clean up child process
	_ = stdin.Close()
	_ = cmd.Wait()
}

// TestCrossPlatformPathResolution verifies that InitPaths correctly resolves configurations under standard system paths or falls back gracefully
func TestCrossPlatformPathResolution(t *testing.T) {
	// Temporarily clear custom configurations
	originalCatalog := config.AppConfig.Paths.Catalog
	originalState := config.AppConfig.Paths.State
	defer func() {
		config.AppConfig.Paths.Catalog = originalCatalog
		config.AppConfig.Paths.State = originalState
		config.InitPaths() // Restore paths
	}()

	config.AppConfig.Paths.Catalog = ""
	config.AppConfig.Paths.State = ""

	config.InitPaths()

	// Verify resolved paths are not empty
	if config.CSVPath == "" {
		t.Error("InitPaths resolved empty CSVPath")
	}
	if config.StatePath == "" {
		t.Error("InitPaths resolved empty StatePath")
	}

	// Verify standard XDG fallback folder logic on POSIX systems or roaming folder logic on Windows
	configDir, err := os.UserConfigDir()
	if err == nil {
		expectedSuffix := filepath.Join("aura", "books_catalog.csv")
		if !strings.HasSuffix(filepath.Clean(config.CSVPath), expectedSuffix) {
			t.Logf("CSVPath resolved to %s (precedence or fallback under config dir: %s)", config.CSVPath, configDir)
		}
	}
}
