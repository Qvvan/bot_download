# Stage 1: Build the Go application
FROM golang:1.20-alpine as builder

# Set the working directory inside the container
WORKDIR /usr/src/app

# Copy go.mod and go.sum files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application code
COPY . .

# Build the Go application
RUN go build -o bot

# Stage 2: Create a lightweight runtime image
FROM alpine:latest

# Install necessary runtime dependencies
RUN apk add --no-cache ffmpeg yt-dlp

# Copy the built binary from the builder stage
COPY --from=builder /usr/src/app/bot /bot

# Expose the port if necessary (not used by Telegram bots)
# EXPOSE 8080

# Define the entrypoint command
CMD ["/bot"]
