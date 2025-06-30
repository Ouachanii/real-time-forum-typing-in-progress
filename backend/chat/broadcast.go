package chat

import (
	"log"
	"time"

	"forum/database" // Assuming database package is accessible
)

func BroadcastNewUser(nickname, firstName, lastName string) {
	mu.Lock()
	defer mu.Unlock()

	msg := map[string]interface{}{
		"type": "userRegistered",
		"user": map[string]string{
			"nickname":  nickname,
			"firstName": firstName,
			"lastName":  lastName,
		},
	}

	for conn, client := range clients {
		if err := client.SendJSON(msg); err != nil {
			log.Printf("Broadcast error: %v", err)
			client.Conn().Close()
			delete(clients, conn)
		}
	}
}
func broadcastOnlineUsers() {
	userList := make([]map[string]string, 0, len(clients))
	for _, client := range clients {
		userList = append(userList, map[string]string{
			"nickname":  client.nickname,
			"firstName": client.firstName,
			"lastName":  client.lastName,
		})
	}

	message := map[string]interface{}{
		"type":  "onlineUsers",
		"users": userList,
	}

	for _, client := range clients {
		if err := client.conn.WriteJSON(message); err != nil {
			client.conn.Close()
			delete(clients, client.conn)
		}
	}
}

// sendPrivateMessage sends a message (chat or typing notification) to a specific user.
func sendPrivateMessage(msg Message) {
	mu.Lock()
	defer mu.Unlock()

	// Find the recipient client
	var recipientClient *Client
	for _, client := range clients { // Changed 'conn, client' to '_, client' as conn is unused here
		if client.nickname == msg.Receiver {
			recipientClient = client
			break
		}
	}

	// If recipient is connected, add timestamp and send the message
	if recipientClient != nil {
		// Add timestamp only for actual chat messages before sending
		if msg.Type == MessageTypeChat {
			msg.Timestamp = time.Now().Format(time.RFC3339) // Add current server time
		}
		// For typing notifications, timestamp might not be needed or relevant, so we skip adding it.

		err := recipientClient.SendJSON(msg) // Send the potentially modified message object
		if err != nil {
			log.Printf("Error sending private message type '%s' to %s: %v", msg.Type, msg.Receiver, err)
			// Optional: Handle error, maybe close connection if write fails repeatedly
			// recipientClient.Conn().Close()
			// delete(clients, recipientClient.Conn())
			// broadcastOnlineUsers() // Update user list if client is removed
		} else {
			// If it was a chat message and sending succeeded, send a notification event and store in DB
			if msg.Type == MessageTypeChat {
				// 1. Send the real-time notification event via WebSocket
				notificationEvent := map[string]string{
					"type":   "notification", // Explicit type for frontend handler
					"sender": msg.Sender,     // Let frontend know who sent it
				}
				if errNotif := recipientClient.SendJSON(notificationEvent); errNotif != nil {
					log.Printf("Error sending notification event to %s: %v", msg.Receiver, errNotif)
					// Decide if failure to send notification event is critical
				}

				// 2. Store the notification in the database
				senderID, errSender := getUserIDByNickname(msg.Sender)
				receiverID, errReceiver := getUserIDByNickname(msg.Receiver)

				if errSender == nil && errReceiver == nil {
					_, err := database.DB.Exec(`
                        INSERT INTO notifications (user_id, sender_id, is_read)
                        VALUES (?, ?, ?)`,
						receiverID, senderID, false) // Store notification as unread initially
					if err != nil {
						log.Printf("Failed to store notification for message from %s to %s: %v", msg.Sender, msg.Receiver, err)
					}
				} else {
					log.Printf("Could not get user IDs to store notification for message from %s to %s", msg.Sender, msg.Receiver)
				}
			}
			// No notification needed for typing start/stop messages
		}
	} else {
		// log.Printf("Recipient %s not found or not connected for message type '%s'", msg.Receiver, msg.Type) // Commented out noisy log
		// If recipient is offline and it's a chat message, still store the notification
		if msg.Type == MessageTypeChat {
			senderID, errSender := getUserIDByNickname(msg.Sender)
			receiverID, errReceiver := getUserIDByNickname(msg.Receiver)

			if errSender == nil && errReceiver == nil {
				_, err := database.DB.Exec(`
                    INSERT INTO notifications (user_id, sender_id, is_read)
                    VALUES (?, ?, ?)`,
					receiverID, senderID, false) // Store notification as unread
				if err != nil {
					log.Printf("Failed to store notification for offline message from %s to %s: %v", msg.Sender, msg.Receiver, err)
				}
			} else {
				log.Printf("Could not get user IDs to store notification for offline message from %s to %s", msg.Sender, msg.Receiver)
			}
		}
	}
}
