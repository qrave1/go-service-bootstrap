# Go Service Bootstrap

This is a TUI (Text User Interface) tool for quickly scaffolding a new Go web service based on Clean/Hexagonal Architecture principles.

## Generated Architecture

The generated project follows principles similar to Clean Architecture and Hexagonal Architecture (Ports and Adapters). This ensures a clear separation of concerns, loose coupling, and high testability.

The main layers are:
- **Domain**: The core of the application, containing business logic and entities.
- **Usecase**: Orchestrates the flow of data and manages business processes.
- **Infrastructure**: Contains concrete implementations of interfaces (like databases).
- **Presentation**: The entry point for users (e.g., HTTP handlers).


## Install

You can install the generator directly using `go install`:

```bash
go install github.com/qrave1/go-service-bootstrap@latest
```

This will download the source code, build the binary, and place it in your `$GOPATH/bin` directory. Make sure this directory is in your system's `PATH`.

## Options

The following options are available in the TUI:

- **HTTP Framework**:
  - Echo
  - Fiber
- **Database (with sqlx)**:
  - PostgreSQL
  - MySQL
  - SQLite
- **WebSocket**:
  - gorilla/websocket
- **Features**:
  - Enable HTML templates (creates a `/web` directory)
- **Task Runner**:
  - Makefile
  - Taskfile

When a database is selected, a corresponding service will automatically be added to the `docker-compose.yml` file for a ready-to-use development environment.
