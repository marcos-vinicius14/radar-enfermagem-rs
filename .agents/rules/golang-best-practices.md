# Go Best Practices

> Guidelines for writing idiomatic, maintainable, secure, and production-quality Go code.

This document defines the coding standards for this project.

The priorities are:

1. Simplicity
2. Readability
3. Correctness
4. Explicitness
5. Maintainability
6. Security
7. Performance

Do not write Go as if it were Java, C#, C++, or JavaScript.

Prefer idiomatic Go.

---

# 1. General Principles

## Keep code simple

Go intentionally favors simple constructs over abstraction-heavy designs.

Prefer:

```go
if err != nil {
	return err
}
```

over unnecessary abstractions hiding control flow.

Avoid:

* unnecessary design patterns
* speculative abstractions
* excessive interfaces
* deep inheritance-like compositions
* generic utilities without concrete need
* premature optimization
* premature package extraction

Code should be obvious before being clever.

---

# 2. Formatting

All Go code must be formatted automatically.

Use:

```bash
gofmt -w .
```

or:

```bash
go fmt ./...
```

Prefer `goimports` when available because it also manages imports.

Never manually align or format code against `gofmt`.

Formatting must not be debated during code review.

---

# 3. Naming

Go naming should be short, descriptive, and contextual.

## Variables

Prefer:

```go
user := findUser()
```

instead of:

```go
userObject := findUser()
```

Prefer:

```go
i
```

inside a short loop instead of:

```go
currentIterationIndex
```

But do not make names cryptic when scope is larger.

Bad:

```go
u := loadUser()
```

Good:

```go
user := loadUser()
```

---

## Avoid redundant names

Bad:

```go
user.UserName
user.UserEmail
order.OrderID
```

Prefer:

```go
user.Name
user.Email
order.ID
```

Package context already provides meaning.

Bad:

```go
package user

type UserService struct{}
```

Prefer:

```go
package user

type Service struct{}
```

Usage becomes:

```go
user.Service
```

instead of:

```go
user.UserService
```

---

# 4. Initialisms

Common initialisms should preserve capitalization.

Prefer:

```go
ID
URL
HTTP
API
JSON
SQL
UUID
```

Examples:

```go
userID
httpClient
apiURL
```

Avoid:

```go
userId
HttpClient
ApiUrl
```

---

# 5. Package Names

Package names should be:

* short
* lowercase
* singular when practical
* meaningful
* free of underscores
* free of generic names

Prefer:

```text
user
order
payment
auth
http
postgres
```

Avoid:

```text
user_service
utils
helpers
common
misc
shared
```

Generic utility packages tend to become dumping grounds.

---

# 6. Package Structure

Do not create packages solely to reproduce Java-style layers.

Avoid structures like:

```text
controllers/
services/
repositories/
models/
utils/
```

unless they genuinely represent useful package boundaries.

Prefer grouping code by domain or responsibility.

Example:

```text
cmd/
    api/
        main.go

internal/
    user/
        service.go
        repository.go
        model.go

    order/
        service.go
        repository.go

    postgres/
        user_repository.go
        order_repository.go

    http/
        user_handler.go
        order_handler.go

go.mod
go.sum
```

For application code, `internal/` is useful for preventing external packages from importing implementation details.

Use:

```text
cmd/<application>
```

for executable entry points.

---

# 7. Files

File names should be lowercase.

Prefer:

```text
user.go
user_test.go
repository.go
handler.go
```

Avoid:

```text
UserService.go
USER_SERVICE.go
```

Do not split files excessively.

A package with several cohesive files is preferable to many tiny files.

---

# 8. Exported Identifiers

Identifiers beginning with uppercase are exported.

Only export what external packages actually need.

Bad:

```go
type UserRepositoryImpl struct{}
```

when the type is internal.

Prefer:

```go
type repository struct{}
```

Keep public APIs small.

---

# 9. Comments

Comments should explain **why**, not restate **what** the code does.

Bad:

```go
// Increment counter by one.
counter++
```

Useful:

```go
// Skip the first row because it contains the CSV header.
rows = rows[1:]
```

Exported declarations should have documentation comments.

```go
// User represents an authenticated system user.
type User struct {
	ID   string
	Name string
}
```

Package documentation should describe the package purpose.

```go
// Package auth provides authentication and authorization primitives.
package auth
```

---

# 10. Functions

Functions should do one coherent thing.

Prefer small functions with clear responsibility.

Bad:

```go
func ProcessUserAndSaveAndSendEmailAndLog(...) {
}
```

Prefer:

```go
func createUser(...) error
func saveUser(...) error
func sendWelcomeEmail(...) error
```

But do not create tiny functions that add no abstraction value.

---

# 11. Function Parameters

Avoid excessive parameter counts.

Bad:

```go
func CreateUser(
	name string,
	email string,
	age int,
	country string,
	city string,
	active bool,
) error
```

When parameters represent one concept, use a struct:

```go
type CreateUserInput struct {
	Name    string
	Email   string
	Age     int
	Country string
	City    string
}

func CreateUser(input CreateUserInput) error {
	// ...
}
```

