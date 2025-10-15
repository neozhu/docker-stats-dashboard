package stream

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Hub struct {
	log       *slog.Logger
	clients   map[*client]struct{}
	register  chan *client
	remove    chan *client
	broadcast chan []byte
}

func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		log:       logger,
		clients:   map[*client]struct{}{},
		register:  make(chan *client),
		remove:    make(chan *client),
		broadcast: make(chan []byte, 32),
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.shutdown()
			return
		case c := <-h.register:
			h.clients[c] = struct{}{}
			h.log.Debug("client connected", slog.Int("clients", len(h.clients)))
		case c := <-h.remove:
			h.disconnect(c)
		case msg := <-h.broadcast:
			h.log.Debug("broadcasting payload", slog.Int("clients", len(h.clients)), slog.Int("bytes", len(msg)))
			for c := range h.clients {
				select {
				case c.send <- msg:
				default:
					h.log.Debug("dropping slow client")
					h.disconnect(c)
				}
			}
		}
	}
}

func (h *Hub) Broadcast(data []byte) {
	h.broadcast <- data
}

func (h *Hub) ServeWS(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		h.log.Warn("failed to upgrade websocket", slog.String("error", err.Error()))
		return
	}

	client := &client{
		conn: conn,
		send: make(chan []byte, 16),
		hub:  h,
	}

	h.register <- client

	go client.writePump()
	go client.readPump()
}

func (h *Hub) disconnect(c *client) {
	if _, ok := h.clients[c]; !ok {
		return
	}
	delete(h.clients, c)
	close(c.send)
	c.conn.Close()
	h.log.Debug("client disconnected", slog.Int("clients", len(h.clients)))
}

func (h *Hub) shutdown() {
	for c := range h.clients {
		close(c.send)
		c.conn.Close()
	}
}

type client struct {
	conn *websocket.Conn
	send chan []byte
	hub  *Hub
}

func (c *client) readPump() {
	defer func() {
		c.hub.remove <- c
	}()

	c.conn.SetReadLimit(512)
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		if _, _, err := c.conn.ReadMessage(); err != nil {
			return
		}
	}
}

func (c *client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
