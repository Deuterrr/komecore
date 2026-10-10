package database

import (
	"context"
	"fmt"
	"log"

	"komecore/internal/infra/db/seeds"
)

func RunSeed(conn *Connection) error {
	ctx := context.Background()

	if err := seeds.SeedRoles(ctx, conn.Pool); err != nil {
		return fmt.Errorf("seed roles: %w", err)
	}
	log.Printf("database: roles seeded")

	if err := seeds.SeedCouriers(ctx, conn.Pool); err != nil {
		return fmt.Errorf("seed couriers: %w", err)
	}
	log.Printf("database: couriers seeded")

	if err := seeds.SeedPaymentInstructions(ctx, conn.Pool); err != nil {
		return fmt.Errorf("seed payment instructions: %w", err)
	}
	log.Printf("database: payment instructions seeded")

	return nil
}
