# HelaLedger API

## Overview
HelaLedger is a high-concurrency, double-entry ledger and wallet transfer backend service engineered in Go. Built to guarantee strict financial integrity under heavy parallel load, HelaLedger prevents race conditions, double-spending, and floating-point precision drift across wallet transactions.

Whether handling burst payment traffic or retrying dropped mobile network requests, HelaLedger enforces ACID compliance, deterministic row locking, and strict idempotency guarantees.

### Key Features
- 🔒 **Double-Entry Ledger Accounting** – Every transfer generates balanced debit and credit postings inside a single atomic database transaction.
- ⚡ **Concurrency & Race-Condition Protection** – Pessimistic row-level locking prevents negative balances during simultaneous wallet debits.
- 🔁 **Idempotent Request Handling** – Guarantees at-most-once execution for retried transfers via `Idempotency-Key` headers and payload hashing.
- 💰 **Fixed-Precision Monetary Arithmetic** – All monetary values are stored and calculated in minor units (`int64` cents) to eliminate floating-point rounding bugs.
- 🐳 **Containerized Environment** – One-command setup with Docker Compose for both the Go API server and PostgreSQL database.

### Tech Stack
- ⚙️ **Backend**: Go and Chi Router
- 🗄️ **Database**: PostgreSQL
- 🐳 **Containerization**: Docker & Docker Compose
- 🧪 **Testing**: Go `testing`, `net/http/httptest`, `testify`

### Architecture & Engineering Decisions

#### Database Locking Strategy (Pessimistic Locking)
HelaLedger uses **pessimistic locking (`SELECT ... FOR UPDATE`)** inside PostgreSQL transactions when executing wallet transfers:
- **Why Pessimistic over Optimistic?** Under high contention on a single hot wallet (e.g., a merchant receiving or disbursing simultaneous payouts), optimistic locking causes high retry churn and wasted CPU cycles due to frequent version conflicts. Pessimistic locking serializes access at the row level, ensuring the balance check and ledger posting happen atomically without failed retries.
- **Deadlock Prevention:** When locking multiple wallets in a single transfer (`from_wallet_id` and `to_wallet_id`), rows are always locked in **deterministic lexicographical UUID order** (`wallet_a < wallet_b`). This prevents circular wait conditions when two wallets transfer funds to each other simultaneously.

#### Network Timeouts & Failure Handling
- **Context Propagation & Query Timeouts:** Every incoming HTTP request binds a `context.Context` with strict deadlines passed down to the PostgreSQL driver. If a client disconnects mid-flight or a query exceeds the threshold, PostgreSQL automatically aborts and rolls back uncommitted ledger postings.
- **Safe Client Retries:** Because mobile clients frequently experience unstable connections, clients safely retry timed-out `POST /v1/transfers` requests using the same `Idempotency-Key`. If the original transaction committed before the connection dropped, HelaLedger returns the cached `201 Created` response without re-debiting the sender. If the original request is still in-flight, the API returns `409 Conflict` / `425 Too Early` to prevent duplicate processing.

---

## User Instructions

### API Endpoints

#### 1. Create a Transfer
`POST /v1/transfers`

**Headers:**
```http
Content-Type: application/json
Idempotency-Key: 550e8400-e29b-41d4-a716-446655440000
```

**Request Body:**
```json
{
  "from_wallet_id": "11111111-1111-1111-1111-111111111111",
  "to_wallet_id": "22222222-2222-2222-2222-222222222222",
  "amount": 2500,
  "currency": "KES"
}
```

**Responses:**
- `201 Created` – Transfer succeeded and ledger postings recorded.
- `400 Bad Request` – Insufficient funds, invalid amount (`<= 0`), or currency mismatch.
- `409 Conflict` – `Idempotency-Key` reused with a different request payload, or an identical request is currently in-flight.

#### 2. Get Wallet Balance
`GET /v1/wallets/:id/balance`

**Response (`200 OK`):**
```json
{
  "id": "11111111-1111-1111-1111-111111111111",
  "balance": 7500,
  "currency": "KES"
}
```

---

## Developer instructions
---
> [!WARNING]
> To run this project locally using Docker, you **MUST** have Docker and Docker Compose installed on your machine. 
> 
> * **Note on Docker Desktop:** While Docker Desktop is the standard choice for macOS and Windows Pro/Enterprise, it cannot be installed on certain systems (such as standard Linux distributions, Windows Home without WSL2, or machines lacking hardware virtualization support). If Docker Desktop is unavailable or unsupported on your system, follow the **Docker Engine** instructions below.

---

### Step 1: Install Docker & Docker Compose

Choose the installation method that matches your operating system and system capabilities:

#### Option A: Docker Desktop (Recommended for macOS & Windows Pro/Enterprise)
If your operating system supports it, Docker Desktop provides a complete containerization environment bundled with Docker Compose. Visit the official [Docker Get Started Guide](https://docs.docker.com/get-docker/).

#### Option B: Docker Engine & Compose (For Linux or Machines Without Docker Desktop Support)
If you are running **Linux** (e.g., Ubuntu, Debian, Fedora) or a machine where Docker Desktop is not supported, you can install the lightweight **Docker Engine** and **Docker Compose plugin** directly. Follow the official distribution-specific guide to install Docker Engine: [Install Docker Engine Guide](https://docs.docker.com/engine/install/).


### Installation & Local Setup

1. **Clone the repository**
```bash
git clone <repository-url>
cd helaLedger-api
```

2. **Configure environment variables**
Copy the example environment file and verify database credentials:
```bash
cp .env.example .env
```

This is the default database url for the dev environment
```env
# Replace the values in < > with your actual credentials. Don't include the angle brackets in the DATABASE_URL in your .env file
DATABASE_URL=postgres://<db-user>:<db-password>@<db-host>:<db-port>/<db-name>?sslmode=disable
```

> [!NOTE]
> The DATABASE_URL relies on the exact service name postgres defined in docker-compose.yaml as the host so the API container can successfully connect to the database container within the Docker network.


3. **Start services via Docker Compose**
```bash
docker-compose up --build -d
```
The API will be available at `http://localhost:8080`.

4. **Run the automated test suite**
Execute the unit and concurrency integration tests with the Go race detector enabled:
```bash
go test -v -race ./...
```

---

## Contributions
Contributions are welcome! To contribute to HelaLedger:
1. Fork the repository and create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```
2. Ensure all existing and new tests pass with `go test -race ./...`.
3. Submit a Pull Request detailing your changes and architectural rationale.

---

## Known Issues
1. **Single-Currency Transfers Only**: Cross-currency FX conversion between wallets with differing base currencies is not yet supported; transfers currently require matching source and destination currencies.
2. **In-Memory / DB Idempotency Cleanup**: Expired idempotency keys are retained indefinitely in PostgreSQL and currently require a scheduled background cleanup worker (TTL pruning) for long-term production scale.