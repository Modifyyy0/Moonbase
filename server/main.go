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

	user, _ := models.FindUserByName("Darren")
	err := user.DeleteUser()
	fmt.Printf("%s\n",err)
	user, _ = models.FindUserByName("Darren")
	err = user.DeleteUser()
	fmt.Printf("%s\n",err)
}