Do not create parameter structs merely to hide poor API design.

---

# 12. Return Values

Prefer returning values directly.

Bad:

```go
func Name() (name string) {
	name = "John"
	return
}
```

Prefer:

```go
func Name() string {
	return "John"
}
```

Named return values are appropriate when they improve documentation or are required for deferred logic.

Avoid naked returns in long functions.

---

# 13. Errors

Errors are values.

Handle them explicitly.

```go
user, err := repository.FindByID(ctx, id)
if err != nil {
	return err
}
```

Never silently discard errors.

Bad:

```go
result, _ := doSomething()
```

unless ignoring the error is intentional and demonstrably safe.

---

# 14. Error Wrapping

Add context when propagating errors.

Prefer:

```go
user, err := repository.FindByID(ctx, id)
if err != nil {
	return fmt.Errorf("find user %s: %w", id, err)
}
```

Use `%w` when callers may need to inspect the underlying error.

---

# 15. errors.Is and errors.As

Do not inspect errors using strings.

Bad:

```go
if err.Error() == "user not found" {
}
```

Prefer sentinel errors:

```go
var ErrUserNotFound = errors.New("user not found")
```

Then:

```go
if errors.Is(err, ErrUserNotFound) {
	// ...
}
```

For typed errors:

```go
var validationErr *ValidationError

if errors.As(err, &validationErr) {
	// ...
}
```

---

# 16. Error Messages

Error messages should normally:

* start lowercase
* avoid punctuation
* contain relevant context

Prefer:

```go
fmt.Errorf("load user: %w", err)
```

Avoid:

```go
fmt.Errorf("Error loading user.")
```

The caller may wrap the error again.

---

# 17. Panic

Do not use `panic` for ordinary error handling.

Bad:

```go
if err != nil {
	panic(err)
}
```

Prefer returning an error.

Use panic only when continuing execution is fundamentally impossible because of a programmer invariant or unrecoverable initialization condition.

---

# 18. Interfaces

Prefer small interfaces.

Ideal interfaces frequently contain one or a few methods.

```go
type UserRepository interface {
	FindByID(context.Context, string) (User, error)
}
```

Avoid large interfaces:

```go
type UserRepository interface {
	Create(...)
	Update(...)
	Delete(...)
	Find(...)
	FindAll(...)
	Search(...)
	Count(...)
	Exists(...)
	...
}
```

Split by consumer requirements when appropriate.

---

# 19. Define Interfaces Where They Are Used

Interfaces should generally belong to the consuming package.

Example:

```go
package user

type Repository interface {
	FindByID(context.Context, string) (User, error)
}
```

The implementation may live elsewhere:

```go
package postgres

type UserRepository struct {
	db *sql.DB
}
```

The implementation does not need to declare that it implements the interface.

Go interfaces are satisfied implicitly.

---

# 20. Accept Interfaces, Return Concrete Types

When useful, functions should accept the smallest interface they require.

Example:

```go
func Decode(r io.Reader) error
```

instead of:

```go
func Decode(file *os.File) error
```

Return concrete types unless callers benefit from abstraction.

---

# 21. Interface Pollution

Do not create an interface for every struct.

Bad:

```go
type UserService interface {
	Create(...) error
}

type UserServiceImpl struct{}
```

If there is only one implementation and consumers do not require polymorphism, use the concrete type.

```go
type Service struct{}
```

Introduce interfaces when they solve a real boundary or testing problem.

---

# 22. Struct Initialization

Prefer keyed fields:

```go
user := User{
	ID:    id,
	Name:  name,
	Email: email,
}
```

Avoid positional literals for structs with multiple fields:

```go
user := User{id, name, email}
```

Keyed fields are safer when structs evolve.

---

# 23. Zero Values

Design types so their zero value is useful whenever practical.

Example:

```go
var buffer bytes.Buffer
```

is immediately usable.

Avoid constructors when zero-value initialization is sufficient.

---

# 24. Constructors

Go has no constructors as a language feature.

Use `NewX` when initialization is required.

```go
func NewService(repo Repository) *Service {
	return &Service{
		repo: repo,
	}
}
```

If initialization cannot fail:

```go
func NewClient(...) *Client
```

If it can fail:

```go
func NewClient(...) (*Client, error)
```

Do not create constructors automatically for every struct.

---

# 25. Pointer vs Value Receivers

Use pointer receivers when:

* the method mutates the receiver
* the struct is large
* copying would be undesirable
* consistency requires pointer semantics

Example:

```go
func (u *User) Rename(name string) {
	u.Name = name
}
```

Use value receivers for small immutable value-like types:

```go
func (m Money) IsZero() bool {
	return m.amount == 0
}
```

Do not mix pointer and value receivers without a reason.

---

# 26. `new` vs `make`

`new(T)` allocates zeroed storage and returns `*T`.

```go
user := new(User)
```

`make` initializes slices, maps, and channels.

```go
users := make([]User, 0, 10)
cache := make(map[string]User)
jobs := make(chan Job, 10)
```

