package marketing

import (
	"go-on-rails/auth"
	"go-on-rails/common"

	"github.com/gofiber/fiber/v2"
)

func AddRoutes(app *fiber.App) {
	app.Get("/protected", func(c *fiber.Ctx) error {
		userId, err := auth.IsLoggedIn(c)
		if err != nil {
			return c.Redirect("/login?redirect=/protected&error=Please+log+in+to+view+this+page")
		}

		var email string
		err = auth.AuthDb.Get(&email, `SELECT email FROM users WHERE id = $1`, userId)
		if err != nil {
			return c.Redirect("/login?redirect=/protected&error=Please+log+in+to+view+this+page")
		}

		return common.RenderTempl(c, protected_page(email))
	})
}
