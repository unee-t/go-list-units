package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"

	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/unee-t/env"
)

type unit struct {
	Id          int
	Name        string
	Description template.HTML
}

type handler struct {
	DSN            string // e.g. "bugzilla:secret@tcp(auroradb.dev.unee-t.com:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL"
	APIAccessToken string // e.g. O8I9svDTizOfLfdVA5ri
	db             *sql.DB
	Code           env.EnvCode
}

func init() {
	if os.Getenv("UP_STAGE") == "" {
		log.SetHandler(text.Default)
	} else {
		log.SetHandler(jsonhandler.Default)
	}
}

func main() {

	h, err := New()
	if err != nil {
		log.WithError(err).Fatal("error setting configuration")
		return
	}

	defer h.db.Close()

	addr := ":" + os.Getenv("PORT")
	app := mux.NewRouter()
	app.HandleFunc("/", h.listjson).Methods("GET").Headers("Accept", "application/json")
	app.HandleFunc("/", h.listhtml).Methods("GET")
	app.HandleFunc("/metrics", h.prometheus)
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func (h handler) getUnits() (units []unit, err error) {

	rows, err := h.db.Query("select id, name, description from products")
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

func (h handler) listhtml(w http.ResponseWriter, r *http.Request) {

	if os.Getenv("UP_STAGE") != "production" {
		w.Header().Set("X-Robots-Tag", "none")
	}

	units, err := h.getUnits()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := template.Must(template.New("").ParseGlob("templates/*.html"))
	t.ExecuteTemplate(w, "index.html", units)

}

func (h handler) listjson(w http.ResponseWriter, r *http.Request) {

	if os.Getenv("UP_STAGE") != "production" {
		w.Header().Set("X-Robots-Tag", "none")
	}

	units, err := h.getUnits()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(units)

}

func (h handler) prometheus(w http.ResponseWriter, r *http.Request) {

	rows, err := h.db.Query("select COUNT(*) from user_group_map")
	if err != nil {
		log.WithError(err).Error("failed to open database")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var count int

	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			log.WithError(err).Error("failed to scan count")
		}
	}

	log.Infof("Count: %d", count)

	fmt.Fprintf(w, "# HELP user_group_map_total shows the number of rows in the user_group_map table.\n# TYPE user_group_map_total counter\nuser_group_map_total %d", count)
}

// New setups the configuration assuming various parameters have been setup in the AWS account
func New() (h handler, err error) {

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("uneet-prod"))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID
	e, err := env.New(cfg)
	if err != nil {
		log.WithError(err).Warn("error getting AWS unee-t env")
	}

	var mysqlhost string
	val, ok := os.LookupEnv("MYSQL_HOST")
	if ok {
		log.Infof("MYSQL_HOST overridden by local env: %s", val)
		mysqlhost = val
	} else {
		mysqlhost = e.Udomain("auroradb")
	}

	h = handler{
		DSN: fmt.Sprintf("%s:%s@tcp(%s:3306)/bugzilla?multiStatements=true&sql_mode=TRADITIONAL",
			e.GetSecret("MYSQL_USER"),
			e.GetSecret("MYSQL_PASSWORD"),
			mysqlhost),
		APIAccessToken: e.GetSecret("API_ACCESS_TOKEN"),
		Code:           e.Code,
	}

	h.db, err = sql.Open("mysql", h.DSN)
	if err != nil {
		log.WithError(err).Fatal("error opening database")
		return
	}

	return

}
