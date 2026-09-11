package main

import (
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"Moonbase/handlers"
	"Moonbase/src/db"
)

func main() {

	err := db.Connect()

	if err != nil {
		log.Printf("Could not connect to the database: %v", err)
		return
	}
	defer db.Close()

	http.HandleFunc("POST /api/login", handlers.LoginHandler)
	http.HandleFunc("POST /api/logout", handlers.LogoutHandler)
	http.HandleFunc("GET /api/me", handlers.HandleMe)

	http.HandleFunc("/api/conversations", handlers.HandleConversations)
	http.HandleFunc("GET /api/conversations/{convoID}", handlers.ConvoInfo)
	http.HandleFunc("DELETE /api/conversations/{convoID}/members/me", handlers.LeaveConvo)
	http.HandleFunc("GET /api/conversations/{convoID}/messages", handlers.GetMessages)
	http.HandleFunc("POST /api/join/{convoID}", handlers.JoinConvo)

	http.HandleFunc("POST /api/newUser", handlers.NewUser)
	http.HandleFunc("DELETE /api/deleteUser", handlers.DeleteUser)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
