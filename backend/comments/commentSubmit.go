package comments

import (
	"encoding/json" 
	"log"
	"net/http"
	"strconv" 
	"time"   

	"forum/backend/middleware" 
	"forum/backend/utils"     
	"forum/database"
)

func CommentSubmit(w http.ResponseWriter, r *http.Request) {
	_, sessionToken, loggedIn, _ := middleware.RequireLogin(w, r) // Add 'middleware.' prefix	
	response := make(map[string]interface{})

	if !loggedIn {
		http.Error(w, "Unauthorized: User is not logged in", http.StatusUnauthorized)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	comment := utils.EscapeString(r.FormValue("comment"))
	postIDStr := r.FormValue("post_id")

	if comment == "" {
		http.Error(w, "Comment field is empty", http.StatusBadRequest)
		return
	}

	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM posts WHERE id = ?)", postID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking post existence: %v", err)
		response["error"] = "Failed to validate post ID"
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !exists {
		response["error"] = "Post ID does not exist"
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	insertCommentQuery := `
		INSERT INTO comments (user_id, post_id, content, created_at)
		SELECT id, ?, ?, ? FROM users WHERE session_token = ?
	`
	_, err = database.DB.Exec(insertCommentQuery, postID, comment, time.Now(), sessionToken)
	if err != nil {
		http.Error(w, "Failed to submit comment", http.StatusInternalServerError)
		log.Printf("Error inserting comment: %v", err)
		return
	}

	response["message"] = "Comment submitted successfully"
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}