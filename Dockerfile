# Builder Image
FROM golang:1.20-alpine3.18 as builder
# Build deps
RUN apk --no-cache add git
# Setup
WORKDIR /app
COPY . ./
# Build
RUN go mod download
RUN go build -v -o main

# Run Image
FROM alpine:3.18
# necessary binaries
RUN apk add --no-cache bash curl file
# Setup
WORKDIR /app
COPY --from=builder /app/main /app/main

EXPOSE 8080/tcp
VOLUME ["/app/data"]
HEALTHCHECK --start-period=5s --interval=30s --timeout=5s --retries=5 \
  CMD curl -f http://localhost:8080/status || exit 1
#Run
CMD ["/app/main"]

