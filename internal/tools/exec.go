package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// ToolResult is the output of a tool execution
type ToolResult struct {
	Content string `json:"content"`
	IsError bool   `json:"is_error"`
}

// Tool is the interface all tools implement
type Tool interface {
	Name() string
	Description() string
	Schema() json.RawMessage
	Execute(ctx context.Context, params json.RawMessage) (ToolResult, error)
}

// ExecTool runs shell commands
type ExecTool struct {
	workdir string
}

type execParams struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout"` // seconds
}

// NewExecTool creates a new exec tool
func NewExecTool(workdir string) *ExecTool {
	return &ExecTool{workdir: workdir}
}

func (t *ExecTool) Name() string        { return "exec" }
func (t *ExecTool) Description() string  { return "Execute a shell command and return its output" }
func (t *ExecTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"command": {"type": "string", "description": "Shell command to execute"},
			"timeout": {"type": "integer", "description": "Timeout in seconds (default 30)"}
		},
		"required": ["command"]
	}`)
}

func (t *ExecTool) Execute(ctx context.Context, params json.RawMessage) (ToolResult, error) {
	var p execParams
	if err := json.Unmarshal(params, &p); err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	timeout := 30
	if p.Timeout > 0 {
		timeout = p.Timeout
	}

	execCtx, cancel := context.WithTimeout(ctx, time.Duration(timeout)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(execCtx, "sh", "-c", p.Command)
	cmd.Dir = t.workdir

	out, err := cmd.CombinedOutput()
	output := strings.TrimSpace(string(out))

	if err != nil {
		if execCtx.Err() == context.DeadlineExceeded {
			return ToolResult{
				Content: fmt.Sprintf("Command timed out after %ds\n%s", timeout, output),
				IsError: true,
			}, nil
		}
		return ToolResult{
			Content: fmt.Sprintf("%s\n%s", output, err.Error()),
			IsError: true,
		}, nil
	}

	return ToolResult{Content: output}, nil
}

// ReadTool reads file contents
type ReadTool struct {
	workdir string
}

type readParams struct {
	Path string `json:"path"`
}

func NewReadTool(workdir string) *ReadTool {
	return &ReadTool{workdir: workdir}
}

func (t *ReadTool) Name() string        { return "read" }
func (t *ReadTool) Description() string  { return "Read the contents of a file" }
func (t *ReadTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to read"}
		},
		"required": ["path"]
	}`)
}

func (t *ReadTool) Execute(ctx context.Context, params json.RawMessage) (ToolResult, error) {
	var p readParams
	if err := json.Unmarshal(params, &p); err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	path := p.Path
	if !strings.HasPrefix(path, "/") {
		path = t.workdir + "/" + path
	}

	data, err := exec.Command("cat", path).Output()
	if err != nil {
		return ToolResult{Content: fmt.Sprintf("Error reading %s: %v", p.Path, err), IsError: true}, nil
	}

	content := string(data)
	if len(content) > 50000 {
		content = content[:50000] + "\n... (truncated)"
	}

	return ToolResult{Content: content}, nil
}

// WriteTool writes content to a file
type WriteTool struct {
	workdir string
}

type writeParams struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

func NewWriteTool(workdir string) *WriteTool {
	return &WriteTool{workdir: workdir}
}

func (t *WriteTool) Name() string        { return "write" }
func (t *WriteTool) Description() string  { return "Write content to a file" }
func (t *WriteTool) Schema() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "Path to the file to write"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["path", "content"]
	}`)
}

func (t *WriteTool) Execute(ctx context.Context, params json.RawMessage) (ToolResult, error) {
	var p writeParams
	if err := json.Unmarshal(params, &p); err != nil {
		return ToolResult{Content: err.Error(), IsError: true}, nil
	}

	path := p.Path
	if !strings.HasPrefix(path, "/") {
		path = t.workdir + "/" + path
	}

	// Create parent directories
	dir := path[:strings.LastIndex(path, "/")]
	exec.Command("mkdir", "-p", dir).Run()

	if err := exec.Command("sh", "-c", fmt.Sprintf("cat > %q", path)).Run(); err != nil {
		// Fallback: use tee
		cmd := exec.Command("tee", path)
		cmd.Stdin = strings.NewReader(p.Content)
		if err := cmd.Run(); err != nil {
			return ToolResult{Content: fmt.Sprintf("Error writing %s: %v", p.Path, err), IsError: true}, nil
		}
	}

	return ToolResult{Content: fmt.Sprintf("Successfully wrote %d bytes to %s", len(p.Content), p.Path)}, nil
}

// Registry holds all available tools
type Registry struct {
	tools map[string]Tool
}

// NewRegistry creates a tool registry with default tools
func NewRegistry(workdir string) *Registry {
	r := &Registry{tools: make(map[string]Tool)}
	r.Register(NewExecTool(workdir))
	r.Register(NewReadTool(workdir))
	r.Register(NewWriteTool(workdir))
	return r
}

// Register adds a tool to the registry
func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

// Get returns a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	tools := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		tools = append(tools, t)
	}
	return tools
}
