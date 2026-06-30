# Cermin Backend Architecture

This document explains how the Cermin backend is structured, how each folder is used, and how requests move through router, handler, service, repository, model, and database layers.

## Overview

Cermin Backend is a Go HTTP API built with:

- Gin for HTTP routing and request/response handling.
- GORM for PostgreSQL database access.
- godotenv and environment variables for configuration.
- bcrypt for password hashing.
- HMAC SHA-256 JWT tokens for authentication.
- Scalar/OpenAPI for API documentation.

The project follows a simple layered architecture:

```text
HTTP client
  -> Gin router
  -> Handler
  -> Service
  -> Repository
  -> GORM model
  -> PostgreSQL database
```

Each layer has a different responsibility:

- Router decides which URL maps to which handler.
- Handler handles HTTP details such as JSON binding, query params, path params, and HTTP status codes.
- Service contains business rules such as duplicate email checks, password hashing, login validation, OAuth login behavior, and response shaping.
- Repository contains database operations and hides GORM queries behind interfaces.
- Model defines database entities and response DTOs.
- Database package opens the PostgreSQL connection and runs GORM auto migration.

## Folder Structure

```text
.
|-- cmd/
|   `-- api/
|       `-- main.go
|-- internal/
|   |-- admin/
|   |   `-- user_handler.go
|   |-- auth/
|   |   |-- apple.go
|   |   |-- google.go
|   |   |-- handler.go
|   |   |-- middleware.go
|   |   `-- service.go
|   |-- config/
|   |   `-- config.go
|   |-- daily-summary/
|   |-- database/
|   |   `-- postgres.go
|   |-- docs/
|   |   |-- docs.go
|   |   `-- openapi.json
|   |-- journal/
|   |   |-- model.go
|   |   `-- repository.go
|   |-- router/
|   |   |-- docs.go
|   |   `-- router.go
|   `-- user/
|       |-- model.go
|       |-- repository.go
|       `-- service.go
|-- migrations/
|-- tests/
|   `-- api/
|-- Dockerfile
|-- docker-compose.yml
|-- Makefile
|-- go.mod
`-- go.sum
```

## Folder Responsibilities

| Folder/File | Function |
| --- | --- |
| `cmd/api/main.go` | Application entrypoint. Loads config, connects database, runs auto migration, builds router, and starts the HTTP server. |
| `internal/` | Private application code. Go prevents external modules from importing packages inside `internal`. |
| `internal/router/` | Central route registration and dependency wiring. It creates repositories, services, and handlers, then maps endpoints to handler methods. |
| `internal/router/docs.go` | Registers `/openapi.json` and `/docs` routes. |
| `internal/admin/` | Admin-facing HTTP handlers. Currently handles admin user CRUD endpoints. |
| `internal/auth/` | Authentication domain. Handles register, login, Google OAuth, Apple OAuth, JWT creation/parsing, and auth middleware. |
| `internal/config/` | Reads environment variables, applies defaults, and builds the PostgreSQL database URL. |
| `internal/database/` | Opens and validates the PostgreSQL connection. Runs GORM `AutoMigrate` for active models. |
| `internal/docs/` | Generated or embedded OpenAPI documentation used by docs routes. |
| `internal/user/` | User domain. Defines user model, response DTOs, repository interface/implementation, and user service logic. |
| `internal/journal/` | Journal domain models and repository interface. The models are included in database auto migration, but HTTP routes/services are not wired yet in the current router. |
| `internal/daily-summary/` | Reserved domain folder. It currently has no source files in this working tree. |
| `migrations/` | SQL migration files for schema changes. Used by the `migrate` CLI through Makefile targets. |
| `tests/api/` | API/integration-style tests for routes such as ping, docs, auth, and admin users. |
| `Makefile` | Common developer commands for dev server, tests, and database migrations. |
| `Dockerfile` | Container build definition for the backend. |
| `docker-compose.yml` | Local service orchestration, typically for app/database development. |
| `auth-flow.md` | Existing detailed documentation for authentication flows. |
| `erd.md` | Existing database/entity relationship documentation. |
| `migrations.md` | Existing database migration documentation. |

The root `api` file is a compiled Mach-O executable, not a source folder.

## Application Startup Flow

Startup begins in `cmd/api/main.go`.

```text
main()
  -> config.Load()
  -> database.Connect(cfg.DatabaseURL)
  -> database.AutoMigrate(db)
  -> router.Setup(db, cfg)
  -> r.Run(":" + cfg.AppPort)
