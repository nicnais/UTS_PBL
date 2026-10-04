package main

import (
	"context"
	"log"
	"net"
	"time"

	"laundry-api/config"
	"laundry-api/database"
)

func run() error {
	settings, err := config.Load()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	pool, err := database.NewPool(ctx)
	cancel()
	if err != nil {
		return err
	}
	defer pool.Close()
	log.Println("Database berhasil terhubung")
	app, err := config.NewApp(pool, settings)
	if err != nil {
		return err
	}
	return app.Listen(net.JoinHostPort(settings.Host, settings.Port))
}
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}