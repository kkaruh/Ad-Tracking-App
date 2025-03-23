# Stage 1: Build the Go Application
FROM golang:1.22 AS go_app

WORKDIR /app

# Pass Arguments
ARG PROFILE
ARG SQL_SDN
ARG PORT
ARG REDIS_URI
ARG KAFKA_URI
ARG LOG_PATH

# Environment Variables
ENV PROFILE=${PROFILE}
ENV SQL_SDN=${SQL_SDN}
ENV PORT=${PORT}
ENV REDIS_URI=${REDIS_URI}
ENV KAFKA_URI=${KAFKA_URI}
ENV LOG_PATH=${LOG_PATH}

# Copy source code
COPY . .

# Print Profile for Debugging
RUN echo "Building image for profile --> ${PROFILE}"

# Download Dependencies
RUN go mod download

# Build the Go Application
RUN CGO_ENABLED=0 go build -o adds_app ./cmd/main.go

FROM alpine:latest

WORKDIR /app

COPY --from=go_app /app/adds_app .

EXPOSE ${PORT}

CMD ["/app/adds_app"]