```

Step by step:

1. `config.Load()` reads `.env` if present, then reads environment variables.
2. `config.Load()` builds `DatabaseURL` from `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME`, and `DB_SSLMODE`.
3. `database.Connect()` opens the PostgreSQL connection with GORM and pings the database.
4. `database.AutoMigrate()` syncs active GORM models to the database.
5. `router.Setup()` creates the Gin engine, registers docs routes, wires dependencies, and registers API routes.
6. The server listens on `APP_PORT`, defaulting to `8080`.

## Router Layer

The router is defined in `internal/router/router.go`.

The router has two main jobs:

1. Create the HTTP route tree.
2. Wire dependencies between repositories, services, and handlers.

Current dependency wiring:

```text
userRepository := user.NewRepository(db)
userService    := user.NewService(userRepository)
userHandler    := admin.NewUserHandler(userService)

authService := auth.NewService(userRepository, cfg.JWTSecret)
authHandler := auth.NewHandler(authService, googleOAuthConfig, appleOAuthConfig)
```

Important point: both `userService` and `authService` share the same `userRepository`. This means authentication and admin user management use the same user table and repository behavior.

Current route groups:

```text
GET  /openapi.json
GET  /docs

/api/v1
  GET  /ping

  /auth
    POST /register
    POST /login
    GET  /google
    GET  /google/callback
    GET  /apple
    GET  /apple/callback
    POST /apple/callback

  /admin/users
    POST   /
    GET    /
    GET    /:id
    PATCH  /:id
    DELETE /:id
```

## Handler Layer

Handlers are responsible for HTTP-specific work.

They should:

- Read JSON bodies with `c.ShouldBindJSON`.
- Read query parameters with `c.Query`.
- Read path parameters with `c.Param`.
- Validate request shape using Gin binding tags.
- Convert request data into service input structs.
- Convert service errors into HTTP status codes.
- Return JSON responses.

They should not:

- Build SQL queries.
- Know GORM details.
- Contain password hashing rules.
- Contain complex business decisions.

Examples:

- `internal/auth/handler.go` handles auth HTTP endpoints.
- `internal/admin/user_handler.go` handles admin user CRUD HTTP endpoints.

For example, register works like this:

```text
POST /api/v1/auth/register
  -> auth.Handler.Register
  -> bind JSON request
  -> call auth.Service.Register
  -> map ErrEmailAlreadyUsed to 409 Conflict
  -> map unexpected error to 500 Internal Server Error
  -> return 201 Created with token and public user
