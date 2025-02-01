# Development stage
FROM golang:1.23.4-alpine

WORKDIR /app

RUN apk add --no-cache git && \
    go install github.com/cosmtrek/air@v1.49.0  # Using a specific version

# Copy Go module files
COPY go.mod .
COPY go.sum .

# Download dependencies
RUN go mod download

# Copy the source code
COPY . .

# Create .air.toml if it doesn't exist
RUN if [ ! -f .air.toml ]; then air init; fi

# Expose the port
EXPOSE 9999

# Use air for development
CMD ["air"]