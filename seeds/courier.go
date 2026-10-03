package seeds

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedCouriers(ctx context.Context, pool *pgxpool.Pool) error {
	seeded, err := courierAlreadySeeded(ctx, pool)
	if err != nil {
		return err
	}

	if seeded {
		log.Println("database: couriers already seeded, skipping")
		return nil
	}

	log.Println("database: seeding couriers")

	query := `
		INSERT INTO couriers (id, code, name, is_active)
		VALUES
			(gen_random_uuid(), 'jne', 'Jalur Nugraha Ekakurir', true),
			(gen_random_uuid(), 'sicepat', 'SiCepat', true),
			(gen_random_uuid(), 'ide', 'IDExpress', true),
			(gen_random_uuid(), 'sap', 'SAP Express', true),
			(gen_random_uuid(), 'ninja', 'Ninja Express', true),
			(gen_random_uuid(), 'jnt', 'J&T Express', true),
			(gen_random_uuid(), 'tiki', 'TIKI', true),
			(gen_random_uuid(), 'wahana', 'Wahana Express', true),
			(gen_random_uuid(), 'pos', 'POS Indonesia', true),
			(gen_random_uuid(), 'sentral', 'Sentral Cargo', true),
			(gen_random_uuid(), 'lion', 'Lion Parcel', true),
			(gen_random_uuid(), 'rex', 'Royal Express Indonesia', true)
	`

	if _, err := pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to seed couriers: %w", err)
	}

	if err := markCourierSeeded(ctx, pool); err != nil {
		return fmt.Errorf("failed to mark courier seed: %w", err)
	}

	return nil
}

func courierAlreadySeeded(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool

	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM seed_versions WHERE name = $1
		)
	`, "courier_v1").Scan(&exists)

	return exists, err
}

func markCourierSeeded(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO seed_versions (name, version)
		VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`, "courier_v1", "1.0")

	return err
}
