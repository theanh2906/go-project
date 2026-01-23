# Go CLI - Backend Project Generator

A CLI tool for generating Go backend projects with Gin framework, inspired by Angular CLI.

## Installation

Build the CLI tool:

```bash
go build -o go-cli.exe
```

Or install it globally:

```bash
go install
```

## Usage

### Create a new project

```bash
go-cli new my-api
# or using short alias
go-cli n my-api
```

### Show version

```bash
go-cli version
# or
go-cli -v
```

### Show help

```bash
go-cli help
# or
go-cli -h
```

## Generated Project Structure

The CLI generates a Spring Boot-style project structure:

```
my-api/
├── cmd/                    # Command-line applications
├── internal/
│   ├── controllers/        # HTTP handlers (like @RestController)
│   ├── services/          # Business logic layer
│   └── models/            # Data models and DTOs
├── config/                # Configuration files
├── routes/                # Route definitions
├── main.go                # Application entry point
├── go.mod                 # Go modules file
├── .gitignore
└── README.md
```

## Features

- 🎨 Interactive CLI with colorful output (similar to Angular CLI)
- 📁 Spring Boot-style project structure
- 🚀 Gin framework integration
- 🎯 RESTful API scaffold with CRUD operations
- 📝 Sample controllers, services, and models
- ✅ Health check endpoint
- 🔧 Configuration management
- 📚 Auto-generated documentation

## Example

```bash
$ go-cli new my-awesome-api

🚀 Creating a new Go backend project...

? Project description: My awesome REST API
? Server port: 8080

✓ Creating project structure...
✓ Generating files...

✓ Project created successfully!

Next steps:
  cd my-awesome-api
  go mod tidy
  go run main.go

Happy coding! 🎉
```

## Generated API Endpoints

The generated project includes the following endpoints:

- `GET /health` - Health check
- `GET /api/users` - Get all users
- `GET /api/users/:id` - Get user by ID
- `POST /api/users` - Create new user
- `PUT /api/users/:id` - Update user
- `DELETE /api/users/:id` - Delete user

## License

MIT
