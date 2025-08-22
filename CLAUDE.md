# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

GPT-Load is a high-performance, enterprise-grade AI API transparent proxy service written in Go, with a Vue 3 frontend. It provides intelligent key management, load balancing, and comprehensive monitoring for integrating multiple AI services (OpenAI, Google Gemini, Anthropic Claude).

## Development Commands

### Backend Development (Go)
```bash
# Build frontend and run server
make run

# Development mode with race detection
make dev

# Manual build and run
go run ./main.go

# Show available Make targets
make help
```

### Frontend Development (Vue 3)
```bash
# Navigate to web directory first
cd web

# Development server
npm run dev

# Build for production
npm run build

# Linting and formatting
npm run lint          # Auto-fix ESLint issues
npm run lint:check    # Check ESLint without fixing
npm run format        # Auto-format with Prettier
npm run format:check  # Check formatting without fixing
npm run type-check    # TypeScript type checking

# Combined checks
npm run check-all     # Run lint:check + format:check + type-check
```

### Docker Development
```bash
# Quick start with Docker
docker run -d --name gpt-load \
    -p 3001:3001 \
    -e AUTH_KEY=sk-123456 \
    -v "$(pwd)/data":/app/data \
    ghcr.io/tbphp/gpt-load:latest

# Docker Compose (recommended)
docker compose up -d
docker compose logs -f
docker compose down && docker compose up -d
```

## Architecture Overview

### Backend Architecture (Go)
- **Dependency Injection**: Uses `go.uber.org/dig` for clean dependency management
- **Web Framework**: Gin framework for HTTP routing and middleware
- **Database Support**: Multi-database support (SQLite, MySQL, PostgreSQL) via GORM
- **Caching**: Redis for distributed caching and coordination (optional)
- **Configuration**: Two-tier config system (static env vars + dynamic hot-reload settings)

### Key Backend Components
- **`internal/app/`**: Main application startup and lifecycle management
- **`internal/proxy/`**: Core proxy server handling AI API requests
- **`internal/channel/`**: AI service adapters (OpenAI, Gemini, Anthropic)
- **`internal/keypool/`**: Intelligent key pool management and validation
- **`internal/services/`**: Business logic services (key management, logging, etc.)
- **`internal/store/`**: Storage abstraction layer (Redis/Memory)
- **`internal/config/`**: Configuration management with hot-reload support

### Frontend Architecture (Vue 3)
- **Framework**: Vue 3 with TypeScript
- **UI Library**: Naive UI for consistent component design
- **Build Tool**: Vite for fast development and building
- **State Management**: Composable-based state with VueUse
- **API Layer**: Axios for HTTP client with structured API modules

### Key Frontend Structure
- **`src/views/`**: Main application pages (Dashboard, Settings, Logs)
- **`src/components/`**: Reusable Vue components organized by feature
- **`src/api/`**: API service modules for backend communication
- **`src/services/`**: Frontend service layer (auth, version management)
- **`src/utils/`**: Utility functions and helpers

## Configuration System

### Static Configuration (.env)
Server, database, Redis, and infrastructure settings that require restart to change.

### Dynamic Configuration (Hot-reload)
System settings and group configurations stored in database, can be modified via web interface without restart.

## Key Features to Understand

### Proxy Routing
Requests are routed via: `http://localhost:3001/proxy/{group_name}/{original_api_path}`
- Maintains full compatibility with original AI API formats
- Group-based routing with automatic load balancing
- Transparent proxy preserving all headers and request/response formats

### Smart Key Management
- Automatic key validation and blacklist management
- Group-based key organization with configurable settings
- Background health checking and recovery mechanisms
- Batch key validation and import capabilities

### Multi-Database Support
The application automatically detects and configures database connections based on the `DATABASE_DSN` environment variable format.

## Testing and Quality

### Backend Testing
```bash
# Run with race detection
go run -race ./main.go

# Manual testing scripts
python batch-key-tester.py  # Batch key validation testing
python key-analyzer.py     # Key analysis and diagnostics
```

### Frontend Testing
```bash
cd web
npm run type-check    # TypeScript validation
npm run lint:check    # ESLint validation
npm run format:check  # Prettier formatting validation
```

## Important Notes

- The embedded web UI is built into the Go binary via `//go:embed web/dist`
- All AI service formats are preserved transparently (OpenAI, Gemini, Anthropic)
- Configuration changes in the web interface apply immediately without restart
- The system supports both standalone and cluster deployment modes
- Default admin access uses `sk-123456` - change `AUTH_KEY` in production