package main

import (
	"fmt"

	"github.com/belldb/internal/db"
)

func main() {
	db := db.NewDB()

	if err := db.Open(); err != nil {
		panic(err)
	}
	defer db.Close()

	// for i := 1; i <= 1765; i++ {
	// 	err := db.Put("s", int64(i), float64(i))
	// 	if err != nil {
	// 		panic(err)
	// 	}
	// }

	points := db.Range("s", int64(1), int64(1765))

	fmt.Println(points)
}
