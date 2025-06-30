package auth

import (
	"database/sql"
	"encoding/json"
	"fmt" // Needed if LogoutHandler uses fmt.Fprintln (though redirection makes it redundant)
	"log"
	"net/http"
	"strings" // Needed for ToLower in LoginHandler
	"time"

	"forum/backend/utils" // For EscapeString
	"forum/database"
	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/bcrypt"
)

func LoginHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		response := map[string]string{"error": "Method not allowed"}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	identifier := utils.EscapeString(r.FormValue("email"))
	password := utils.EscapeString(r.FormValue("password"))
	lowerIdentifier := strings.ToLower(identifier)

	const maxIdentifier = 100
	const maxPassword = 100

	if len(identifier) > maxIdentifier {
		response := map[string]string{"error": fmt.Sprintf("Nickname/Email cannot be longer than %d characters", maxIdentifier)}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	if len(password) > maxPassword {
		response := map[string]string{"error": fmt.Sprintf("Password cannot be longer than %d characters", maxPassword)}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	var userID int
	var storedPassword, sessionToken, nickname string
	err := database.DB.QueryRow(
		"SELECT id, password, session_token, nickname FROM users WHERE email = ? OR nickname = ?",
		lowerIdentifier,
		identifier,
	).Scan(&userID, &storedPassword, &sessionToken, &nickname)
	if err != nil {
		if err == sql.ErrNoRows {
			response := map[string]string{"error": "Invalid nickname/email or password"}
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(response)
		} else {
			log.Printf("Database error: %v", err)
			response := map[string]string{"error": "Internal server error"}
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(response)
		}
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password)); err != nil {
		response := map[string]string{"error": "Invalid nickname/email or password"}
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(response)
		return
	}

	newSessionToken, _ := uuid.NewV4()
	sessionToken = newSessionToken.String()

	_, err = database.DB.Exec(
		"UPDATE users SET session_token = ? WHERE id = ?",
		sessionToken,
		userID,
	)
	if err != nil {
		log.Printf("Error updating session token: %v", err)
		response := map[string]string{"error": "Internal server error"}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:    "session_token",
		Value:   sessionToken,
		Expires: time.Now().Add(24 * time.Hour),
		Path:    "/",
	})

	response := map[string]string{
		"message":  "Login successful!",
		"nickname": nickname,
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
