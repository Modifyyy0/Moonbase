package main

import (
	"Moonbase/src/db"
	"Moonbase/src/models"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	if err := db.Connect(); err != nil {
		log.Fatal(err)
	}
	defer db.Close()
}
