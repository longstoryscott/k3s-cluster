# syntax=docker/dockerfile:1
FROM golang:1.21 as builder
WORKDIR /app
COPY . .
RUN go build -o proxy-api main.go

FROM gcr.io/distroless/base-debian11
WORKDIR /app
COPY --from=builder /app .
COPY ../config.yaml /etc/passgo/config.yaml
ENTRYPOINT ["/app/passgo"]
