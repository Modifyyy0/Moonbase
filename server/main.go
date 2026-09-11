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

	http.HandleFunc("POST /login", handlers.LoginHandler)
	http.HandleFunc("POST /logout", handlers.LogoutHandler)
	http.HandleFunc("GET /me", handlers.HandleMe)

	http.HandleFunc("/conversations", handlers.HandleConversations)
	http.HandleFunc("GET /conversations/{convoID}", handlers.ConvoInfo)
	http.HandleFunc("DELETE /conversations/{convoID}/members/me", handlers.LeaveConvo)
	http.HandleFunc("GET /conversations/{convoID}/messages", handlers.GetMessages)
	http.HandleFunc("POST /join/{convoID}", handlers.JoinConvo)

	http.HandleFunc("POST /newUser", handlers.NewUser)
	http.HandleFunc("DELETE /deleteUser", handlers.DeleteUser)

	fmt.Println("Server running at http://localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
