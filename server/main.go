package main

import (
	"Moonbase/src/websocket"
	"Moonbase/src/db"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"Moonbase/handlers"
	"Moonbase/src/db"
)

var allowedOrigins = map[string]bool{
	"http://localhost:3000": true,
	"http://localhost:5173": true,
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}

		if r.Method == http.MethodOptions {
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

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
	log.Fatal(http.ListenAndServe(":8080", corsMiddleware(http.DefaultServeMux)))
}
