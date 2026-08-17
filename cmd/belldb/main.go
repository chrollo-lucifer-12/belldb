package main

import (
	"fmt"

	"github.com/belldb/internal/db"
)

func main() {
	database := db.NewDB()

	if err := database.Open(); err != nil {
		panic(err)
	}
	defer database.Close()

	for i := 1; i <= 1000; i++ {
		if err := database.Put("cpu_usage", int64(i), float64(i)); err != nil {
			panic(err)
		}
	}

	database.WaitForFlush()

	v, err := database.Get("cpu_usage", 1)
	if err != nil {
		panic(err)
	}

	fmt.Println(v)

}