Prefer literals when clearer.

```go
users := []User{}
cache := map[string]User{}
```

---

# 27. Slices

Prefer slices over arrays for most application code.

```go
users := []User{}
```

Preallocate when size is known or reasonably predictable:

```go
users := make([]User, 0, len(rows))
```

Then:

```go
users = append(users, user)
```

This can avoid unnecessary allocations.

---

# 28. Nil Slices

A nil slice is normally valid and should often be preferred to an empty allocated slice.

```go
var users []User
```

Both generally support:

```go
len(users)
append(users, user)
range users
```

Do not allocate empty slices unnecessarily.

Be aware that serialization formats may distinguish `nil` from empty slices.

---

# 29. Maps

Check map presence explicitly when needed:

```go
user, ok := users[id]
if !ok {
	// not found
}
```

Do not depend solely on the zero value if zero is also valid.

---

# 30. Defer

Use `defer` for cleanup close to resource acquisition.

```go
file, err := os.Open(name)
if err != nil {
	return err
}
defer file.Close()
```

For resources whose `Close` error matters:

```go
if err := file.Close(); err != nil {
	return fmt.Errorf("close file: %w", err)
}
```

Understand that deferred calls execute when the surrounding function returns.

---

# 31. Context

Functions performing:

* network operations
* database operations
* external API calls
* long-running work
* cancellable work

should generally accept `context.Context`.

Convention:

```go
func FindUser(ctx context.Context, id string) (User, error)
```

`context.Context` should normally be the first parameter.

Use:

```go
ctx
```

as its variable name.

---

# 32. Never Store Context in Structs

Avoid:

```go
type Service struct {
	ctx context.Context
}
```

Prefer:

```go
func (s *Service) Execute(ctx context.Context) error {
	// ...
}
```

Context belongs to the lifetime of the operation, not normally the lifetime of an object.

---

# 33. Never Pass nil Context

Bad:

```go
service.Execute(nil)
```

Use:

```go
context.Background()
```

for top-level contexts.

Use:

```go
context.TODO()
```

only when the correct context has not yet been determined.

---

# 34. Cancel Contexts

When creating cancellable contexts:

```go
ctx, cancel := context.WithTimeout(parent, 5*time.Second)
defer cancel()
```

Always call the returned cancel function.

This releases associated resources.

---

# 35. Context Values

Do not use context as an arbitrary parameter bag.

Bad:

```go
ctx = context.WithValue(ctx, "userID", id)
ctx = context.WithValue(ctx, "limit", 10)
ctx = context.WithValue(ctx, "sort", "name")
```

Context values are intended primarily for request-scoped metadata crossing API/process boundaries.

Normal business parameters belong in explicit function arguments.

---

# 36. Goroutines

Start goroutines only when their lifecycle is understood.

Bad:

```go
go process()
```

without knowing:

* when it stops
* how errors are handled
* how cancellation occurs
* who owns it

Every goroutine should have a clear termination condition.

---

# 37. Goroutine Leaks

Never create goroutines that can block forever.

Dangerous:

```go
go func() {
	result := <-ch
	process(result)
}()
```

if nobody is guaranteed to send to `ch`.

Use cancellation when appropriate:

```go
select {
case result := <-ch:
	process(result)

case <-ctx.Done():
	return
}
```

---

# 38. Channels

Use channels to communicate between goroutines.

Do not use channels simply because Go has them.

A mutex may be simpler for protecting shared state.

Use the synchronization primitive that best matches the problem.

---

# 39. Channel Ownership

The goroutine that creates and sends on a channel should generally control closing it.

Receivers should generally not close channels they do not own.

Never send on a closed channel.

---

# 40. Buffered Channels

Use buffer sizes intentionally.

Avoid arbitrary numbers:

```go
make(chan Job, 1000)
```

unless there is a reason for `1000`.

Buffer sizes affect:

* backpressure
* memory usage
* scheduling behavior
* latency

---

# 41. Mutexes

Use `sync.Mutex` for straightforward shared-state synchronization.

```go
type Cache struct {
	mu    sync.RWMutex
	items map[string]Item
}
```

Protect all access consistently.

Do not copy structs containing mutexes.

---

# 42. Concurrency Safety

Run the race detector during development and CI where practical:

```bash
go test -race ./...
```

Code that appears correct without the race detector may still contain data races.

---

# 43. WaitGroup

Use `sync.WaitGroup` when waiting for goroutines.

```go
var wg sync.WaitGroup

for _, job := range jobs {
	wg.Add(1)

	go func(job Job) {
		defer wg.Done()
		process(job)
	}(job)
}

wg.Wait()
```

Ensure every `Add` corresponds to a `Done`.

---

# 44. Concurrency Is Not Parallelism

Do not introduce goroutines automatically to make code "faster".

Concurrency is useful for structuring independent work.

Parallel execution only helps when the workload and runtime characteristics benefit from it.

Measure before optimizing.

---

# 45. Select

Use `select` for coordinating multiple channel operations.

