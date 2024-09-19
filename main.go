package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	// This is just to make it easier to test
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/signalfx/splunk-otel-go/instrumentation/database/sql/splunksql"
	_ "github.com/signalfx/splunk-otel-go/instrumentation/github.com/lib/pq/splunkpq"
	"go.opentelemetry.io/otel"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func startPostgres() *postgres.PostgresContainer {
	container, err := postgres.Run(context.Background(),
		"docker.io/postgres:16-alpine",
		postgres.WithDatabase("postgres"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		fmt.Printf("failed to start container: %s\n", err)
		os.Exit(1)
	}

	return container
}

func connect(connStr string) *sql.DB {
	conn, err := splunksql.Open("postgres", connStr)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	_, err = conn.Exec("SELECT 1")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	return conn
}

func main() {
	container := startPostgres()
	defer container.Terminate(context.Background())

	connStr, err := container.ConnectionString(
		context.Background(),
		"sslmode=disable",
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	conn1 := connect(connStr)
	defer conn1.Close()

	// If one comments the following line, the program does not
	// panic anymore!
	conn2 := connect(connStr)
	defer conn2.Close()

	reader := sdkmetric.NewManualReader()
	mp := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(mp)

	fmt.Println("All good")
}
