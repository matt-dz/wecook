// Package api sets up and starts the API
// server with routing, middleware, and Swagger documentation.
package api

import (
	"fmt"
	"net/http"

	"github.com/matt-dz/wecook/docs"
	"github.com/matt-dz/wecook/internal/api/middleware"
	api "github.com/matt-dz/wecook/internal/api/openapi"
	"github.com/matt-dz/wecook/internal/api/routes/admin"
	"github.com/matt-dz/wecook/internal/api/routes/auth"
	"github.com/matt-dz/wecook/internal/api/routes/ping"
	"github.com/matt-dz/wecook/internal/api/routes/recipes"
	"github.com/matt-dz/wecook/internal/api/routes/users"
	"github.com/matt-dz/wecook/internal/env"
	"github.com/matt-dz/wecook/internal/role"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/go-chi/chi/v5"
	oapimw "github.com/oapi-codegen/nethttp-middleware"
)

const (
	defaultPort = "8080"
)

func addRoutes(router *chi.Mux) {
	router.Route("/api", func(r chi.Router) {
		r.Get("/ping", ping.HandlePing)
		r.Post("/login", users.HandleUserLogin)
		r.Post("/auth/session/refresh", auth.HandleRefreshSession)
		r.With(middleware.AuthorizeRequest(role.RoleUser)).
			Get("/auth/session/verify", auth.HandleVerifySession)

		r.Post("/setup/admin", admin.HandleAdminSetup)
		r.Route("/admin", func(r chi.Router) {
			r.Use(middleware.AuthorizeRequest(role.RoleAdmin))

			r.Post("/user", users.HandleCreateUser)
			r.Post("/", admin.HandleCreateAdmin)
		})

		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthorizeRequest(role.RoleUser))
			r.Post("/recipes", recipes.CreateRecipe)
			r.Post("/recipes/ingredients", recipes.CreateRecipeIngredient)
			r.Post("/recipes/steps", recipes.CreateRecipeStep)
			r.Get("/recipes/personal", recipes.GetPersonalRecipes)
			r.Get("/recipes/personal/{recipeID}", recipes.GetPersonalRecipe)
			r.Post("/recipes/{recipeID}/cover", recipes.UpdateRecipeCover)
			r.Delete("/recipes/{recipeID}", recipes.DeleteRecipe)
			r.Delete("/recipes/{recipeID}/ingredients/{ingredientID}", recipes.DeleteRecipeIngredient)
			r.Delete("/recipes/{recipeID}/steps/{stepID}", recipes.DeleteRecipeStep)
			r.Patch("/recipes/{recipeID}/steps/{stepID}", recipes.UpdateRecipeStep)
			r.Patch("/recipes/{recipeID}/ingredients/{ingredientID}", recipes.UpdateRecipeIngredient)
			r.Patch("/recipes/{recipeID}", recipes.UpdateRecipe)
			r.Put("/recipes/{recipeID}", recipes.UpdateRecipeFull)
		})
		r.Get("/recipes/{recipeID}", recipes.GetRecipe)
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
	router.Use(middleware.AddCors)
	router.Use(oapimw.OapiRequestValidatorWithOptions(swagger, &oapimw.Options{
		Options: openapi3filter.Options{
			AuthenticationFunc: middleware.OAPIAuthFunc,
		},
		ErrorHandlerWithOpts: middleware.OAPIErrorHandler,
	}))

	api.HandlerFromMux(
		api.NewStrictHandler(server, nil),
		router)
	s := &http.Server{
		Handler: router,
		Addr:    "0.0.0.0:" + serverPort,
	}

	env.Logger.Info(fmt.Sprintf("Listening at localhost:%s", serverPort))
	env.Logger.Info(fmt.Sprintf("Swagger UI available at http://localhost:%s/api/swagger/index.html", serverPort))
	return s.ListenAndServe()
}
