package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"

	"github.com/apex/log"
	jsonhandler "github.com/apex/log/handlers/json"
	"github.com/apex/log/handlers/text"
	"github.com/aws/aws-sdk-go-v2/aws/endpoints"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/rds/rdsutils"
	"github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/schema"
	"github.com/jmoiron/sqlx"
)

type unit struct {
	ID          int
	Name        string
	Description template.HTML
}

type uQuery struct {
	IDs    []int  `schema:"ids"`
	Limit  int    `schema:"limit"`
	Cursor int    `schema:"cursor"` // used in deletions
	Query  string `schema:"query"`
}

type handler struct {
	db *sqlx.DB
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
	app.HandleFunc("/tables", h.showtables).Methods("GET")
	app.HandleFunc("/delete", h.deleteUnit).Methods("POST")
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
		args.Cursor, "%"+args.Query+"%", args.Limit)
	return
}
func (h handler) showtables(w http.ResponseWriter, r *http.Request) {
	tables := []string{}
	h.db.Select(&tables, "SHOW TABLES")
	fmt.Fprintf(w, "%#v", tables)
	for _, t := range tables {

		var id int
		ctx := log.WithFields(log.Fields{
			"table": t,
		})
		err := h.db.QueryRow(fmt.Sprintf(`SELECT id
	FROM %s
	WHERE id = 32767`, t)).Scan(&id)
		if err != nil {
			ctx.WithError(err).Debug("no result")
		}
		if id > 0 {
			ctx.WithField("id", id).Info("Result")
		}
	}
}

func (h handler) listhtml(w http.ResponseWriter, r *http.Request) {
	var decoder = schema.NewDecoder()
	var query uQuery
	err := decoder.Decode(&query, r.URL.Query())

	if query.Limit == 0 {
		query.Limit = 100
	}

	units, err := h.getUnits(query)
	log.Debugf("units: %#v", units)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var fns = template.FuncMap{
		"cursor": func(units []unit) int {
			if len(units) >= 1 {
				return units[len(units)-1].ID
			}
			return 0
		},
	}

	t := template.Must(template.New("").Funcs(fns).ParseGlob("templates/*.html"))
	err = t.ExecuteTemplate(w, "index.html", struct {
		Units []unit
		Query uQuery
	}{
		units,
		query,
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

	cfg, err := external.LoadDefaultAWSConfig(external.WithSharedConfigProfile("uneet-dev"))
	if err != nil {
		log.WithError(err).Fatal("setting up credentials")
		return
	}
	cfg.Region = endpoints.ApSoutheast1RegionID

	err = RegisterRDSMysqlCerts(http.DefaultClient)
	if err != nil {
		log.WithError(err).Fatal("failed to register certificates")
	}

	provider := cfg.Credentials
	endpoint := "twoam2-cluster.cluster-c5eg6u2xj9yy.ap-southeast-1.rds.amazonaws.com:3306"
	user := "mydbuser"

	log.Info(endpoint)
	authToken, err := rdsutils.BuildAuthToken(endpoint, "ap-southeast-1", user, provider)

	DSN := fmt.Sprintf("%s:%s@tcp(%s)/%s?allowCleartextPasswords=true&tls=rds&multiStatements=true",
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
func (h handler) deleteUnit(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.WithError(err).Error("unable to parse form")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var decoder = schema.NewDecoder()
	var query uQuery
	err = decoder.Decode(&query, r.PostForm)

	if err != nil {
		log.WithError(err).Error("trouble getting args")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.WithField("number to delete", len(query.IDs)).Info("delete")
	for _, id := range query.IDs {
		_, err = h.runsql("cleanup_remove_a_unit_bzfe.sql", id)

		if err != nil {
			log.WithError(err).Error("sql failed")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	var encoder = schema.NewEncoder()
	v := url.Values{}
	err = encoder.Encode(query, v)
	v.Del("ids")
	if err != nil {
		log.WithError(err).Error("url failed to encode")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/?"+v.Encode(), http.StatusFound)
}

func (h handler) runsql(sqlfile string, unitID int) (rowsAffected int64, err error) {

	if unitID == 0 {
		return rowsAffected, fmt.Errorf("id is unset")
	}

	sqlscript, err := ioutil.ReadFile(fmt.Sprintf("sql/%s", sqlfile))
	if err != nil {
		return
	}

	log.Infof("Running %s with unit id %d", sqlfile, unitID)
	fillInArg := fmt.Sprintf(string(sqlscript), unitID)

	res, err := h.db.Exec(fillInArg)
	if err != nil {
		log.WithError(err).Error("error running SQL")
	}
	ioutil.WriteFile("/tmp/debug.sql", []byte(fillInArg), 0644)
	return res.RowsAffected()
}
