# sebuf Examples

This directory contains complete working examples demonstrating different aspects of sebuf usage.

## Examples Overview

### 📚 [Basic Examples](./basic/)
- **[Simple CRUD API](./basic/crud-api/)** - Task management API with all four tools
- **[Authentication Service](./basic/auth-service/)** - Login/logout with oneof helpers
- **[File Upload API](./basic/file-upload/)** - Handling binary data and metadata

### 🚀 [Framework Integration](./frameworks/)
- **[Gin Integration](./frameworks/gin/)** - Complete API with Gin HTTP framework
- **[Echo Integration](./frameworks/echo/)** - Echo framework with middleware
- **[Chi Router](./frameworks/chi/)** - Chi router with custom middleware

### 🏗️ [Advanced Patterns](./patterns/)
- **[Microservices](./patterns/microservices/)** - Multiple services with shared types
- **[API Gateway](./patterns/api-gateway/)** - Gateway aggregating multiple services
- **[Event-Driven](./patterns/events/)** - Using protobuf for event schemas

### 🚢 [Deployment](./deployment/)
- **[Docker](./deployment/docker/)** - Containerized deployment
- **[Kubernetes](./deployment/k8s/)** - K8s manifests and examples
- **[Serverless](./deployment/serverless/)** - AWS Lambda and Google Cloud Functions

### 🔧 [Development Workflows](./workflows/)
- **[Local Development](./workflows/local-dev/)** - Complete dev environment setup
- **[CI/CD Pipeline](./workflows/ci-cd/)** - GitHub Actions and GitLab CI
- **[Testing Strategies](./workflows/testing/)** - Unit, integration, and e2e testing

## Quick Start

Each example includes:
- Complete protobuf definitions
- Generated code (via `make generate`)
- Working server implementation
- Test suite
- Documentation and README

### Running an Example

```bash
# Choose any example
cd examples/basic/crud-api

# Install dependencies
go mod tidy

# Generate code
make generate

# Run the server
make run

# Run tests
make test
```

### Example Structure

```
example-name/
├── api/                    # Protobuf definitions
│   └── service.proto
├── cmd/                    # Main application
│   └── server/
│       └── main.go
├── internal/               # Implementation
│   └── service/
├── tests/                  # Test files
├── docs/                   # Generated documentation
├── Makefile               # Build automation
├── go.mod
└── README.md              # Example-specific docs
```

## Contributing Examples

Have a great example to share? We'd love to include it!

1. **Follow the structure** - Use the standard example layout
2. **Include tests** - Make sure your example is thoroughly tested
3. **Document thoroughly** - Add comprehensive README and comments
4. **Real-world focus** - Examples should solve actual problems

See [Contributing Guidelines](../../CONTRIBUTING.md) for more details.

## Example Categories

### By Complexity
- 🟢 **Beginner** - New to sebuf or protobuf
- 🟡 **Intermediate** - Familiar with basics, learning advanced patterns  
- 🔴 **Advanced** - Complex production scenarios

### By Use Case
- 🌐 **Web APIs** - REST-like HTTP APIs
- 📱 **Mobile Backend** - APIs optimized for mobile apps
- 🔗 **Microservices** - Service-to-service communication
- 📊 **Data Processing** - ETL and analytics workflows
- 🎮 **Real-time** - WebSocket and streaming APIs

## Popular Examples

Based on community feedback, these are the most helpful examples:

1. **[CRUD API](./basic/crud-api/)** - Perfect starting point
2. **[Gin Integration](./frameworks/gin/)** - Most popular Go HTTP framework
3. **[Authentication Service](./basic/auth-service/)** - Essential for most APIs
4. **[Docker Deployment](./deployment/docker/)** - Production deployment
5. **[Testing Strategies](./workflows/testing/)** - Testing sebuf-generated code

## Getting Help

- **Issues with examples**: [File a GitHub issue](https://github.com/SebastienMelki/sebuf/issues)
- **Questions**: [GitHub Discussions](https://github.com/SebastienMelki/sebuf/discussions)
- **Contributing**: See [Contributing Guide](../../CONTRIBUTING.md)

---

**Happy coding!** 🚀