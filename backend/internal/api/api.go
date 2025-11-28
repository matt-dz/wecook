// Package api sets up and starts the API
// server with routing, middleware, and Swagger documentation.
package api

import (
	"fmt"
	"net/http"

	_ "github.com/matt-dz/wecook/docs"
	"github.com/matt-dz/wecook/internal/api/middleware"
	"github.com/matt-dz/wecook/internal/api/routes/admin"
	"github.com/matt-dz/wecook/internal/api/routes/ping"
	"github.com/matt-dz/wecook/internal/api/routes/users"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/role"

	"github.com/go-chi/chi/v5"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

const (
	serverPort = 8080
)

func addDocs(r *chi.Mux, serverAddr string) {
	swagger := httpSwagger.Handler(
		httpSwagger.URL(fmt.Sprintf("http://%s/api/swagger/doc.json", serverAddr)),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	)

	r.Mount("/api/swagger", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Handle preflight
		if req.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Allow GET to serve Swagger
		if req.Method == http.MethodGet {
			swagger.ServeHTTP(w, req)
			return
		}

		// Block anything else
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}))
}

func addRoutes(router *chi.Mux) {
	router.Route("/api", func(r chi.Router) {
		r.Get("/ping", ping.HandlePing)

		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.AuthorizeRequest(role.RoleAdmin))

			r.Post("/user", users.HandleCreateUser)
			r.Post("/", admin.HandleCreateAdmin)
		})
	})
}

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
	router.Use(middleware.AddRequestID)
	router.Use(middleware.LogRequest(env.Logger))
	router.Use(middleware.InjectEnv(env))
	router.Use(middleware.AddCors)

	addRoutes(router)
	addDocs(router, fmt.Sprintf("0.0.0.0:%d", serverPort))
	http.Handle("/", router)

	env.Logger.Info(fmt.Sprintf("Listening at 0.0.0.0:%d", serverPort))
	env.Logger.Info(fmt.Sprintf("Swagger UI available at http://0.0.0.0:%d/api/swagger/index.html", serverPort))
	return http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil)
}
