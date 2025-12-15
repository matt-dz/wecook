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
build: fmt lint docs
	@echo "🔨  Building $(SERVER_BIN)…"
	go build -o $(BIN_DIR)/$(SERVER_BIN) $(SERVER_DIR)/main.go
	@echo "✓  Built $(BIN_DIR)/$(SERVER_BIN)"

.PHONY: docker-build
docker-build:
	@echo "🔨🐳 Building docker image \"$(SERVER_BIN)\" with tag \"$(DOCKER_NAME)\""
	docker build -f Dockerfile -t $(DOCKER_NAME) .
	@echo "✓  Built $(SERVER_BIN)"

.PHONY: run
run: docs
	@echo "🚀  Starting..."
	go run $(SERVER_DIR)/main.go

.PHONY: fmt
fmt: sql-fmt
	@echo "🎨  Formatting source code..."
	golangci-lint fmt -v

.PHONY: lint
lint:
	@echo "🔍  Linting source code..."
	golangci-lint run -v

.PHONY: docs
docs:
	@echo "📝 Generating OpenAPI client..."
	go tool oapi-codegen -config docs/cfg.yaml docs/api.yaml

.PHONY: sql-fmt
sql-fmt:
	@echo "🎨 Formatting SQL"
	pg_format -i internal/sql/query.sql internal/sql/schema.sql

.PHONY: sqlc
sqlc: sql-fmt
	@echo "🗄️  Generating SQLC code..."
	sqlc generate

.PHONY: clean
clean:
	@echo "🧹 Cleaning $(BIN_DIR)"
	rm $(BIN_DIR)/*

.PHONY: dbmock
dbmock: sqlc
	@echo "🤖 Generating database mock..."
	mockgen -source=internal/database/querier.go \
		-destination internal/dbmock/dbmock.go \
		-package dbmock

.PHONY: fileservermock
fileservermock:
	@echo "🤖 Generating fileserver mock..."
	mockgen -source=internal/fileserver/fileserver.go \
		-destination internal/fileserver/fileservermock.go \
		-package fileserver

.PHONY: filestoremock
filestoremock:
	@echo "🤖 Generating filestore mock..."
	mockgen -source=internal/filestore/filestore.go \
		-destination internal/filestore/filestoremock.go \
		-package filestore

.PHONY: mocks
mocks: dbmock fileservermock filestoremock

.PHONY: test
test: mocks
	@echo "🧪 Running tests..."
	go test -v ./...

.PHONY: help
help:
	@cat Makefile
