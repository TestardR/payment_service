# Payment Processor

A payment processing system that acts as an intermediary between clients and a banking system. The system receives payment requests via REST API, persists them in dual storage (SQLite database and XML files), and processes bank responses to update payment statuses.

## Architecture

This project follows Clean Architecture and Domain-Driven Design (DDD) principles:

- **Domain Layer** (`internal/domain/`): Core business logic, entities, and value objects
- **Application Layer** (`internal/application/`): Use cases, command/query handlers, and ports
- **Infrastructure Layer** (`internal/infrastructure/`): External concerns like databases, HTTP, and file systems

## Project Structure

```
internal/
├── domain/
│   ├── payment/           # Payment aggregate and domain services
│   └── shared/            # Shared value objects (IBAN, Amount, etc.)
├── application/
│   ├── command/           # Command handlers
│   ├── query/             # Query handlers
│   ├── port/              # Application interfaces
│   └── service/           # Application services
└── infrastructure/
    ├── persistence/
    │   ├── sqlite/        # SQLite repository implementations
    │   └── xml/           # XML file generation
    ├── http/
    │   ├── handler/       # HTTP request handlers
    │   └── middleware/    # HTTP middleware
    └── worker/            # Background worker for CSV processing
```

## Getting Started

### Prerequisites

- Go 1.24.4 or later
- SQLite 3.x

### Building

```bash
make build
```

### Running

```bash
make run
```

### Testing

```bash
make test
```

### Development

```bash
make lint    # Run linting tools
make fmt     # Format code
```

## Features

- **Payment Request API**: REST API with basic authentication
- **Dual Persistence**: SQLite database + XML file storage with ACID transactions
- **Background Processing**: CSV file monitoring for bank response updates
- **Data Validation**: IBAN format validation and input sanitization
- **Idempotency**: Duplicate request prevention

## API Endpoints

- `POST /payments` - Submit a payment request

## Development Workflow

- Use conventional commits for all commit messages
- Keep pull requests small and focused
- Follow Go best practices and use provided linting tools
