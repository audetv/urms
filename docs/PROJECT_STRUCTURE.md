# URMS-OS Project Structure
**Single Repository (Monorepo) Approach**

## 🎯 Root Structure
```text
urms/
├── 📁 backend/ # Go backend services
├── 📁 frontend/ # Vue 3 frontend
├── 📁 shared/ # Shared types/configs
├── 📁 docs/ # Documentation
├── 📁 deployments/ # Docker, k8s, scripts
├── 📁 scripts/ # Build/validation scripts
└── 📄 ARCHITECTURE_PRINCIPLES.md
```
## 🔧 Backend Structure (Go)
```text
backend/
├── 📁 cmd/ # Entry points
│ ├── api/ # HTTP API server
│ ├── worker/ # Background workers
│ └── migration/ # Database migrations
├── 📁 internal/ # Private Go code
│ ├── 📁 core/ # BUSINESS CORE
│ │ ├── domain/ # Entities, VO, Aggregates
│ │ │ ├── ticket.go
│ │ │ ├── customer.go
│ │ │ └── valueobjects/
│ │ ├── ports/ # INTERFACES
│ │ │ ├── repositories.go
│ │ │ ├── services.go
│ │ │ └── gateways.go
│ │ └── services/ # Business logic
│ │ ├── ticket_service.go
│ │ └── classification_service.go
│ └── 📁 infrastructure/ # EXTERNAL ADAPTERS
│ ├── http/ # Web layer
│ │ ├── handlers/ # HTTP handlers
│ │ ├── middleware/ # Auth, logging
│ │ └── routers/ # Gin/Fiber routers
│ ├── persistence/ # Data layer
│ │ ├── postgres/ # PostgreSQL repos
│ │ ├── redis/ # Cache repos
│ │ └── migrations/ # DB migration files
│ └── external/ # External services
│ ├── email/ # Email providers
│ ├── ai/ # AI model services
│ └── messaging/ # Telegram, Webhooks
├── 📁 pkg/ # Public Go libraries
│ ├── config/ # Configuration loader
│ ├── logger/ # Structured logging
│ └── health/ # Health checks
└── 📁 api/ # API specifications
├── openapi/ # OpenAPI specs
└── protobuf/ # gRPC proto files
```
## 🎨 Frontend Structure (Vue 3)
```text
frontend/
├── 📁 src/
│ ├── 📁 core/ # Frontend business logic
│ │ ├── domain/ # Frontend entities
│ │ ├── ports/ # Frontend interfaces
│ │ └── services/ # Business services
│ ├── 📁 infrastructure/ # Frontend adapters
│ │ ├── api/ # HTTP client adapters
│ │ ├── storage/ # Local storage
│ │ └── plugins/ # Vue plugins
│ ├── 📁 ui/ # Presentation layer
│ │ ├── components/ # Reusable components
│ │ ├── views/ # Page components
│ │ └── layouts/ # Layout components
│ └── 📁 shared/ # Shared utilities
└── vite.config.ts
```
## 🔗 Shared Configuration
```text
shared/
├── 📁 types/ # Shared TypeScript types
├── 📁 config/ # Shared configuration
└── 📁 contracts/ # API contract
```
## 🐳 Deployment & Development
```text
deployments/
├── 📁 docker/ # Docker configurations
├── 📁 kubernetes/ # K8s manifests
├── 📁 docker-compose/ # Local development
└── 📁 scripts/ # Deployment scripts
```
```text
scripts/
├── validate_architecture.sh # Architecture checks
├── setup_dev_env.sh # Dev environment
└── codegen.sh # Code generation
```