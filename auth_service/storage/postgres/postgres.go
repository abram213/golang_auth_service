package postgres

import (
	"fmt"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DBPostgres struct {
	*sqlx.DB
}

func New(uri string) (*DBPostgres, error) {
	db, err := sqlx.Connect("postgres", uri)
	if err != nil {
		return nil, fmt.Errorf("conn to postgres err: %v\nuri: %s", err, uri)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(5)
	return &DBPostgres{db}, nil
}

func (db *DBPostgres) Close() error {
	return db.Close()
}
