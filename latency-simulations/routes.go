package latency_simulations

import (
	"go-on-rails/common"

	"github.com/gofiber/fiber/v2"
)

func AddRoutes(app *fiber.App) {
	app.Get("/", func(c *fiber.Ctx) error {
		sortBy := c.Query("sort_by", "label")
		sortOrder := c.Query("sort_order", "asc")

		// Validate sort parameters
		validSortColumns := map[string]bool{
			"label":          true,
			"median_latency": true,
			"p10_latency":    true,
			"p25_latency":    true,
			"p75_latency":    true,
			"p90_latency":    true,
			"p95_latency":    true,
			"count":          true,
		}

		if !validSortColumns[sortBy] {
			sortBy = "label"
		}

		if sortOrder != "asc" && sortOrder != "desc" {
			sortOrder = "asc"
		}

		var logs []LatencyLog
		err := db.Select(&logs, `select * from latency_logs order by `+sortBy+" "+sortOrder)
		if err != nil {
			return c.Status(500).SendString(err.Error())
		}

		return common.RenderTempl(c, home_page(logs))
	})
}
