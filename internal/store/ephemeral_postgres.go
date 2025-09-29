package store

// TODO this doesn't belong in store

import (
	"context"
	"fmt"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type MockPostgres struct {
	ConnectionString string
	container        *postgres.PostgresContainer
}

func NewMockPostgres(ctx context.Context) (*MockPostgres, error) {
	container, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:15.3-alpine"),
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second),
		),
	)

	if err != nil {
		return nil, fmt.Errorf("Could not start postgres container: %v", err)
	}

	connectionString, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("Could not get connection string: %v", err)
	}

	return &MockPostgres{
		ConnectionString: connectionString,
		container:        container,
	}, nil
}

func (m *MockPostgres) Terminate(ctx context.Context) error {
	return m.container.Terminate(ctx)
}
