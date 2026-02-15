package files

import "fmt"

func (g *FileGenerator) generateDockerComposeFile() string {
	serviceName := g.Info.ServiceName
	if serviceName == "" {
		serviceName = "app"
	}
	port := g.Info.Port
	if port == "" {
		port = "8080"
	}
	imageName := g.Info.ImageName
	if imageName == "" {
		imageName = serviceName
	}

	return fmt.Sprintf(`version: '3.8'

services:
  %s:
    image: %s
    container_name: %s
    build:
      context: .
      dockerfile: Dockerfile
    ports:
      - "%s:%s"
    environment:
      - PORT=%s
      - ENVIRONMENT=production
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:%s/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 10s
    networks:
      - app-network

networks:
  app-network:
    driver: bridge

volumes:
  app-data:
`, serviceName, imageName, serviceName, port, port, port, port)
}
