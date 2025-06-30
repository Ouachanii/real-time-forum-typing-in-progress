package chat

import (
	"encoding/json"
	"log"
	"net/http"

	"forum/database" // Assuming database package is accessible
)

func MarkNotificationsRead(w http.ResponseWriter, r *http.Request) {
	var request struct {
		Receiver string `json:"receiver"`
		Sender   string `json:"sender"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// DELETE instead of UPDATE
	_, err := database.DB.Exec(`
        DELETE FROM notifications 
        WHERE user_id = (SELECT id FROM users WHERE nickname = ?)
        AND sender_id = (SELECT id FROM users WHERE nickname = ?)`,
		request.Receiver, request.Sender)
	if err != nil {
		http.Error(w, "Deletion failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
func GetNotifications(w http.ResponseWriter, r *http.Request) {
	nickname := r.URL.Query().Get("nickname")

	// ONLY return truly unread notifications
	rows, err := database.DB.Query(`
		SELECT n.id, u.nickname as sender
		FROM notifications n
		JOIN users u ON n.sender_id = u.id
		WHERE n.user_id = (SELECT id FROM users WHERE nickname = ?)
		AND n.is_read = FALSE
		ORDER BY n.created_at DESC`,
		nickname)
	if err != nil {
		log.Println("Error getting unread notifications users:", err)
		return
	}
	defer rows.Close()

	notifications, err := fetchUnreadNotifications(nickname) // Use your existing function
	if err != nil {
		http.Error(w, "Failed to fetch notifications", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}
func GetAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := r.URL.Query().Get("nickname")
	if currentUser == "" {
		http.Error(w, "Missing nickname", http.StatusBadRequest)
		return
	}

	users, err := queryUsers(currentUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, users)
}
func FetchMessagesHandler(w http.ResponseWriter, r *http.Request) {
	currentUser := r.URL.Query().Get("nickname")
	otherUser := r.URL.Query().Get("otherUser")
	offset := r.URL.Query().Get("offset")
	limit := r.URL.Query().Get("limit")

	if currentUser == "" || otherUser == "" {
		http.Error(w, "Missing nickname or otherUser", http.StatusBadRequest)
		return
	}

	// Set default values if not provided
	if offset == "" {
		offset = "0"
	}
	if limit == "" {
		limit = "10"
	}

	messages, err := queryMessages(currentUser, otherUser, offset, limit)
	if err != nil {
		http.Error(w, "Failed to fetch messages: "+err.Error(), http.StatusInternalServerError)
		return
	}

	jsonResponse(w, messages)
}
func jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Println("Error encoding JSON response:", err)
	}
}
