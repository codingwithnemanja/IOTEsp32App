package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

func main() {
	connStr := "postgres://nemanja:mojasifra@db:5432/iot_db?sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	for i := 0; i < 5; i++ {
		err = db.Ping()
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("USPEH: Povezan na PostgreSQL bazu!")

	select {}
}
