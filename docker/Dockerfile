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

# install make
RUN apk add make

# Build the Go web server binary
RUN make 

# Stage 2: Create a minimal image for running the application
FROM alpine:latest

# Add some default quality of life
RUN apk add vim curl 

# Accept build-time variables for DATABASE_URL and PORT
ARG DATABASE_URL
ARG PORT

# Set environment variables
ENV DATABASE_URL=${DATABASE_URL}
ENV PORT=${PORT}


# Set the working directory inside the container
WORKDIR /app

# Copy the pre-built Go binary from the GitHub Action build step
COPY --from=builder /app/bin/plunger-server /app/plunger-server
COPY --from=builder /app/scripts/entrypoint.sh /app/entrypoint.sh
COPY --from=builder /app/config/config_template.json /app/config_template.json

RUN chmod +x /app/entrypoint.sh

# Expose the port for the Go web server
EXPOSE ${PORT}
EXPOSE 6060

# Run the binary and pass in the necessary environment variables
ENTRYPOINT [ "/app/entrypoint.sh" ]

