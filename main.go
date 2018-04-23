package main

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"

	jsonhandler "github.com/apex/log/handlers/json"

	"github.com/apex/log"
	"github.com/apex/log/handlers/text"
	"github.com/gorilla/pat"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

type unit struct {
	Id          int
	Name        string
	Description template.HTML
}

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
	app.Get("/list", listhtml)
	app.Get("/json", listjson)
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func getUnits() (units []unit, err error) {

	// db, err := sql.Open("mysql", "root:uniti@/bugzilla")
	db, err := sql.Open("mysql", os.Getenv("DSN"))
	if err != nil {
		return units, err
	}

	defer db.Close()

	// Open doesn't open a connection. Validate DSN data:
	err = db.Ping()
	if err != nil {
		log.WithError(err).Error("failed to open database")
		return units, err
	}

	rows, err := db.Query("select id, name, description from products")
	if err != nil {
		log.WithError(err).Error("failed to open database")
		return units, err
	}

	defer rows.Close()

	for rows.Next() {
		var u unit
		err := rows.Scan(&u.Id, &u.Name, &u.Description)

		if err != nil {
			log.WithError(err).Error("failed to scan")
			return units, err
		}
		log.Infof("%d %s", u.Id, u.Name, u.Description)
		units = append(units, u)
	}

	err = rows.Err()

	if err != nil {
		log.WithError(err).Error("row iterator issue")
		return units, err
	}

	return units, err

}

func listhtml(w http.ResponseWriter, r *http.Request) {

	if os.Getenv("UP_STAGE") != "production" {
		w.Header().Set("X-Robots-Tag", "none")
	}

	units, err := getUnits()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := template.Must(template.New("").ParseGlob("templates/*.html"))
	t.ExecuteTemplate(w, "index.html", units)

}

func listjson(w http.ResponseWriter, r *http.Request) {

	if os.Getenv("UP_STAGE") != "production" {
		w.Header().Set("X-Robots-Tag", "none")
	}

	units, err := getUnits()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(units)

}
