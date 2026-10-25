package testdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// TruncateTables truncates all user tables in the public schema except schema_migrations,
// resetting identity sequences and cascading through foreign keys.
func TruncateTables(ctx context.Context, pool *pgxpool.Pool) error {
	query := `
		DO $$
		DECLARE
			stmt TEXT;
		BEGIN
			SELECT 'TRUNCATE TABLE ' || string_agg(quote_ident(tablename), ', ') || ' RESTART IDENTITY CASCADE'
			INTO stmt
			FROM pg_tables
			WHERE schemaname = 'public'
			  AND tablename NOT IN ('schema_migrations');

			IF stmt IS NOT NULL THEN
				EXECUTE stmt;
			END IF;
		END $$;
	`
	_, err := pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to truncate tables: %w", err)
	}
	return nil
}
