package utils

import (
	"context"
	"log"
	"time"

	"github.com/arangodb/go-driver"
)

func InitialiseServices(ctx context.Context, conn driver.Client, database string) (driver.Collection){
	db, err := CreateDatabaseIfNotExistsWithRetry(ctx, conn, database, 5, 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	collection, err := CreateCollectionIfNotExistsWithRetry(ctx, db, "users", 5, 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	return collection
}