package latency_simulations

import (
	"github.com/jmoiron/sqlx"
)

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

	// drop table if it exists; ensures a clean slate
	_, err = db.Exec(`DROP TABLE IF EXISTS latency_logs`)
	if err != nil {
		panic(err)
	}

	// create table if it doesn't exist
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS latency_logs (
		label TEXT,
		median_latency REAL,
		p10_latency REAL,
		p25_latency REAL,
		p75_latency REAL,
		p90_latency REAL,
		p95_latency REAL,
		count REAL
	)`)
	if err != nil {
		panic(err)
	}

	// create index on label
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_label ON latency_logs (label)`)
	if err != nil {
		panic(err)
	}

	initSQLite()
	initPostgres()
}

type LatencyLog struct {
	Label         string  `db:"label"`
	MedianLatency float64 `db:"median_latency"`
	P10Latency    float64 `db:"p10_latency"`
	P25Latency    float64 `db:"p25_latency"`
	P75Latency    float64 `db:"p75_latency"`
	P90Latency    float64 `db:"p90_latency"`
	P95Latency    float64 `db:"p95_latency"`
	Count         float64 `db:"count"`
}

// Logs the latency stats to the database.
func logLatency(label string, latency LatencyStats) error {
	_, err := db.NamedExec(`INSERT INTO latency_logs (label, median_latency, p10_latency, p25_latency, p75_latency, p90_latency, p95_latency, count) VALUES (:label, :median_latency, :p10_latency, :p25_latency, :p75_latency, :p90_latency, :p95_latency, :count)`, LatencyLog{
		Label:         label,
		MedianLatency: latency.MedianLatency,
		P10Latency:    latency.P10Latency,
		P25Latency:    latency.P25Latency,
		P75Latency:    latency.P75Latency,
		P90Latency:    latency.P90Latency,
		P95Latency:    latency.P95Latency,
		Count:         latency.Count,
	})
	return err
}
