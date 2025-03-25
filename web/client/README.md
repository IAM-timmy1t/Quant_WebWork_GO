📦 QUANT_WW_GO

A high-performance, modular, and secure bridge framework for cross-language communication between Go and web-based systems, featuring:

    ✅ Go backend with gRPC, REST, GraphQL

    ✅ React (TypeScript) frontend for monitoring & interaction

    ✅ Prometheus + Grafana for full observability

    ✅ Dockerized, with CI/CD via GitHub Actions

📁 Project Structure

QUANT_WW_GO/
├── cmd/                  # Main Go entry points
├── internal/             # Go internal logic (API, bridge, metrics, etc.)
├── deployments/          # Monitoring configs (Grafana, Prometheus)
├── tests/                # Go integration & verification tests
├── web/
│   └── client/           # React frontend (Vite + TypeScript)
├── .github/              # GitHub Actions CI/CD workflows
├── Dockerfile            # Multi-stage build (Go backend + frontend)
├── docker-compose.yml    # Full deployment stack

🚀 Quick Start (Full Stack)
1. 📦 Install Dependencies (Go + Frontend)
<details> <summary><strong>🧪 Go Backend</strong></summary>

go mod tidy
go mod verify

</details> <details> <summary><strong>🎨 Frontend (React)</strong></summary>

cd web/client
npm install

</details>
2. 🔨 Build & Launch (with Docker)

docker-compose up --build

This launches:

    Go backend (cmd/server/main.go)

    React frontend (auto-built via Vite)

    Prometheus + Grafana (monitoring)

    gRPC-Web services + REST + GraphQL

    WebSockets + Token Analyzer

    ✅ App runs at: http://localhost:8080
    ✅ Grafana at: http://localhost:3000 (user: admin / pass: admin)

3. 🧪 Run Tests
<details> <summary>✅ Go Unit Tests</summary>

cd internal/bridge/adapters
go test ./...

</details> <details> <summary>✅ React Tests</summary>

cd web/client
npm run test        # Jest unit tests
npx cypress open    # E2E tests (Bridge Verification)

</details> <details> <summary>✅ Backend Verification</summary>

cd tests
go run bridge_verification.go manual

</details>
4. 📊 Monitoring Setup

    Setup with:

./setup_monitoring.sh

    Then visit:

        Grafana: http://localhost:3000

        Prometheus: http://localhost:9090

🧠 Features

    🔌 Bridge Module: Plug-and-play protocol support (gRPC, WebSocket, REST)

    🌐 API Layer: REST + GraphQL with middleware, validation, CORS

    📊 Monitoring: Custom Prometheus metrics from Go and React

    🔐 Security: Rate limiting, token validation, risk scoring

    ⚙️ Discovery: Service registry, health checking

    🔍 Test Coverage: Jest + Cypress + Go unit tests

    📦 CI/CD: GitHub Actions auto builds, runs tests, lints code

🔁 Development Workflow
Task	Command
Build project	docker-compose build
Start project	docker-compose up
Frontend dev server	cd web/client && npm run dev
Format frontend code	npm run format
Lint frontend	npm run lint
Run backend tests	go test ./...
Run frontend unit tests	npm run test
Run Cypress e2e tests	npx cypress open
🧩 Useful Scripts
Script	Description
setup_monitoring.sh	Sets up Prometheus + Grafana
update_deps.bat	Updates project Go modules
verify_monitoring.bat	Verifies metrics and dashboards
setup.sh (frontend)	Installs React dependencies
🛡️ Security

    Content-Security-Policy headers in middleware.go

    Sanitization of inputs + DOMPurify for frontend

    JWT + API key validation via security_handlers.go

    Rate limiter: firewall/rate_limiter.go

🔍 Troubleshooting
Problem	Fix
React build not visible	Ensure vite output path matches Go static folder
Metrics not loading	Check Prometheus targets in prometheus.yml
Grafana shows blank	Ensure dashboards loaded from provisioning/
Missing Go deps	Run go mod tidy
❤️ Contributors

    Backend Lead: You

    Frontend Integration: You

    Bridge Protocol Architect: You

    Monitoring/CI/CD Engineer: You

    If others contribute, update this section accordingly.

📝 License

MIT (or update to your organization’s license policy)
✅ Final Checklist

Frontend lives in web/client

Monitoring configs live in deployments/monitoring

Docker Compose integrates backend, frontend, observability

CI/CD configured via .github/workflows/ci-cd.yml
