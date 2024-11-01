package latency_simulations

import (
	"go-on-rails/common"
	"time"

	"github.com/gofiber/fiber/v2"
)

func AddRoutes(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		sortBy := c.Query("sort_by", "label")
		sortOrder := c.Query("sort_order", "asc")

		// Validate sort parameters
		validSortColumns := map[string]bool{
			"label":  true,
			"median": true,
			"p10":    true,
			"p25":    true,
			"p75":    true,
			"p90":    true,
			"p95":    true,
			"count":  true,
		}

		if !validSortColumns[sortBy] {
			sortBy = "label"
		}

		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "asc"
		}

		// Map the URL parameter to the actual column name
		columnMapping := map[string]string{
			"label":  "label",
			"median": "median_latency",
			"p10":    "p10_latency",
			"p25":    "p25_latency",
			"p75":    "p75_latency",
			"p90":    "p90_latency",
			"p95":    "p95_latency",
			"count":  "count",
		}

		sqlColumn := columnMapping[sortBy]
		var logs []LatencyLog
		err := db.Select(&logs, `SELECT * FROM latency_logs ORDER BY `+sqlColumn+` `+sortOrder)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		// Set cache headers
		common.SetCacheHeader(c, common.CacheOptions{
			MaxAge:               time.Minute * 5, // Cache for 5 minutes
			StaleWhileRevalidate: time.Minute * 1, // Allow stale content for 1 minute while revalidating
			StaleIfError:         time.Minute * 5, // Allow stale content for 5 minutes on error
		})

		return common.RenderTempl(c, home_page(logs))
	})

	app.Get("/simulate", func(c *fiber.Ctx) error {
		err := simulateAll()
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}
		return c.Redirect("/")
	})
}