```go
select {
case result := <-results:
	return result, nil

case <-ctx.Done():
	return Result{}, ctx.Err()
}
```

Be aware that `select` chooses among ready cases pseudo-randomly.

---

# 46. Loops

Prefer simple `range` loops.

```go
for _, user := range users {
	process(user)
}
```

Use index when required:

```go
for i := range users {
	users[i].Active = true
}
```

---

# 47. Early Returns

Prefer early returns over nested control flow.

Bad:

```go
if user != nil {
	if user.Active {
		if user.Email != "" {
			send(user)
		}
	}
}
```

Prefer:

```go
if user == nil {
	return
}

if !user.Active {
	return
}

if user.Email == "" {
	return
}

send(user)
```

---

# 48. Avoid `else` After Return

Bad:

```go
if err != nil {
	return err
} else {
	process()
}
```

Prefer:

```go
if err != nil {
	return err
}

process()
```

---

# 49. Switch

Prefer `switch` when it improves clarity over long `if/else` chains.

```go
switch status {
case StatusPending:
	// ...
case StatusApproved:
	// ...
case StatusRejected:
	// ...
default:
	// ...
}
```

Go does not require explicit `break`.

---

# 50. Type Switch

Use type switches when behavior genuinely depends on dynamic type:

```go
switch v := value.(type) {
case string:
	handleString(v)

case int:
	handleInt(v)

default:
	return fmt.Errorf("unsupported type %T", v)
}
```

Do not overuse `interface{}`/`any`.

---

# 51. `any`

`any` is an alias for `interface{}`.

Use it only when values genuinely may be of arbitrary types.

Bad:

```go
func Save(value any)
```

when the domain has a known type.

Prefer:

```go
func Save(user User)
```

Strong types are preferable.

---

# 52. Generics

Use generics when the algorithm genuinely applies to multiple types.

Good candidates include:

* containers
* reusable algorithms
* type-safe transformations

Do not use generics merely to eliminate a few duplicated lines.

Prefer concrete code when it is clearer.

Bad abstraction:

```go
func Process[T any](value T) T
```

with no meaningful generic behavior.

---

# 53. Constraints

Use the narrowest useful constraint.

Avoid `any` when operations require specific capabilities.

Generic APIs should be easier to use than duplicated implementations, not harder.

---

# 54. Embedding

Embedding can promote behavior:

```go
type LoggingService struct {
	*Service
	Logger
}
```

Use it for composition when the relationship is meaningful.

Do not use embedding merely to simulate inheritance.

---

# 55. Dependency Injection

Go generally does not need dependency injection frameworks.

Prefer explicit constructors:

```go
type Service struct {
	repository Repository
	logger     *slog.Logger
}

func NewService(
	repository Repository,
	logger *slog.Logger,
) *Service {
	return &Service{
		repository: repository,
		logger:     logger,
	}
}
```

Explicit wiring is usually easier to understand and maintain.

---

# 56. Global State

Avoid mutable global state.

Bad:

```go
var database *sql.DB
```

Prefer explicit dependencies:

```go
type Application struct {
	DB *sql.DB
}
```

Constants and immutable package-level values are generally fine.

---

# 57. `init`

Use `init()` sparingly.

Avoid hidden application initialization.

Prefer explicit initialization in:

```go
func main()
```

or constructors.

Hidden side effects make startup and tests harder to reason about.

---

# 58. Configuration

Load configuration at application startup.

Validate required configuration immediately.

Prefer a typed configuration:

```go
type Config struct {
	HTTPPort string
	DBURL    string
}
```

Avoid scattering environment-variable access throughout the codebase.

---

# 59. Secrets

Never commit:

* passwords
* tokens
* private keys
* API secrets
* database credentials

Use:

* environment variables
* secret managers
* platform-provided secret mechanisms

Never log secrets.

---

# 60. Logging

Prefer structured logging.

Modern Go applications can use `log/slog`.

Example:

```go
logger.Info(
	"user created",
	"user_id", user.ID,
)
```

Prefer fields over concatenated strings.

Bad:

```go
logger.Printf("created user " + user.ID)
```

Avoid logging the same error repeatedly at different layers.

Typically:

* lower layers return errors
* application boundaries decide what to log

---

# 61. HTTP Handlers

Handlers should primarily:

1. parse input
2. validate input
3. invoke application/domain logic
4. map result/error
5. return HTTP response

Avoid embedding business logic inside handlers.

---

# 62. HTTP Timeouts

Production HTTP servers should configure timeouts where appropriate.

Example:

```go
server := &http.Server{
	Addr:              ":8080",
	ReadHeaderTimeout: 5 * time.Second,
	ReadTimeout:       10 * time.Second,
	WriteTimeout:      10 * time.Second,
	IdleTimeout:       60 * time.Second,
}
```

Do not blindly copy timeout values.

Select them according to application behavior.

---

# 63. HTTP Clients

Avoid uncontrolled use of:

```go
http.DefaultClient
```

for production external calls where timeout behavior matters.

Prefer explicitly configured clients:

```go
client := &http.Client{
	Timeout: 10 * time.Second,
}
```

