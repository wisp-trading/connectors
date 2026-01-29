<!-- This file is markdown documentation, not Go code -->

# GitHub Copilot Configuration for wisp

## CRITICAL: Always Check Existing Code First

### Before Writing ANY Code:

Search the codebase using `@workspace` for similar functionality

## When to Use @workspace

ALWAYS search `@workspace` when:
- Writing new services/handlers
- Adding configuration
- Implementing error handling
- Creating dependencies

DON'T search `@workspace` for:
- Standard library usage
- Syntax questions
- IDE/terminal help
- Language feature questions

Check these packages in order:

`pkg/websocket/base/` - Core handlers and base services

`pkg/websocket/security/` - Validation and rate limiting

`pkg/websocket/connection/` - Connection lifecycle management

`pkg/websocket/performance/` - Metrics and monitoring

If functionality exists, USE IT. Never duplicate.

### When I Ask for Code:

First response: "I found existing code in [location]. Should we use/extend it?"

If no existing code: Proceed with implementation

STOP and ask before duplicating functionality

### When I Mention MCP Services:

Use the exact MCP service I reference

Ask: "Should I use the MCP service [name] for this?"

Never ignore MCP context - it's there for a reason

## Mandatory Development Pattern: Test-Driven Development (TDD)

### Every Code Change Must Follow This Sequence:

Write the interface first (if service/handler)

Write Ginkgo/Gomega tests BEFORE implementation

Implement to pass tests

Refactor while keeping tests green

### Test Requirements (Ginkgo + Gomega):

```go
// Example test structure for every new function
var _ = Describe("ServiceName", func() {
	var (
		service ServiceInterface
		mockDep *MockDependency
	)

	BeforeEach(func() {
		mockDep = NewMockDependency()
		service = NewService(mockDep)
	})

	Describe("FunctionName", func() {
		Context("when given valid input", func() {
			It("should return expected output", func() {
				result, err := service.FunctionName(validInput)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(Equal(expectedOutput))
			})
		})

		Context("when given nil input", func() {
			It("should return an error", func() {
				result, err := service.FunctionName(nil)
				Expect(err).To(HaveOccurred())
				Expect(result).To(BeZero())
			})
		})

		Context("when given empty input", func() {
			It("should return an error", func() {
				result, err := service.FunctionName(emptyInput)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("edge case: large input", func() {
			It("should handle gracefully", func() {
				result, err := service.FunctionName(largeInput)
				Expect(err).ToNot(HaveOccurred())
				Expect(result).To(BeNumerically(">", 0))
			})
		})
	})
})
```

Test Standards:

Minimum 4 test contexts: happy path, nil input, empty input, edge case

Use Gomega matchers: `Expect(err).ToNot(HaveOccurred())`, `Expect(x).To(Equal(y))`

Coverage target: 80% minimum

Mock external dependencies with gomock or testify/mock

Use BeforeEach/AfterEach for setup/teardown

Organize with Describe/Context/It for readability

## Dependency Injection with uber-go/fx

### All Services MUST:

**Define an interface before implementation:**

```go
// ALWAYS define interface first
type WebSocketService interface {
	Connect(ctx context.Context) error
	Subscribe(channels []string) error
	Close() error
}

// Then implementation
type webSocketServiceImpl struct {
	config      Config
	logger      *zap.Logger
	baseService *base.Service
	// ... dependencies injected via constructor
}
```

**Use constructor functions registered with fx.Provide:**

```go
func NewWebSocketService(
	config Config,
	logger *zap.Logger,
	baseService *base.Service,
	connMgr *connection.Manager,
) WebSocketService {
	return &webSocketServiceImpl{
		config:      config,
		logger:      logger,
		baseService: baseService,
		connMgr:     connMgr,
	}
}

// Register in module
var Module = fx.Module("websocket",
	fx.Provide(
		NewWebSocketService,
		NewConnectionManager,
		NewHandlerRegistry,
	),
)
```

**Inject dependencies, never instantiate directly:**

```go
// CORRECT: Inject in constructor
func NewHandler(ws WebSocketService, db Database) *Handler {
	return &Handler{
		ws: ws,
		db: db,
	}
}

// WRONG: Direct instantiation inside function
func NewHandler() *Handler {
	ws := &webSocketServiceImpl{} // NO! This breaks DI
	return &Handler{ws: ws}
}
```

