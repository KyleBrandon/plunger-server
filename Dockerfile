FROM alpine:latest

# Accept build-time variables for DATABASE_URL and PORT
ARG DATABASE_URL
ARG PORT

# Set environment variables
ENV DATABASE_URL=${DATABASE_URL}
ENV PORT=${PORT}

# Set the working directory inside the container
WORKDIR /app/

# Copy the pre-built Go binary from the GitHub Action build step
COPY plunger-server .

# Expose the port for the Go web server
EXPOSE ${PORT}

# Run the binary and pass in the necessary environment variables
CMD ["./plunger-server"]