Propagate request contexts.

---

# 64. Database Access

Use contexts:

```go
row := db.QueryRowContext(ctx, query, id)
```

Prefer:

```go
ExecContext
QueryContext
QueryRowContext
```

over context-free alternatives in request-driven code.

---

# 65. SQL Injection

Never construct SQL using untrusted string interpolation.

Bad:

```go
query := "SELECT * FROM users WHERE email = '" + email + "'"
```

Prefer parameterized queries:

```go
row := db.QueryRowContext(
	ctx,
	"SELECT id, email FROM users WHERE email = $1",
	email,
)
```

---

# 66. Transactions

Keep transactions focused.

Example:

```go
tx, err := db.BeginTx(ctx, nil)
if err != nil {
	return err
}

defer tx.Rollback()

if err := executeOperations(ctx, tx); err != nil {
	return err
}

if err := tx.Commit(); err != nil {
	return err
}

return nil
```

`Rollback` after successful `Commit` should be harmless for standard `database/sql` transactions.

---

# 67. Resource Management

Close resources deterministically.

Examples:

```go
defer rows.Close()
```

```go
defer response.Body.Close()
```

Always check iteration errors:

```go
for rows.Next() {
	// ...
}

if err := rows.Err(); err != nil {
	return err
}
```

---

# 68. JSON

Define explicit request/response types.

Prefer:

```go
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}
```

over:

```go
map[string]any
```

Strong types improve validation, documentation, and maintainability.

---

# 69. API Models vs Domain Models

Do not automatically expose database or domain structs as HTTP contracts.

Example:

```go
type User struct {
	ID           string
	PasswordHash string
}
```

should not automatically become:

```json
{
  "id": "...",
  "passwordHash": "..."
}
```

Use explicit transport DTOs when the boundary requires it.

---

# 70. Validation

Validate data at system boundaries.

Examples:

* HTTP input
* CLI arguments
* configuration
* external messages

Domain invariants should also be enforced inside domain/application logic.

Do not depend exclusively on frontend validation.

---

# 71. Testing Strategy: Testing Trophy

Adopt the **Testing Trophy** philosophy as our guiding strategy:

1. **Static Analysis**: Linters (`go vet`, `golangci-lint`), Go compiler, strong typing, sqlc validation.
2. **Unit Tests (Lean & Focused)**: Reserved for pure algorithms, math, text normalization, and complex calculation rules. Avoid extensive unit mocking of repositories and services.
3. **Integration Tests (Primary Focus & Core of the Trophy)**: **Prioritize integration tests over unit tests.** Test components integrated with real dependencies (such as PostgreSQL running in Docker) to validate real queries, constraints, transactions, and behaviors. Integration tests offer the highest ROI and confidence against regression.
4. **End-to-End Tests (E2E)**: High-level checks for critical business flows.

Use Go's built-in `testing` package.

Example:

```go
func TestJobRepository_InsertAndFindByID(t *testing.T) {
	// ...
}
```

Tests should clearly express:

* input
* expected behavior
* actual behavior


---

# 72. Table-Driven Tests

Use table-driven tests when testing multiple scenarios.

```go
func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr bool
	}{
		{
			name:    "valid email",
			email:   "user@example.com",
			wantErr: false,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)

			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateEmail() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
```

Use table-driven tests when they improve readability, not as a mandatory pattern.

---

# 73. Test Behavior, Not Implementation

Avoid tests that are tightly coupled to internal implementation details.

Prefer testing observable behavior.

Refactoring internals should not unnecessarily break tests.

---

# 74. Avoid Assertion DSLs by Default

Go tests are usually clearer with standard language constructs.

Prefer:

```go
if got != want {
	t.Fatalf("got %v, want %v", got, want)
}
```

over an unnecessary assertion framework.

Third-party assertion libraries may still be justified if the project intentionally standardizes on one and they improve readability.

---

# 75. Test Failure Messages

Failure output must make debugging easy.

Prefer:

```go
t.Fatalf("CreateUser() error = %v, want nil", err)
```

or:

```go
t.Errorf("Name = %q, want %q", got, want)
```

Use the conventional ordering:

```text
got, want
```

---

# 76. Test Helpers

Helpers should call:

```go
t.Helper()
```

Example:

```go
func newTestUser(t *testing.T) User {
	t.Helper()

	return User{
		ID: "123",
	}
}
```

This improves reported failure locations.

---

# 77. Parallel Tests

Use:

```go
t.Parallel()
```

only when tests are safe to run concurrently.

Do not use parallel tests when they share mutable:

* databases
* files
* ports
* environment variables
* global state

without proper isolation.

---

# 78. Fuzz Testing

Use fuzzing for parsers, decoders, validators, protocols, and code handling untrusted input.

Example shape:

```go
func FuzzParse(f *testing.F) {
	f.Add("valid-input")

	f.Fuzz(func(t *testing.T, value string) {
		_, _ = Parse(value)
	})
}
```

Fuzzing is especially valuable for finding unexpected edge cases and security-sensitive failures.

