package main

import (
	"log"

	"komecore/internal/bootstrap"
)

func main() {
	cfg := bootstrap.LoadConfig()
	app, err := bootstrap.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer app.Close()

	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
