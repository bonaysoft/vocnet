# GitHub Copilot Instructions

This project is a Go backend application using Clean Architecture principles with PostgreSQL database, gRPC services, and HTTP/JSON API through grpc-gateway.

## Project Overview

- **Language**: Go (latest stable version)
- **Architecture**: Clean Architecture (Clean Arch)
- **Database**: PostgreSQL
- **API Style**: gRPC (primary) + RESTful HTTP/JSON (via grpc-gateway)
- **Interface Definition**: Protocol Buffers
- **Package Manager**: Go Modules

## Architecture Guidelines

### Clean Architecture Layers

Follow the Clean Architecture pattern with these layers:

1. **Entities** (`internal/entity/`): Business entities and core business rules
2. **Use Cases** (`internal/usecase/`): Application business rules and orchestration
3. **Interface Adapters** (`internal/adapter/`): gRPC services, HTTP handlers, and gateways
4. **Frameworks & Drivers** (`internal/infrastructure/`): External frameworks, databases, gRPC servers

### Directory Structure

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
│   │   ├── gateway/       # HTTP gateway handlers
│   │   └── repository/    # Data access implementations
│   ├── infrastructure/
│   │   ├── database/      # Database connections and migrations
│   │   ├── config/        # Configuration management
│   │   └── server/        # gRPC and HTTP server setup
│   └── pkg/               # Shared utilities
├── sql/
│   ├── schema/            # Database schema files
│   ├── queries/           # SQL query files for sqlc
│   └── migrations/        # Database migration files
└── docs/                  # Project documentation
```

## Coding Standards

### General Go Practices

- Follow Go naming conventions (camelCase for private, PascalCase for public)
- Use meaningful variable and function names
- Write self-documenting code with clear comments
- Implement proper error handling with wrapped errors
- Use context.Context for request scoping and cancellation
- Follow the principle of dependency injection

### Error Handling

```go
// Wrap errors with context
if err != nil {
    return fmt.Errorf("failed to create user: %w", err)
}

// Define custom error types for business logic
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed for %s: %s", e.Field, e.Message)
}
```

### Database Operations

- Use sqlc for type-safe database queries and code generation
- Implement proper transaction handling
- Use PostgreSQL-specific features when beneficial
- Include database migrations for schema changes
- Use connection pooling appropriately
- Write SQL queries in `.sql` files for sqlc generation

```go
// Example repository interface (sqlc generated)
type UserRepository interface {
    CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
    GetUser(ctx context.Context, id int64) (User, error)
    UpdateUser(ctx context.Context, arg UpdateUserParams) error
    DeleteUser(ctx context.Context, id int64) error
    ListUsers(ctx context.Context, arg ListUsersParams) ([]User, error)
}

// Example sqlc query file (queries/user.sql)
-- name: CreateUser :one
INSERT INTO users (name, email, created_at) 
VALUES ($1, $2, $3) 
RETURNING *;

-- name: GetUser :one
SELECT * FROM users 
WHERE id = $1 LIMIT 1;

-- name: UpdateUser :exec
UPDATE users 
SET name = $2, email = $3, updated_at = $4 
WHERE id = $1;

