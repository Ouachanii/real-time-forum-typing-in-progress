package interactions

import (
	"fmt" 
	"log"
	"strconv"

	"forum/database"
)

func commentLike(userID int, commentIDStr string, isLike *bool) map[string]interface{} {
	commentID, err := strconv.Atoi(commentIDStr)
	if err != nil {
		log.Printf("Invalid comment ID: %v", err)
		return map[string]interface{}{
			"error": "Invalid comment ID",
		}
	}
	var exists bool
	err = database.DB.QueryRow("SELECT EXISTS (SELECT 1 FROM comments WHERE id = ?)", commentID).Scan(&exists)
	if err != nil {
		log.Printf("Error checking comment existence: %v", err)
		return map[string]interface{}{
			"error": "Failed to validate comment ID",
		}
	}

	if !exists {
		fmt.Println("Comment ID does not exist")
		return map[string]interface{}{
			"error": "Comment ID does not exist",
		}
	}

	if isLike == nil {
		_, err = database.DB.Exec("DELETE FROM comment_likes WHERE user_id = ? AND comment_id = ?", userID, commentID)
		if err != nil {
			log.Printf("Error deleting comment like: %v", err)
			return map[string]interface{}{
				"error": "Failed to remove comment like",
			}
		}
		return map[string]interface{}{
			"message":       "Like removed",
			"updatedIsLike": nil,
		}
	} else {

		query := `
            INSERT INTO comment_likes (user_id, comment_id, is_like, created_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(user_id, comment_id)
			DO UPDATE SET 
    		is_like = excluded.is_like, 
    		created_at = CURRENT_TIMESTAMP
			`
		_, err = database.DB.Exec(query, userID, commentID, isLike)
		if err != nil {
			log.Printf("Error inserting/updating comment like: %v", err)
			return map[string]interface{}{
				"error": "Failed to submit interaction",
			}
		}
		log.Printf("Executing comment like query with userID: %d, commentID: %d, isLike: %v", userID, commentID, isLike)

		log.Println("Interaction added/updated successfully")
		return map[string]interface{}{
			"message":       "Interaction updated successfully",
			"updatedIsLike": *isLike,
		}
	}
}
