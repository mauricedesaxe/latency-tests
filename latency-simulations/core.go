package latency_simulations

import (
	"fmt"
	"go-on-rails/common"
	"math/rand"
	"sync"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/montanaflynn/stats"
)

var allLock sync.Mutex

type SimulationType string

const (
	SQLite      SimulationType = "sqlite"
	SameBox     SimulationType = "same_box"
	IntraAZ     SimulationType = "intra_az"
	InterAZ     SimulationType = "inter_az"
	InterRegion SimulationType = "inter_region"
)

func simulateAll() {
	allLock.Lock()
	defer allLock.Unlock()

	var err error

	sqliteSim, err := simulate(SQLite)
	if err != nil {
		panic(err)
	}

	sameBoxSim, err := simulate(SameBox)
	if err != nil {
		panic(err)
	}

	intraAZSim, err := simulate(IntraAZ)
	if err != nil {
		panic(err)
	}

	interAZSim, err := simulate(InterAZ)
	if err != nil {
		panic(err)
	}

	interRegionSim, err := simulate(InterRegion)
	if err != nil {
		panic(err)
	}

	tx, err := db.Beginx()
	if err != nil {
		panic(err)
	}
	defer func() {
		if err != nil {
			tx.Rollback()
			panic(err)
		}
	}()

	// drop table if it exists; ensures a clean slate
	_, err = tx.Exec(`DROP TABLE IF EXISTS latency_logs`)
	if err != nil {
		return
	}

	// create table if it doesn't exist
	_, err = tx.Exec(`CREATE TABLE IF NOT EXISTS latency_logs (
			label TEXT NOT NULL PRIMARY KEY,
			median_latency REAL,
			p10_latency REAL,
			p25_latency REAL,
			p75_latency REAL,
			p90_latency REAL,
			p95_latency REAL,
			count REAL
		)`)
	if err != nil {
		return
	}

	// create index on label
	_, err = tx.Exec(`CREATE INDEX IF NOT EXISTS idx_label ON latency_logs (label)`)
	if err != nil {
		return
	}

	logs := []struct {
		label string
		stats LatencyStats
	}{
		{"SQLite Read1", sqliteSim.Read1},
		{"SQLite Read2", sqliteSim.Read2},
		{"SQLite Write1", sqliteSim.Write1},
		{"SameBox Read1", sameBoxSim.Read1},
		{"SameBox Read2", sameBoxSim.Read2},
		{"SameBox Write1", sameBoxSim.Write1},
		{"IntraAZ Read1", intraAZSim.Read1},
		{"IntraAZ Read2", intraAZSim.Read2},
		{"IntraAZ Write1", intraAZSim.Write1},
		{"InterAZ Read1", interAZSim.Read1},
		{"InterAZ Read2", interAZSim.Read2},
		{"InterAZ Write1", interAZSim.Write1},
		{"InterRegion Read1", interRegionSim.Read1},
		{"InterRegion Read2", interRegionSim.Read2},
		{"InterRegion Write1", interRegionSim.Write1},
	}
	for _, log := range logs {
		err = logLatency(tx, log.label, log.stats)
		if err != nil {
			return
		}
	}

	err = tx.Commit()
	if err != nil {
		return
	}
}

type LatencyStats struct {
	MedianLatency float64
	P10Latency    float64
	P25Latency    float64
	P75Latency    float64
	P90Latency    float64
	P95Latency    float64
	Count         float64
}

type Simulation struct {
	Read1  LatencyStats
	Read2  LatencyStats
	Write1 LatencyStats
}

// Runs the latency simulation for the given simulation type
func simulate(simulationType SimulationType) (Simulation, error) {
	// if sqlite, use the default db
	if simulationType == SQLite {
		return simulateSQLiteLatency(db)
	}

	// otherwise, use the appropriate db url
	var dbURL string
	switch simulationType {
	case SameBox:
		dbURL = common.Env.SAME_BOX_POSTGRES_URL
	case IntraAZ:
		dbURL = common.Env.INTRA_AZ_POSTGRES_URL
	case InterAZ:
		dbURL = common.Env.INTER_AZ_POSTGRES_URL
	case InterRegion:
		dbURL = common.Env.INTER_REGION_POSTGRES_URL
	default:
		return Simulation{}, nil
	}

	// instantiate a new postgres db
	localDb, err := sqlx.Open("postgres", dbURL)
	if err != nil {
		return Simulation{}, err
	}

	// run the simulation
	return simulatePostgresLatency(localDb)
}

const (
	productCount          = 1_000
	reviewCountPerProduct = 10
	queryCount            = 100
)

func simulateSQLiteLatency(db *sqlx.DB) (Simulation, error) {
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

func simulatePostgresLatency(db *sqlx.DB) (Simulation, error) {
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

func calculateLatencyStatsNs(latencies []time.Duration) (LatencyStats, error) {
	durations := make([]float64, len(latencies))
	for i, d := range latencies {
		durations[i] = float64(d.Nanoseconds())
	}

	medianLatency, err := stats.Median(stats.Float64Data(durations))
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
		}, err
	}

	p10Latency, err := stats.Percentile(stats.Float64Data(durations), 10)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
		}, err
	}

	p25Latency, err := stats.Percentile(stats.Float64Data(durations), 25)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
		}, err
	}

	p75Latency, err := stats.Percentile(stats.Float64Data(durations), 75)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
			P75Latency:    p75Latency,
		}, err
	}

	p90Latency, err := stats.Percentile(stats.Float64Data(durations), 90)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
			P75Latency:    p75Latency,
			P90Latency:    p90Latency,
		}, err
	}

	p95Latency, err := stats.Percentile(stats.Float64Data(durations), 95)
	if err != nil {
		return LatencyStats{
			MedianLatency: medianLatency,
			P10Latency:    p10Latency,
			P25Latency:    p25Latency,
			P75Latency:    p75Latency,
			P90Latency:    p90Latency,
		}, err
	}

	return LatencyStats{
		MedianLatency: medianLatency,
		P10Latency:    p10Latency,
		P25Latency:    p25Latency,
		P75Latency:    p75Latency,
		P90Latency:    p90Latency,
		P95Latency:    p95Latency,
		Count:         float64(len(latencies)),
	}, nil
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
func logLatency(db *sqlx.Tx, label string, latency LatencyStats) error {
	_, err := db.NamedExec(`
		INSERT INTO latency_logs (label, median_latency, p10_latency, p25_latency, p75_latency, p90_latency, p95_latency, count) 
		VALUES (:label, :median_latency, :p10_latency, :p25_latency, :p75_latency, :p90_latency, :p95_latency, :count)
		ON CONFLICT (label) DO UPDATE SET
			median_latency = :median_latency,
			p10_latency = :p10_latency,
			p25_latency = :p25_latency,
			p75_latency = :p75_latency,
			p90_latency = :p90_latency,
			p95_latency = :p95_latency,
			count = :count`, LatencyLog{
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
