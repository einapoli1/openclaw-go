# OpenClaw Go — Architecture

## Overview

OpenClaw Go is a minimal reimplementation of the OpenClaw gateway in Go. It provides:

1. **Gateway** — WebSocket server that accepts client connections and routes messages to agents
2. **Agent Runtime** — Manages agent sessions, context, and tool execution
3. **Tool System** — Pluggable tools (exec, file I/O, web search, etc.)
4. **Channel Plugins** — Signal, Discord, Telegram, WhatsApp adapters
5. **Config** — JSON5 configuration compatible with the original

## Core Packages

```
cmd/
  openclaw/         # CLI entrypoint
internal/
  gateway/          # WebSocket server, routing, auth
  agent/            # Agent runtime, session management
  tools/            # Tool registry and implementations
  channels/         # Channel plugin system
  config/           # Configuration loading and validation
  session/          # Session store (file-backed)
  model/            # LLM provider abstraction (Anthropic, OpenAI)
pkg/
  protocol/         # Wire protocol types (shared with clients)
```

## Key Design Decisions

### 1. Single binary
Everything compiles to one binary. No npm install, no node_modules.

### 2. Goroutine per session
Each agent session runs in its own goroutine. Tool calls are executed concurrently where safe.

### 3. Provider abstraction
The `model` package defines an interface:
```go
type Provider interface {
    Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error)
    Stream(ctx context.Context, req *ChatRequest) (<-chan StreamEvent, error)
}
```

### 4. Tool interface
```go
type Tool interface {
    Name() string
    Description() string
    Schema() json.RawMessage
    Execute(ctx context.Context, params json.RawMessage) (ToolResult, error)
}
```

### 5. Channel interface
```go
type Channel interface {
    Name() string
    Start(ctx context.Context) error
    Stop() error
    Send(ctx context.Context, msg OutgoingMessage) error
    Receive() <-chan IncomingMessage
}
```

## Phase 1 — MVP
- [ ] Config loading (JSON5)
- [ ] Anthropic provider (messages API)
- [ ] Basic gateway (WebSocket)
- [ ] Exec tool
- [ ] File read/write tools
- [ ] Single agent session
- [ ] CLI interface

## Phase 2 — Feature Parity
- [ ] Multi-agent support
- [ ] Channel plugins (Signal first)
- [ ] Web search tool
- [ ] Memory/recall system
- [ ] Heartbeat system
- [ ] Cron jobs
- [ ] Sub-agent spawning

## Phase 3 — Beyond
- [ ] Native tool sandboxing (seccomp/sandbox-exec)
- [ ] Embedded web UI
- [ ] Plugin system (Go plugins or WASM)
