package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/einapoli1/openclaw-go/internal/model"
	"github.com/einapoli1/openclaw-go/internal/tools"
)

// ToolUseContent represents a tool_use content block from the model
type ToolUseContent struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

// ToolResultContent represents a tool_result content block sent back
type ToolResultContent struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

// RunLoop executes the full agent loop: send message, handle tool calls, repeat
func (m *Manager) RunLoop(ctx context.Context, sessionID, agentID, message string, registry *tools.Registry) (string, error) {
	session := m.GetOrCreate(sessionID, agentID)
	session.mu.Lock()
	defer session.mu.Unlock()

	// Add user message
	session.Messages = append(session.Messages, model.Message{
		Role:    "user",
		Content: []model.Content{{Type: "text", Text: message}},
	})

	system := m.loadSystemPrompt(session.Config)

	// Build tool definitions for the model
	var modelTools []model.Tool
	for _, t := range registry.List() {
		modelTools = append(modelTools, model.Tool{
			Name:        t.Name(),
			Description: t.Description(),
			InputSchema: t.Schema(),
		})
	}

	maxIterations := 20
	for i := 0; i < maxIterations; i++ {
		req := &model.ChatRequest{
			Model:     session.Config.Model,
			Messages:  session.Messages,
			System:    system,
			MaxTokens: 4096,
			Tools:     modelTools,
		}

		resp, err := m.provider.Chat(ctx, req)
		if err != nil {
			return "", fmt.Errorf("model chat (iteration %d): %w", i, err)
		}

		log.Printf("[agent:%s] iteration %d: %d in / %d out, stop=%s",
			agentID, i, resp.Usage.InputTokens, resp.Usage.OutputTokens, resp.StopReason)

		// Add assistant response to history
		session.Messages = append(session.Messages, model.Message{
			Role:    "assistant",
			Content: resp.Content,
		})

		// If stop reason is "end_turn" or no tool use, we're done
		if resp.StopReason != "tool_use" {
			var finalText string
			for _, c := range resp.Content {
				if c.Type == "text" {
					finalText += c.Text
				}
			}
			session.UpdatedAt = session.CreatedAt
			return finalText, nil
		}

		// Execute tool calls
		var toolResults []model.Content
		for _, c := range resp.Content {
			if c.Type != "tool_use" {
				continue
			}

			tool, ok := registry.Get(c.Name)
			if !ok {
				toolResults = append(toolResults, model.Content{
					Type:      "tool_result",
					ToolUseID: c.ID,
					Text:      fmt.Sprintf("unknown tool: %s", c.Name),
					IsError:   true,
				})
				continue
			}

			log.Printf("[agent:%s] tool call: %s", agentID, c.Name)
			result, err := tool.Execute(ctx, c.Input)
			if err != nil {
				result = tools.ToolResult{
					Content: fmt.Sprintf("tool error: %v", err),
					IsError: true,
				}
			}

			toolResults = append(toolResults, model.Content{
				Type:      "tool_result",
				ToolUseID: c.ID,
				Text:      result.Content,
				IsError:   result.IsError,
			})
		}

		// Add tool results as user message
		session.Messages = append(session.Messages, model.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	return "", fmt.Errorf("agent loop exceeded %d iterations", maxIterations)
}
