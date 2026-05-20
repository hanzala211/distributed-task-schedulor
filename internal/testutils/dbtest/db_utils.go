package dbtest

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
)

type PostgresTestSuite struct {
	DB        *sql.DB
	Container *tcpostgres.PostgresContainer
}

func SetupSuite() *PostgresTestSuite {
	ctx := context.Background()
	ctr, err := tcpostgres.Run(ctx, "postgres:15-alpine",
		tcpostgres.WithDatabase("tasks_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies())
	if err != nil {
		panic(err)
	}
	connStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		testcontainers.TerminateContainer(ctr)
		panic(err)
	}
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		testcontainers.TerminateContainer(ctr)
		panic(err)
	}
	if err = db.PingContext(ctx); err != nil {
		db.Close()
		testcontainers.TerminateContainer(ctr)
		panic(err)
	}
	return &PostgresTestSuite{
		DB:        db,
		Container: ctr,
	}
}

func (p *PostgresTestSuite) Close() error {
	if p.DB != nil {
		_ = p.DB.Close()
	}
	if p.Container != nil {
		return testcontainers.TerminateContainer(p.Container)
	}
	return nil
}

func (p *PostgresTestSuite) RunMigrations() error {
	driver, err := migratepostgres.WithInstance(p.DB, &migratepostgres.Config{})
	if err != nil {
		return err
	}
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, "..", "..", "cmd", "migrations")
	sourceURL := "file://" + path
	m, err := migrate.NewWithDatabaseInstance(sourceURL, "postgres", driver)
	if err != nil {
		return err
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func (p *PostgresTestSuite) ResetDB(t *testing.T) {
	t.Helper()

	ctx := context.Background()

	_, err := p.DB.ExecContext(ctx, `
		TRUNCATE TABLE task_dependencies, tasks RESTART IDENTITY CASCADE;
	`)
	require.NoError(t, err)
}
