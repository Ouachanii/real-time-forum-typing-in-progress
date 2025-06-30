package interactions

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"forum/backend/middleware"
	"forum/database"
)

func Interact(w http.ResponseWriter, r *http.Request) {
	_, sessionToken, loggedIn, _ := middleware.RequireLogin(w, r) // Add 'middleware.' prefix
	if !loggedIn {
		http.Error(w, "Unauthorized: User is not logged in", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	isLikeStr := r.FormValue("is_like")
	postIDStr := r.FormValue("post_id")
	commentIDStr := r.FormValue("comment_id")

	var isLike *bool
	if isLikeStr != "" {
		parsedIsLike, err := strconv.ParseBool(isLikeStr)
		if err != nil {
			http.Error(w, "Invalid is_like value", http.StatusBadRequest)
			return
		}
		isLike = &parsedIsLike
	}

	var userID int
	err := database.DB.QueryRow("SELECT id FROM users WHERE session_token = ?", sessionToken).Scan(&userID)
	if err != nil {
		log.Printf("Error fetching user ID: %v", err)
		http.Error(w, "Unauthorized: Invalid session", http.StatusUnauthorized)
		return
	}

	var response map[string]interface{}
	if postIDStr != "" {
		response = postLike(userID, postIDStr, isLike)
	} else if commentIDStr != "" {
		response = commentLike(userID, commentIDStr, isLike)
	} else {
		http.Error(w, "Either post_id or comment_id must be specified", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