---

# 79. Benchmarks

Use benchmarks when performance matters.

```go
func BenchmarkParser(b *testing.B) {
	for b.Loop() {
		Parse(data)
	}
}
```

Measure before optimizing.

Do not optimize code based only on intuition.

---

# 80. Profiling

Use Go profiling tools before performance changes.

Common tools include:

```bash
go test -bench=.
go test -bench=. -benchmem
go tool pprof
```

Prefer evidence-driven optimization.

---

# 81. Dependencies

Prefer the standard library when it sufficiently solves the problem.

Do not add a dependency for trivial functionality.

Evaluate dependencies for:

* maintenance
* security
* API quality
* transitive dependency count
* license
* update frequency

---

# 82. Modules

Every modern project should use Go modules.

Initialize with:

```bash
go mod init github.com/example/project
```

Maintain dependencies with:

```bash
go mod tidy
```

Commit:

```text
go.mod
go.sum
```

to version control.

---

# 83. Dependency Versions

Avoid unnecessary upgrades.

Inspect dependencies:

```bash
go list -m all
```

Check available upgrades:

```bash
go list -m -u all
```

Upgrade intentionally.

---

# 84. `go mod tidy`

Run:

```bash
go mod tidy
```

after dependency changes.

It removes unnecessary requirements and adds missing ones.

CI should detect uncommitted module changes when appropriate.

---

# 85. Semantic Versioning

Published modules should use semantic versions:

```text
v1.0.0
v1.1.0
v1.1.1
```

A published module version should never be mutated.

Publish a new version instead.

---

# 86. Major Versions

For module versions `v2+`, follow Go module major-version semantics.

Example import path:

```text
github.com/example/project/v2
```

when applicable.

---

# 87. Security Scanning

Use Go's official vulnerability scanner.

Install:

```bash
go install golang.org/x/vuln/cmd/govulncheck@latest
```

Run:

```bash
govulncheck ./...
```

Use it locally and in CI.

---

# 88. Static Analysis

At minimum run:

```bash
go vet ./...
```

Recommended project checks:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
govulncheck ./...
```

Projects may additionally use `staticcheck` or `golangci-lint`.

Linters should improve correctness and maintainability rather than enforce arbitrary complexity.

---

# 89. CI

A baseline CI pipeline should include:

```bash
go mod tidy
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
govulncheck ./...
```

CI should fail if formatting or dependency files differ from committed content.

---

# 90. Build Reproducibility

Build using module metadata committed to the repository.

Do not depend on packages installed manually on a developer machine.

A fresh clone should be buildable using documented commands.

---

# 91. Main Package

Keep `main` small.

Bad:

```go
func main() {
	// hundreds of lines of initialization and business logic
}
```

Prefer:

```go
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// application wiring
	return nil
}
```

Business logic belongs outside `main`.

---

# 92. Graceful Shutdown

Long-running services should handle shutdown intentionally.

Typical signals:

```go
signal.NotifyContext(
	context.Background(),
	os.Interrupt,
	syscall.SIGTERM,
)
```

Propagate cancellation to:

* HTTP servers
* workers
* consumers
* database operations
* external requests

---

# 93. Time

Prefer explicit durations:

```go
5 * time.Second
```

instead of:

```go
5000 * time.Millisecond
```

Use `time.Time` for timestamps rather than string representations inside domain/application code when possible.

Serialize only at boundaries.

---

# 94. UUIDs and IDs

Prefer domain-specific types where they improve correctness:

```go
type UserID string
```

instead of passing arbitrary strings everywhere.

Do not introduce wrapper types merely for ceremony.

---

# 95. Constants

Use typed constants when appropriate.

```go
type Status string

const (
	StatusPending  Status = "pending"
	StatusApproved Status = "approved"
)
```

This is preferable to scattered magic strings.

---

# 96. Avoid Magic Values

Bad:

```go
if retries > 5 {
}
```

Prefer:

```go
const maxRetries = 5

if retries > maxRetries {
}
```

Use named constants when the value has domain meaning.

---

# 97. Enums

Go has no enum keyword.

Use typed constants:

```go
type Role string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"
)
```

Do not expose invalid states unnecessarily.

---

# 98. Functional Options

Functional options may be useful for APIs with many optional configuration values.

Example:

```go
client := NewClient(
	WithTimeout(5*time.Second),
	WithRetries(3),
)
```

Do not use functional options for constructors with only a couple of obvious required parameters.

---

# 99. Builder Pattern

Avoid Java-style builders unless they solve a real API problem.

Bad:

```go
NewUserBuilder().
	WithName(...).
	WithEmail(...).
	Build()
