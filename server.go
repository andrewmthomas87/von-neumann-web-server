package main

import (
	"context"
	"log"

	"github.com/andrewmthomas87/von-neumann-web-server/game"
	"github.com/andrewmthomas87/von-neumann-web-server/manager"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/websocket/v2"
	"github.com/pion/webrtc/v3"
	"github.com/spf13/viper"
)

func main() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$HOME/.von-neumann-web-server/")
	viper.AddConfigPath(".")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal(err)
	}

	viper.SetDefault("addr", ":9000")
	viper.SetDefault("cors.allowOrigins", "http://localhost:3000")

	m := manager.New()

	app := fiber.New()

	app.Use(recover.New(), logger.New(), cors.New(cors.Config{AllowOrigins: viper.GetString("cors.allowOrigins")}))

	app.Get("/list-servers", func(c *fiber.Ctx) error {
		servers := m.List()
		ids := make([]string, len(servers))
		for i, s := range servers {
			ids[i] = s.ID()
		}

		return c.JSON(ids)
	})

	app.Post("/connect/:id", func(c *fiber.Ctx) error {
		id := c.Params("id")

		sd := new(webrtc.SessionDescription)
		if err := c.BodyParser(sd); err != nil {
			return err
		}

		sd, err := m.Connect(id, sd)
		if err != nil {
			return err
		}

		return c.JSON(sd)
	})

	app.Get("/ws", websocket.New(func(c *websocket.Conn) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		s := game.NewServer(c)
		m.Register(s)
		defer m.Unregister(s)

		_ = s.Run(ctx)
	}))

	log.Fatal(app.Listen(viper.GetString("addr")))
}
