package chat

import (
	"database/sql"
	"log"

	"forum/backend/utils" // Assuming utils package path
	"forum/database"    // Assuming database package is accessible
)

func getUserIDByNickname(nickname string) (int, error) {
	var id int
	err := database.DB.QueryRow("SELECT id FROM users WHERE nickname = ?", nickname).Scan(&id)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func fetchUnreadNotifications(nickname string) ([]map[string]interface{}, error) {
	rows, err := database.DB.Query(`
        SELECT 
            n.id,
            u.nickname as sender
        FROM notifications n
        JOIN users u ON n.sender_id = u.id
        WHERE n.user_id = (SELECT id FROM users WHERE nickname = ?)
        ORDER BY n.created_at DESC`,
		nickname)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []map[string]interface{}
	for rows.Next() {
		var id int
		var sender string
		if err := rows.Scan(&id, &sender); err != nil {
			return nil, err
		}
		notifications = append(notifications, map[string]interface{}{
			"type":   "notification",
			"sender": sender,
			"db_id":  id, // For marking as read later
		})
	}
	return notifications, nil
}

func getUsersWithConversations(userID int) ([]string, error) {
	query := `
        SELECT DISTINCT u.nickname 
        FROM users u
        JOIN chats c ON u.id = c.sender_id OR u.id = c.receiver_id
        WHERE (c.sender_id = ? OR c.receiver_id = ?) AND u.id != ?
    `
	rows, err := database.DB.Query(query, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nicknames []string
	for rows.Next() {
		var nickname string
		if err := rows.Scan(&nickname); err != nil {
			return nil, err
		}
		nicknames = append(nicknames, nickname)
	}
	return nicknames, nil
}

// Get users who don't have conversations with the given user
func getUsersWithoutConversations(userID int) ([]string, error) {
	query := `
        SELECT u.nickname 
        FROM users u
        WHERE u.id != ? AND u.id NOT IN (
            SELECT DISTINCT CASE 
                WHEN c.sender_id = ? THEN c.receiver_id 
                ELSE c.sender_id 
            END
            FROM chats c
            WHERE c.sender_id = ? OR c.receiver_id = ?
        )
    `
	rows, err := database.DB.Query(query, userID, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nicknames []string
	for rows.Next() {
		var nickname string
		if err := rows.Scan(&nickname); err != nil {
			return nil, err
		}
		nicknames = append(nicknames, nickname)
	}
	return nicknames, nil
}

func queryUsers(currentUser string) ([]User, error) {
	rows, err := database.DB.Query(`
		SELECT u.id, u.nickname, u.first_name, u.last_name, 
			CASE WHEN s.is_online THEN 1 ELSE 0 END as is_online, s.last_seen
		FROM users u LEFT JOIN user_status s ON u.id = s.user_id
		WHERE u.nickname != ? ORDER BY CASE WHEN s.is_online THEN 0 ELSE 1 END, u.first_name, u.last_name`,
		currentUser)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		var lastSeen sql.NullString
		if err := rows.Scan(&user.ID, &user.Nickname, &user.FirstName, &user.LastName, &user.IsOnline, &lastSeen); err != nil {
			return nil, err
		}
		if lastSeen.Valid {
			user.LastSeen = lastSeen.String
		}
		users = append(users, user)
	}
	return users, nil
}

func saveMessage(sender, receiver, content string) {
	var senderID, receiverID int
	content = utils.EscapeString(content)
	if err := database.DB.QueryRow("SELECT id FROM users WHERE nickname = ?", sender).Scan(&senderID); err != nil {
		log.Println("Error getting sender ID:", err)
		return
	}
	if err := database.DB.QueryRow("SELECT id FROM users WHERE nickname = ?", receiver).Scan(&receiverID); err != nil {
		log.Println("Error getting receiver ID:", err)
		return
	}
	if _, err := database.DB.Exec("INSERT INTO chats (sender_id, receiver_id, message) VALUES (?, ?, ?)", senderID, receiverID, content); err != nil {
		log.Println("Error saving message:", err)
	}
}

func queryMessages(currentUser, otherUser, offset, limit string) ([]Message, error) {
	rows, err := database.DB.Query(`
		SELECT u_sender.nickname, u_receiver.nickname, chats.message, chats.sent_at, 
			u_sender.first_name, u_sender.last_name
		FROM chats
		JOIN users u_sender ON chats.sender_id = u_sender.id
		JOIN users u_receiver ON chats.receiver_id = u_receiver.id
		WHERE (u_sender.nickname = ? AND u_receiver.nickname = ?) OR 
			(u_sender.nickname = ? AND u_receiver.nickname = ?)
		ORDER BY chats.sent_at DESC
		LIMIT ? OFFSET ?`,
		currentUser, otherUser, otherUser, currentUser, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var msgs []Message
	for rows.Next() {
		var msg Message
		if err := rows.Scan(&msg.Sender, &msg.Receiver, &msg.Content, &msg.Timestamp, &msg.SenderFirstName, &msg.SenderLastName); err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}

	return msgs, nil
}

