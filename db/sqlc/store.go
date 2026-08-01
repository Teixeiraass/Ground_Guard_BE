package db

import "database/sql"

// Store provides all functions to execute db queries
type Store interface {
	Querier
}

// SqlStore provides all functions to execute SQL queries and transaction
type SqlStore struct {
	*Queries
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return &SqlStore{
		db:      db,
		Queries: New(db),
	}
}
