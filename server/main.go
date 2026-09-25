package main

import (
	"Moonbase/src/db"
	"fmt"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"Moonbase/handlers"
)

var allowedOrigins = map[string]bool{
	"http://localhost:3000": true,
	"http://localhost:5173": true,
	"http://127.0.0.1:5500": true,
	"http://localhost:5500": true,
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowedOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)

		}
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		w.Header().Set("Vary", "Origin")

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
	http.HandleFunc("GET /api/users", handlers.GetUsers)

	http.HandleFunc("/api/conversations", handlers.HandleConversations)
	http.HandleFunc("GET /api/conversations/{convoID}", handlers.ConvoInfo)
	http.HandleFunc("DELETE /api/conversations/{convoID}/members/me", handlers.LeaveConvo)
	http.HandleFunc("GET /api/conversations/{convoID}/messages", handlers.GetMessages)
	http.HandleFunc("POST /api/join/{convoID}", handlers.JoinConvo)

	http.HandleFunc("DELETE /api/deleteUser", handlers.DeleteUser)

	http.HandleFunc("/ws", handlers.HandleWebsocket)

	clientFiles := http.FileServer(http.Dir("../client"))
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.Redirect(w, r, "/login.html", http.StatusFound)
			return
		}
		clientFiles.ServeHTTP(w, r)
	})

	fmt.Println("Server running at http://localhost:6767")
	log.Fatal(http.ListenAndServe(":6767", corsMiddleware(http.DefaultServeMux)))
}
