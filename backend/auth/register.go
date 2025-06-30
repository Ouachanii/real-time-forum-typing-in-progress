package auth

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv" 

	"forum/backend/chat" 
	"forum/backend/utils"
	"forum/database"
	"github.com/gofrs/uuid/v5"
	"golang.org/x/crypto/bcrypt"
)

func RegisterHandler(w http.ResponseWriter, r *http.Request) {
	response := make(map[string]string)

	if r.Method != http.MethodPost {
		response = map[string]string{"error": "Method not allowed"}
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(response)
		return
	}

	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Failed to parse form data", http.StatusBadRequest)
		return
	}

	nickname := utils.EscapeString(r.FormValue("nickname"))
	email := utils.EscapeString(r.FormValue("email"))
	password := utils.EscapeString(r.FormValue("password"))
	firstName := utils.EscapeString(r.FormValue("first_name"))
	lastName := utils.EscapeString(r.FormValue("last_name"))
	ageStr := r.FormValue("age")
	gender := utils.EscapeString(r.FormValue("gender"))

	age, err := strconv.Atoi(ageStr)
	if err != nil {
		response = map[string]string{"error": "Invalid age value"}
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(response)
		return
	}

	errors, valid := Validation(nickname, email, password, firstName, lastName, age, gender)
	if !valid {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Validation error",
			"fields": errors,
		})
		return
	}

	var existingNickname, existingEmail string
	err = database.DB.QueryRow(
		"SELECT nickname, email FROM users WHERE email = ? OR nickname = ?",
		email,
		nickname,
	).Scan(&existingNickname, &existingEmail)

	if err == nil {
		var conflictField, conflictMessage string
		if existingNickname == nickname {
			conflictField = "nickname"
			conflictMessage = "Nickname already exists"
		} else if existingEmail == email {
			conflictField = "email"
			conflictMessage = "Email already exists"
		}

		response = map[string]string{
			"error": conflictMessage,
			"field": conflictField,
		}
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(response)
		return
	} else if err != sql.ErrNoRows {
		log.Printf("Database error: %v", err)
		response = map[string]string{"error": "Database error"}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		response = map[string]string{"error": "Error hashing password"}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	sessionToken, _ := uuid.NewV4()

	result, err := database.DB.Exec(
		"INSERT INTO users (nickname, email, password, first_name, last_name, age, gender, session_token) VALUES (?, ?, ?, ?, ?, ?, ?, ?)",
		nickname,
		email,
		hashedPassword,
		firstName,
		lastName,
		age,
		gender,
		sessionToken,
	)
	if err != nil {
		log.Printf("Error inserting user: %v", err)
		response = map[string]string{"error": "Registration failed"}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	userID, err := result.LastInsertId()
	if err != nil {
		log.Printf("Error retrieving last insert ID: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Add the user to the chat with a welcome message to chats
	_, err = database.DB.Exec(`
        INSERT INTO chats (sender_id, receiver_id, message, sent_at, meta_data)
        VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)`,
		userID, 0, "Welcome to the chat!", nickname)
	if err != nil {
		log.Printf("Error inserting user into chat table: %v", err)
		response = map[string]string{"error": "Chat registration failed"}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	chat.BroadcastNewUser(nickname, firstName, lastName)

	response = map[string]string{"message": "Registration successful! Please log in."}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}
