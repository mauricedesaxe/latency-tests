package latency_simulations

import "github.com/jmoiron/sqlx"

var db *sqlx.DB

func init() {
	var err error
	db, err = sqlx.Open("sqlite3", "./db/latency_simulations.sqlite?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=-2000")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA foreign_keys = ON")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA journal_mode = WAL")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA synchronous = NORMAL")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA cache_size = -2000")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec("PRAGMA busy_timeout = 5000")
	if err != nil {
		panic(err)
	}
}
