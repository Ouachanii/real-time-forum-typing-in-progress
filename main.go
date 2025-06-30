package main

import (
	"log"
	"net/http"

	"forum/database"

	"forum/backend/auth"
	"forum/backend/categories"
	"forum/backend/chat"
	"forum/backend/comments"
	"forum/backend/interactions"
	"forum/backend/middleware"
	"forum/backend/posts"
)

func main() {

		if err := database.InitDB(); err != nil {
			log.Fatalf("Database initialization failed: %v", err)
		}
		defer database.DB.Close()
	
		http.Handle("/frontend/", http.StripPrefix("/frontend/", http.FileServer(http.Dir("./frontend"))))
	
	
		http.HandleFunc("/", posts.HomePage)
	
		// Post related routes
		http.HandleFunc("/show_posts", posts.ShowPosts)  
		http.HandleFunc("/post_submit", posts.PostSubmit)
	
		// Comment related routes
		http.HandleFunc("/comment_submit", comments.CommentSubmit)
	
		// Interaction (Likes/Dislikes) route
		http.HandleFunc("/interact", interactions.Interact)
	
		// Category route
		http.HandleFunc("/get_categories", categories.GetCategories) // Handler from 'categories' package
	
		// Auth routes
		http.HandleFunc("/login", auth.LoginHandler)      
		http.HandleFunc("/register", auth.RegisterHandler)
		http.HandleFunc("/logout", middleware.LogoutHandler) 
	
		// Session check route
		http.HandleFunc("/check-session", middleware.CheckSessionHandler)
	
		http.HandleFunc("/get_all_users", chat.GetAllUsersHandler)    
		http.HandleFunc("/fetch_messages", chat.FetchMessagesHandler) 
		http.HandleFunc("/ws", chat.HandleConnections)                 
		http.HandleFunc("/mark-read", chat.MarkNotificationsRead)     
		http.HandleFunc("/get-notifications", chat.GetNotifications)


	log.Println("http://localhost:3344/")
	log.Fatal(http.ListenAndServe(":3344", nil))
}
