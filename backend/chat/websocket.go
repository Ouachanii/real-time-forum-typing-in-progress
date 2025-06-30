package chat

import (
	"log"
	"net/http"
	"sync"

	"forum/database"

	"github.com/gorilla/websocket"
)

var (
	upgrader = websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	clients  = make(map[*websocket.Conn]*Client)
	mu       sync.Mutex
)

func HandleConnections(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}
	defer conn.Close()

	nickname := r.URL.Query().Get("nickname")
	if nickname == "" {
		return
	}

	if err := updateUserStatus(nickname, true); err != nil {
		log.Println("Error updating user status:", err)
		return
	}

	// Get the user's ID based on nickname (you'll need to implement this)
	userID, err := getUserIDByNickname(nickname)
	if err != nil {
		log.Println("Error getting user ID:", err)
		return
	}

	// Fetch and send pending notifications on connection
	notifications, err := fetchUnreadNotifications(nickname)
	if err != nil {
		log.Println("Error fetching notifications:", err)
	} else {
		for _, notif := range notifications {
			// Send each notification
			if err := conn.WriteJSON(notif); err != nil {
				log.Printf("Failed to send pending notification to %s: %v", nickname, err)
				continue
			}
		}
	}

	// Get users who have conversations with this user
	usersWithConversations, err := getUsersWithConversations(userID)
	if err != nil {
		log.Println("Error getting conversation users:", err)
	}

	// Get users who don't have conversations with this user
	usersWithoutConversations, err := getUsersWithoutConversations(userID)
	if err != nil {
		log.Println("Error getting non-conversation users:", err)
	}

	client, err := createClient(conn, nickname)
	if err != nil {
		log.Println("Error creating client:", err)
		return
	}

	mu.Lock()
	clients[conn] = client
	broadcastOnlineUsers()
	mu.Unlock()

	// Send conversation data to the client
	conn.WriteJSON(map[string]interface{}{
		"type": "conversation_data",
		"data": map[string]any{
			"with_conversations":    usersWithConversations,
			"without_conversations": usersWithoutConversations,
		},
	})

	defer cleanupClient(conn, nickname)

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			break
		}

		mu.Lock()
		if client, ok := clients[conn]; ok {
			msg.Sender = client.nickname
			msg.SenderFirstName = client.firstName
			msg.SenderLastName = client.lastName
		} else {
			mu.Unlock()
			log.Printf("Client not found for connection from supposed user %s", nickname)
			continue
		}
		mu.Unlock()

		if msg.Sender == "" || msg.Receiver == "" {
			continue
		}

		switch msg.Type {
		case MessageTypeChat:
			if msg.Content == "" {
				log.Printf("Received empty chat message from %s to %s", msg.Sender, msg.Receiver)
				continue
			}
			saveMessage(msg.Sender, msg.Receiver, msg.Content)
			sendPrivateMessage(msg)

		case MessageTypeTypingStart, MessageTypeTypingStop:
			sendPrivateMessage(msg)

		default:
			log.Printf("Received message with unknown or missing type '%s' from %s", msg.Type, msg.Sender)
		}
	}
}

func createClient(conn *websocket.Conn, nickname string) (*Client, error) {
	var firstName, lastName string
	err := database.DB.QueryRow(
		"SELECT first_name, last_name FROM users WHERE nickname = ?", nickname,
	).Scan(&firstName, &lastName)
	if err != nil {
		return nil, err
	}
	return &Client{conn: conn, firstName: firstName, lastName: lastName, nickname: nickname}, nil
}

func cleanupClient(conn *websocket.Conn, nickname string) {
	mu.Lock()
	delete(clients, conn)
	mu.Unlock()

	if err := updateUserStatus(nickname, false); err != nil {
		log.Println("Error updating user status to offline:", err)
	}
	broadcastOnlineUsers()
}

func updateUserStatus(nickname string, online bool) error {
	_, err := database.DB.Exec(`
		INSERT OR REPLACE INTO user_status (user_id, is_online, last_seen)
		SELECT id, ?, CURRENT_TIMESTAMP FROM users WHERE nickname = ?`,
		online, nickname)
	return err
}
