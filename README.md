# SSO Service

gRPC microservice for authentication and single sign-on. Part of the Messenger microservice ecosystem.

## Overview

Handles user registration and login with bcrypt password hashing and JWT token generation. Assigns default "user" role via the Permissions Service on registration.

- **Port:** 44044
- **Language:** Go
- **Database:** PostgreSQL
- **Protocol:** gRPC (protobuf)

## Architecture

```
cmd/
  sso/main.go           Entrypoint — loads config, wires dependencies, starts gRPC server
  migrator/main.go      Runs database migrations
internal/
  config/               YAML config via cleanenv
  app/                  Dependency wiring and gRPC server setup
  grpc/auth/            gRPC handler implementations
  services/auth/        Business logic (register, login with bcrypt + JWT)
  storage/postgresql/   PostgreSQL queries (pgxpool)
  domain/models/        Domain structs (User, App)
  clients/permissions/  gRPC client to Permissions Service
  lib/jwt/              JWT token creation (HS256)
  lib/logger/sl/        slog error helper
proto/sso/              Protobuf definitions
migrations/             SQL migration files
tests/                  Integration tests
```

## API

### Auth

| Method | Request | Response | Description |
|---|---|---|---|
| `Register` | `RegisterRequest` | `RegisterResponse` | Register a new user (assigns default "user" role) |
| `Login` | `LoginRequest` | `LoginResponse` | Authenticate and receive a JWT token |

### JWT Token Claims

```json
{
  "uid": 1,
  "email": "user@example.com",
  "exp": 1740000000,
  "app_id": 1
}
```

Tokens are signed with HS256 using the app's secret key.

## Configuration

Config loaded via `--config` flag, `CONFIG_PATH` env var, or default path. See `config/config_local.example.yaml`.

```yaml
env: "local"
storage_path: "postgres://postgres:password@localhost:5432/auth"
permissions_address: "localhost:50052"
grpc:
  port: 44044
  timeout: 10h
```

## Getting Started

### Prerequisites

- Go
- PostgreSQL

### Setup

```bash
# 1. Create the database
createdb auth

# 2. Seed a test app in the apps table:
#    INSERT INTO apps (id, name, secret) VALUES (1, 'myapp', 'your-secret-key');

# 3. Run migrations
go run cmd/migrator/main.go

# 4. Start the service
go run cmd/sso/main.go
```

### Testing

```bash
go test ./... -v
```

Tests require a running server at `localhost:44044`. The test suite seeds a test app (`id=1, name=test, secret=test-secret`) via its own migration.

## Makefile

| Target | Description |
|---|---|
| `proto-gen` | Generate Go code from proto files |
| `proto-clean` | Remove generated .pb.go files |
| `proto-tools` | Install protoc-gen-go and protoc-gen-go-grpc |