```

## Service Layer

Services contain application business rules. They sit between handlers and repositories.

### Auth Service

Defined in `internal/auth/service.go`.

Responsibilities:

- Register local users.
- Check duplicate email before creating a user.
- Hash passwords with bcrypt.
- Validate login credentials.
- Create JWT tokens.
- Parse and validate JWT tokens.
- Login or create users from Google OAuth data.
- Login or create users from Apple OAuth data.
- Prevent OAuth user creation when the email is already used by another auth provider.

Important auth errors:

- `ErrEmailAlreadyUsed`
- `ErrInvalidCredentials`
- `ErrInvalidToken`
- `ErrExpiredToken`

### User Service

Defined in `internal/user/service.go`.

Responsibilities:

- Create users from admin endpoints.
- List users with pagination.
- Get a user by ID.
- Update user data.
- Delete users.
- Check duplicate emails on create/update.
- Hash admin-created or admin-updated passwords.
- Convert internal `User` models into `AdminUser` response DTOs.

Important user errors:

- `ErrEmailAlreadyUsed`
- `ErrUserNotFound`

## Repository Layer

Repositories isolate database access.

The main repository pattern is in `internal/user/repository.go`.

```go
type Repository interface {
	Create(ctx context.Context, request CreateUserRequest) (*User, error)
	List(ctx context.Context, request ListUsersRequest) ([]User, int64, error)
	FindByID(ctx context.Context, id int64) (*User, error)
	FindByEmail(ctx context.Context, email string) (*User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*User, error)
	FindByAppleID(ctx context.Context, appleID string) (*User, error)
	Update(ctx context.Context, id int64, request UpdateUserRequest) (*User, error)
	Delete(ctx context.Context, id int64) error
}
```

The concrete implementation is `GormRepository`.

Responsibilities:

- Convert service requests into GORM operations.
- Use `WithContext(ctx)` so request context reaches database calls.
- Convert `gorm.ErrRecordNotFound` into domain error `ErrUserNotFound`.
- Apply database query behavior such as search, sorting, pagination, insert, update, and delete.

This pattern makes services easier to test because services depend on an interface, not directly on GORM.

## Model Layer

Models define the data shape used by GORM and API responses.

### User Models

Defined in `internal/user/model.go`.

- `User` is the database model.
- `PublicUser` is the auth-facing response shape.
- `AdminUser` is the admin-facing response shape.
- `ToPublicUser()` hides sensitive fields such as password hash.
- `ToAdminUser()` exposes admin-safe fields such as timestamps.

The `User` model maps to the users table and includes:

- `ID`
- `Name`
- `Email`
- `PasswordHash`
- `AuthProvider`
- `GoogleID`
- `AppleID`
- `CreatedAt`
- `UpdatedAt`

### Journal Models

Defined in `internal/journal/model.go`.

The journal domain currently defines persistence models for:

- `JournalEntry`
- `JournalReflection`
- `ReflectionSummary`
- `ReflectionHiddenLanguage`
- `ReflectionEmotionScore`

These models describe journal entries, AI or reflection output, hidden language notes, and emotion scores. They are included in `database.AutoMigrate()`, so their database tables can be created by GORM. Current HTTP routing does not expose journal endpoints yet.

## Authentication Middleware

Defined in `internal/auth/middleware.go`.

`RequireAuth(service)` returns a Gin middleware that:

1. Reads the `Authorization` header.
2. Expects the format `Bearer <token>`.
3. Calls `service.ParseJWT(token)`.
4. Rejects missing, invalid, or expired tokens with `401 Unauthorized`.
5. Stores authenticated user data in Gin context:
   - `auth_user_id`
   - `auth_email`
6. Calls `c.Next()` to continue the request.

Helper functions:

- `CurrentUserID(c)` returns the authenticated user ID if present.
- `MustCurrentUserID(c)` returns the authenticated user ID or panics if middleware was not applied correctly.

The middleware exists, but the current router does not yet apply it to route groups.

## Database Layer

Defined in `internal/database/postgres.go`.

Responsibilities:

- Open a PostgreSQL connection using GORM.
- Validate the connection with `Ping()`.
- Run `AutoMigrate()` for active models.

Current auto-migrated models:

- `user.User`
- `journal.JournalEntry`
- `journal.JournalReflection`
- `journal.ReflectionSummary`
- `journal.ReflectionHiddenLanguage`
- `journal.ReflectionEmotionScore`

The project also has SQL migrations in `migrations/`. The Makefile supports migration commands:

- `make migrate-create name=...`
- `make migrate-up`
- `make migrate-down`
- `make migrate-force version=...`
- `make migrate-version`

Because both SQL migrations and GORM auto migration exist, schema changes should be handled carefully. Prefer one source of truth for production schema changes, usually SQL migrations, and keep GORM models aligned with those migrations.

## Request Flow Examples

### Local Register

```text
POST /api/v1/auth/register
  -> router sends request to authHandler.Register
  -> handler validates JSON body
  -> service checks if email exists
  -> service hashes password
  -> repository creates user row
  -> service creates JWT
  -> handler returns 201 with token and public user
