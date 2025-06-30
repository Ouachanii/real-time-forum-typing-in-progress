package interactions

import (
	"fmt" 
	"log"
	"strconv"

	"forum/database"
)

func postLike(userID int, postIDStr string, isLike *bool) map[string]interface{} {
	postID, err := strconv.Atoi(postIDStr)
	if err != nil {
		log.Printf("Invalid post ID: %v", err)
		return map[string]interface{}{
			"error": "Invalid post ID",
		}
	}
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM posts WHERE id = ?)", postID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking post existence: %v", err)
		return map[string]interface{}{
			"error": "Failed to validate post ID",
		}
	}

	if !exists {
		fmt.Println("post ID does not exist")
		return map[string]interface{}{
			"error": "post ID does not exist",
		}
	}

	if isLike == nil {
		_, err = database.DB.Exec("DELETE FROM post_likes WHERE user_id = ? AND post_id = ?", userID, postID)
		if err != nil {
			log.Printf("Error deleting post like: %v", err)
			return map[string]interface{}{
				"error": "Invalid comment ID",
			}
		}
		return map[string]interface{}{
			"message":       "Like removed",
			"updatedIsLike": nil,
		}
	} else {
		query := `
		            INSERT INTO post_likes (user_id, post_id, is_like, created_at)
					VALUES (?, ?, ?, CURRENT_TIMESTAMP)
					ON CONFLICT(user_id, post_id)
					DO UPDATE SET 
		    		is_like = excluded.is_like, 
		    		created_at = CURRENT_TIMESTAMP
					`
		_, err = database.DB.Exec(query, userID, postID, isLike)
		if err != nil {
			return map[string]interface{}{
				"error": "Failed to submit interaction",
			}
		}

		return map[string]interface{}{
			"message":       "Interaction updated successfully",
			"updatedIsLike": *isLike,
		}
	}
}