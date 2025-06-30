package posts

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"forum/backend/comments"   
	"forum/backend/middleware" 
	"forum/backend/models"
	"forum/database"
)

func ShowPosts(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]interface{})

	if r.Method != http.MethodGet {
		response["error"] = "Invalid request method."
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	_, sessionToken, _, err := middleware.RequireLogin(w, r) // Add 'middleware.' prefix
	if err != nil {
		response["error"] = "Unauthorized access. Please log in."
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	category := r.URL.Query().Get("category")
	var postStmt string
	var postRows *sql.Rows

	if category == "all" || category == "" {
		postStmt = "SELECT id, title, content, created_at FROM Posts ORDER BY created_at DESC"
		postRows, err = database.DB.Query(postStmt)
	} else {
		postStmt = `
			SELECT p.id, p.title, p.content, p.created_at
			FROM Posts p
			INNER JOIN post_categories pc ON p.id = pc.post_id
			INNER JOIN categories c ON pc.category_id = c.id
			WHERE c.name = ?
			ORDER BY p.created_at DESC
		`
		postRows, err = database.DB.Query(postStmt, category)
	}

	if err != nil {
		log.Printf("Error querying posts: %v", err)
		http.Error(w, "Error retrieving posts", http.StatusInternalServerError)
		return
	}
	defer postRows.Close()

	var posts []models.PostWithLike
	for postRows.Next() {
		var p models.Post
		var postWithLike models.PostWithLike
		var postID int
		var createdAt time.Time
		err = postRows.Scan(&postID, &p.Title, &p.Content, &createdAt)
		if err != nil {
			log.Printf("Error scanning post: %v", err)
			continue
		}
		p.PostID = postID
		p.CreatedAt = createdAt

		var userID int
		userIdStmt := "SELECT user_id FROM posts WHERE id = ?"
		err = database.DB.QueryRow(userIdStmt, postID).Scan(&userID)
		if err != nil {
			response["error"] = "Unauthorized access. Please log in."
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		authorStmt := "SELECT nickname FROM users WHERE id = ?"
		err = database.DB.QueryRow(authorStmt, userID).Scan(&p.Author)
		if err != nil {
			response["error"] = "Failed to get author information."
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
			return
		}

		if sessionToken != "guest" {
			var isLike sql.NullBool
			err = database.DB.QueryRow(`
				SELECT is_like FROM post_likes
				WHERE post_id = ? AND user_id = ?
			`, postID, userID).Scan(&isLike)
			if err != nil && err != sql.ErrNoRows {
				log.Printf("Error retrieving like status for post %d: %v", postID, err)
				continue
			}
			if isLike.Valid {
				if isLike.Bool {
					postWithLike.IsLike = 1
				} else {
					postWithLike.IsLike = 2
				}
			} else {
				postWithLike.IsLike = -1
			}
		}

		err = database.DB.QueryRow(`
			SELECT
				COUNT(CASE WHEN is_like = true THEN 1 END) AS like_count,
				COUNT(CASE WHEN is_like = false THEN 1 END) AS dislike_count
			FROM post_likes
			WHERE post_id = ?
		`, postID).Scan(&postWithLike.LikeCount, &postWithLike.DislikeCount)
		if err != nil {
			log.Printf("Error retrieving like/dislike counts for post %d: %v", postID, err)
			continue
		}

		catStmt := `
			SELECT c.name
			FROM categories c
			INNER JOIN post_categories pc ON c.id = pc.category_id
			WHERE pc.post_id = ?`
		catRows, err := database.DB.Query(catStmt, postID)
		if err != nil {
			log.Printf("Error querying categories for post %d: %v", postID, err)
			continue
		}

		var categories []string
		for catRows.Next() {
			var category string
			if err := catRows.Scan(&category); err != nil {
				log.Printf("Error scanning category for post %d: %v", postID, err)
				continue
			}
			categories = append(categories, category)
		}
		catRows.Close()

		comments, err := comments.ShowComments(postID, w, r) 
		if err != nil {
			log.Printf("Error retrieving comments for post %d: %v", postID, err)
			comments = []models.CommentWithLike{}
		}

		p.Categories = categories
		p.Comments = comments

		postWithLike.Post = p

		posts = append(posts, postWithLike)
	}

	if len(posts) == 0 {
		log.Println("No posts found.")
		posts = []models.PostWithLike{}
	}

	w.Header().Set("Content-Type", "application/json")
	err = json.NewEncoder(w).Encode(posts)
	if err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		response["error"] = "Error processing posts"
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}
}
