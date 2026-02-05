package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var db *sql.DB
var trenutnaBoja = "black"

type LogEntry struct {
	ID       int       `json:"id"`
	Tip      string    `json:"tip"`
	Vrednost string    `json:"vrednost"`
	Vreme    time.Time `json:"vreme"`
}

func main() {
	connStr := "postgres://nemanja:mojasifra@db:5432/iot_db?sslmode=disable"
	var err error
	db, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatal(err)
	}

	db.Exec(`CREATE TABLE IF NOT EXISTS iot_logs (
		id SERIAL PRIMARY KEY, 
		tip TEXT, 
		vrednost TEXT, 
		vreme TIMESTAMP DEFAULT NOW()
	)`)

	http.HandleFunc("/", homePage)
	http.HandleFunc("/podaci", primiPodatke)
	http.HandleFunc("/set-boja", setBoja)
	http.HandleFunc("/get-stanje", getStanje)

	fmt.Println("Server pokrenut na portu 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, `
	<!DOCTYPE html>
	<html>
	<head>
		<title>ESP32 Kontrola</title>
		<style>
			body { font-family: sans-serif; text-align: center; background: #f4f4f4; }
			.temp-box { font-size: 48px; font-weight: bold; color: #2c3e50; margin: 20px; }
			button { padding: 15px 25px; margin: 5px; font-size: 16px; cursor: pointer; border: none; color: white; border-radius: 5px; }
			.black { background: black; } .green { background: green; } .red { background: red; } .off { background: gray; }
			table { margin: 20px auto; border-collapse: collapse; width: 80%%; background: white; }
			th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
		</style>
	</head>
	<body>
		<h1>ESP32 Dashboard</h1>
		<div class="temp-box">Temperatura: <span id="temp">--</span>°C</div>
		
		<div>
			<button class="black" onclick="promeniBoju('black')">CRNO</button>
			<button class="green" onclick="promeniBoju('green')">ZELENO</button>
			<button class="red" onclick="promeniBoju('red')">CRVENO</button>
			<button class="off" onclick="promeniBoju('off')">ISKLJUČI</button>
		</div>

		<h3>Logovi (Posljednjih 10)</h3>
		<table id="logTable">
			<tr><th>Tip</th><th>Vrijednost</th><th>Vrijeme</th></tr>
		</table>

		<script>
			function promeniBoju(nova) {
				fetch('/set-boja?boja=' + nova);
			}

			function osveziPodatke() {
				fetch('/get-stanje').then(r => r.json()).then(data => {
					document.getElementById('temp').innerText = data.zadnja_temp;
					let table = document.getElementById('logTable');
					table.innerHTML = '<tr><th>Tip</th><th>Vrijednost</th><th>Vrijeme</th></tr>';
					data.logovi.forEach(l => {
						let row = table.insertRow();
						row.innerHTML = "<td>"+l.tip+"</td><td>"+l.vrednost+"</td><td>"+new Date(l.vreme).toLocaleString()+"</td>";
					});
				});
			}
			setInterval(osveziPodatke, 2000); // Osvežava svake 2 sekunde
		</script>
	</body>
	</html>`)
}

func primiPodatke(w http.ResponseWriter, r *http.Request) {
	val := r.URL.Query().Get("temp")
	db.Exec("INSERT INTO iot_logs (tip, vrednost) VALUES ('TEMPERATURA', $1)", val)
	fmt.Fprint(w, "OK")
}

func setBoja(w http.ResponseWriter, r *http.Request) {
	boja := r.URL.Query().Get("boja")
	trenutnaBoja = boja
	db.Exec("INSERT INTO iot_logs (tip, vrednost) VALUES ('BOJA', $1)", boja)
	fmt.Fprint(w, "OK")
}

func getStanje(w http.ResponseWriter, r *http.Request) {
	var zadnjaTemp string
	db.QueryRow("SELECT vrednost FROM iot_logs WHERE tip='TEMPERATURA' ORDER BY id DESC LIMIT 1").Scan(&zadnjaTemp)

	rows, _ := db.Query("SELECT tip, vrednost, vreme FROM iot_logs ORDER BY id DESC LIMIT 10")
	var logs []LogEntry
	for rows.Next() {
		var l LogEntry
		rows.Scan(&l.Tip, &l.Vrednost, &l.Vreme)
		logs = append(logs, l)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"zadnja_temp": zadnjaTemp,
		"boja":        trenutnaBoja,
		"logovi":      logs,
	})
}
