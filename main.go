package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	// Statički link ka bazi (Docker DNS koristi "db")
	connStr := "postgres://nemanja:mojasifra@db:5432/iot_db?sslmode=disable"

	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	// Čekanje da se baza podigne
	for i := 0; i < 5; i++ {
		if err = db.Ping(); err == nil {
			break
		}
		fmt.Println("Baza nije spremna, čekam...")
		time.Sleep(2 * time.Second)
	}

	// Kreiranje tabele za merenja
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS merenja (
		id SERIAL PRIMARY KEY, 
		vrednost TEXT, 
		vreme TIMESTAMP DEFAULT NOW()
	)`)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("USPEH: Tabela je spremna!")

	// REST API RUTE
	http.HandleFunc("/podaci", primiPodatke) // ESP32 šalje ovde: POST /podaci?temp=25
	http.HandleFunc("/led", vratiBoju)       // ESP32 čita odavde: GET /led

	fmt.Println("Server pokrenut na portu 8080...")

	// ListenAndServe drži program budnim zauvek (nema više deadlock-a)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func primiPodatke(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		temp := r.URL.Query().Get("temp")
		_, err := db.Exec("INSERT INTO merenja (vrednost) VALUES ($1)", temp)
		if err != nil {
			http.Error(w, "Greška u bazi", 500)
			return
		}
		fmt.Fprintf(w, "Uspešno upisano: %s", temp)
	}
}

func vratiBoju(w http.ResponseWriter, r *http.Request) {
	// Ovde kasnije možeš čitati iz baze, sada šaljemo statički
	fmt.Fprintf(w, "CRVENA")
}
