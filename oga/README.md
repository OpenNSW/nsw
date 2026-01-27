# OGA Portal Backend

A standalone Go service for managing OGA (Other Government Agency) verification tasks. Services can inject data into this portal for review, and reviewers can approve or reject the submissions.

## Features

- **Data Injection**: External services can POST data for OGA review
- **Local Storage**: Uses SQLite for data persistence
- **Review Workflow**: Approve/Reject functionality with reviewer notes
- **Response Callback**: Automatically sends review results back to the originating service
- **RESTful API**: Simple HTTP API for integration

## Architecture

```
External Service  -->  POST /api/oga/inject  -->  OGA Portal (SQLite DB)
                                                        |
                                                        v
                                             UI fetches via GET APIs
                                                        |
                                                        v
                                             Reviewer Approves/Rejects
                                                        |
                                                        v
External Service  <--  POST {serviceUrl}/api/tasks  <--  OGA Portal
```

## API Endpoints

### 0. Health Check

**GET** `/health`

Simple health check endpoint to verify the service is running.

**Response:**
```json
{
  "status": "ok",
  "service": "oga-portal"
}
```

### 1. Inject Data (For External Services)

**POST** `/api/oga/inject`

Inject data into the OGA portal for review.

**Request Body:**
```json
{
  "taskId": "927adaaa-b959-4648-880a-16508acafc12",
  "consignmentId": "cefda05e-3071-4e94-b001-328094e570a7",
  "serviceUrl": "http://your-service.com/api/tasks",
  "data": {
    "field1": "value1",
    "field2": "value2"
  }
}
```

**Response:**
```json
{
  "success": true,
  "message": "Data injected successfully",
  "taskId": "927adaaa-b959-4648-880a-16508acafc12"
}
```

### 2. Get All Applications (For UI)

**GET** `/api/oga/applications`

Fetch all applications. Optionally filter by status.

**Query Parameters:**
- `status` (optional): Filter by status (PENDING, APPROVED, REJECTED)

**Response:**
```json
[
  {
    "taskId": "927adaaa-b959-4648-880a-16508acafc12",
    "consignmentId": "cefda05e-3071-4e94-b001-328094e570a7",
    "serviceUrl": "http://your-service.com/api/tasks",
    "data": {
      "field1": "value1",
      "field2": "value2"
    },
    "status": "PENDING",
    "reviewerNotes": "",
    "reviewedAt": null,
    "createdAt": "2024-01-27T10:00:00Z",
    "updatedAt": "2024-01-27T10:00:00Z"
  }
]
```

### 3. Get Single Application (For UI)

**GET** `/api/oga/applications/{taskId}`

Fetch a specific application by task ID.

**Response:**
```json
{
  "taskId": "927adaaa-b959-4648-880a-16508acafc12",
  "consignmentId": "cefda05e-3071-4e94-b001-328094e570a7",
  "serviceUrl": "http://your-service.com/api/tasks",
  "data": {
    "field1": "value1",
    "field2": "value2"
  },
  "status": "PENDING",
  "reviewerNotes": "",
  "reviewedAt": null,
  "createdAt": "2024-01-27T10:00:00Z",
  "updatedAt": "2024-01-27T10:00:00Z"
}
```

### 4. Review Application (For UI)

**POST** `/api/oga/applications/{taskId}/review`

Approve or reject an application. This will update the database and send the review result back to the originating service.

**Request Body:**
```json
{
  "decision": "APPROVED",
  "reviewerNotes": "All documents verified"
}
```

`decision` must be either `APPROVED` or `REJECTED`.

**Response:**
```json
{
  "success": true,
  "message": "Application reviewed successfully"
}
```

**Callback to Service:**

After a successful review, the OGA portal will POST the following to the `serviceUrl` that was provided during data injection:

```json
{
  "task_id": "927adaaa-b959-4648-880a-16508acafc12",
  "consignment_id": "cefda05e-3071-4e94-b001-328094e570a7",
  "payload": {
    "action": "OGA_VERIFICATION",
    "content": {
      "decision": "APPROVED",
      "reviewerNotes": "All documents verified",
      "reviewedAt": "2024-01-27T10:15:00Z"
    }
  }
}
```

## Configuration

Configuration is done via environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `OGA_PORT` | HTTP server port | `8081` |
| `OGA_DB_PATH` | Path to SQLite database file | `./oga_applications.db` |

## Getting Started

### Build

```bash
go build -o bin/oga ./cmd/server
```

### Run

```bash
# With default configuration
./bin/oga

# With custom configuration
OGA_PORT=9000 OGA_DB_PATH=/data/oga.db ./bin/oga
```

### Using Docker

```bash
# Build the Docker image
docker build -t oga-portal .

# Run the container
docker run -p 8081:8081 -v oga-data:/data oga-portal

# Or use docker-compose
docker-compose up -d

# View logs
docker-compose logs -f

# Stop the service
docker-compose down
```

## Example Usage

### Injecting Data from a Service

```bash
curl -X POST http://localhost:8081/api/oga/inject \
  -H "Content-Type: application/json" \
  -d '{
    "taskId": "927adaaa-b959-4648-880a-16508acafc12",
    "consignmentId": "cefda05e-3071-4e94-b001-328094e570a7",
    "serviceUrl": "http://your-service.com/api/tasks",
    "data": {
      "importerName": "ABC Corp",
      "documentNumber": "DOC-12345"
    }
  }'
```

### Fetching Pending Applications

```bash
curl http://localhost:8081/api/oga/applications?status=PENDING
```

### Reviewing an Application

```bash
curl -X POST http://localhost:8081/api/oga/applications/927adaaa-b959-4648-880a-16508acafc12/review \
  -H "Content-Type: application/json" \
  -d '{
    "decision": "APPROVED",
    "reviewerNotes": "All documents verified successfully"
  }'
```

## Development

### Project Structure

```
oga/
├── cmd/
│   └── server/
│       └── main.go          # Application entry point
├── handler.go               # HTTP handlers
├── service.go               # Business logic
├── store.go                 # Database operations
├── utils.go                 # Utility functions
├── go.mod                   # Go module definition
└── README.md               # This file
```

### Database Schema

The SQLite database contains a single `applications` table:

| Column | Type | Description |
|--------|------|-------------|
| task_id | UUID | Primary key, task identifier |
| consignment_id | UUID | Related consignment |
| service_url | VARCHAR(512) | URL to send review response to |
| data | TEXT (JSON) | Injected data from service |
| status | VARCHAR(50) | PENDING, APPROVED, or REJECTED |
| reviewer_notes | TEXT | Optional notes from reviewer |
| reviewed_at | DATETIME | Timestamp when reviewed |
| created_at | DATETIME | Record creation time |
| updated_at | DATETIME | Last update time |

## License

MIT