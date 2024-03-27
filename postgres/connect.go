package postgres

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectDB(t string, u string) *sqlx.DB {
	db, err := sqlx.Connect(t, u)
	if err != nil {
		log.Fatalln(err)
	}
	return db
}
