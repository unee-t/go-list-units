package main

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds/rdsutils"
	"github.com/go-sql-driver/mysql"
)

func newAWSSession(profile string, region string) (*session.Session, error) {
	cfg := aws.Config{Region: aws.String(region), CredentialsChainVerboseErrors: aws.Bool(true)}
	sessionOpts := session.Options{Profile: profile, Config: cfg}
	return session.NewSessionWithOptions(sessionOpts)
}

func main() {
	sess, err := newAWSSession("uneet-dev", "ap-southeast-1")
	if err != nil {
		panic(err)
	}

	userPtr := flag.String("user", "mydbuser", "user of the credentials")
	regionPtr := flag.String("region", "ap-southeast-1", "region to be used when grabbing sts creds")
	endpointPtr := flag.String("endpoint", "twoam2-cluster.cluster-c5eg6u2xj9yy.ap-southeast-1.rds.amazonaws.com", "DB endpoint to be connected to")
	portPtr := flag.Int("port", 3306, "DB port to be connected to")
	tablePtr := flag.String("table", "user_group_map", "DB table to query against")
	dbNamePtr := flag.String("dbname", "bugzilla", "DB name to query against")
	flag.Parse()

	// Check required flags. Will exit with status code 1 if
	// required field isn't set.
	if err := requiredFlags(
		userPtr,
		regionPtr,
		endpointPtr,
		portPtr,
		dbNamePtr,
	); err != nil {
		fmt.Printf("Error: %v\n\n", err)
		flag.PrintDefaults()
		os.Exit(1)
	}

	err = registerRDSMysqlCerts(http.DefaultClient)
	if err != nil {
		panic(err)
	}

	creds := sess.Config.Credentials

	v := url.Values{}
	// required fields for DB connection
	v.Add("tls", "rds")
	v.Add("allowCleartextPasswords", "true")
	endpoint := fmt.Sprintf("%s:%d", *endpointPtr, *portPtr)

	b := rdsutils.NewConnectionStringBuilder(endpoint, *regionPtr, *userPtr, *dbNamePtr, creds)
	connectStr, err := b.WithTCPFormat().WithParams(v).Build()

	if err != nil {
		panic(err)
	}

	log.Println(connectStr)

	const dbType = "mysql"
	db, err := sql.Open(dbType, connectStr)
	// if an error is encountered here, then most likely security groups are incorrect
	// in the database.
	if err != nil {
		panic(fmt.Errorf("failed to open connection to the database"))
	}

	rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s  LIMIT 1", *tablePtr))
	if err != nil {
		panic(fmt.Errorf("failed to select from table, %q, with %v", *tablePtr, err))
	}

	for rows.Next() {
		columns, err := rows.Columns()
		if err != nil {
			panic(fmt.Errorf("failed to read columns from row: %v", err))
		}

		fmt.Printf("rows colums:\n%d\n", len(columns))
	}
}

func requiredFlags(flags ...interface{}) error {
	for _, f := range flags {
		switch f.(type) {
		case nil:
			return fmt.Errorf("one or more required flags were not set")
		}
	}
	return nil
}

func registerRDSMysqlCerts(c *http.Client) error {
	resp, err := c.Get("https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem")
	if err != nil {
		return err
	}

	pem, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	rootCertPool := x509.NewCertPool()
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		return fmt.Errorf("failed to append cert to cert pool!")
	}

	return mysql.RegisterTLSConfig("rds", &tls.Config{RootCAs: rootCertPool, InsecureSkipVerify: true})
}
