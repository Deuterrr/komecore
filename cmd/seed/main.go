package main

import (
	"flag"
	"log"
	"strings"

	"komecore/internal/bootstrap"
	"komecore/internal/config"
	database "komecore/internal/infra/db"
)

func main() {
	targetFlag := flag.String("target", "", "database target: postgres | supabase | sqlserver (defaults to DB_TARGET env or postgres)")
	flag.Parse()

	cfg := bootstrap.LoadConfig()

	target := strings.ToLower(strings.TrimSpace(*targetFlag))
	if target == "" {
		target = strings.ToLower(strings.TrimSpace(cfg.DBTarget))
	}
	if target == "" {
		target = "postgres"
	}

	dbCfg := config.ResolveDBConfig(target)
	if dbCfg.DSN == nil || *dbCfg.DSN == "" {
		log.Fatalf("database: DSN for target %q is not configured", target)
	}

	log.Printf("\x1b[0;94;49mseed start [target=%s]\x1b[0;39;49m", target)

	conn, err := database.NewConnection(dbCfg)
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer conn.Close()

	if err := database.RunSeed(conn); err != nil {
		log.Fatalf("seed failed: %v", err)
	}

	log.Printf("\x1b[0;94;49mseed complete [target=%s]\x1b[0;39;49m", target)
}
