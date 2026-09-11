package postgres

import (
	"errors"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

// RunMigration runs database migrations using the provided dbURL and migrations directory.
func RunMigration(dbURL, migrationsDir string) error {
	// Clean the dbURL (sometimes it has query params that golang-migrate parses differently)
	// Actually golang-migrate supports pgx/postgres URLs normally.
	
	sourceURL := fmt.Sprintf("file://%s", migrationsDir)
	
	m, err := migrate.New(sourceURL, dbURL)
	if err != nil {
		return fmt.Errorf("could not create migrate instance: %w", err)
	}
	defer m.Close()

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("could not run migrate up: %w", err)
	}

	if errors.Is(err, migrate.ErrNoChange) {
		log.Println("Database migration: no change")
	} else {
		log.Println("Database migration: successfully applied")
	}

	return nil
}
