.PHONY: all build clean test test-short test-integration test-coverage run swag lint fmt help docker-build docker-run load-test openapi-gen openapi-test build-lambda

# 変数定義
APP_NAME=stock-api
GO_FILES=$(shell find . -type f -name "*.go" -not -path "./vendor/*")
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"
ECR_REPO ?= $(shell aws ecr describe-repositories --query "repositories[?repositoryName=='$(APP_NAME)-lambda'].repositoryUri" --output text)

# デフォルトターゲット
all: clean fmt lint test build

# ビルド
build:
	@echo "Building $(APP_NAME)..."
	@go build $(LDFLAGS) -o $(APP_NAME) .

# Lambda用にビルド (Dockerfileで使用するため)
build-lambda:
    @echo "Building for AWS Lambda..."
    @GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o lambda-handler main.go handlers.go database.go
    @echo "Lambda handler built: lambda-handler"

# クリーンアップ
clean:
	@echo "Cleaning..."
	@rm -f $(APP_NAME) function.zip coverage.out
	@go clean

# テスト
test:
	@echo "Running all tests..."
	@go test -v ./...

# 短いテスト (統合テストをスキップ)
test-short:
	@echo "Running short tests..."
	@go test -v -short ./...

# データベース統合テスト
test-integration:
	@echo "Running integration tests..."
	@go test -v -tags=integration ./...

# swaggerテスト
test-swagger:
	@echo "Running swagger tests..."
	@go test -v -run TestSwagger ./...

# カバレッジレポート
test-coverage:
	@echo "Generating test coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report saved to coverage.html"

# アプリケーションの実行
run:
	@echo "Running $(APP_NAME)..."
	@go run $(LDFLAGS) .

# スワッガードキュメント生成
swag:
	@echo "Generating Swagger documentation..."
	@command -v swag >/dev/null 2>&1 || { echo "Installing swag..."; go install github.com/swaggo/swag/cmd/swag@latest; }
	@swag init -g main.go

# リント
lint:
	@echo "Linting code..."
	@command -v golangci-lint >/dev/null 2>&1 || { echo "Installing golangci-lint..."; go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; }
	@golangci-lint run

# コードフォーマット
fmt:
	@echo "Formatting code..."
	@gofmt -s -w $(GO_FILES)

# Docker関連
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):$(VERSION) .

docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env $(APP_NAME):$(VERSION)

# データベース環境のセットアップ
db-setup:
	@echo "Setting up database environment..."
	@docker-compose up -d db
	@echo "Waiting for database to start..."
	@sleep 5
	@cat schema.sql | docker-compose exec -T db mysql -uroot -proot stock_db

# データベース環境の停止
db-stop:
	@echo "Stopping database environment..."
	@docker-compose down

# 依存パッケージの更新
mod-update:
	@echo "Updating dependencies..."
	@go get -u ./...
	@go mod tidy

# テスト用データの投入
db-seed:
	@echo "Seeding test data..."
	@cat seed.sql | docker-compose exec -T db mysql -uroot -proot stock_db

# Postmanによる負荷テスト
load-test:
	@echo "Postmanによる負荷テストを実行しています..."
	@cd tests/postman && chmod +x run-load-test.sh && ./run-load-test.sh

# OpenAPIコード生成
openapi-gen:
	@echo "Generating code from OpenAPI schema..."
	oapi-codegen --package api --generate types api/types_gen.go swagger.yaml
	oapi-codegen --package api --generate server api/server_gen.go swagger.yaml
	oapi-codegen --package api --generate client api/client_gen.go swagger.yaml

# OpenAPIテスト実行
openapi-test:
	@echo "Running OpenAPI tests..."
	go test -v -tags=oapi ./...

# Lambda Dockerイメージ関連
docker-lambda-build:
    @echo "Building Lambda Docker image..."
    @docker build -t $(APP_NAME)-lambda:$(VERSION) .
    @echo "Lambda Docker image built: $(APP_NAME)-lambda:$(VERSION)"

docker-lambda-push:
    @echo "Pushing Lambda Docker image to ECR..."
    @aws ecr get-login-password --region ap-northeast-1 | docker login --username AWS --password-stdin $(ECR_REPO)
    @docker tag $(APP_NAME)-lambda:$(VERSION) $(ECR_REPO)/$(APP_NAME)-lambda:$(VERSION)
    @docker push $(ECR_REPO)/$(APP_NAME)-lambda:$(VERSION)
    @echo "Lambda Docker image pushed: $(ECR_REPO)/$(APP_NAME)-lambda:$(VERSION)"

docker-lambda-run:
    @echo "Running Lambda Docker container locally..."
    @docker run -p 9000:8080 \
        --env-file .env \
        $(APP_NAME)-lambda:$(VERSION)
    @echo "Lambda endpoint available at: http://localhost:9000/2015-03-31/functions/function/invocations"

# AWS SAM Localを使用したLambda実行（代替方法）
sam-local:
    @echo "Running Lambda with SAM Local..."
    @sam local start-api --host localhost --port 3000

# ヘルプ
help:
    @echo "Available commands:"
    @echo "  make build             - Build the application"
    @echo "  make build-lambda      - Build Lambda handler locally"
    @echo "  make docker-lambda-build - Build Lambda Docker container"
    @echo "  make docker-lambda-run   - Run Lambda container locally"
    @echo "  make docker-lambda-push  - Push Lambda container to ECR"
    @echo "  make sam-local         - Run Lambda with SAM Local"
    # 既存のhelpコマンドの内容...
