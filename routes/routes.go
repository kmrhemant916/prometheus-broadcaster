package routes

import (
	"github.com/arangodb/go-driver"
	"github.com/gin-gonic/gin"
	"github.com/kmrhemant916/prometheus-broadcaster/config"
	"github.com/kmrhemant916/prometheus-broadcaster/controllers"
	"github.com/kmrhemant916/prometheus-broadcaster/middlewares"
	"github.com/nats-io/nats.go"
)

func SetupRoutes(natsConn *nats.Conn, db driver.Client, config *config.Config, collection driver.Collection) (*gin.Engine){
	router := gin.Default()
	app := controllers.NewApp(db, natsConn, config, collection)
	router.POST("/health", app.Health)
	router.POST("/register", app.Register)
	protectedRoutes := router.Group("/api")
	protectedRoutes.Use(middlewares.AuthHeader(config.JWTKey))
	protectedRoutes.POST("/publish", app.Publish)
	return router
}