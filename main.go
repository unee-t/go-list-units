package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/gorilla/mux"
	"github.com/unee-t/env"

	"github.com/apex/log"

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
	app := mux.NewRouter()
	app.HandleFunc("/", listjson).Methods("GET").Headers("Accept", "application/json")
	app.HandleFunc("/", listhtml).Methods("GET")
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func getUnits() (units []unit, err error) {

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("uneet-dev"))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID
	e, err := env.New(cfg)
	if err != nil {
		log.WithError(err).Warn("error getting unee-t env")
	}

	db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL",
		e.GetSecret("MYSQL_USER"),
		e.GetSecret("MYSQL_PASSWORD"),
		e.Udomain("auroradb")))
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