```

### Local Login

```text
POST /api/v1/auth/login
  -> router sends request to authHandler.Login
  -> handler validates JSON body
  -> service finds user by email
  -> service compares bcrypt password hash
  -> service creates JWT
  -> handler returns 200 with token and public user
```

### Google OAuth Login

```text
GET /api/v1/auth/google
  -> redirects client to Google OAuth URL

GET /api/v1/auth/google/callback
  -> validates state
  -> exchanges code for access token
  -> fetches Google user info
  -> service finds or creates user by google_id
  -> service creates JWT
  -> returns token and public user
```

### Apple OAuth Login

```text
GET /api/v1/auth/apple
  -> redirects client to Apple OAuth URL

GET or POST /api/v1/auth/apple/callback
  -> validates state when configured
  -> exchanges code for Apple token response
  -> parses Apple user info
  -> service finds or creates user by apple_id
  -> service creates JWT
  -> returns token and public user
```

### Admin List Users

```text
GET /api/v1/admin/users?page=1&per_page=10&search=...
  -> router sends request to userHandler.List
  -> handler parses query params
  -> service normalizes pagination
  -> repository queries users with optional search
  -> service converts models to AdminUser DTOs
  -> handler returns paginated JSON
```

## How to Add a New Feature

Use the existing layer pattern.

For a new domain such as journals:

1. Add or update models in `internal/journal/model.go`.
2. Add repository methods in `internal/journal/repository.go`.
3. Add a concrete GORM repository implementation if it does not exist yet.
4. Add service methods for business rules.
5. Add handler methods for HTTP input/output.
6. Register routes in `internal/router/router.go`.
7. Add or update SQL migrations in `migrations/`.
8. Add API tests in `tests/api/`.
9. Update `internal/docs/openapi.json` if API docs are maintained manually.

Recommended package shape for a full domain:

```text
internal/example/
  model.go
  repository.go
  service.go
  handler.go
```

## Naming Rules

Use names that describe the layer and the action. Keep names boring and predictable.

### Package Names

- Use singular domain package names: `user`, `auth`, `journal`.
- Use short lowercase package names.
- Do not use names like `users`, `userService`, or `user_repository` for packages.
- Keep one domain per package when possible.

Good:

```text
internal/user
internal/journal
internal/auth
```

Avoid:

```text
internal/users
internal/user_service
internal/repositories
```

### Repository Interface Names

Inside a domain package, name the main repository interface `Repository`.

```go
type Repository interface {
	Create(ctx context.Context, request CreateUserRequest) (*User, error)
	FindByID(ctx context.Context, id int64) (*User, error)
}
```

Because the interface lives inside `internal/user`, other packages read it as `user.Repository`. This is clear without repeating the domain name in the type.

Use a more specific name only when one package has multiple repository roles:

```go
type TokenRepository interface {
	CreateRefreshToken(ctx context.Context, request CreateRefreshTokenRequest) error
}
```

Concrete repository implementations should include their storage technology:

```go
type GormRepository struct {
	db *gorm.DB
}
```

Use constructor names like:

```go
func NewRepository(db *gorm.DB) *GormRepository
```

### Service Names

Inside a domain package, name the main service `Service`.

```go
type Service struct {
	users Repository
}

