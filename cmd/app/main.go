package main

import (
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	"github.com/markbates/goth"
	"github.com/markbates/goth/providers/google"
	"github.com/shareed2k/goth_fiber"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"regit/pkg/model"
	"regit/pkg/query"
)

func main() {
	godotenv.Load()

	gormdb, _ := gorm.Open(mysql.Open(os.Getenv("DSN")))
	gormdb.AutoMigrate(&model.User{}, &model.Message{})

	query.SetDefault(gormdb)

	app := fiber.New()

	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendString("Hello, World!")
	})

	goth.UseProviders(
		google.New(os.Getenv("OAUTH_KEY"), os.Getenv("OAUTH_SECRET"), "http://127.0.0.1:3388/auth/callback/google"),
	)

	store := session.New(session.Config{
		CookieDomain:   "http://localhost:5173",
		CookieSameSite: "None",
		CookieHTTPOnly: true,
		CookieSecure:   false,
	})
	storeSame := session.New(session.Config{})

	app.Use(cors.New(cors.Config{
		AllowOrigins:     "*",
		AllowHeaders:     "Origin, Content-Type, Accept",
		AllowCredentials: true,
	}))
	store.RegisterType(uuid.UUID{})

	getUser := func(ctx *fiber.Ctx) (*model.User, error) {
		sess, err := store.Get(ctx)
		if err != nil {
			return nil, err
		}

		userId, ok := sess.Get("user_id").(uuid.UUID)
		if !ok {
			return nil, fiber.NewError(http.StatusForbidden)
		}

		if userId == uuid.Nil {
			return nil, fiber.NewError(http.StatusForbidden)
		}

		user, err := query.User.WithContext(ctx.UserContext()).Where(query.User.ID.Eq(userId)).First()
		if err != nil {
			return nil, fiber.NewError(http.StatusForbidden, "fail query")
		}

		return user, nil
	}

	app.Get("/login/:provider", func(ctx *fiber.Ctx) error {
		if r := ctx.Query("r"); r != "" {
			sess, err := storeSame.Get(ctx)
			if err != nil {
				return err
			}

			sess.Set("next", r)

			if err := sess.Save(); err != nil {
				return err
			}
		}

		return ctx.Next()
	}, goth_fiber.BeginAuthHandler)
	app.Get("/auth/callback/:provider", func(ctx *fiber.Ctx) error {
		userAuth, err := goth_fiber.CompleteUserAuth(ctx)
		if err != nil {
			return err
		}

		sess, err := storeSame.Get(ctx)
		if err != nil {
			return err
		}

		user, err := query.User.WithContext(ctx.UserContext()).Where(query.User.UserID.Eq(userAuth.UserID)).First()
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// new user
		if errors.Is(err, gorm.ErrRecordNotFound) {
			user = &model.User{
				UserID:    userAuth.UserID,
				Email:     userAuth.Email,
				NickName:  userAuth.NickName,
				FirstName: userAuth.FirstName,
				LastName:  userAuth.LastName,
			}
			if err := query.User.WithContext(ctx.UserContext()).Create(user); err != nil {
				return err
			}
		}

		next, ok := sess.Get("next").(string)

		if err := sess.Destroy(); err != nil {
			return err
		}

		sess, err = store.Get(ctx)
		if err != nil {
			return err
		}

		sess.Set("user_id", user.ID)

		if err := sess.Save(); err != nil {
			return err
		}

		if ok && next != "" {
			nextURL, err := base64.StdEncoding.DecodeString(next)
			if err != nil {
				return err
			}
			return ctx.Redirect(string(nextURL))
		}

		return ctx.Redirect("/profile")
	})

	app.Get("/profile", func(ctx *fiber.Ctx) error {
		user, err := getUser(ctx)
		if err != nil {
			return err
		}

		return ctx.Status(http.StatusOK).JSON(user.Response())
	})

	app.Put("/messages", func(ctx *fiber.Ctx) error {
		body := struct {
			Message string `json:"message"`
		}{}

		if err := ctx.BodyParser(&body); err != nil {
			return err
		}

		user, err := getUser(ctx)
		if err != nil {
			return err
		}

		message := &model.Message{
			UserID:  user.ID,
			User:    user,
			Message: body.Message,
		}

		if err := query.Message.Create(message); err != nil {
			return err
		}

		return ctx.Status(http.StatusOK).JSON(message.Response())
	})

	app.Get("/messages", func(ctx *fiber.Ctx) error {
		messages, err := query.Message.Find()
		if err != nil {
			return err
		}
		var msgs model.Messages
		msgs = messages

		return ctx.Status(http.StatusOK).JSON(msgs.Response())
	})

	app.Get("/logout", func(ctx *fiber.Ctx) error {
		sess, err := store.Get(ctx)
		if err != nil {
			return err
		}

		if err := goth_fiber.Logout(ctx); err != nil {
			return err
		}

		if err := sess.Destroy(); err != nil {
			return err
		}

		return ctx.SendStatus(http.StatusOK)
	})

	log.Fatal(app.Listen(":3388"))
}
