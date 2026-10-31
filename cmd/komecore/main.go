package main

import (
	"flag"
	"fmt"
	"io"
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

	o, err := parseArgs(os.Args[1:])
	if err != nil {
		log.Fatalf("invalid command or arguments: %v", err)
	}

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
	migrateDown    bool
	migrateVersion bool
	target         string
}

func parseArgs(args []string) (opts, error) {
	if len(args) == 0 {
		return opts{}, nil
	}

	switch args[0] {
	case "serve":
		return opts{}, nil

	case "version", "-v", "--version":
		return opts{showVersion: true}, nil

	case "migrate":
		return parseMigrateSubcommand(args[1:])

	case "seed":
		return parseSeedSubcommand(args[1:])

	default:
		return opts{}, fmt.Errorf("unknown command %q (supported: serve, migrate, seed, version)", args[0])
	}
}

func parseMigrateSubcommand(args []string) (opts, error) {
	var o opts
	fs := flag.NewFlagSet("migrate", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.StringVar(&o.target, "target", "", "database target")

	var flagArgs []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && (arg == "-target" || arg == "--target") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}

	action := "up"
	if len(positional) > 0 {
		action = positional[0]
	}

	switch action {
	case "up":
		o.migrate = true
	case "rollback", "step-down":
		o.rollback = true
	case "down":
		o.migrateDown = true
	case "version":
		o.migrateVersion = true
	default:
		return o, fmt.Errorf("unknown migration action %q (supported: up, rollback, down, version)", action)
	}

	return o, nil
}

func parseSeedSubcommand(args []string) (opts, error) {
	var o opts
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	fs.BoolVar(&o.seedDev, "dev", false, "include development fixtures")
	fs.BoolVar(&o.force, "force", false, "force execution in production")
	fs.StringVar(&o.target, "target", "", "database target")

	var flagArgs []string
	var positional []string

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			flagArgs = append(flagArgs, arg)
			if !strings.Contains(arg, "=") && (arg == "-target" || arg == "--target") && i+1 < len(args) {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			positional = append(positional, arg)
		}
	}

	if err := fs.Parse(flagArgs); err != nil {
		return o, err
	}

	o.seed = true
	if len(positional) > 0 {
		switch positional[0] {
		case "dev":
			o.seedDev = true
		case "base", "prod":
			// base seeds only
		default:
			return o, fmt.Errorf("unknown seed action %q (supported: dev, base)", positional[0])
		}
	}

	return o, nil
}

func executeOpts(o opts) bool {
	if o.showVersion {
		fmt.Printf("komecore %s\n", version.String())
		return true
	}

	withDev := o.seedDev
	isSeeding := o.seed || withDev

	if isSeeding || o.migrate || o.rollback || o.migrateDown || o.migrateVersion {
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

		if o.migrateDown {
			if err := runMigrationDown(dbCfg); err != nil {
				log.Fatalf("migration down failed: %v", err)
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
