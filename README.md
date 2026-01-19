# NSW
**_National Single Window for Trade Facilitation_**

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://www.apache.org/licenses/LICENSE-2.0)
![Last Commit](https://img.shields.io/github/last-commit/OpenNSW/nsw)

**NSW** is a centralized platform designed to streamline international trade by providing a single entry point for traders to interact with various Government Agencies (OGAs). By decoupling process orchestration from domain-specific data, NSW ensures a scalable and flexible ecosystem for managing consignments, certifications, and regulatory approvals.

**MVP Focus:** The initial release targets Tea and Coconut exports, focusing on high revenue-generating HS codes with shared processes. The system handles consignment-level workflows such as Country of Origin certificates and Export Licenses, while injecting pre-consignment requirements (Business Registration, Environmental Protection License, TIN) via one-time verification.

<p align="center">
  •   <a href="#why-nsw">Why NSW?</a> •
  <a href="#key-features">Key Features</a> •
  <a href="#getting-started">Getting Started</a> •
  <a href="#project-structure">Project Structure</a> •
  <a href="#system-architecture">System Architecture</a> •
  <a href="#delivery-milestones">Delivery Milestones</a> •
  <a href="#contributing">Contributing</a> •
  <a href="#license">License</a> •
</p>

## Why NSW?

The National Single Window eliminates the complexity of manual, fragmented trade processes. It acts as the "orchestrator" for trade, allowing traders to submit documentation once and track the entire lifecycle of their consignment across multiple agencies like the Tea Board and Coconut Board.

Key architectural benefits include:

* **State vs. Data Decoupling:** The Core Workflow Engine (CWE) manages the process state, while OGA Service Modules (OGA SM) handle specific domain data.
* **Isolated OGA Modules:** Each agency maintains its own database and portal logic, ensuring data sovereignty and system stability.
* **Interoperability:** Seamlessly integrates with the National Data Exchange (NDX) for common data and ASYCUDA for customs finalization.
* **One-Time Verification:** Injects pre-consignment requirements (BR, TIN, EPL) directly into the workflow to reduce repetitive submissions.

## Key Features

NSW offers powerful capabilities that streamline international trade processes:

| Feature | Status |
|---------|--------|
| **Core Workflow Engine (CWE)** – BPMN 2.0 based process orchestration managing process states without awareness of OGA-specific data schemas | In Progress |
| **NSW Core Portal** – Single entry point for Traders to initiate consignments and track global status | In Progress |
| **Tea Board Service Module** – Independent, pluggable unit with OGA-specific logic, database, and officer portal for Tea exports | In Progress |
| **Coconut Board Service Module** – Independent, pluggable unit with OGA-specific logic, database, and officer portal for Coconut products | In Progress |
| **One-Time Verification** – Pre-consignment document injection (Business Registration, TIN, Environmental Protection License) | In Progress |
| **Automated Notifications** – Email and SMS alerts via Postal service background worker | Planned |
| **ASYCUDA Interface** – Automated handoff to Customs system upon completion of all OGA approvals | Planned |
| **NDX Integration** – Fetching common data (BR number, VAT number) from external government providers | Under Evaluation |
| **Identity Provider Integration** – Centralized account management for Traders, OGA officers, and NSW Admins | In Progress |
| **Observability Stack** – Built-in OpenTelemetry for metrics, tracing, and logging | Planned |

## Getting Started

### Prerequisites
- Go 1.21 or higher
- Docker & Docker Compose
- Access to an OIDC-compliant Identity Provider (e.g., Asgardeo) for managing Traders, OGA officers, and NSW Admins
- PostgreSQL database (for NSW Core DB - stores process state)
- Access to Postal service (for Email/SMS notifications)
- Access to ASYCUDA system (for customs finalization)
- NDX API access (optional for MVP - integration timing under evaluation)

### Minimum Requirements
- 8GB of memory
- 4 CPU cores
- 50GB disk space

### Installation

1. Clone the repository and navigate to the project directory:

```bash
git clone https://github.com/OpenNSW/nsw.git
cd nsw
```

2. Set up environment variables:

```bash
# Copy example environment file
cp .env.example .env
# Edit .env with your configuration
```

3. Build the project:

```bash
make build
```

4. Run tests:

```bash
make test
```

5. Start services:

```bash
# Start all services using Docker Compose
docker-compose up -d
```

### Proposed Project Structure

The NSW monorepo is organized as follows:

```
nsw/
├── cmd/             # Entry points for the applications
│   ├── nsw-api/     # Trader & Admin Portal backend
│   ├── cwe-engine/  # Core Workflow Engine (BPMN handler)
│   └── worker-notify/ # Background worker for email/SMS
├── internal/        # Private code; can't be imported by other projs
│   ├── api/         # HTTP/gRPC handlers, middleware, and routing
│   ├── workflow/    # BPMN 2.0 state machine & CWE implementation
│   ├── integration/ # External service clients
│   │   ├── ndx/     # Client for NDX (reads from OGA providers)
│   │   └── asycuda/ # Customs interface logic
│   ├── platform/    # Shared infrastructure
│   │   ├── auth/    # IDP integration
│   │   ├── obs/     # OpenTelemetry setup (Metrics, Tracing, Logging)
│   │   └── database/ # NSW Core DB connections (Process state)
│   └── oga/         # Core logic for handling pluggable OGA modules
├── pkg/             # Public libs (importable by OGA SMs or others)
│   └── types/       # Common NSW data structures & interfaces
├── api/             # API definitions and schemas
│   ├── openapi/     # Swagger/OpenAPI 3.0 specs
│   └── proto/       # Protobuf definitions if using gRPC
├── oga-modules/     # Pluggable OGA Service Modules
│   ├── tea-board/   # Dedicated logic for Tea exports
│   └── coconut-board/ # Dedicated logic for Coconut products
└── deployments/     # Orchestration and CI/CD
    ├── docker/      # Multi-stage Dockerfiles
    └── scripts/     # Observability stack scripts
```

## System Architecture

The NSW system is built on a distributed microservices architecture to maintain high availability and modularity.

### Core Components

* **Identity Provider (IDP):** Manages all accounts for Traders, OGA officers, and NSW Admins. Provides centralized authentication and authorization.
* **NSW Core Portal:** Single entry point for Traders to initiate consignments and track global status across all OGAs.
* **Core Workflow Engine (CWE):** The "Brain" of the system that manages process states (e.g., "Waiting for Approval") without being aware of OGA-specific data schemas. Defined using BPMN 2.0 (Business Process Model and Notation)
* **OGA Service Modules (OGA SM):** Independent, pluggable units containing OGA-specific logic, databases, and officer portals. Each OGA SM maintains its own database as the source of truth for domain-specific data (e.g., Tea Blend Specifications).
* **National Data Exchange (NDX):** The bridge for reading common data from external government providers (BR number, VAT number, etc.). Integration timing is under evaluation for MVP
* **ASYCUDA Integration:** Automated handoff to Customs system upon completion of all OGA approvals.

### Architecture Principles

* **State vs. Data Decoupling:** The CWE manages Process State (e.g., "Waiting for Approval", "Approved", "Rejected"), while OGA Service Modules manage Domain Data (e.g., Tea Blend Specifications, Certificate details). This separation ensures the CWE remains agnostic to OGA-specific schemas.
* **Isolated OGA Modules:** Each agency maintains its own database and portal logic, ensuring data sovereignty and system stability. OGA-specific data resides in the SM's own database, helping decouple the system and maintain data isolation.
* **Dual Portal Views:** Each OGA portal provides two views: one consumed by OGA officers for review and approval, and another consumed by Traders to enter data required for the specific OGA process.
* **Source of Truth:** OGA-specific data resides in the SM's own database. Common data will be accessed through NDX (e.g., BR number, VAT number).
* **Callback-Based Workflow:** OGA SMs send success/fail callbacks to the CWE to advance the state machine. NSW fetches data about processes when needed directly from OGA SM (in future through NDX).

### The Consignment Journey

1. **Initialization:** Trader selects an HS Code through the NSW Core Portal; CWE triggers the relevant BPMN workflow.
2. **Submission:** Trader submits OGA-specific forms via the Portal directly to the OGA SM.
3. **Notification:** The OGA SM alerts the relevant officer via the Postal (Email) or SMS service.
4. **Review:** OGA Official reviews the submission within their isolated SM Portal.
5. **Decision:** OGA Officer approves or denies the request within their portal.
6. **Callback:** The SM sends a success/fail callback to the CWE to advance the state machine.
7. **Finalization:** Once all OGA states are complete, the CWE initiates the Customs (ASYCUDA) interface.

## Contributing

Thank you for wanting to contribute to the National Single Window project. Please see [CONTRIBUTING.md](docs/CONTRIBUTING.md) for more details.

## License 

Distributed under the Apache 2.0 License. See [LICENSE](LICENSE) for more information.