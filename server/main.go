package main

import (
	"database/sql"
	"log"
	"net/http"

	_ "github.com/go-sql-driver/mysql"

	"encoding/json"

	"Moonbase/src/models"
)

var db *sql.DB

type user struct {
	Name string `json:"name"`
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		getUser(w, r)
	case http.MethodPost:
		createUser(w, r)
	}
}

func getUser(w http.ResponseWriter, r *http.Request) {

}

func createUser(w http.ResponseWriter, r *http.Request) {

	var u user

	err := json.NewDecoder(r.body).Decode(&u)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = models.CreateUser(u.Name)

}

func main() {
	var err error

	dsn := "root:12345678@tcp(localhost:3303)/messanging_app"

	db, err = sql.Open("mysql", dsn)

	if err != nil {
		log.Fatalf("Failed to connect to DB", err)
		return
	}

	defer db.Close()

	http.HandleFunc("/users", handleRequest)

	log.Fatal(http.ListenAndServe(":8080", nil))

}