func NewService(users Repository) *Service
```

Other packages use it as `user.Service` or `auth.Service`, so the package name provides the domain context.

### Handler Names

Handlers should be named by the resource or domain they handle.

Examples:

```go
type Handler struct {
	service *Service
}
```

This is good inside `internal/auth` because other packages read it as `auth.Handler`.

```go
type UserHandler struct {
	service *user.Service
}
```

This is good inside `internal/admin` because the admin package may later contain multiple handlers.

### Request Type Names

Use different request/input types for each layer.

Handler request structs should be unexported because they are only used by one handler file:

```go
type createUserRequest struct {
	Name     string `json:"name" binding:"required,max=100"`
	Email    string `json:"email" binding:"required,email,max=150"`
	Password string `json:"password" binding:"required,min=8"`
}
```

Service input structs should be exported when handlers from another package need to use them:

```go
type CreateAdminUserInput struct {
	Name     string
	Email    string
	Password string
}
```

Repository request structs should describe database operation input:

```go
type CreateUserRequest struct {
	Name         string
	Email        string
	PasswordHash *string
	AuthProvider string
}
```

Recommended naming pattern:

| Layer | Type Pattern | Example |
| --- | --- | --- |
| Handler | `<action><resource>Request` | `createUserRequest` |
| Service | `<Action><Resource>Input` | `CreateAdminUserInput` |
| Repository | `<Action><Model>Request` | `CreateUserRequest` |
| List service result | `List<Resource>Result` | `ListAdminUsersResult` |
| Public response DTO | `<Purpose><Resource>` | `PublicUser`, `AdminUser` |

Do not reuse the same struct for HTTP request, service input, and repository request. Each layer has different validation and security needs.

### Response Type Names

Use response DTOs to control what leaves the API.

Current examples:

```go
type PublicUser struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Email        string `json:"email"`
	AuthProvider string `json:"auth_provider"`
}

type AdminUser struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Email        string    `json:"email"`
	AuthProvider string    `json:"auth_provider"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
```

Use conversion functions to keep response shaping consistent:

```go
func ToPublicUser(user *User) PublicUser
func ToAdminUser(user *User) AdminUser
```

Never return database models directly when the model contains sensitive fields, internal fields, foreign-key relations, or fields the client should not depend on.

## Custom Response Types

Create custom response types when the API response is more than a single model.

Examples:

```go
type AuthResult struct {
	Token string          `json:"token"`
	User  user.PublicUser `json:"user"`
}
```

```go
type ListAdminUsersResult struct {
	Data    []AdminUser `json:"data"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	PerPage int         `json:"per_page"`
}
```

Use this pattern for paginated responses:

```go
type ListExampleResult struct {
	Data    []ExampleResponse `json:"data"`
	Total   int64             `json:"total"`
	Page    int               `json:"page"`
	PerPage int               `json:"per_page"`
}
```

Use this pattern for create/update responses:

```go
type ExampleResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
```

Use this pattern for simple success responses only when there is no useful resource to return:

```go
c.JSON(http.StatusOK, gin.H{"message": "success"})
```

Prefer typed response structs over repeated `gin.H` for normal business responses. `gin.H` is fine for small error responses and simple messages.

## HTTP Response Rules

Handlers own HTTP status codes. Services should return domain errors, not HTTP responses.

Recommended status codes:

| Case | Status |
| --- | --- |
| Create success | `201 Created` |
| Read/list success | `200 OK` |
| Update success | `200 OK` |
| Delete success with no body | `204 No Content` |
| Invalid JSON/body/query/path param | `400 Bad Request` |
| Missing or invalid auth token | `401 Unauthorized` |
| Authenticated but not allowed | `403 Forbidden` |
| Resource not found | `404 Not Found` |
| Duplicate unique resource, such as email | `409 Conflict` |
| Upstream OAuth/provider failure | `502 Bad Gateway` |
| Unexpected internal error | `500 Internal Server Error` |

Current error response shape:

```go
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
```

Keep this shape consistent unless the project decides to introduce a shared error response type.

If a shared response type is introduced later, use a small helper in a shared package such as `internal/response`:

```go
type ErrorResponse struct {
	Error string `json:"error"`
}

func Error(c *gin.Context, status int, err error) {
	c.JSON(status, ErrorResponse{Error: err.Error()})
}
```

Then handlers can use:

```go
response.Error(c, http.StatusConflict, err)
```

Do not put HTTP status code logic inside repositories. Avoid putting it inside services unless the service is explicitly an HTTP-facing service, which this project does not use.

## Error Handling Rules

Define domain errors close to the domain that owns them.

Examples:

```go
var ErrUserNotFound = errors.New("user not found")
```

```go
var (
	ErrEmailAlreadyUsed   = errors.New("email already used")
	ErrInvalidCredentials = errors.New("invalid credentials")
)
```

Use `errors.Is` in handlers and services:

```go
if errors.Is(err, user.ErrUserNotFound) {
	c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	return
}
```

Wrap infrastructure errors with context:

```go
return nil, fmt.Errorf("failed to open database: %w", err)
```

Do not expose internal infrastructure details to clients unless they are useful and safe. For unexpected errors, returning `err.Error()` is acceptable in local development, but production APIs often replace it with a generic message and log the real error server-side.

Recommended handler structure:

```go
result, err := h.service.DoSomething(c.Request.Context(), input)
if errors.Is(err, domain.ErrNotFound) {
	c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	return
}
if errors.Is(err, domain.ErrAlreadyExists) {
	c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
	return
}
if err != nil {
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
	return
}

