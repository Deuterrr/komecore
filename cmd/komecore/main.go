package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"komecore/internal/bootstrap"
	"komecore/internal/config"
	database "komecore/internal/infra/db"
	"komecore/pkg/version"
)

func main() {
	config.LoadDotEnv()

	o := parseOpts()
	if executeOpts(o) {
		return
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

type opts struct {
	showVersion    bool
	seed           bool
	seedDev        bool
	force          bool
	migrate        bool
	rollback       bool
	migrateVersion bool
	target         string
}

func parseOpts() opts {
	var o opts

	flag.BoolVar(&o.showVersion, "version", false, "print version information and exit")
	flag.BoolVar(&o.showVersion, "v", false, "print version information and exit (shorthand)")

	flag.BoolVar(&o.seed, "seed", false, "run database seed routines and exit")
	flag.BoolVar(&o.seedDev, "seed-dev", false, "run database seeds including development fixtures and exit")
	flag.BoolVar(&o.seedDev, "dev", false, "alias for -seed-dev (shorthand)")
	flag.BoolVar(&o.force, "force", false, "force execution (e.g. bypass production safety guards for dev seeds)")

	flag.BoolVar(&o.migrate, "migrate", false, "apply all pending database migrations and exit")
	flag.BoolVar(&o.rollback, "migrate-rollback", false, "rollback 1 database migration step and exit")
	flag.BoolVar(&o.migrateVersion, "migrate-version", false, "print current database migration version and exit")
	flag.StringVar(&o.target, "target", "", "database target: postgres | supabase | sqlserver (defaults to DB_TARGET env or postgres)")

	flag.Parse()
	return o
}

func executeOpts(o opts) bool {
	if o.showVersion {
		fmt.Printf("komecore %s\n", version.String())
		return true
	}

	withDev := o.seedDev
	isSeeding := o.seed || withDev

	if isSeeding || o.migrate || o.rollback || o.migrateVersion {
		target := strings.ToLower(strings.TrimSpace(o.target))
		dbCfg := config.ResolveDBConfig(target)
		if dbCfg.DSN == nil || *dbCfg.DSN == "" {
			log.Fatalf("database: DSN for target %q is not configured", target)
		}

		if o.migrateVersion {
			v, dirty, err := getMigrationVersion(dbCfg)
			if err != nil {
				log.Fatalf("migration version check failed: %v", err)
			}
			if dirty {
				fmt.Printf("%d (dirty)\n", v)
			} else {
				fmt.Printf("%d\n", v)
			}
			return true
		}

		if o.migrate {
			if err := runMigration(dbCfg); err != nil {
				log.Fatalf("migration failed: %v", err)
			}
			return true
		}

		if o.rollback {
			if err := runRollback(dbCfg); err != nil {
				log.Fatalf("migration rollback failed: %v", err)
			}
			return true
		}

		if isSeeding {
			conn, err := database.NewConnection(dbCfg)
			if err != nil {
				log.Fatalf("database connection failed: %v", err)
			}
			defer conn.Close()

			if err := runSeed(conn, withDev, o.force, os.Getenv("APP_ENV")); err != nil {
				log.Fatalf("seed failed: %v", err)
			}
			return true
		}
	}

	return false
}
