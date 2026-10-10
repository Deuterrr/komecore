package seeds

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func SeedRoles(ctx context.Context, pool *pgxpool.Pool) error {
	seeded, err := roleAlreadySeeded(ctx, pool)
	if err != nil {
		return err
	}

	if seeded {
		log.Println("database: roles already seeded, skipping")
		return nil
	}

	log.Println("database: seeding roles")

	query := `
		INSERT INTO roles (id, code, name)
		VALUES
			(gen_random_uuid(), 'staff_admin', 'Staff Admin'),
			(gen_random_uuid(), 'staff', 'Staff')
	`

	if _, err := pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("failed to seed roles: %w", err)
	}

	if err := markRoleSeeded(ctx, pool); err != nil {
		return fmt.Errorf("failed to mark role seed: %w", err)
	}

	return nil
}

func roleAlreadySeeded(ctx context.Context, pool *pgxpool.Pool) (bool, error) {
	var exists bool

	err := pool.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM seed_versions WHERE name = $1
		)
	`, "role_v1").Scan(&exists)

	return exists, err
}

func markRoleSeeded(ctx context.Context, pool *pgxpool.Pool) error {
	_, err := pool.Exec(ctx, `
		INSERT INTO seed_versions (name, version)
		VALUES ($1, $2)
		ON CONFLICT (name) DO NOTHING
	`, "role_v1", "1.0")

	return err
}
