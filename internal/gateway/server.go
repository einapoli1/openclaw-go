package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// Server is the gateway WebSocket server
type Server struct {
	addr    string
	token   string
	clients map[*Client]bool
	mu      sync.RWMutex
	srv     *http.Server
}

// Client represents a connected WebSocket client
type Client struct {
	conn   *websocket.Conn
	send   chan []byte
	server *Server
}

// RPCRequest is an incoming RPC call
type RPCRequest struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

// RPCResponse is an outgoing RPC response
type RPCResponse struct {
	ID      string      `json:"id"`
	Payload interface{} `json:"payload,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// NewServer creates a new gateway server
func NewServer(addr, token string) *Server {
	return &Server{
		addr:    addr,
		token:   token,
		clients: make(map[*Client]bool),
	}
}

// Start begins listening for connections
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleDashboard)
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	s.srv = &http.Server{
		Addr:    s.addr,
		Handler: mux,
	}

	log.Printf("[gateway] listening on ws://%s", s.addr)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.srv.Shutdown(shutdownCtx)
	}()

	if err := s.srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("listen: %w", err)
	}
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"version": "0.0.1",
		"uptime":  time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `<!doctype html>
<html><head><title>OpenClaw Go</title></head>
<body style="font-family:system-ui;max-width:600px;margin:40px auto;padding:0 20px">
<h1>⚡ OpenClaw Go</h1>
<p>Gateway is running.</p>
<p>Clients connected: %d</p>
</body></html>`, len(s.clients))
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Token auth
	if s.token != "" {
		tok := r.URL.Query().Get("token")
		if tok == "" {
			tok = r.Header.Get("Authorization")
		}
		if tok != s.token && tok != "Bearer "+s.token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[gateway] upgrade error: %v", err)
		return
	}

	client := &Client{
		conn:   conn,
		send:   make(chan []byte, 256),
		server: s,
	}

	s.mu.Lock()
	s.clients[client] = true
	s.mu.Unlock()

	log.Printf("[gateway] client connected (%d total)", len(s.clients))

	go client.writePump()
	go client.readPump()
}

func (c *Client) readPump() {
	defer func() {
		c.server.mu.Lock()
		delete(c.server.clients, c)
		c.server.mu.Unlock()
		c.conn.Close()
		log.Printf("[gateway] client disconnected (%d remaining)", len(c.server.clients))
	}()

	c.conn.SetReadLimit(1 << 20) // 1MB
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("[gateway] read error: %v", err)
			}
			break
		}

		var req RPCRequest
		if err := json.Unmarshal(message, &req); err != nil {
			c.sendError("", "invalid JSON")
			continue
		}

		c.handleRPC(req)
	}
}

func (c *Client) writePump() {
	defer c.conn.Close()
	for msg := range c.send {
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

func (c *Client) handleRPC(req RPCRequest) {
	switch req.Method {
	case "ping":
		c.sendResponse(req.ID, map[string]string{"pong": "ok"})
	case "status":
		c.sendResponse(req.ID, map[string]interface{}{
			"version": "0.0.1",
			"clients": len(c.server.clients),
		})
	default:
		c.sendError(req.ID, fmt.Sprintf("unknown method: %s", req.Method))
	}
}

func (c *Client) sendResponse(id string, payload interface{}) {
	resp := RPCResponse{ID: id, Payload: payload}
	data, _ := json.Marshal(resp)
	c.send <- data
}

func (c *Client) sendError(id string, errMsg string) {
	resp := RPCResponse{ID: id, Error: errMsg}
	data, _ := json.Marshal(resp)
	c.send <- data
}
