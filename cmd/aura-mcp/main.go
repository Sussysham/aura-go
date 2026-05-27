package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"aura-go/internal/catalog"
	"aura-go/internal/config"
	"aura-go/internal/reader"
	"aura-go/internal/state"
)

func main() {
	// Initialize configurations and paths cleanly
	configPath := "aura.toml"
	if exePath, err := os.Executable(); err == nil {
		configPath = filepath.Join(filepath.Dir(exePath), "aura.toml")
	}
	// Silence standard prints to Stderr to prevent stdio JSON-RPC corruption
	_ = config.LoadConfig(configPath)

	// Ensure StatePath and CSVPath exist
	stateData, err := state.LoadState(config.StatePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️ Failed to load library state: %v\n", err)
		stateData = &state.LibraryState{
			Statuses:    make(map[string]string),
			Notes:       make(map[string]string),
			ReadingTime: make(map[string]int),
			BookProgress: make(map[string]int),
		}
	}

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			sendError(nil, -32700, "Parse error: invalid JSON", nil)
			continue
		}

		handleRequest(&req, stateData)
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		fmt.Fprintf(os.Stderr, "Error scanning stdin: %v\n", err)
	}
}

func handleRequest(req *JSONRPCRequest, stateData *state.LibraryState) {
	switch req.Method {
	case "initialize":
		// Standard MCP initialization response
		res := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]interface{}{},
				"serverInfo": map[string]string{
					"name":    "aura-mcp",
					"version": "1.0.0",
				},
			},
		}
		sendResponse(res)

	case "tools/list":
		// List of 5 high-value native catalog & reading tools
		tools := []Tool{
			{
				Name:        "aura_list_books",
				Description: "List all files in your local Aura catalog, including titles, formats, authors, and categorizations.",
				InputSchema: InputSchema{
					Type:       "object",
					Properties: map[string]Property{},
				},
			},
			{
				Name:        "aura_search_library",
				Description: "Search the local catalog using a keyword. Filters by book title, author, category, or tags.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"query": {
							Type:        "string",
							Description: "The search term or query keyword.",
						},
					},
					Required: []string{"query"},
				},
			},
			{
				Name:        "aura_get_metadata",
				Description: "Retrieve state metadata for a book, including its custom tags, notes, reading status, progress, and accumulated reading hours.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"sha1": {
							Type:        "string",
							Description: "The unique SHA1 fingerprint of the book.",
						},
					},
					Required: []string{"sha1"},
				},
			},
			{
				Name:        "aura_read_book",
				Description: "Natively extract and read plain-text content from a local PDF, EPUB, MOBI, DOCX, MD, or TXT book. Supports pagination via line limits.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"path": {
							Type:        "string",
							Description: "The absolute or relative local filesystem path to the book.",
						},
						"startLine": {
							Type:        "integer",
							Description: "Optional. Starting line index (1-indexed) to read to prevent context overflow.",
						},
						"endLine": {
							Type:        "integer",
							Description: "Optional. Ending line index (inclusive, 1-indexed) to read.",
						},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "aura_read_section",
				Description: "Retrieve a specific line-range section of a local file. Ideal for reading specific chapters or targeted summaries.",
				InputSchema: InputSchema{
					Type: "object",
					Properties: map[string]Property{
						"path": {
							Type:        "string",
							Description: "The absolute or relative path to the book on disk.",
						},
						"startLine": {
							Type:        "integer",
							Description: "Starting line index (1-indexed).",
						},
						"endLine": {
							Type:        "integer",
							Description: "Ending line index (inclusive, 1-indexed).",
						},
					},
					Required: []string{"path", "startLine", "endLine"},
				},
			},
		}

		res := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: map[string]interface{}{
				"tools": tools,
			},
		}
		sendResponse(res)

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			sendError(req.ID, -32602, "Invalid params", nil)
			return
		}

		result := executeTool(params.Name, params.Arguments, stateData)
		res := JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  result,
		}
		sendResponse(res)

	default:
		sendError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method), nil)
	}
}

