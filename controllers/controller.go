package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/kmrhemant916/prometheus-broadcaster/utils"
)

type User struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (app *App)Register(c *gin.Context) {
	var user User
	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	query := "FOR u IN users FILTER u.username == @username RETURN u"
	bindVars := map[string]interface{}{
		"username": user.Username,
	}
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
	cursor, err := utils.QueryWithRetry(ctx, app.Collection, query, bindVars, 5, 3*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close()

	if cursor.HasMore() {
		c.JSON(http.StatusConflict, gin.H{"error": "User already exists"})
		return
	}

	// Create the user document in the users collection
	_, err = app.Collection.CreateDocument(ctx, user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": user.Username,
		"exp":      time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString([]byte(app.Config.JWTKey))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}

func (app *App) Publish(c *gin.Context) {
	var body struct {
		Alerts []struct {
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
		} `json:"alerts"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	for _, alert := range body.Alerts {
		alertData := struct {
			Labels      map[string]string `json:"labels"`
			Annotations map[string]string `json:"annotations"`
		}{
			Labels:      alert.Labels,
			Annotations: alert.Annotations,
		}

		data, err := json.Marshal(alertData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal alert data"})
			return
		}
		err = app.NATSConn.Publish("alerts", data)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish message"})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"message": "Messages published successfully"})
}

func (app *App) Health(c *gin.Context) {
    if err := app.NATSConn.Publish("health-check", []byte("test")); err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to publish test message to NATS"})
        return
    }
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    db, err := app.DB.Database(ctx, app.Config.ArangoDB.Database)
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open database"})
        return
    }
    cursor, err := db.Query(ctx, "RETURN 1", map[string]interface{}{})
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query ArangoDB"})
        return
    }
    defer cursor.Close()
    c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
