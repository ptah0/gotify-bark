# Builder Image
FROM golang:1.27-alpine3.24 AS builder
# Build deps
RUN apk --no-cache add git
# Setup
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
# Build
RUN go build -v -o gotify-bark ./cmd/gotify-bark

# Run Image
FROM alpine:3.24 AS runtime
# necessary binaries
RUN apk add --no-cache curl
# Setup
WORKDIR /app
COPY --from=builder /app/gotify-bark /app/gotify-bark

EXPOSE 8080/tcp
HEALTHCHECK --start-period=5s --interval=30s --timeout=5s --retries=5 \
  CMD curl -f http://localhost:8080/status || exit 1
#Run
CMD ["/app/gotify-bark"]
