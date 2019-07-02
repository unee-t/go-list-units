package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/rds/rdsutils"
	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/jmoiron/sqlx"
	"github.com/unee-t/env"
)

type unit struct {
	ID          int
	Name        string
	Description template.HTML
}

type uQuery struct {
	ID    int    `schema:"id"`
	Limit int    `schema:"limit"`
	Query string `schema:"query"`
}

type handler struct {
	db  *sqlx.DB
	Env env.Env
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
	app.HandleFunc("/ping", h.ping).Methods("GET")
	if err := http.ListenAndServe(addr, app); err != nil {
		log.WithError(err).Fatal("error listening")
	}

}

func (h handler) getUnits(args uQuery) (units []unit, err error) {
	log.Infof("args: %#v", args)

	// https://stackoverflow.com/a/3799293/4534
	err = h.db.Select(&units, `SELECT id, name, description
	FROM products
	WHERE id > ?
	AND name LIKE ?
	ORDER BY id
	LIMIT ?`,
		args.ID, "%"+args.Query+"%", args.Limit)
	return
}

func (h handler) listhtml(w http.ResponseWriter, r *http.Request) {
	var decoder = schema.NewDecoder()
	var query uQuery
	err := decoder.Decode(&query, r.URL.Query())

	if query.Limit == 0 {
		query.Limit = 100
	}

	units, err := h.getUnits(query)
	// log.Infof("units: %#v", units)
	if len(units) >= 1 {
		query.ID = units[len(units)-1].ID
	}
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var count int
	err = h.db.Get(&count, "select COUNT(*) from products")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	t := template.Must(template.New("").ParseGlob("templates/*.html"))
	err = t.ExecuteTemplate(w, "index.html", struct {
		Units   []unit
		Query   uQuery
		Count   int
		Account string
	}{
		units,
		query,
		count,
		h.Env.AccountID,
	})

	if err != nil {
		log.WithError(err).Error("template")
	}

}

func (h handler) listjson(w http.ResponseWriter, r *http.Request) {

	if os.Getenv("UP_STAGE") != "production" {
		w.Header().Set("X-Robots-Tag", "none")
	}

	units, err := h.getUnits(uQuery{})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(units)

}

// New setups the configuration assuming various parameters have been setup in the AWS account
func New() (h handler, err error) {

	profile := "uneet-dev"
	if os.Getenv("AWS_PROFILE") != "" {
		profile = os.Getenv("AWS_PROFILE")
	}

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile(profile))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID

	h.Env, err = env.New(cfg)
	if err != nil {
		log.WithError(err).Warn("error getting unee-t env")
	}

	err = RegisterRDSMysqlCerts(http.DefaultClient)
	if err != nil {
		log.WithError(err).Fatal("failed to register certificates")
	}

	provider := cfg.Credentials
	endpoint := h.Env.GetSecret("MYSQL_HOST") + ":3306"
	user := "mydbuser"

	log.Infof("Profile: %s Endpoint: %s", profile, endpoint)
	authToken, err := rdsutils.BuildAuthToken(endpoint, "ap-southeast-1", user, provider)

	DSN := fmt.Sprintf("%s:%s@tcp(%s)/%s?allowCleartextPasswords=true&tls=rds",
		user, authToken, endpoint, "bugzilla",
	)

	h.db, err = sqlx.Connect("mysql", DSN)
	if err != nil {
		log.WithError(err).Fatal("error opening database")
		return
	}

	return

}

func (h handler) ping(w http.ResponseWriter, r *http.Request) {
	err := h.db.Ping()
	if err != nil {
		log.WithError(err).Error("failed to ping database")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	fmt.Fprintf(w, "OK")
}

func RegisterRDSMysqlCerts(c *http.Client) error {
	// resp, err := c.Get("https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem")
	// if err != nil {
	// 	panic(err)
	// }

	pem, err := ioutil.ReadFile("./iam/rds-combined-ca-bundle.pem")
	if err != nil {
		panic(err)
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		panic(err)
	}

	err = mysql.RegisterTLSConfig("rds", &tls.Config{RootCAs: rootCertPool, InsecureSkipVerify: true})
	if err != nil {
		panic(err)
	}
	return nil
}
