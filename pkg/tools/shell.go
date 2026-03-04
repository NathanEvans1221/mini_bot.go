package tools

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type ExecTool struct {
	Sandbox *Sandbox
}

func (t *ExecTool) Name() string        { return "exec" }
func (t *ExecTool) Description() string { return "Execute a shell command inside the workspace" }
func (t *ExecTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string", "description": "The shell command to execute"},
			"timeout": map[string]any{"type": "integer", "description": "Timeout in seconds, default 30"},
		},
		"required": []string{"command"},
	}
}

var allowedCommands = map[string]bool{
	"ls": true, "dir": true, "cd": true, "pwd": true,
	"cat": true, "head": true, "tail": true, "less": true,
	"grep": true, "egrep": true, "fgrep": true,
	"find": true, "xargs": true,
	"wc": true, "sort": true, "uniq": true, "cut": true,
	"echo": true, "printf": true,
	"stat": true, "file": true, "md5sum": true, "sha256sum": true,
	"tree": true, "ls -la": true, "ls -l": true,
}

var dangerPatterns = []string{
	`rm\s+-rf\s+/`,
	`del\s+/f`,
	`rmdir\s+/s`,
	`format\b`,
	`mkfs\b`,
	`diskpart\b`,
	`dd\s+if=`,
	`shutdown\b`,
	`reboot\b`,
	`poweroff\b`,
	`:\(\)\{\s+:\|:&\s+\};:`,
	`;\s*rm\s+`,
	`\|\s*rm\s+`,
	`&&\s*rm\s+`,
	`\|\|\s*rm\s+`,
	`\$\(.*\)`,
	"`.*`",
}

var dangerCommands = []string{
	"curl", "wget", "nc", "ncat", "bash", "powershell", "sh",
	"python", "python3", "perl", "ruby", "php", "node",
	"npm", "pip", "cargo", "go", "rustc",
	"ssh", "scp", "sftp", "ftp", "telnet",
	"chmod", "chown", "chgrp",
	"kill", "killall", "pkill",
	"mount", "umount",
	"fdisk", "parted", "sfdisk",
	"sudo", "su",
	"git", "svn", "hg",
	"docker", "kubectl", "helm",
	"tar", "gzip", "bzip2", "xz", "zip", "unzip",
	"sed", "awk", "vim", "nano", "emacs",
	"ln", "unlink", "touch", "mkdir", "rmdir",
}

func validateCommand(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}

	if len(cmd) > 500 {
		return false
	}

	baseCmd := strings.ToLower(fields[0])

	if _, ok := allowedCommands[baseCmd]; ok {
		return true
	}
	if strings.HasPrefix(baseCmd, "ls-") {
		return true
	}

	for _, dc := range dangerCommands {
		if baseCmd == dc {
			return false
		}
	}

	return len(fields) > 0
}

func sanitizeCommand(cmd string) string {
	sanitized := cmd
	dangerChars := []string{";", "|", "&&", "||", "$(", "`", "\\\n"}
	for _, dc := range dangerChars {
		sanitized = strings.ReplaceAll(sanitized, dc, "")
	}
	return strings.TrimSpace(sanitized)
}

func isCommandSafe(cmd string) bool {
	for _, pattern := range dangerPatterns {
		matched, _ := regexp.MatchString(pattern, cmd)
		if matched {
			return false
		}
	}
	return true
}

func (t *ExecTool) Execute(ctx context.Context, args map[string]any) *ToolResult {
	cmdStr, _ := args["command"].(string)

	sanitized := sanitizeCommand(cmdStr)
	if sanitized != cmdStr {
		cmdStr = sanitized
	}

	if !validateCommand(cmdStr) || !isCommandSafe(cmdStr) {
		return &ToolResult{ForLLM: "Error: command rejected due to safety sandbox restrictions.", IsError: true}
	}

	timeoutSec := float64(30)
	if to, ok := args["timeout"].(float64); ok && to > 0 {
		timeoutSec = to
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	// Determine the shell based on OS
	var c *exec.Cmd
	if runtime.GOOS == "windows" {
		c = exec.CommandContext(timeoutCtx, "cmd", "/c", cmdStr)
	} else {
		c = exec.CommandContext(timeoutCtx, "sh", "-c", cmdStr)
	}
	c.Dir = t.Sandbox.Workspace

	outBytes, err := c.CombinedOutput()
	resultStr := string(outBytes)

	if timeoutCtx.Err() == context.DeadlineExceeded {
		return &ToolResult{ForLLM: fmt.Sprintf("Timeout after %.0f seconds.\nOutput: %s", timeoutSec, resultStr), IsError: true}
	}

	if err != nil {
		return &ToolResult{ForLLM: fmt.Sprintf("Error: %v\nOutput: %s", err, resultStr), IsError: true}
	}

	return &ToolResult{ForLLM: fmt.Sprintf("Command exited successfully.\nOutput: %s", resultStr), IsError: false}
}
