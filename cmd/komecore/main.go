package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"komecore/internal/bootstrap"
	"komecore/pkg/version"
)

func main() {
	showVersion := flag.Bool("version", false, "print version information and exit")
	flag.BoolVar(showVersion, "v", false, "print version information and exit (shorthand)")
	flag.Parse()

	if *showVersion {
		fmt.Printf("komecore %s\n", version.String())
		os.Exit(0)
	}

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
