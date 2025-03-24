# Stage 1: Build the Go Application
FROM golang:1.22 AS go_app

WORKDIR /app

COPY . .

RUN echo "Building image"

RUN go mod download

RUN CGO_ENABLED=0 go build -o adds_app ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=go_app /app/adds_app .

EXPOSE 8080

CMD ["/app/adds_app"]
