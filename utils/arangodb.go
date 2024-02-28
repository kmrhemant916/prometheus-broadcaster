package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"time"

	"github.com/arangodb/go-driver"
	arangodbHTTP "github.com/arangodb/go-driver/http"
)

func Connection(host string, database string, password string, username string) (driver.Client, error) {
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}
    endpoints, err := arangodbHTTP.NewConnection(arangodbHTTP.ConnectionConfig{
        Endpoints: []string{host},
		TLSConfig: tlsConfig,
    })
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Connecting to ArangoDB at %s...", host)
	conn, err := ConnectWithRetry(endpoints, username, password, 5, 3*time.Second)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("Connected to ArangoDB")
    return conn, nil
}

func ConnectWithRetry(endpoints driver.Connection, username, password string, retries int, delay time.Duration) (driver.Client, error) {
    var client driver.Client
    var err error
    for i := 0; i < retries; i++ {
        log.Printf("Attempting to connect (attempt %d/%d)", i+1, retries)
        client, err = driver.NewClient(driver.ClientConfig{
            Connection:     endpoints,
            Authentication: driver.BasicAuthentication(username, password),
        })
        if err == nil {
            return client, nil
        }
        log.Printf("Connection attempt failed: %v", err)
        time.Sleep(delay)
    }
    return nil, fmt.Errorf("failed to connect after %d attempts", retries)
}

func QueryWithRetry(ctx context.Context, collection driver.Collection, query string, bindVars map[string]interface{}, retries int, delay time.Duration) (driver.Cursor, error) {
    var cursor driver.Cursor
    var err error
    for i := 0; i < retries; i++ {
        log.Printf("Querying database (attempt %d/%d)", i+1, retries)
        cursor, err = collection.Database().Query(ctx, query, bindVars)
        if err == nil {
            return cursor, nil
        }
        log.Printf("Query attempt failed: %v", err)
        time.Sleep(delay)
    }
    return nil, fmt.Errorf("failed to query database after %d attempts", retries)
}

func CreateDatabaseIfNotExists(ctx context.Context, client driver.Client, dbName string) (driver.Database, error) {
	exists, err := client.DatabaseExists(ctx, dbName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return client.CreateDatabase(ctx, dbName, nil)
	}
	return client.Database(ctx, dbName)
}

func CreateCollectionIfNotExists(ctx context.Context, db driver.Database, collectionName string) (driver.Collection, error) {
	exists, err := db.CollectionExists(ctx, collectionName)
	if err != nil {
		return nil, err
	}
	if !exists {
		return db.CreateCollection(ctx, collectionName, nil)
	}
	return db.Collection(ctx, collectionName)
}

func CreateDatabaseIfNotExistsWithRetry(ctx context.Context, client driver.Client, dbName string, retries int, delay time.Duration) (driver.Database, error) {
    var db driver.Database
    var err error
    for i := 0; i < retries; i++ {
        log.Printf("Checking if database %s exists (attempt %d/%d)", dbName, i+1, retries)
        db, err = CreateDatabaseIfNotExists(ctx, client, dbName)
        if err == nil {
            return db, nil
        }
        log.Printf("Database creation attempt failed: %v", err)
        time.Sleep(delay)
    }
    return nil, fmt.Errorf("failed to create database %s after %d attempts", dbName, retries)
}

func CreateCollectionIfNotExistsWithRetry(ctx context.Context, db driver.Database, collectionName string, retries int, delay time.Duration) (driver.Collection, error) {
    var collection driver.Collection
    var err error
    for i := 0; i < retries; i++ {
        log.Printf("Checking if collection %s exists (attempt %d/%d)", collectionName, i+1, retries)
        collection, err = CreateCollectionIfNotExists(ctx, db, collectionName)
        if err == nil {
            return collection, nil
        }
        log.Printf("Collection creation attempt failed: %v", err)
        time.Sleep(delay)
    }
    return nil, fmt.Errorf("failed to create collection %s after %d attempts", collectionName, retries)
}
