package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/einapoli1/openclaw-go/internal/config"
	"github.com/einapoli1/openclaw-go/internal/model"
)

// Session represents an active agent conversation
type Session struct {
	ID        string
	AgentID   string
	Config    config.AgentConfig
	Messages  []model.Message
	CreatedAt time.Time
	UpdatedAt time.Time
	mu        sync.Mutex
}

// Manager manages agent sessions
type Manager struct {
	sessions map[string]*Session
	provider model.Provider
	configs  map[string]config.AgentConfig
	mu       sync.RWMutex
}

// NewManager creates a new session manager
func NewManager(provider model.Provider, agents []config.AgentConfig) *Manager {
	configs := make(map[string]config.AgentConfig)
	for _, a := range agents {
		configs[a.ID] = a
	}
	return &Manager{
		sessions: make(map[string]*Session),
		provider: provider,
		configs:  configs,
	}
}

// GetOrCreate returns an existing session or creates a new one
func (m *Manager) GetOrCreate(sessionID, agentID string) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	if s, ok := m.sessions[sessionID]; ok {
		return s
	}

	agentCfg, ok := m.configs[agentID]
	if !ok {
		agentCfg = config.AgentConfig{
			ID:        agentID,
			Identity:  config.AgentIdentity{Name: agentID},
			Workspace: "~/.openclaw/workspace",
			Model:     "claude-sonnet-4-20250514",
		}
	}

	s := &Session{
		ID:        sessionID,
		AgentID:   agentID,
		Config:    agentCfg,
		Messages:  []model.Message{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.sessions[sessionID] = s
	return s
}

// Chat sends a message to the agent and returns the response
func (m *Manager) Chat(ctx context.Context, sessionID, agentID, message string) (string, error) {
	session := m.GetOrCreate(sessionID, agentID)
	session.mu.Lock()
	defer session.mu.Unlock()

	// Add user message
	session.Messages = append(session.Messages, model.Message{
		Role: "user",
		Content: []model.Content{
			{Type: "text", Text: message},
		},
	})

	// Load system prompt from SOUL.md if it exists
	system := m.loadSystemPrompt(session.Config)

	// Build request
	req := &model.ChatRequest{
		Model:     session.Config.Model,
		Messages:  session.Messages,
		System:    system,
		MaxTokens: 4096,
	}

	// Call the model
	resp, err := m.provider.Chat(ctx, req)
	if err != nil {
		return "", fmt.Errorf("model chat: %w", err)
	}

	// Extract text response
	var responseText string
	for _, c := range resp.Content {
		if c.Type == "text" {
			responseText += c.Text
		}
	}

	// Add assistant message to history
	session.Messages = append(session.Messages, model.Message{
		Role:    "assistant",
		Content: resp.Content,
	})
	session.UpdatedAt = time.Now()

	log.Printf("[agent:%s] %d tokens in, %d tokens out",
		agentID, resp.Usage.InputTokens, resp.Usage.OutputTokens)

	return responseText, nil
}

// loadSystemPrompt reads SOUL.md from the agent's workspace
func (m *Manager) loadSystemPrompt(cfg config.AgentConfig) string {
	workspace := cfg.Workspace
	if len(workspace) >= 2 && workspace[:2] == "~/" {
		home, _ := os.UserHomeDir()
		workspace = filepath.Join(home, workspace[2:])
	}

	soulPath := filepath.Join(workspace, "SOUL.md")
	data, err := os.ReadFile(soulPath)
	if err != nil {
		return fmt.Sprintf("You are %s.", cfg.Identity.Name)
	}
	return string(data)
}

// ListSessions returns all active sessions
func (m *Manager) ListSessions() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}
