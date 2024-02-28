package controllers

import (
	"github.com/arangodb/go-driver"
	"github.com/kmrhemant916/prometheus-broadcaster/config"
	"github.com/nats-io/nats.go"
)

type App struct {
	DB driver.Client
	NATSConn *nats.Conn
	Config  *config.Config
	Collection driver.Collection
}

func NewApp(db driver.Client, natsConn *nats.Conn, config *config.Config, collection driver.Collection) (*App) {
	app := &App{
		DB:          db,
		NATSConn:    natsConn,
		Config: 	 config,
		Collection:  collection,
	}
	return app
}