c.JSON(http.StatusOK, result)
```

Always return immediately after writing an error response.

## Clean Code Best Practices

### Keep Layer Boundaries Clear

- Handler: HTTP parsing, validation, status codes, JSON response.
- Service: business rules, orchestration, domain decisions.
- Repository: database queries and persistence details.
- Model: GORM schema and API-safe DTO conversion.
- Config/database/router: infrastructure wiring.

If a function starts doing work from multiple layers, split it.

### Keep Functions Small

A handler should usually:

1. Parse input.
2. Call service.
3. Map errors.
4. Return response.

A service method should usually:

1. Validate business conditions.
2. Call repositories or external providers.
3. Transform data into response DTOs.
4. Return result or domain error.

### Prefer Explicit Types

Use typed structs for service input and output. Avoid passing large `map[string]any` values through the application.

Good:

```go
type LoginInput struct {
	Email    string
	Password string
}
```

Avoid:

```go
func Login(ctx context.Context, payload map[string]any) (*AuthResult, error)
```

### Keep Validation in the Right Place

- HTTP shape validation belongs in handler request structs with Gin binding tags.
- Business validation belongs in services.
- Database constraints belong in migrations and GORM model tags.

Example:

- Handler validates that `email` is present and has email format.
- Service validates that the email is not already used.
- Database enforces the unique index.

### Avoid Leaking Sensitive Data

Never return:

- Password hashes.
- OAuth provider raw tokens.
- JWT secrets.
- Private keys.
- Internal-only database fields unless the endpoint explicitly needs them.

Always return DTOs such as `PublicUser` or `AdminUser`.

### Keep Dependencies Pointing Inward

Recommended dependency direction:

```text
router -> handler -> service -> repository -> database/model
```

Avoid dependencies in the opposite direction. For example, a repository should not import a handler package, and a model should not know about Gin.

### Make New Code Testable

- Put business rules in services so they can be unit tested.
- Depend on repository interfaces in services.
- Keep GORM details inside concrete repository implementations.
- Test HTTP behavior with API tests when route behavior matters.

### Prefer Consistency Over Cleverness

Follow the patterns already used in this codebase unless there is a strong reason to change them. A simple repeated pattern is better than a clever abstraction that future contributors need to decode.

## Conventions

- Keep HTTP logic in handlers.
- Keep business rules in services.
- Keep SQL/GORM logic in repositories.
- Keep database models and response DTO conversion in model files.
- Pass `context.Context` from handlers into services and repositories.
- Convert low-level database errors into domain errors at repository boundaries.
- Return safe DTOs from services instead of exposing sensitive fields such as password hashes.
- Register dependencies in `router.Setup()` unless the project grows enough to need a separate dependency injection package.
- Add tests at the layer that owns the behavior:
  - service tests for business rules;
  - API tests for routing, status codes, and JSON behavior;
  - repository tests when database query behavior becomes complex.

## Current Architecture Notes

- `internal/journal` has models and a repository interface, but it is not fully wired into services, handlers, or routes yet.
- `internal/auth/middleware.go` provides JWT middleware, but no current route group applies it yet.
- `internal/daily-summary` exists as a folder but currently has no source files.
- `/docs` and `/openapi.json` are registered outside `/api/v1`.
- `/api/v1/admin/users` currently has no auth middleware in the router, so it is publicly reachable unless protected elsewhere by deployment infrastructure.
