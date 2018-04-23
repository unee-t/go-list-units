package main

import (
	"net/http"
	"os"

	jsonhandler "github.com/apex/log/handlers/json"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/gorilla/pat"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var (
	id   int
	name string
)

func init() {
	if os.Getenv("UP_STAGE") == "" {
		log.SetHandler(text.Default)
	} else {
		log.SetHandler(jsonhandler.Default)
	}
}

func main() {
	addr := ":" + os.Getenv("PORT")
	app := pat.New()
	app.Get("/list", list)
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func list(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("mysql", "root:uniti@/bugzilla")
	if err != nil {
		log.WithError(err).Error("failed to open database")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		log.WithError(err).Error("failed to open database")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rows, err := db.Query("select id, name from products")
	if err != nil {
		log.WithError(err).Error("failed to open database")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	for rows.Next() {
		err := rows.Scan(&id, &name)

		if err != nil {
			log.WithError(err).Error("failed to scan")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		log.Infof("OK %d %s", id, name)
	}

	err = rows.Err()

	if err != nil {
		log.WithError(err).Error("row iterator issue")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}
