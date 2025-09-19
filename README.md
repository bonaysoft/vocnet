# Rockd

A Go backend application built with Clean Architecture principles, featuring gRPC services, HTTP/JSON API, and PostgreSQL database.

## Features

- **Clean Architecture**: Well-structured codebase following Clean Architecture principles
- **gRPC & HTTP**: Dual API support with gRPC primary and HTTP/JSON via grpc-gateway
- **Type-safe Database**: Using sqlc for type-safe database operations
- **PostgreSQL**: Robust relational database with migrations
- **Protocol Buffers**: API-first design with protobuf definitions
- **Comprehensive Testing**: Unit tests with mocks and integration tests
- **Docker Support**: Containerized deployment with Docker and docker-compose
- **Development Tools**: Makefile with common development tasks

## Architecture

The project follows Clean Architecture with these layers:

```
├── cmd/                    # Application entry points
├── api/
│   ├── proto/             # Protocol Buffer definitions
│   ├── gen/               # Generated gRPC and gateway code
│   └── openapi/           # Auto-generated OpenAPI documentation
├── internal/
│   ├── entity/            # Business entities
│   ├── usecase/           # Business logic use cases
│   ├── adapter/
│   │   ├── grpc/          # gRPC service implementations
│   │   └── repository/    # Data access implementations
│   ├── infrastructure/
│   │   ├── database/      # Database connections
│   │   ├── config/        # Configuration management
│   │   └── server/        # Server setup
│   └── mocks/             # Generated mock files
├── sql/
│   ├── schema/            # Database schema files
│   ├── queries/           # SQL query files for sqlc
│   └── migrations/        # Database migration files
└── docs/                  # Project documentation
```

## Prerequisites

- Go 1.21 or later
- PostgreSQL 13+
- Protocol Buffers compiler (protoc)
- Docker and Docker Compose (optional)

## Quick Start

### 1. Setup Development Environment

```bash
# Clone the repository
git clone <repository-url>
cd rockd

# Install development tools and dependencies
make setup
```

### 2. Start Database

```bash
# Start PostgreSQL using Docker
make db-up

# Run migrations
make migrate-up
```

### 3. Generate Code

```bash
# Generate protobuf code
make generate

# Generate database code
make sqlc

# Generate mocks
make mocks
```

### 4. Run the Application

```bash
# Run in development mode
make run

# Or build and run binary
make build
./bin/rockd-server
```

The application will start with:
- gRPC server on port 9090
- HTTP gateway on port 8080

### 5. Test the API

```bash
# Create a user
curl -X POST http://localhost:8080/api/v1/users \
  -H "Content-Type: application/json" \
  -d '{"name": "John Doe", "email": "john@example.com"}'

# Get a user
curl http://localhost:8080/api/v1/users/1

# List users
curl http://localhost:8080/api/v1/users?page=1&per_page=10
```

## Development

### Available Make Commands

```bash
make help                 # Show all available commands
make setup               # Setup development environment
make build               # Build the application
make run                 # Run the application
make test                # Run tests
make test-coverage       # Generate test coverage report
make generate            # Generate protobuf code
make sqlc                # Generate database code
make mocks               # Generate mock files
make lint                # Run linter
make fmt                 # Format code
make clean               # Clean build artifacts
```

### Database Management

```bash
make db-up               # Start PostgreSQL database
make db-down             # Stop PostgreSQL database
make migrate-up          # Run migrations up
make migrate-down        # Run migrations down
make migrate-force       # Force migration version
```

### Docker Support

```bash
# Build Docker image
make docker-build

# Run with Docker Compose
docker-compose up

# Run with Docker Compose in background
docker-compose up -d
```

## Configuration

The application uses environment variables for configuration. See `.env` file for available options:

```env
# Server configuration
SERVER_HOST=localhost
GRPC_PORT=9090
HTTP_PORT=8080

# Database configuration
DB_HOST=localhost
DB_PORT=5432
DB_NAME=rockd
DB_USER=postgres
DB_PASSWORD=postgres
DB_SSLMODE=disable

# Logging configuration
LOG_LEVEL=info
LOG_FORMAT=json
```

## API Documentation

### gRPC

The gRPC API is defined in Protocol Buffer files located in `api/proto/`. The generated Go code is in `api/gen/`.

### HTTP/JSON

The HTTP API is automatically generated from gRPC definitions using grpc-gateway. OpenAPI documentation is generated in `api/openapi/`.

#### User Service Endpoints

- `POST /api/v1/users` - Create a user
- `GET /api/v1/users/{id}` - Get a user by ID
- `PUT /api/v1/users/{id}` - Update a user
- `DELETE /api/v1/users/{id}` - Delete a user
- `GET /api/v1/users` - List users with pagination

## Testing

### Unit Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test ./internal/usecase -v
```

### Integration Tests

Integration tests require a running PostgreSQL database:

```bash
# Start test database
make db-up

# Run integration tests
go test ./tests/integration -v
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for your changes
5. Ensure tests pass (`make test`)
6. Format code (`make fmt`)
7. Commit your changes (`git commit -am 'Add amazing feature'`)
8. Push to the branch (`git push origin feature/amazing-feature`)
9. Open a Pull Request

## Code Generation

This project uses several code generation tools:

- **protoc**: Generates gRPC and HTTP gateway code from `.proto` files
- **sqlc**: Generates type-safe Go code from SQL queries
- **mockgen**: Generates mock implementations for testing

Run `make generate sqlc mocks` to regenerate all code.

## Project Structure

The project follows Clean Architecture principles:

- **Entities**: Core business entities and rules (`internal/entity/`)
- **Use Cases**: Application business rules (`internal/usecase/`)
- **Interface Adapters**: gRPC services and repositories (`internal/adapter/`)
- **Frameworks & Drivers**: External frameworks and database (`internal/infrastructure/`)

## License

This project is licensed under the MIT License - see the LICENSE file for details.