Use interfaces for all injected dependencies:

Makes testing easier (can inject mocks)

Decouples implementation

Services depend on abstractions, not concrete types

## WebSocket Service Architecture (MANDATORY)

### Message Processing Pattern
All WebSocket services MUST follow this flow:

```go
func (s *Service) handleMessage(msg []byte) error {
	// 1. Use BaseService for rate limiting & validation
	if err := s.baseService.ProcessMessage(msg); err != nil {
		return fmt.Errorf("message validation failed: %w", err)
	}

	// 2. Route through HandlerRegistry
	handler, err := s.handlerRegistry.GetHandler(msgType)
	if err != nil {
		return fmt.Errorf("no handler for message type: %w", err)
	}

	// 3. Execute handler
	return handler.Handle(ctx, msg)
}
```

**Never:**

- Process messages without `BaseService.ProcessMessage()`
- Handle messages outside `HandlerRegistry`
- Manage connections without `ConnectionManager`
- Implement reconnect logic (use `ReconnectManager`)

### Configuration Pattern

Store ALL constants in `pkg/connectors/{exchange}/config.go`

Never hardcode: URLs, timeouts, rate limits, retry counts

Use baseline: `connection.DefaultConfig()` and override only what's needed

```go
// CORRECT: Configuration in config.go
const (
	HyperliquidWSURL = "wss://api.hyperliquid.xyz/ws"
	DefaultTimeout   = 30 * time.Second
	MaxReconnects    = 5
)

// WRONG: Hardcoded in service
func (s *Service) Connect() {
	conn, _ := websocket.Dial("wss://api.hyperliquid.xyz/ws", ...) // NO!
}
```

## Error Handling

### Error Channel Pattern:

```go
type WebSocketService struct {
	errChan chan error
}

func (s *Service) Run(ctx context.Context) error {
	for {
		select {
		case err := <-s.errChan:
			s.logger.Error("websocket error", zap.Error(err))
			// Handle error (reconnect, alert, etc.)
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
```

**Standards:**

- Use error channel for async errors
- Log at appropriate levels: Debug/Info/Warn/Error
- Never silently drop errors - always log or return
- Wrap errors with context: `fmt.Errorf("context: %w", err)`

## Code Review Checklist (Ask These Before Generating Code)

Before suggesting ANY code, explicitly check:

Is this already implemented in `pkg/websocket/`?

Did I search `@workspace` for similar code?

Did I write the interface first?

Did I write Ginkgo tests before implementation?

Does it use `BaseService` for message processing?

Does it use `HandlerRegistry` for routing?

Are constants in `config.go`?

Are all dependencies injected via fx?

Did I use interfaces for dependencies?

Did I check if MCP services are available?

## When I Deviate or Ask for Shortcuts

Remind me:

"Check `pkg/websocket/` first - don't duplicate existing code"

"Should we write tests first (TDD)?"

"This dependency should be injected via fx, not instantiated"

"Let's define the interface before implementation"

"Is there an MPC service for this functionality?"

## When I Ignore Your Questions or Repeat Mistakes

Directly state:

"You asked this before and I provided [X]. Are we changing direction?"

"This conflicts with existing code in [location]. Should we refactor the existing code?"

"I previously suggested using [existing service]. Why are we not using it?"

Push back if I'm making architectural mistakes

## Communication Style

Be direct, not apologetic - I value correctness over politeness

Reference specific files/packages when suggesting existing code

Ask clarifying questions before generating code

Challenge my decisions if they violate these standards

Don't repeat past mistakes - remind me of previous solutions

## Code Generation Standards

```go
// Every exported type/function needs GoDoc
// WebSocketService handles real-time market data streaming
type WebSocketService interface {
	// Connect establishes WebSocket connection with exponential backoff
	Connect(ctx context.Context) error
}

// Struct fields: injected services
// Function params: transient state
func NewService(logger *zap.Logger) *Service {
	return &Service{logger: logger}
}

// Keep functions under 50 lines
// If longer, extract helper functions
```

**Go Best Practices:**

- Use uppercase for exported identifiers
- Prefer composition over embedding
- Return errors, don't panic
- Use context for cancellation
- Close resources in defer statements

