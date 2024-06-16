# Use the official Golang image to build the application
FROM golang:1.20 as build

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go.mod and go.sum to download dependencies
COPY server/go.mod server/go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY server/*.go ./

# Build the Go app
RUN go build -o main .

# Use a minimal image to serve the application
FROM debian:bullseye-slim

# Install CA certificates to handle HTTPS requests
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/main .

# Copy static files from client directory
COPY client /app/client

# Set environment variables
ENV STATIC_DIR /app/client
ENV PORT 4242

# Expose port 4242 to the outside world
EXPOSE 4242

# Run the application
CMD ["./main"]

