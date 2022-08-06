# Builder Image
FROM golang:1.19-alpine3.16 as builder
# Build deps
RUN apk --no-cache add git
# Setup
WORKDIR /app
COPY . ./
# Build
RUN go mod download
RUN go build -v -o main

# Run Image
FROM alpine:3.16
# necessary binaries
RUN apk add --no-cache bash curl file 
# Setup
WORKDIR /app
COPY --from=builder /app/main /app/main
# Run
CMD ["/app/main"]

