// Package api sets up and starts the API
// server with routing, middleware, and Swagger documentation.
package api

import (
	"fmt"
	"net/http"

	"github.com/matt-dz/wecook/docs"
	apiError "github.com/matt-dz/wecook/internal/api/error"
	"github.com/matt-dz/wecook/internal/api/middleware"
	api "github.com/matt-dz/wecook/internal/api/openapi"
	"github.com/matt-dz/wecook/internal/api/requestid"
	"github.com/matt-dz/wecook/internal/env"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimw "github.com/oapi-codegen/nethttp-middleware"
)

const (
	defaultPort = "8080"
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
	serverPort := env.Get("SERVER_PORT")
	if serverPort == "" {
		serverPort = defaultPort
	}

	server := api.NewServer()
	router := chi.NewMux()
	spec, err := docs.Docs.ReadFile("api.yaml")
	if err != nil {
		return fmt.Errorf("reading openapi spec: %w", err)
	}
	swagger, err := openapi3.NewLoader().LoadFromData(spec)
	if err != nil {
		return fmt.Errorf("creating openapi loader: %w", err)
	}
	swagger.Servers = nil

	router.Use(middleware.AddRequestID)
	router.Use(middleware.LogRequest(env.Logger))
	router.Use(middleware.InjectEnv(env))
	router.Use(middleware.Recoverer)
	router.Use(middleware.AddCors)
	router.Use(oapimw.OapiRequestValidatorWithOptions(swagger, &oapimw.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: middleware.OAPIAuthFunc,
		},
		ErrorHandlerWithOpts: middleware.OAPIErrorHandler,
	}))

	// Customize strict handler to return errors in custom format
	strictHandlerOptions := api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(r.Context()))
			// Request decoding errors are client errors (invalid JSON, etc.)
			_ = apiError.EncodeError(w, apiError.BadRequest, err.Error(), requestID)
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			requestID := fmt.Sprintf("%d", requestid.ExtractRequestID(r.Context()))
			// Response encoding errors are server errors
			_ = apiError.EncodeInternalError(w, requestID)
		},
	}

	api.HandlerFromMux(
		api.NewStrictHandlerWithOptions(server, nil, strictHandlerOptions),
		router)
	s := &http.Server{
		Handler: router,
		Addr:    "0.0.0.0:" + serverPort,
	}

	env.Logger.Info(fmt.Sprintf("Listening at localhost:%s", serverPort))
	env.Logger.Info(fmt.Sprintf("Swagger UI available at http://localhost:%s/api/swagger/index.html", serverPort))
	return s.ListenAndServe()
}
