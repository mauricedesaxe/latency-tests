package latency_simulations

import (
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var sqliteLock sync.Mutex

func initSQLite() {
	sqliteLock.Lock()
	defer sqliteLock.Unlock()

	db, err := sql.Open("sqlite3", "./db/latency_simulations.sqlite?_journal_mode=WAL&_synchronous=NORMAL&_busy_timeout=5000&_cache_size=-2000")
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
	defer db.Close()

	simulation, err := simulateSQLiteLatency(db)
	if err != nil {
		panic(err)
	}
	fmt.Printf("SQLite Read1: %+v\n", simulation.Read1)
	logLatency("SQLite Read1", simulation.Read1)
	fmt.Printf("SQLite Read2: %+v\n", simulation.Read2)
	logLatency("SQLite Read2", simulation.Read2)
	fmt.Printf("SQLite Write1: %+v\n", simulation.Write1)
	logLatency("SQLite Write1", simulation.Write1)
}

func simulateSQLiteLatency(db *sql.DB) (Simulation, error) {
	var err error

	// drop tables if they exist; ensures a clean slate
	_, err = db.Exec(`DROP TABLE IF EXISTS product_reviews`)
	if err != nil {
		return Simulation{}, err
	}
	_, err = db.Exec(`DROP TABLE IF EXISTS products`)
	if err != nil {
		return Simulation{}, err
	}

	// create table for products
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS products (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		price REAL NOT NULL
	)`)
	if err != nil {
		return Simulation{}, err
	}

	// add index on name
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_products_name ON products (name)`)
	if err != nil {
		return Simulation{}, err
	}

	// create table for product reviews
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS product_reviews (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		product_id INTEGER NOT NULL,
		review TEXT NOT NULL
	)`)
	if err != nil {
		return Simulation{}, err
	}

	// add index on product_id
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_product_reviews_product_id ON product_reviews (product_id)`)
	if err != nil {
		return Simulation{}, err
	}

	// seed with products
	tx, err := db.Begin()
	if err != nil {
		return Simulation{}, err
	}
	for i := 0; i < productCount; i++ {
		_, err := tx.Exec(`INSERT INTO products (name, price) VALUES (?, ?)`, fmt.Sprintf("product%d", i), rand.Float64()*100)
		if err != nil {
			tx.Rollback()
			return Simulation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Simulation{}, err
	}

	// seed each product with reviews
	tx, err = db.Begin()
	if err != nil {
		return Simulation{}, err
	}
	for i := 0; i < productCount; i++ {
		for j := 0; j < reviewCountPerProduct; j++ {
			_, err := tx.Exec(`INSERT INTO product_reviews (product_id, review) VALUES (?, ?)`, i, fmt.Sprintf("review%d", j))
			if err != nil {
				tx.Rollback()
				return Simulation{}, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return Simulation{}, err
	}

	// run 100 read queries where you get the 100 most expensive products and measure the latency
	latenciesRead1 := []time.Duration{}
	for i := 0; i < queryCount; i++ {
		start := time.Now()
		var id int
		var name string
		var price float64
		err = db.QueryRow(`SELECT id, name, price FROM products ORDER BY price DESC LIMIT 1`).Scan(&id, &name, &price)
		if err != nil {
			return Simulation{}, err
		}
		latenciesRead1 = append(latenciesRead1, time.Since(start))
	}
	statsRead1, err := calculateLatencyStatsNs(latenciesRead1)
	if err != nil {
		return Simulation{
			Read1: statsRead1,
		}, err
	}

	// run 100 read queries where you get a random product and measure the latency
	latenciesRead2 := []time.Duration{}
	for i := 0; i < queryCount; i++ {
		start := time.Now()
		var id int
		var name string
		var price float64
		err = db.QueryRow(`SELECT id, name, price FROM products WHERE name = ? LIMIT 1`, fmt.Sprintf("product%d", rand.Intn(1_000))).Scan(&id, &name, &price)
		if err != nil {
			return Simulation{
				Read1: statsRead1,
			}, err
		}
		latenciesRead2 = append(latenciesRead2, time.Since(start))
	}
	statsRead2, err := calculateLatencyStatsNs(latenciesRead2)
	if err != nil {
		return Simulation{
			Read1: statsRead1,
			Read2: statsRead2,
		}, err
	}

	// add 100 new products and measure the latency
	latenciesWrite1 := []time.Duration{}
	for i := 0; i < queryCount; i++ {
		start := time.Now()
		_, err = db.Exec(`INSERT INTO products (name, price) VALUES (?, ?)`, fmt.Sprintf("product%d", i), rand.Float64()*100)
		if err != nil {
			return Simulation{
				Read1: statsRead1,
				Read2: statsRead2,
			}, err
		}
		latenciesWrite1 = append(latenciesWrite1, time.Since(start))
	}
	statsWrite1, err := calculateLatencyStatsNs(latenciesWrite1)
	if err != nil {
		return Simulation{
			Read1:  statsRead1,
			Read2:  statsRead2,
			Write1: statsWrite1,
		}, err
	}

	return Simulation{
		Read1:  statsRead1,
		Read2:  statsRead2,
		Write1: statsWrite1,
	}, nil
}