-- name: DeleteUser :exec
DELETE FROM users 
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users 
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
```

### RESTful API Design

- Use appropriate HTTP methods (GET, POST, PUT, DELETE, PATCH)
- Follow RESTful URL patterns (`/api/v1/users`, `/api/v1/users/{id}`)
- Return appropriate HTTP status codes
- Use consistent JSON response formats
- Implement proper pagination for list endpoints

```go
// Example response structure
type APIResponse struct {
    Success bool        `json:"success"`
    Data    interface{} `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
    Meta    *Meta       `json:"meta,omitempty"`
}

type APIError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
}

type Meta struct {
    Page       int `json:"page"`
    PerPage    int `json:"per_page"`
    Total      int `json:"total"`
    TotalPages int `json:"total_pages"`
}
```

### gRPC Service Design

- Define services and messages in Protocol Buffer files
- Use appropriate gRPC methods (Unary, Server streaming, Client streaming, Bidirectional)
- Follow protobuf naming conventions (PascalCase for messages, snake_case for fields)
- Include proper error handling with gRPC status codes
- Use grpc-gateway annotations for HTTP/JSON mapping

```protobuf
// Example proto definition
syntax = "proto3";

package user.v1;

import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "*"
    };
  }
  
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }
  
  rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {
    option (google.api.http) = {
      get: "/api/v1/users"
    };
  }
}

message User {
  int64 id = 1;
  string name = 2;
  string email = 3;
  google.protobuf.Timestamp created_at = 4;
  google.protobuf.Timestamp updated_at = 5;
}

message CreateUserRequest {
  string name = 1;
  string email = 2;
}

message CreateUserResponse {
  User user = 1;
}
```

## Dependencies and Libraries

### Recommended Libraries

- **gRPC**: google.golang.org/grpc
- **gRPC Gateway**: github.com/grpc-ecosystem/grpc-gateway/v2
- **Protocol Buffers**: google.golang.org/protobuf
- **Database**: lib/pq (PostgreSQL driver), sqlc (preferred for type-safe queries)
- **Configuration**: viper
- **Logging**: logrus or zap
- **Testing**: testify
- **Migration**: golang-migrate/migrate
- **Validation**: go-playground/validator (for HTTP), protoc-gen-validate (for protobuf)
- **Code Generation**: sqlc (for database queries), protoc (for gRPC/protobuf)
- **Mocking**: gomock (for interface mocking)

### Environment Configuration

- Use environment variables for configuration
- Provide sensible defaults
- Validate configuration on startup
- Support multiple environments (dev, staging, prod)

## sqlc Configuration and Usage

### sqlc Setup

Use sqlc for type-safe database operations. Create a `sqlc.yaml` configuration file:

```yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "sql/queries"
    schema: "sql/schema"
    gen:
      go:
        package: "db"
        out: "internal/infrastructure/database/db"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_db_tags: true
        emit_prepared_queries: false
        emit_interface: true
        emit_exact_table_names: false
        emit_empty_slices: true
```

### SQL Query Organization

- Place all SQL schema files in `sql/schema/`
- Place all SQL query files in `sql/queries/`
- Use descriptive filenames (e.g., `user.sql`, `order.sql`)
- Group related queries in the same file

### Query Naming Conventions

```sql
-- name: CreateUser :one
-- name: GetUser :one  
-- name: ListUsers :many
-- name: UpdateUser :exec
-- name: DeleteUser :exec
-- name: CountUsers :one
```

### Query Types

- `:one` - Returns a single row
- `:many` - Returns multiple rows  
- `:exec` - Execute query without return
- `:execrows` - Execute and return affected rows count
- `:execlastid` - Execute and return last insert ID

### Transaction Handling with sqlc

```go
// Use generated Queries struct with transaction
func (r *userRepository) CreateUserWithProfile(ctx context.Context, userParam CreateUserParams, profileParam CreateProfileParams) error {
    return r.db.WithTx(ctx, func(q *db.Queries) error {
        user, err := q.CreateUser(ctx, userParam)
        if err != nil {
            return err
        }
        
        profileParam.UserID = user.ID
        _, err = q.CreateProfile(ctx, profileParam)
        return err
    })
}
```

## gRPC and Protocol Buffers Configuration

### Protocol Buffer Setup

Use Protocol Buffers for service definitions. Create proto files in `api/proto/`:

```protobuf
// api/proto/user/v1/user.proto
syntax = "proto3";

package user.v1;

import "google/api/annotations.proto";
import "google/protobuf/timestamp.proto";
import "validate/validate.proto";

option go_package = "github.com/yourorg/yourproject/api/gen/user/v1;userv1";

service UserService {
  rpc CreateUser(CreateUserRequest) returns (CreateUserResponse) {
    option (google.api.http) = {
      post: "/api/v1/users"
      body: "*"
    };
  }
  
  rpc GetUser(GetUserRequest) returns (GetUserResponse) {
    option (google.api.http) = {
      get: "/api/v1/users/{id}"
    };
  }
}

message User {
  int64 id = 1;
  string name = 2 [(validate.rules).string.min_len = 1];
  string email = 3 [(validate.rules).string.email = true];
  google.protobuf.Timestamp created_at = 4;
  google.protobuf.Timestamp updated_at = 5;
}
```

### Code Generation

Use a Makefile or script for code generation:

```makefile
# Makefile
.PHONY: generate
generate:
	@echo "Generating protobuf files..."
	protoc --proto_path=api/proto \
		--go_out=api/gen --go_opt=paths=source_relative \
		--go-grpc_out=api/gen --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=api/gen --grpc-gateway_opt=paths=source_relative \
		--openapiv2_out=api/openapi --openapiv2_opt=logtostderr=true \
		--validate_out="lang=go:api/gen" --validate_opt=paths=source_relative \
		api/proto/**/*.proto

.PHONY: sqlc
sqlc:
	@echo "Generating sqlc files..."
	sqlc generate
```

### gRPC Service Implementation

Implement gRPC services in the adapter layer:

```go
// internal/adapter/grpc/user_service.go
type UserService struct {
	userv1.UnimplementedUserServiceServer
	userUsecase usecase.UserUsecase
}

func NewUserService(userUsecase usecase.UserUsecase) *UserService {
	return &UserService{
		userUsecase: userUsecase,
	}
}

func (s *UserService) CreateUser(ctx context.Context, req *userv1.CreateUserRequest) (*userv1.CreateUserResponse, error) {
	// Validate request
	if err := req.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	
	// Call use case
	user, err := s.userUsecase.CreateUser(ctx, &entity.User{
		Name:  req.Name,
		Email: req.Email,
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	
	// Convert to protobuf
	return &userv1.CreateUserResponse{
		User: s.toProtoUser(user),
	}, nil
}
```

### Server Setup

Configure both gRPC and HTTP servers:

```go
// internal/infrastructure/server/server.go
func NewServer(cfg *config.Config, userService *grpc.UserService) *Server {
	// Create gRPC server
	grpcServer := grpc.NewServer()
	userv1.RegisterUserServiceServer(grpcServer, userService)
	
	// Create gRPC-Gateway mux
	ctx := context.Background()
	mux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	
	err := userv1.RegisterUserServiceHandlerFromEndpoint(ctx, mux, "localhost:9090", opts)
	if err != nil {
		log.Fatal(err)
	}
	
	return &Server{
		grpcServer: grpcServer,
		httpServer: &http.Server{
			Handler: mux,
		},
	}
}
```

## Mock Configuration and Testing

### GoMock Setup

Use GoMock for generating mocks from interfaces. Install GoMock:

```bash
go install github.com/golang/mock/mockgen@latest
```

### Mock Generation

Add `//go:generate` comments to generate mocks:

```go
// internal/usecase/repository/user.go
//go:generate mockgen -source=user.go -destination=../../mocks/user_repository_mock.go -package=mocks

type UserRepository interface {
    CreateUser(ctx context.Context, arg CreateUserParams) (User, error)
    GetUser(ctx context.Context, id int64) (User, error)
    UpdateUser(ctx context.Context, arg UpdateUserParams) error
    DeleteUser(ctx context.Context, id int64) error
    ListUsers(ctx context.Context, arg ListUsersParams) ([]User, error)
}

// internal/usecase/user.go  
//go:generate mockgen -source=user.go -destination=../mocks/user_usecase_mock.go -package=mocks

type UserUsecase interface {
    CreateUser(ctx context.Context, user *entity.User) (*entity.User, error)
    GetUser(ctx context.Context, id int64) (*entity.User, error)
    UpdateUser(ctx context.Context, user *entity.User) error
    DeleteUser(ctx context.Context, id int64) error
    ListUsers(ctx context.Context, limit, offset int) ([]*entity.User, error)
}
```

### Mock Directory Structure

Organize mocks in a dedicated directory:

```
├── internal/
│   ├── mocks/                 # Generated mock files
│   │   ├── user_repository_mock.go
│   │   ├── user_usecase_mock.go
│   │   └── external_service_mock.go
│   ├── usecase/
│   ├── adapter/
│   └── entity/
```

### Mock Usage in Tests

#### Unit Test Example (Use Case Layer)

```go
// internal/usecase/user_test.go
func TestUserUsecase_CreateUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockUserRepository(ctrl)
    userUsecase := NewUserUsecase(mockRepo)

    tests := []struct {
        name    string
        input   *entity.User
        setup   func()
        wantErr bool
    }{
        {
            name:  "successful creation",
            input: &entity.User{Name: "John", Email: "john@example.com"},
            setup: func() {
                mockRepo.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Return(User{ID: 1, Name: "John", Email: "john@example.com"}, nil).
                    Times(1)
            },
            wantErr: false,
        },
        {
            name:  "repository error",
            input: &entity.User{Name: "John", Email: "john@example.com"},
            setup: func() {
                mockRepo.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Return(User{}, errors.New("database error")).
                    Times(1)
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            
            result, err := userUsecase.CreateUser(context.Background(), tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.Equal(t, "John", result.Name)
            }
        })
    }
}
```

#### gRPC Service Test Example

```go
// internal/adapter/grpc/user_service_test.go
func TestUserService_CreateUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockUsecase := mocks.NewMockUserUsecase(ctrl)
    userService := NewUserService(mockUsecase)

    tests := []struct {
        name    string
        request *userv1.CreateUserRequest
        setup   func()
        wantErr bool
    }{
        {
            name: "successful creation",
            request: &userv1.CreateUserRequest{
                Name:  "John",
                Email: "john@example.com",
            },
            setup: func() {
                mockUsecase.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Return(&entity.User{
                        ID:    1,
                        Name:  "John",
                        Email: "john@example.com",
                    }, nil).
                    Times(1)
            },
            wantErr: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            
            response, err := userService.CreateUser(context.Background(), tt.request)
            
            if tt.wantErr {
                assert.Error(t, err)
                assert.Nil(t, response)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, response)
                assert.Equal(t, "John", response.User.Name)
            }
        })
    }
}
```

### Mock Generation in Makefile

Update your Makefile to include mock generation:

```makefile
.PHONY: mocks
mocks:
	@echo "Generating mocks..."
	go generate ./...

.PHONY: test
test: mocks
	@echo "Running tests..."
	go test -v -race -coverprofile=coverage.out ./...

.PHONY: test-coverage
test-coverage: test
	@echo "Generating coverage report..."
	go tool cover -html=coverage.out -o coverage.html
```

### Mock Best Practices

1. **Generate mocks for all interfaces** - Especially repository and use case interfaces
2. **Use gomock.Any()** for flexible argument matching
3. **Set up expectations clearly** - Define exact call counts and return values
4. **Clean up controllers** - Always call `defer ctrl.Finish()`
5. **Test both success and error scenarios** - Cover all code paths
6. **Use table-driven tests** - Organize multiple test cases efficiently
7. **Mock external dependencies** - Database, HTTP clients, external services

## Testing Guidelines

### Unit Tests

- Write tests for all business logic
- Use table-driven tests when appropriate
- Mock external dependencies using GoMock
- Aim for high test coverage (>80%)
- Test both success and error scenarios

```go
func TestUserUseCase_CreateUser(t *testing.T) {
    ctrl := gomock.NewController(t)
    defer ctrl.Finish()

    mockRepo := mocks.NewMockUserRepository(ctrl)
    userUsecase := NewUserUsecase(mockRepo)

    tests := []struct {
        name    string
        input   *entity.User
        setup   func()
        wantErr bool
    }{
        {
            name:    "valid user",
            input:   &entity.User{Email: "test@example.com"},
            setup: func() {
                mockRepo.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Return(User{ID: 1}, nil).
                    Times(1)
            },
            wantErr: false,
        },
        {
            name:    "repository error",
            input:   &entity.User{Email: "test@example.com"},
            setup: func() {
                mockRepo.EXPECT().
                    CreateUser(gomock.Any(), gomock.Any()).
                    Return(User{}, errors.New("db error")).
                    Times(1)
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            tt.setup()
            result, err := userUsecase.CreateUser(context.Background(), tt.input)
            
            if tt.wantErr {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

### Integration Tests

- Test database interactions with real database
- Test gRPC endpoints end-to-end
- Use test containers for PostgreSQL when possible
- Mock external services only
- Clean up test data after each test

### Testing Strategy

- **Unit Tests**: Mock all external dependencies (repository, external services)
- **Integration Tests**: Use real database, mock external services
- **End-to-End Tests**: Test complete workflows with minimal mocking
- **gRPC Tests**: Test service layer with mocked use cases

## Security Best Practices

- Validate all input data
- Use parameterized queries to prevent SQL injection
- Implement proper authentication and authorization
- Hash passwords using bcrypt
- Use HTTPS in production
- Implement rate limiting
- Log security-relevant events

## Performance Considerations

- Use database indexes appropriately
- Implement caching where beneficial
- Use connection pooling
- Profile and monitor application performance
- Implement graceful shutdown
- Use context for request timeouts

## Documentation

- Write clear function and package documentation
- Include examples in documentation
- Maintain API documentation (OpenAPI/Swagger)
- Update README with setup and usage instructions

## When Writing Code

1. **Always** follow the Clean Architecture principles
2. **Always** include proper error handling
3. **Always** write tests for new functionality
4. **Always** validate input data
5. **Consider** performance implications
6. **Consider** security implications
7. **Prefer** explicit over implicit
8. **Prefer** composition over inheritance
9. **Use** dependency injection for better testability
10. **Keep** functions small and focused

## Example Code Structure

When creating new features, follow this pattern:

1. Define Protocol Buffer messages and services in `api/proto/`
2. Generate Go code using protoc and related plugins
3. Define entities in `internal/entity/`
4. Create repository interfaces in the use case layer
5. Implement use cases in `internal/usecase/`
6. Implement repository in `internal/adapter/repository/`
7. Create gRPC service implementations in `internal/adapter/grpc/`
8. Set up server and wire dependencies in `internal/infrastructure/server/`

Remember to maintain the dependency rule: inner layers should not depend on outer layers.
