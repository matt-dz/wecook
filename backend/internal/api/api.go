// Package api sets up and starts the API
// server with routing, middleware, and Swagger documentation.
package api

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/matt-dz/wecook/internal/env"
)

const (
	serverPort = 8080
)

// Start godoc
//
//	@title						WeCook API
//	@version					1.0
//	@description				API Server for the WeCook application.
//
//	@securityDefinitions.apikey	BearerAuth
//	@in							header
//	@name						Authorization
//
//	@host						localhost:8080
//	@BasePath					/api
func Start(env *env.Env) error {
	router := chi.NewRouter()
	// router.Use(middleware.InjectEnvironment(env))
	// router.Use(middleware.LogRequest)
	// router.Use(middleware.RecoverMiddleware)
	// if os.Getenv("ENV") != "production" {
	// 	router.Use(middleware.HandleCors)
	// }

	// apiRouter := router.PathPrefix("/api").Subrouter()
	// addRoutes(apiRouter)
	http.Handle("/", router)

	env.Log.Info(fmt.Sprintf("Listening at 0.0.0.0:%d", serverPort))
	env.Log.Info(fmt.Sprintf("Swagger UI available at http://0.0.0.0:%d/api/swagger/index.html", serverPort))
	return http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil)
}
