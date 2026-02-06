package main

import (
	"context"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Log struct {
	Temp     string
	DeviceID string
	Vreme    string
}

type Stats struct {
	Avg string
	Min string
	Max string
}

var dbPool *pgxpool.Pool

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://nemanja:mojasifra@db:5432/iot_db?sslmode=disable"
	}

	var err error
	for i := 0; i < 10; i++ {
		dbPool, err = pgxpool.New(context.Background(), dbURL)
		if err == nil {
			err = dbPool.Ping(context.Background())
			if err == nil {
				break
			}
		}
		time.Sleep(2 * time.Second)
	}

	if err != nil {
		os.Exit(1)
	}
	defer dbPool.Close()

	dbPool.Exec(context.Background(), `CREATE TABLE IF NOT EXISTS logovi (
       id SERIAL PRIMARY KEY, 
       temperatura TEXT, 
       device_id TEXT, 
       vreme TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`)

	http.HandleFunc("/esp", func(w http.ResponseWriter, r *http.Request) {
		temp := r.URL.Query().Get("temp")
		mac := r.URL.Query().Get("mac")
		if temp != "" {
			dbPool.Exec(context.Background(), "INSERT INTO logovi (temperatura, device_id) VALUES ($1, $2)", temp+"C", mac)
		}
		var zadnja string
		dbPool.QueryRow(context.Background(), "SELECT device_id FROM logovi WHERE temperatura = 'Komanda' ORDER BY id DESC LIMIT 1").Scan(&zadnja)
		fmt.Fprint(w, zadnja)
	})

	http.HandleFunc("/control", func(w http.ResponseWriter, r *http.Request) {
		boja := r.URL.Query().Get("color")
		if boja != "" {
			dbPool.Exec(context.Background(), "INSERT INTO logovi (temperatura, device_id) VALUES ($1, $2)", "Komanda", boja)
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		rows, err := dbPool.Query(context.Background(), "SELECT temperatura, device_id, TO_CHAR(vreme, 'HH:MI') FROM logovi ORDER BY id DESC LIMIT 10")
		if err == nil {
			defer rows.Close()
		}

		var logs []Log
		zadnjaTemp := "--"
		for rows != nil && rows.Next() {
			var l Log
			rows.Scan(&l.Temp, &l.DeviceID, &l.Vreme)
			logs = append(logs, l)
		}
		if len(logs) > 0 {
			for _, l := range logs {
				if l.Temp != "Komanda" {
					zadnjaTemp = l.Temp
					break
				}
			}
		}

		var st Stats
		dbPool.QueryRow(context.Background(), `
          SELECT 
             COALESCE(ROUND(AVG(NULLIF(regexp_replace(temperatura, '[^0-9.]', '', 'g'), '')::numeric), 1)::text, '--'),
             COALESCE(MIN(temperatura), '--'),
             COALESCE(MAX(temperatura), '--')
          FROM logovi WHERE temperatura != 'Komanda'`).Scan(&st.Avg, &st.Min, &st.Max)

		tmpl := `
       <!DOCTYPE html>
       <html lang="sr">
       <head>
          <meta charset="UTF-8">
          <title>IoT Panel</title>
          <meta name="viewport" content="width=device-width, initial-scale=1">
          <link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/font-awesome/6.0.0/css/all.min.css">
          <style>
             :root { --primary: #6366f1; --bg: #0f172a; --card: #1e293b; --text: #f8fafc; }
             body { font-family: 'Segoe UI', sans-serif; background-color: var(--bg); color: var(--text); margin: 0; padding: 20px; display: flex; flex-direction: column; align-items: center; }
             .dashboard { max-width: 800px; width: 100%; }
             .header { text-align: left; margin-bottom: 30px; border-left: 4px solid var(--primary); padding-left: 15px; }
             
             .main-card { background: var(--card); padding: 30px; border-radius: 20px; text-align: center; box-shadow: 0 10px 25px rgba(0,0,0,0.3); margin-bottom: 20px; position: relative; overflow: hidden; }
             .main-temp { font-size: 80px; font-weight: 800; background: linear-gradient(to right, #818cf8, #c084fc); -webkit-background-clip: text; -webkit-text-fill-color: transparent; margin: 10px 0; }
             
             .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: 15px; margin-bottom: 20px; }
             .stat-card { background: var(--card); padding: 20px; border-radius: 15px; text-align: center; }
             .stat-label { font-size: 12px; text-transform: uppercase; color: #94a3b8; letter-spacing: 1px; }
             .stat-val { font-size: 20px; font-weight: bold; margin-top: 5px; display: block; }

             .controls { display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; margin-bottom: 25px; }
             .btn { padding: 15px; border-radius: 12px; border: none; font-weight: bold; cursor: pointer; text-decoration: none; color: white; display: flex; align-items: center; justify-content: center; gap: 8px; transition: 0.3s; font-size: 14px; }
             .btn-bela { background: #64748b; } .btn-zelena { background: #22c55e; }
             .btn-crvena { background: #ef4444; } .btn-off { background: #334155; }
             .btn:hover { opacity: 0.8; transform: translateY(-2px); }

             .history { background: var(--card); border-radius: 20px; padding: 20px; overflow-x: auto; }
             table { width: 100%; border-collapse: collapse; min-width: 400px; }
             th { text-align: left; color: #94a3b8; font-size: 12px; padding: 10px; border-bottom: 1px solid #334155; }
             td { padding: 12px 10px; border-bottom: 1px solid #334155; font-size: 14px; }
             
             @media (max-width: 600px) {
                body { padding: 10px; }
                .main-temp { font-size: 60px; }
                .controls { grid-template-columns: repeat(2, 1fr); }
             }
          </style>
          <script>
             function update() {
                fetch("/").then(r => r.text()).then(html => {
                   let doc = new DOMParser().parseFromString(html, 'text/html');
                   document.querySelector('.main-card').innerHTML = doc.querySelector('.main-card').innerHTML;
                   document.querySelector('.grid').innerHTML = doc.querySelector('.grid').innerHTML;
                   document.querySelector('.history').innerHTML = doc.querySelector('.history').innerHTML;
                });
             }
             setInterval(update, 3000);
          </script>
       </head>
       <body>
          <div class="dashboard">
             <div class="header">
                <h1 style="margin:0; font-size: 24px;">Smart Home Hub</h1>
                <span style="color: #94a3b8; font-size: 14px;">Live monitoring sistema</span>
             </div>

             <div class="main-card">
                <div class="stat-label">Trenutna Temperatura</div>
                <div class="main-temp">{{.Zadnja}}</div>
                <div style="color: #22c55e;"><i class="fas fa-circle" style="font-size: 8px;"></i> Aktivan uređaj</div>
             </div>

             <div class="controls">
                <a href="/control?color=Bela" class="btn btn-bela"><i class="fas fa-lightbulb"></i> Bela</a>
                <a href="/control?color=Zelena" class="btn btn-zelena"><i class="fas fa-leaf"></i> Zelena</a>
                <a href="/control?color=Crvena" class="btn btn-crvena"><i class="fas fa-fire"></i> Crvena</a>
                <a href="/control?color=Off" class="btn btn-off"><i class="fas fa-power-off"></i> Isključi</a>
             </div>

             <div class="grid">
                <div class="stat-card">
                   <div class="stat-label">Prosek (Dan)</div>
                   <span class="stat-val" style="color: #818cf8;">{{.St.Avg}}°C</span>
                </div>
                <div class="stat-card">
                   <div class="stat-label">Najniža</div>
                   <span class="stat-val" style="color: #38bdf8;">{{.St.Min}}</span>
                </div>
                <div class="stat-card">
                   <div class="stat-label">Najviša</div>
                   <span class="stat-val" style="color: #f87171;">{{.St.Max}}</span>
                </div>
             </div>

             <div class="history">
                <h3 style="margin-top:0; font-size: 16px;"><i class="fas fa-history"></i> Poslednja očitanja</h3>
                <table>
                   <thead>
                      <tr><th>Uređaj</th><th>Vrednost</th><th>Vreme</th></tr>
                   </thead>
                   <tbody>
                      {{range .Logs}}
                      <tr>
                         <td><code style="background:#334155; padding:3px 6px; border-radius:4px;">{{.DeviceID}}</code></td>
                         <td style="font-weight:bold;">{{.Temp}}</td>
                         <td style="color:#94a3b8;">{{.Vreme}}</td>
                      </tr>
                      {{end}}
                   </tbody>
                </table>
             </div>
          </div>
       </body>
       </html>`
		t := template.Must(template.New("w").Parse(tmpl))
		t.Execute(w, struct {
			Logs   []Log
			Zadnja string
			St     Stats
		}{logs, zadnjaTemp, st})
	})

	http.ListenAndServe(":8080", nil)
}
