package main

import (
	"github.com/belldb/internal/db"
)

func main() {
	database := db.NewDB()

	if err := database.Open(); err != nil {
		panic(err)
	}
	defer database.Close()

}
