package files

import "fmt"

func (g *FileGenerator) generateEnvExample() string {
	port := g.Info.Port
	if port == "" {
		port = "8080"
	}
	serviceName := g.Info.ServiceName
	if serviceName == "" {
		serviceName = "app"
	}

	return fmt.Sprintf(`# Application
APP_NAME=%s
ENVIRONMENT=development
PORT=%s

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb
DB_USER=postgres
DB_PASSWORD=your_password_here
DB_SSL_MODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your_jwt_secret_here
JWT_EXPIRY=24h

# Logging
LOG_LEVEL=debug
LOG_FORMAT=json

# External Services
API_KEY=your_api_key_here
API_URL=https://api.example.com
`, serviceName, port)
}
