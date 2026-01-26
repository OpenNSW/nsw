# NSW Backend

Go backend service for the NSW workflow management system.

## Prerequisites

- Go 1.22 or higher
- PostgreSQL 12 or higher

## Setup

### 1. Environment Configuration

Copy the example environment file and update with your values:

```bash
cp .env.example .env
```

Edit `.env` with your database credentials:

```bash
DB_HOST=localhost
DB_PORT=5432
DB_USERNAME=postgres
DB_PASSWORD=your_password
DB_NAME=nsw_db
DB_SSLMODE=require
SERVER_PORT=8080
```

### 2. Database Setup

Create the database:

```bash
createdb nsw_db
```

Run the migration script:

```bash
# Load environment variables
set -a; source .env; set +a

# Run migration
PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USERNAME -d $DB_NAME -f internal/database/migrations/001_initial_schema.sql
```

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run the Application

```bash
# Load environment variables
set -a; source .env; set +a

# Run the server
go run ./cmd/server/main.go
```

The server will start on the port specified in `SERVER_PORT` (default: 8080).

## Project Structure

```
backend/
├── cmd/
│   └── server/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/
│   │   └── config.go            # Configuration management
│   ├── database/
│   │   ├── database.go          # Database connection
│   │   └── migrations/          # SQL migration scripts
│   ├── task/                    # Task management
│   └── workflow/                # Workflow management
│       ├── manager.go           # Workflow manager
│       ├── model/               # Data models
│       ├── router/              # HTTP handlers
│       └── service/             # Business logic
├── go.mod
└── go.sum
```

## API Endpoints

- `POST /api/tasks` - Execute a task
- `GET /api/workflow-template` - Get workflow template by HS code and type
- `POST /api/consignments` - Create a new consignment
- `GET /api/consignments/{consignmentID}` - Get consignment by ID

## Database Schema

The application uses PostgreSQL with the following tables:

- `hs_codes` - Harmonized System codes
- `workflow_templates` - Workflow definitions
- `workflow_template_maps` - HS code to workflow mappings
- `consignments` - Consignment records
- `tasks` - Workflow task instances

See `internal/database/migrations/README.md` for detailed schema information.

## Development

### Running Tests

```bash
go test ./...
```

### Building

```bash
go build -o bin/server ./cmd/server
```

### Database Health Check

The application performs a health check on startup. If the database is unavailable, the application will fail to start.

## Graceful Shutdown

The application supports graceful shutdown via SIGINT (Ctrl+C) or SIGTERM signals.
