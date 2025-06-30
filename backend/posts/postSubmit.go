package posts

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings" 
	"time"

	"forum/backend/middleware"
	"forum/backend/utils"     
	"forum/database"
)

func PostSubmit(w http.ResponseWriter, r *http.Request) {
	nickname, sessionToken, loggedIn, _ := middleware.RequireLogin(w, r)
	response := make(map[string]interface{})

	if !loggedIn {
		log.Println("User not logged in")
		response["error"] = "You need to log in to submit a post."
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	if r.Method != http.MethodPost {
		response["error"] = "Invalid request method."
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	title := utils.EscapeString(r.FormValue("title"))
	content := utils.EscapeString(r.FormValue("content"))
	categoryNames := r.Form["category"]

	const maxTitle = 100
	const maxContent = 1000

	if strings.TrimSpace(title) == "" || strings.TrimSpace(content) == "" || len(categoryNames) == 0 {
		response["error"] = "All fields (title, content, and category) are required."
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(title) > maxTitle {
		response["error"] = fmt.Sprintf("Title cannot be longer than %d characters.", maxTitle)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(content) > maxContent {
		response["error"] = fmt.Sprintf("Content cannot be longer than %d characters.", maxContent)
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var exists bool
	err := database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE nickname = ?)", nickname).Scan(&exists)
	if err != nil {
		log.Printf("Error checking user existence: %v", err)
		response["error"] = "Failed to validate user."
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if !exists {
		response["error"] = "User not found."
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var lastPostTime time.Time
	err = database.DB.QueryRow(`
		SELECT created_at
		FROM Posts
		WHERE user_id = (SELECT id FROM users WHERE session_token = ?)
		ORDER BY created_at DESC
		LIMIT 1
	`, sessionToken).Scan(&lastPostTime)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("Error checking last post time: %v", err)
		response["error"] = "Failed to validate post frequency."
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	if err != sql.ErrNoRows {
		timeSinceLastPost := time.Since(lastPostTime)
		const postCooldown = 1 * time.Second
		if timeSinceLastPost < postCooldown {
			response["error"] = fmt.Sprintf(
				"You can only create a post every 1 second. Please wait %d seconds.",
				int(postCooldown.Seconds()-timeSinceLastPost.Seconds()),
			)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			return
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		response["error"] = "Database error during category linking."
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	insertPostQuery := `
		INSERT INTO Posts (user_id, title, content, created_at)
		SELECT id, ?, ?, ? FROM users WHERE session_token = ?
	`
	result, err := tx.Exec(insertPostQuery, title, content, time.Now(), sessionToken)
	if err != nil {
		log.Printf("Error inserting post: %v", err)
		response["error"] = "Failed to submit post."
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		tx.Rollback()
		return
	}

	postID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving post ID: %v", err)
		response["error"] = "Failed to retrieve post ID."
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		tx.Rollback()
		return
	}

	for _, categoryName := range categoryNames {
		var categoryID int
		err := tx.QueryRow("SELECT id FROM categories WHERE name = ?", categoryName).Scan(&categoryID)
		if err == sql.ErrNoRows {
			response["error"] = fmt.Sprintf("Category '%s' not found.", categoryName)
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(response)
			tx.Rollback()
			return
		} else if err != nil {
			log.Printf("Error during category lookup: %v", err)
			response["error"] = "Database error during category lookup."
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			tx.Rollback()
			return
		}

		insertPostCategoryQuery := `
			INSERT INTO post_categories (post_id, category_id)
			VALUES (?, ?)
		`
		_, err = tx.Exec(insertPostCategoryQuery, postID, categoryID)
		if err != nil {
			log.Printf("Error inserting post-category link: %v", err)
			response["error"] = "Failed to link post with categories."
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			tx.Rollback()
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		log.Printf("Error committing transaction: %v", err)
		response["error"] = "Failed to finalize post submission."
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response["message"] = "Post submitted successfully."
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}