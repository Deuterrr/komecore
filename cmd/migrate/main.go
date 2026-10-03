package main

import (
	"flag"
	"log"
	"net/http"
	"strings"

	"komecore/internal/bootstrap"
	"komecore/internal/config"
	database "komecore/internal/infra/db"
	"komecore/internal/infra/storage"
)

func main() {
	targetFlag := flag.String("target", "", "migration target: postgres | supabase | sqlserver (defaults to DB_TARGET env or postgres)")
	withStorage := flag.Bool("storage", false, "run storage bucket migration (Supabase or GCS)")
	rollbackFlag := flag.Bool("rollback", false, "rollback 1 migration step")
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

	log.Printf("\x1b[0;94;49mmigration start [target=%s dialect=%s]\x1b[0;39;49m", target, dbCfg.Dialect)

	if *rollbackFlag {
		if err := database.RunRollback(dbCfg); err != nil {
			log.Fatalf("migration rollback failed: %v", err)
		}
	} else {
		if err := database.RunMigration(dbCfg); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	}

	if *withStorage {
		storageProvider, err := bootstrap.ResolveStorageProvider(
			cfg.Storage,
			&http.Client{},
		)
		if err != nil {
			log.Fatalf("storage provider init failed: %v", err)
		}

		if _, isNoop := storageProvider.(*storage.NoopProvider); isNoop {
			log.Printf("storage: skipped (no storage credentials configured)")
		} else {
			if err := storage.RunMigration(storageProvider); err != nil {
				log.Fatalf("storage migration failed: %v", err)
			}
		}
	}

	log.Printf("\x1b[0;94;49mmigration complete [target=%s]\x1b[0;39;49m", target)
}
