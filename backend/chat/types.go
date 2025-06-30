package chat

import "github.com/gorilla/websocket"

// Message types
const (
	MessageTypeChat        = "chat_message"
	MessageTypeTypingStart = "typing_start"
	MessageTypeTypingStop  = "typing_stop"
)

type Message struct {
	Type            string `json:"type"`
	Sender          string `json:"sender"`
	Receiver        string `json:"receiver"`
	Content         string `json:"content,omitempty"` 
	Timestamp       string `json:"timestamp,omitempty"`
	SenderFirstName string `json:"firstName,omitempty"`
	SenderLastName  string `json:"lastName,omitempty"`
}

type Client struct {
	conn      *websocket.Conn
	firstName string
	lastName  string
	nickname  string
}

type User struct {
	ID        int    `json:"id"`
	Nickname  string `json:"nickname"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	IsOnline  bool   `json:"isOnline"`
	LastSeen  string `json:"lastSeen,omitempty"`
}


// this method to safely access the connection
func (c *Client) Conn() *websocket.Conn {
	return c.conn
}

// this method to send JSON
func (c *Client) SendJSON(v interface{}) error {
	return c.conn.WriteJSON(v)
}
