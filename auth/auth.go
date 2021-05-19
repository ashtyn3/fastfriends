package auth

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type DBCreds struct {
	Host     string
	Password string
	Port     string
	Database string
	User     string
}

type Network string

type Creds struct {
	DB         DBCreds
	NetworkUrl Network
}

func (db DBCreds) Init() (*sql.DB, string) {
	psqlconn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", db.Host, db.Port, db.User, db.Password, db.Database)
	sq, dbErr := sql.Open("postgres", psqlconn)
	if dbErr != nil {
		log.Fatalln(dbErr)
	}
	return sq, psqlconn
}

var path string

func Auth() Creds {
	if os.Getenv("envPath") != "" {
		path = os.Getenv("envPath")
	} else {
		path = "../.env"
	}
	err := godotenv.Load(path)

	if err != nil {
		log.Fatalln("Error loading .env file")
	}

	if os.Getenv("db_host") == "" || os.Getenv("db_port") == "" || os.Getenv("db_user") == "" || os.Getenv("db_password") == "" || os.Getenv("db_name") == "" {
		log.Fatalln("missing crucial database information to authenticate.")
	}
	if err != nil {
		log.Fatalln(err)
	}
	c := Creds{
		DB: DBCreds{
			Host:     os.Getenv("db_host"),
			Port:     os.Getenv("db_port"),
			User:     os.Getenv("db_user"),
			Password: os.Getenv("db_password"),
			Database: os.Getenv("db_name"),
		},
	}
	return c
}
