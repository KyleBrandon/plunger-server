# Stage 1: Build the Go binary
FROM golang:1.22-alpine AS builder

# Set environment variables
ENV GOOS=linux GOARCH=arm GOARM=7

# Set the working directory inside the container
WORKDIR /app

# Copy the Go module files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the source code into the container
COPY . .

# Build the Go web server binary
RUN go build -o /plunger-server


# Stage 2: Create a minimal image for running the application
FROM alpine:latest

# Accept build-time variables for DATABASE_URL and PORT
ARG DATABASE_URL
ARG PORT

# Set environment variables
ENV DATABASE_URL=${DATABASE_URL}
ENV PORT=${PORT}

# Set the working directory inside the container
WORKDIR /app

# Copy the pre-built Go binary from the GitHub Action build step
COPY --from=builder /plunger-server .
COPY config.json .

# Expose the port for the Go web server
EXPOSE ${PORT}

# Run the binary and pass in the necessary environment variables
CMD ["./plunger-server"]

