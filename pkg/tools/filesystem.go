package tools

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var safePathChars = regexp.MustCompile(`^[a-zA-Z0-9_/\\.:\-]+$`)

func isValidPathChars(path string) bool {
	if strings.Contains(path, "..") {
		return false
	}
	return safePathChars.MatchString(path)
}

type ReadFileTool struct {
	Sandbox *Sandbox
}

func (t *ReadFileTool) Name() string        { return "read_file" }
func (t *ReadFileTool) Description() string { return "Read entire file content" }
func (t *ReadFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Path to the file relative to workspace"},
		},
		"required": []string{"path"},
	}
}
func (t *ReadFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	if !isValidPathChars(path) {
		return &ToolResult{ForLLM: "Error: path contains invalid characters", IsError: true}
	}

	safePath, err := t.Sandbox.CheckPath(path)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}

	content, err := os.ReadFile(safePath)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to read file: %v", err), IsError: true}
	}

	// Prepend line numbers to output to help AI replace exact lines
	lines := strings.Split(string(content), "\n")
	var numbered strings.Builder
	for i, line := range lines {
		// Stop adding newline at the end of the last empty slice if original ended in \n
		if i == len(lines)-1 && line == "" && len(lines) > 1 {
			break
		}
		numbered.WriteString(fmt.Sprintf("%d: %s\n", i+1, line))
	}

	return &ToolResult{ForLLM: numbered.String(), IsError: false}
}

type WriteFileTool struct {
	Sandbox *Sandbox
}

func (t *WriteFileTool) Name() string        { return "write_file" }
func (t *WriteFileTool) Description() string { return "Overwrite or create a file with given content" }
func (t *WriteFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "File path"},
			"content": map[string]any{"type": "string", "description": "New content"},
		},
		"required": []string{"path", "content"},
	}
}
func (t *WriteFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if !isValidPathChars(path) {
		return &ToolResult{ForLLM: "Error: path contains invalid characters", IsError: true}
	}

	safePath, err := t.Sandbox.CheckPath(path)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}

	if err := os.WriteFile(safePath, []byte(content), 0600); err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to write file: %v", err), IsError: true}
	}
	return &ToolResult{ForLLM: "File written successfully.", IsError: false}
}

type AppendFileTool struct {
	Sandbox *Sandbox
}

func (t *AppendFileTool) Name() string        { return "append_file" }
func (t *AppendFileTool) Description() string { return "Append content to the end of a file" }
func (t *AppendFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string", "description": "File path"},
			"content": map[string]any{"type": "string", "description": "Content to append"},
		},
		"required": []string{"path", "content"},
	}
}
func (t *AppendFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	content, _ := args["content"].(string)

	if !isValidPathChars(path) {
		return &ToolResult{ForLLM: "Error: path contains invalid characters", IsError: true}
	}

	safePath, err := t.Sandbox.CheckPath(path)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}

	f, err := os.OpenFile(safePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to open file: %v", err), IsError: true}
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to append: %v", err), IsError: true}
	}
	return &ToolResult{ForLLM: "Content appended successfully.", IsError: false}
}

type EditFileTool struct {
	Sandbox *Sandbox
}

func (t *EditFileTool) Name() string { return "edit_file" }
func (t *EditFileTool) Description() string {
	return "Replace content in a line range (1-indexed, inclusive)"
}
func (t *EditFileTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":        map[string]any{"type": "string", "description": "File path"},
			"start_line":  map[string]any{"type": "integer", "description": "Starting line number (1-indexed)"},
			"end_line":    map[string]any{"type": "integer", "description": "Ending line number (1-indexed, inclusive)"},
			"new_content": map[string]any{"type": "string", "description": "New content to replace within the range"},
		},
		"required": []string{"path", "start_line", "end_line", "new_content"},
	}
}
func (t *EditFileTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	startLineFl, _ := args["start_line"].(float64)
	endLineFl, _ := args["end_line"].(float64)
	newContent, _ := args["new_content"].(string)

	if !isValidPathChars(path) {
		return &ToolResult{ForLLM: "Error: path contains invalid characters", IsError: true}
	}

	startLine := int(startLineFl)
	endLine := int(endLineFl)

	safePath, err := t.Sandbox.CheckPath(path)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}

	bytesData, err := os.ReadFile(safePath)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to read file: %v", err), IsError: true}
	}

	lines := strings.Split(string(bytesData), "\n")
	if startLine < 1 || startLine > len(lines) {
		return &ToolResult{ForLLM: "start_line out of bounds", IsError: true}
	}
	if endLine < startLine || endLine > len(lines) {
		return &ToolResult{ForLLM: "end_line out of bounds", IsError: true}
	}

	var newLines []string
	newLines = append(newLines, lines[:startLine-1]...)
	if newContent != "" {
		newLines = append(newLines, newContent)
	}
	newLines = append(newLines, lines[endLine:]...)

	if err := os.WriteFile(safePath, []byte(strings.Join(newLines, "\n")), 0600); err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to write changes: %v", err), IsError: true}
	}

	return &ToolResult{ForLLM: "File edited successfully.", IsError: false}
}

type ListDirTool struct {
	Sandbox *Sandbox
}

func (t *ListDirTool) Name() string        { return "list_dir" }
func (t *ListDirTool) Description() string { return "List contents of a directory" }
func (t *ListDirTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{"type": "string", "description": "Directory path relative to workspace"},
		},
		"required": []string{"path"},
	}
}
func (t *ListDirTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	path, _ := args["path"].(string)
	if !isValidPathChars(path) {
		return &ToolResult{ForLLM: "Error: path contains invalid characters", IsError: true}
	}

	safePath, err := t.Sandbox.CheckPath(path)
	if err != nil {
		return &ToolResult{ForLLM: err.Error(), IsError: true}
	}

	entries, err := os.ReadDir(safePath)
	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Failed to read dir: %v", err), IsError: true}
	}

	var sb strings.Builder
	for _, entry := range entries {
		info, _ := entry.Info()
		t := "F"
		if entry.IsDir() {
			t = "D"
		}
		sb.WriteString(fmt.Sprintf("[%s] %s (Size: %d bytes)\n", t, entry.Name(), info.Size()))
	}

	return &ToolResult{ForLLM: sb.String(), IsError: false}
}
