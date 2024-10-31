package latency_simulations

import (
	"database/sql"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func init() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}

	db, err := sql.Open("postgres", os.Getenv("LOCAL_POSTGRES_URL"))
	if err != nil {
		panic(err)
	}

	simulation, err := simulatePostgresLatency(db)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Postgres Read1: %+v\n", simulation.Read1)
	fmt.Printf("Postgres Read2: %+v\n", simulation.Read2)
	fmt.Printf("Postgres Write1: %+v\n", simulation.Write1)
}

func simulatePostgresLatency(db *sql.DB) (Simulation, error) {
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
		id SERIAL PRIMARY KEY,
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
		id SERIAL PRIMARY KEY,
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

	// seed with 1000 products
	tx, err := db.Begin()
	if err != nil {
		return Simulation{}, err
	}
	for i := 0; i < productCount; i++ {
		_, err := tx.Exec(`INSERT INTO products (name, price) VALUES ($1, $2)`, fmt.Sprintf("product%d", i), rand.Float64()*100)
		if err != nil {
			tx.Rollback()
			return Simulation{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return Simulation{}, err
	}

	// seed each product with 10 reviews
	tx, err = db.Begin()
	if err != nil {
		return Simulation{}, err
	}
	for i := 0; i < productCount; i++ {
		for j := 0; j < reviewCountPerProduct; j++ {
			_, err := tx.Exec(`INSERT INTO product_reviews (product_id, review) VALUES ($1, $2)`, i, fmt.Sprintf("review%d", j))
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
		err = db.QueryRow(`SELECT id, name, price FROM products WHERE name = $1 LIMIT 1`, fmt.Sprintf("product%d", rand.Intn(productCount))).Scan(&id, &name, &price)
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
		_, err = db.Exec(`INSERT INTO products (name, price) VALUES ($1, $2)`, fmt.Sprintf("product%d", i), rand.Float64()*100)
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