```

Prefer:

```go
User{
	Name:  name,
	Email: email,
}
```

or a constructor if validation is necessary.

---

# 100. Getters and Setters

Do not automatically create getters and setters.

Bad:

```go
func (u *User) GetName() string
func (u *User) SetName(name string)
```

Use fields directly when appropriate:

```go
user.Name
```

If behavior or invariants are required, use domain-oriented methods:

```go
func (u *User) Rename(name string) error
```

---

# 101. Avoid ServiceImpl Naming

Avoid Java patterns such as:

```go
UserService
UserServiceImpl
```

Prefer:

```go
type Service struct{}
```

inside package:

```go
package user
```

Concrete names should describe what a type is, not that it is an implementation.

---

# 102. Avoid RepositoryImpl Naming

Bad:

```go
type UserRepositoryImpl struct{}
```

Prefer implementation-specific names:

```go
type UserRepository struct{}
```

inside:

```go
package postgres
```

Usage:

```go
postgres.UserRepository
```

---

# 103. Domain Logic

Keep domain rules close to the domain types or application logic responsible for enforcing them.

Avoid spreading the same rule across:

* handler
* service
* repository
* database adapter

There should be a clear source of truth.

---

# 104. Dependency Direction

Business logic should not unnecessarily depend on delivery or infrastructure concerns.

For example, domain/application packages should generally not need to know:

```text
net/http
PostgreSQL driver details
RabbitMQ implementation details
JSON transport details
```

Expose the smallest abstraction required at meaningful boundaries.

Do not turn this principle into unnecessary architecture layers.

---

# 105. Clean Architecture

Go can use architectural principles such as ports and adapters or clean architecture, but avoid ceremonial implementations.

Good:

```text
domain
application
postgres
http
```

Bad when unnecessary:

```text
entities
usecases
controllers
presenters
gateways
factories
interactors
repositories
repositories_impl
adapters
ports
```

Use architecture proportional to the system complexity.

---

# 106. Repository Pattern

Do not introduce repositories when direct `database/sql` usage inside a focused package would be simpler.

Repositories are useful when they establish a meaningful persistence boundary.

Interfaces should represent what consumers need.

Example:

```go
type UserRepository interface {
	FindByEmail(ctx context.Context, email string) (User, error)
	Save(ctx context.Context, user User) error
}
```

---

# 107. Dependency Cycles

Go prohibits import cycles.

If packages require each other, reconsider package responsibilities.

Do not create artificial packages merely to bypass cycles.

Usually the proper solution is to move the shared abstraction to the package that owns it or redesign the dependency direction.

---

# 108. Public API Design

Public APIs should be difficult to misuse.

Prefer:

```go
func ParseEmail(value string) (Email, error)
```

over exposing partially initialized states.

Minimize exported identifiers.

Changing private implementation is cheap.

Changing public APIs is expensive.

---

# 109. Compatibility

For published modules, preserve backward compatibility whenever possible.

Removing or changing exported:

* functions
* methods
* fields
* interfaces
* package paths

may break consumers.

Treat public APIs deliberately.

---

# 110. Security

Assume external input is untrusted.

Validate:

* HTTP input
* uploaded files
* database input from external systems
* messages
* command-line input
* paths
* URLs

Limit:

* request sizes
* concurrency
* timeouts
* resource usage

Use standard cryptographic libraries.

Do not implement custom cryptography.

---

# 111. Path Handling

Use:

```go
filepath.Join(...)
```

for filesystem paths.

Avoid manual concatenation:

```go
base + "/" + filename
```

Validate user-controlled paths to prevent directory traversal where relevant.

---

# 112. Randomness

For security-sensitive randomness use:

```go
crypto/rand
```

not:

```go
math/rand
```

Use `math/rand` only for non-security randomness such as simulations.

---

# 113. Passwords

Never store plaintext passwords.

Use a suitable password hashing algorithm/library.

Never log:

* plaintext passwords
* hashes unnecessarily
* authentication tokens

Do not implement custom password hashing.

---

# 114. Logging Errors

Avoid:

```go
logger.Error("database error", "error", err)
return err
```

at every layer.

This can result in the same error being logged several times.

Prefer propagating contextual errors and logging at an appropriate application boundary.

---

# 115. Error Context

Each layer should add meaningful context without duplicating noise.

Example:

```go
return fmt.Errorf("save user: %w", err)
```

Then:

```go
return fmt.Errorf("create account: %w", err)
```

A final error might read conceptually:

```text
create account: save user: unique constraint violation
```

This preserves the causal chain.

---

# 116. Metrics

Use metrics for system behavior, not logs alone.

Typical metrics include:

* request latency
* error rates
* queue depth
* operation counts
* external dependency latency
* worker processing time

Do not include high-cardinality values such as raw user IDs as metric labels unless explicitly justified.

---

# 117. Observability

Production services should make failures diagnosable using an appropriate combination of:

* structured logs
* metrics
* traces
* contextual errors

Do not log entire sensitive payloads solely for debugging convenience.

---

# 118. Comments vs Code

Prefer expressive code over explanatory comments.

Bad:

```go
// Check if user is active
if user.Status == 1 {
}
```

Prefer:

```go
if user.IsActive() {
}
```

Comments should capture information the code cannot express clearly.

---

# 119. TODO Comments

Useful TODOs should identify the reason or issue.

Prefer:

```go
// TODO(#123): remove compatibility path after clients migrate.
```

Avoid:

```go
// TODO: fix later
```

---

# 120. Dead Code

Delete dead code.

Do not leave large commented-out blocks.

Version control already preserves history.

---

# 121. Generated Code

Generated files should clearly identify themselves.

Convention:

```go
// Code generated by <tool>; DO NOT EDIT.
```

Do not manually modify generated code.

---

# 122. Build Tags

Use build tags only when necessary.

Example:

```go
//go:build integration
```

Keep build constraints explicit and documented.

---

# 123. Cgo

Avoid `cgo` unless native integration is necessary.

It increases complexity around:

* builds
* portability
* cross-compilation
* runtime behavior
* deployment

Prefer pure Go dependencies when equivalent solutions exist.

---

# 124. Standard Library First

Before introducing a dependency, check whether the standard library already provides the capability.

Common packages worth knowing:

```text
context
errors
fmt
io
os
path/filepath
net/http
encoding/json
database/sql
sync
time
testing
log/slog
crypto/*
```

---

# 125. Tooling

Standard development commands:

```bash
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
go mod tidy
go build ./...
```

Security:

```bash
govulncheck ./...
```

Useful diagnostics:

```bash
go test -bench=. -benchmem
go tool pprof
go list
go env
```

---

# 126. Definition of Done

Before code is considered complete:

```text
[ ] Code is formatted
[ ] go vet passes
[ ] Unit tests pass
[ ] Race-sensitive tests pass
[ ] Error paths are tested
[ ] Context is propagated correctly
[ ] Resources are closed
[ ] No unnecessary goroutines exist
[ ] No secrets are logged or committed
[ ] Dependencies are justified
[ ] go.mod/go.sum are clean
[ ] govulncheck passes
[ ] Public API additions are intentional
[ ] Naming is idiomatic Go
[ ] No unnecessary abstractions were introduced
```

---

# 127. Recommended CI Commands

Minimum:

```bash
go mod tidy
go fmt ./...
go vet ./...
go test ./...
go test -race ./...
govulncheck ./...
```

For stricter projects:

```bash
staticcheck ./...
```

or a carefully configured:

```bash
golangci-lint run
```

Do not enable dozens of conflicting linters without understanding their trade-offs.

---

# 128. AI Coding Agent Rules

When generating Go code for this repository, follow these rules.

## MUST

* Produce idiomatic Go.
* Format code according to `gofmt`.
* Handle every meaningful error.
* Wrap errors with `%w` when propagation requires context.
* Use `errors.Is` and `errors.As` instead of matching error strings.
* Propagate `context.Context`.
* Put `context.Context` first in argument lists.
* Call context cancellation functions.
* Keep interfaces small.
* Define interfaces near consumers.
* Prefer concrete types unless abstraction is needed.
* Prefer standard library solutions.
* Write tests for business rules.
* Use table-driven tests when they improve clarity.
* Close resources.
* use parameterized SQL.
* avoid goroutine leaks.
* protect shared mutable state.
* keep `main` focused on wiring.
* minimize exported APIs.
* use explicit dependency injection.
* keep dependencies directional and obvious.

## MUST NOT

* Write Java-style Go.
* Create `SomethingImpl` types.
* Create interfaces for every service.
* Introduce getters/setters automatically.
* Create `utils`, `helpers`, or `common` dumping-ground packages.
* Use `panic` for normal failures.
* ignore errors without justification.
* compare `err.Error()` strings.
* store contexts in structs without exceptional justification.
* use arbitrary context values as function arguments.
* create goroutines without lifecycle management.
* use global mutable state unnecessarily.
* add dependencies for trivial functionality.
* introduce generics without a concrete reusable type problem.
* add architecture layers solely for theoretical purity.
* expose database models directly as API contracts by default.
* build SQL using string concatenation with external values.
* implement custom cryptography.
* commit secrets.
* log credentials or tokens.
* optimize without measurement.

---

# 129. Preferred Style Example

```go
package user

import (
	"context"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("user not found")

type User struct {
	ID    string
	Name  string
	Email string
}

type Repository interface {
	FindByID(ctx context.Context, id string) (User, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) FindByID(
	ctx context.Context,
	id string,
) (User, error) {
	if id == "" {
		return User{}, errors.New("id is required")
	}

	user, err := s.repository.FindByID(ctx, id)
	if err != nil {
		return User{}, fmt.Errorf("find user %q: %w", id, err)
	}

	return user, nil
}
```

The important properties are:

* explicit dependencies
* small interface
* context propagation
* explicit validation
* contextual error wrapping
* no unnecessary abstraction
* no framework-specific coupling

---

# 130. Core Philosophy

Idiomatic Go tends toward:

```text
simple > clever

explicit > magical

concrete > abstract

small interfaces > large interfaces

composition > inheritance

standard library > unnecessary dependency

early return > deep nesting

errors as values > exception-style control flow

measured optimization > speculative optimization

clear concurrency ownership > uncontrolled goroutines

small public API > excessive exports
```

The goal is not to produce the shortest possible code.

The goal is to produce code whose behavior is easy to understand, maintain, test, and operate.
