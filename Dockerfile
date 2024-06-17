# Stage 1: Build the Go app
FROM golang:1.22.2-alpine AS build

# Install necessary build tools
RUN apk add --no-cache git

# Set the working directory for the build
WORKDIR /app/server

# Copy the Go modules manifest and download dependencies
COPY server/go.mod server/go.sum ./
RUN go mod download

# Copy the source code
COPY server/ ./

# Build the Go app
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/main .

# Stage 2: Create the final image
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk add --no-cache ca-certificates

# Set the working directory for the final container
WORKDIR /app

# Copy the built binary from the build stage
COPY --from=build /app/main /app/server/main


# Copy static files from client directory
COPY client /app/client

# Ensure the binary has the correct permissions
RUN chmod +x /app/server/main


# Expose the application port
EXPOSE 4242

# Command to run the executable
ENTRYPOINT ["/app/server/main"]
