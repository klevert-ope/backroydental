# Start from the official Golang image
FROM golang:1.23.2-alpine AS build

LABEL maintainer="Klevert Opee"
LABEL description="Backend API"
LABEL version=1.0

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies
RUN go mod download

# Copy the source code from the current directory to the Working Directory inside the container
COPY . .

# Build the Go app from the cmd directory
RUN go build -o cmd/main ./cmd

# Start a new stage from scratch
FROM alpine:3.20

# Create a group and user to run the application with non-root privileges
RUN addgroup -S app && adduser -S app -G app

# Set the Working Directory to /app and change the ownership to the app user
WORKDIR /app
RUN chown app:app /app

# Copy the Pre-built binary file from the previous stage
COPY --from=build /app/cmd/main /app/main

# Change the ownership of the binary file and migrations directory to the app user
RUN chown -R app:app /app/main

# Expose port 8000 to the outside world
EXPOSE 8000

# Switch to the app user
USER app

# Command to run the executable
ENTRYPOINT ["./main"]
