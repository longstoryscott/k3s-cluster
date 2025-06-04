package storage

import (
	"context"
	"log"
	"proxyllama/config"
	"proxyllama/util"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type TestFunc func(t *testing.T)

func DbTest(t *testing.T, tests ...TestFunc) {
	confFile := "testdata/.config.yaml"
	ctx := context.Background()

	pgc, err := postgres.Run(ctx,
		"timescale/timescaledb-ha:pg17",
		postgres.WithInitScripts("testdata/init_test_db.sh"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := testcontainers.TerminateContainer(pgc); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}()

	config.GetConfig(&confFile)

	psqlconn, err := pgc.ConnectionString(ctx)
	if err != nil {
		panic("failed to get connection string")
	}

	InitializeStorage()

	if err := InitDB(psqlconn); err != nil {
		util.HandleError(err)
	}
	util.LogInfo("Connected to PostgreSQL database", logrus.Fields{
		"connection": psqlconn,
	})
	for _, test := range tests {
		t.Run("DbTest", func(tt *testing.T) {
			test(tt)
		})
	}
}
