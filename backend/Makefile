ifneq (,$(wildcard ./.env))
    include .env
    export
endif

SERVER_DIR:=cmd/wecook
SERVER_BIN:=wecook
DOCKER_NAME:=wecook
BIN_DIR:=bin

.PHONY: all
all: build

.PHONY: build
build:
	@echo "🔨  Building $(SERVER_BIN)…"
	go build -o $(BIN_DIR)/$(SERVER_BIN) $(SERVER_DIR)/main.go
	@echo "✓  Built $(BIN_DIR)/$(SERVER_BIN)"

.PHONY: docker-build
docker-build:
	@echo "🔨🐳 Building docker image \"$(SERVER_BIN)\" with tag \"$(DOCKER_NAME)\""
	docker build -f Dockerfile -t $(DOCKER_NAME) .
	@echo "✓  Built $(SERVER_BIN)"

.PHONY: run
run: swagger
	@echo "🚀  Starting..."
	go run $(SERVER_DIR)/main.go

.PHONY: fmt
fmt: swagger-fmt sql-fmt
	@echo "🎨  Formatting source code..."
	golangci-lint fmt -v

.PHONY: lint
lint:
	@echo "🔍  Linting source code..."
	golangci-lint run -v

.PHONY: swagger
swagger:
	@echo "📜  Generating swagger docs..."
	swag init -g internal/api/api.go \
		--parseInternal --parseDependency

.PHONY: swagger-fmt
swagger-fmt:
	@echo "🎨  Formatting swagger docs..."
	swag fmt

.PHONY: sql-fmt
sql-fmt:
	@echo "🎨 Formatting SQL"
	pg_format -i query.sql schema.sql

.PHONY: sqlc
sqlc:
	@echo "🗄️  Generating SQLC code..."
	sqlc generate

.PHONY: clean
clean:
	@echo "🧹 Cleaning $(BIN_DIR)"
	rm $(BIN_DIR)/*

.PHONY: keys
keys:
	@python3 scripts/generate-keys.py \
		--jwks-path ./keys/jwks.json \
		--private-keys-path ./keys/private-keys.json

.PHONY: help
help:
	@cat Makefile