func executeTool(name string, arguments json.RawMessage, stateData *state.LibraryState) ToolCallResult {
	books, err := catalog.LoadBooks(config.CSVPath)
	if err != nil {
		books = []catalog.Book{}
	}

	switch name {
	case "aura_list_books":
		if len(books) == 0 {
			return errorResult("Your library is currently empty. Scan directories inside Aura TUI first.")
		}
		var sb strings.Builder
		sb.WriteString("📚 Local Aura Library Catalog:\n\n")
		for i, b := range books {
			sb.WriteString(fmt.Sprintf("[%d] Title: %s\n    Author: %s | Format: %s\n    Path: %s\n    SHA1: %s\n\n",
				i+1, b.Title, b.Author, b.Format, b.FullPath, b.SHA1))
		}
		return textResult(sb.String())

	case "aura_search_library":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult("Failed to parse search query arguments.")
		}

		q := strings.ToLower(args.Query)
		var matches []catalog.Book
		for _, b := range books {
			bStatus := stateData.Statuses[b.SHA1]
			if bStatus == "" {
				bStatus = "Inbox"
			}
			if strings.Contains(strings.ToLower(b.Title), q) ||
				strings.Contains(strings.ToLower(b.Author), q) ||
				strings.Contains(strings.ToLower(b.LocationCategory), q) ||
				strings.Contains(strings.ToLower(b.Tags), q) ||
				strings.Contains(strings.ToLower(bStatus), q) {
				matches = append(matches, b)
			}
		}

		if len(matches) == 0 {
			return textResult(fmt.Sprintf("No books matched query keyword: %q", args.Query))
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("🔍 Matches found for %q in local catalog:\n\n", args.Query))
		for i, b := range matches {
			sb.WriteString(fmt.Sprintf("[%d] Title: %s\n    Author: %s | Category: %s\n    Path: %s\n    SHA1: %s\n\n",
				i+1, b.Title, b.Author, b.LocationCategory, b.FullPath, b.SHA1))
		}
		return textResult(sb.String())

	case "aura_get_metadata":
		var args struct {
			SHA1 string `json:"sha1"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult("Failed to parse sha1 argument.")
		}

		var target *catalog.Book
		for _, b := range books {
			if b.SHA1 == args.SHA1 {
				target = &b
				break
			}
		}

		if target == nil {
			return errorResult(fmt.Sprintf("Book with SHA1 fingerprint %s not found in catalog.", args.SHA1))
		}

		bStatus := stateData.Statuses[target.SHA1]
		if bStatus == "" {
			bStatus = "Inbox"
		}
		note := stateData.Notes[target.SHA1]
		if note == "" {
			note = "(None)"
		}
		rTime := stateData.ReadingTime[target.SHA1]
		progress := stateData.BookProgress[target.SHA1]

		metadataStr := fmt.Sprintf("📖 Metadata for: %s\n"+
			"──────────────────────────────────────────────────\n"+
			"• Author: %s\n"+
			"• Category: %s\n"+
			"• Tags: %s\n"+
			"• Format: %s | Size: %.2f MB\n"+
			"• Full Path: %s\n"+
			"• SHA1: %s\n"+
			"──────────────────────────────────────────────────\n"+
			"• Active Focus Status: %s\n"+
			"• Saved Reading Note: %s\n"+
			"• Native Reader Progress: Line %d\n"+
			"• Logged Active Study Time: %d Hours, %d Minutes\n",
			target.Title, target.Author, target.LocationCategory, target.Tags, target.Format, target.SizeMB,
			target.FullPath, target.SHA1, bStatus, note, progress, rTime/3600, (rTime%3600)/60)

		return textResult(metadataStr)

	case "aura_read_book", "aura_read_section":
		var args struct {
			Path      string `json:"path"`
			StartLine int    `json:"startLine"`
			EndLine   int    `json:"endLine"`
		}
		if err := json.Unmarshal(arguments, &args); err != nil {
			return errorResult("Failed to parse read arguments.")
		}

		// Locate book to extract text
		var targetPath string
		var targetBook catalog.Book
		var found bool

		// Check full path directly or resolve via relative title scan
		if _, err := os.Stat(args.Path); err == nil {
			targetPath = args.Path
			// Find catalog book details
			for _, b := range books {
				if filepath.Clean(b.FullPath) == filepath.Clean(args.Path) {
					targetBook = b
					found = true
					break
				}
			}
			if !found {
				// Create temporary minimal book for extraction
				targetBook = catalog.Book{
					FullPath: targetPath,
					FileName: filepath.Base(targetPath),
					Format:   filepath.Ext(targetPath),
				}
			}
		} else {
			// Search catalog to match relative paths or titles
			for _, b := range books {
				if strings.Contains(strings.ToLower(b.Title), strings.ToLower(args.Path)) ||
					strings.Contains(strings.ToLower(b.FullPath), strings.ToLower(args.Path)) {
					targetBook = b
					targetPath = b.FullPath
					found = true
					break
				}
			}
			if !found {
				return errorResult(fmt.Sprintf("Failed to resolve local book file: %s", args.Path))
			}
		}

		// Extract raw book text dynamically using Aura's unified parsing routines!
		rawText, err := reader.ExtractBookRawText(targetBook)
		if err != nil {
			return errorResult(fmt.Sprintf("Failed to extract book text natively: %v", err))
		}

		lines := strings.Split(rawText, "\n")
		start := 1
		end := len(lines)

		// Set line boundaries if provided
		if args.StartLine > 0 {
			start = args.StartLine
		}
		if args.EndLine > 0 {
			end = args.EndLine
		}
		if start > len(lines) {
			start = len(lines)
		}
		if end > len(lines) {
			end = len(lines)
		}
		if start < 1 {
			start = 1
		}
		if end < start {
			end = start
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📖 Native Text Extraction of [%s] (Lines %d-%d of %d):\n\n",
			targetBook.Title, start, end, len(lines)))

		for idx := start - 1; idx < end; idx++ {
			sb.WriteString(fmt.Sprintf("%04d | %s\n", idx+1, lines[idx]))
		}

		return textResult(sb.String())

	default:
		return errorResult(fmt.Sprintf("Unknown tool name: %s", name))
	}
}

func textResult(text string) ToolCallResult {
	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: text,
			},
		},
	}
}

func errorResult(msg string) ToolCallResult {
	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: fmt.Sprintf("⚠️ [ERROR] %s", msg),
			},
		},
		IsError: true,
	}
}

func sendResponse(res interface{}) {
	data, err := json.Marshal(res)
	if err == nil {
		os.Stdout.Write(data)
		os.Stdout.Write([]byte("\n"))
	}
}

func sendError(id interface{}, code int, message string, data interface{}) {
	res := JSONRPCErrorResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	sendResponse(res)
}
