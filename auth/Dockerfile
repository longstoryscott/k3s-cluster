# Stage 1: Build the Go binary
FROM golang:1.24 AS builder

WORKDIR /app
COPY ./api .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o usrmgr .

# Stage 2: Minimal runtime image
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/usrmgr .

# Expose the port the server listens on
EXPOSE 3333

CMD ["./usrmgr"]
# Copy the configuration file
