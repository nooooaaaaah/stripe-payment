# Stage 1: Build the Go binary
FROM golang:1.22.2-alpine AS build

# Install git for fetching dependencies
RUN apk add --no-cache git

# Set the current working directory inside the container
WORKDIR /app/server

# Copy go.mod and go.sum files to download dependencies
COPY server/go.mod server/go.sum ./

# Download dependencies
RUN go mod download

# Copy the source code into the container
COPY server/ ./

# Build the Go app statically
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /app/server/main .

# Stage 2: Create the runtime container
FROM alpine:latest

# Install certificates and bash for Alpine
RUN apk add --no-cache ca-certificates bash

# Set the current working directory inside the container
WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/server/main /app/server/main

# Copy static files from the client directory
COPY client /app/client

# Set environment variables
ENV STATIC_DIR /app/client
ENV PORT 4242

# Expose port 4242 to the outside world
EXPOSE 4242

# Set the entry point to execute the Go binary
ENTRYPOINT ["/app/server/main"]
