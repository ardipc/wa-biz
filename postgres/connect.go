package postgres

import (
	"log"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

func ConnectDB() *sqlx.DB {
	db, err := sqlx.Connect("postgres", "user=postgres.cymagsnihvppzuqevvge password=MEr43y5F78QfURwg host=aws-0-ap-southeast-1.pooler.supabase.com port=5432 dbname=postgres")
	if err != nil {
		log.Fatalln(err)
	}
	return db
}
