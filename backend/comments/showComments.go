package comments

import (
	"database/sql"
	"fmt"      
	"log"
	"net/http"
	"time"

	"forum/backend/middleware" 
	"forum/backend/models"   
	"forum/database"
)


func ShowComments(postID int, w http.ResponseWriter, r *http.Request) ([]models.CommentWithLike, error) {
	_, sessionToken, _, err := middleware.RequireLogin(w, r) 	
	if err != nil {
		fmt.Println("Error in cookie:", err)
		http.Error(w, "Unauthorized access. Please log in.", http.StatusUnauthorized)
		return nil, err
	}

	commentStmt := `
		SELECT c.id, c.content, c.created_at, u.nickname 
		FROM comments c
		INNER JOIN users u ON c.user_id = u.id
		WHERE c.post_id = ? 
		ORDER BY c.created_at DESC
	`
	commentRows, err := database.DB.Query(commentStmt, postID)
	if err != nil {
		return nil, fmt.Errorf("error querying comments: %v", err)
	}
	defer commentRows.Close()

	var comments []models.CommentWithLike
	for commentRows.Next() {
		var c models.Comment
		var commentWithLike models.CommentWithLike
		var commentID int
		var createdAt time.Time

		err = commentRows.Scan(&commentID, &c.Content, &createdAt, &c.Author)
		if err != nil {
			log.Printf("Error scanning comment: %v", err)
			continue
		}
		c.CreatedAt = createdAt

		if sessionToken != "guest" {
			var userID int
			err = database.DB.QueryRow("SELECT id FROM users WHERE session_token = ?", sessionToken).Scan(&userID)
			if err != nil {
				http.Error(w, "Unauthorized access. Please log in.", http.StatusUnauthorized)
				return nil, err
			}

			var isLike sql.NullBool
			err = database.DB.QueryRow(`
				SELECT is_like FROM comment_likes
				WHERE comment_id = ? AND user_id = ?
			`, commentID, userID).Scan(&isLike)

			if err != nil && err != sql.ErrNoRows {
				log.Printf("Error retrieving like status for comment %d: %v", commentID, err)
				continue
			}

			if isLike.Valid {
				if isLike.Bool {
					commentWithLike.IsLike = 1
				} else {
					commentWithLike.IsLike = 2
				}
			} else {
				commentWithLike.IsLike = -1
			}
		}

		err = database.DB.QueryRow(`
			SELECT
				COUNT(CASE WHEN is_like = true THEN 1 END),
				COUNT(CASE WHEN is_like = false THEN 1 END)
			FROM comment_likes
			WHERE comment_id = ?
		`, commentID).Scan(&commentWithLike.LikeCount, &commentWithLike.DislikeCount)
		if err != nil {
			log.Printf("Error retrieving like/dislike counts for comment %d: %v", commentID, err)
			continue
		}

		commentWithLike.Comment = c
		commentWithLike.CommentID = commentID

		comments = append(comments, commentWithLike)
	}

	return comments, nil
}
