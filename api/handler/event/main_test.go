package event

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
	"testing"
	"treffly/db/sqlc"
	"treffly/util"
)

var (
	testStore db.Store
	testConfig util.Config
)

func TestMain(m *testing.M) {
	var err error
	testConfig, err = util.LoadConfig("../../..")
	if err != nil {
		panic("cannot load config: " + err.Error())
	}

	dbpool, err := pgxpool.New(context.Background(), testConfig.DBSource)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create connection pool: %v\n", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	testStore = db.NewStore(dbpool)
	m.Run()
} 