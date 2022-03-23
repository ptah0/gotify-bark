# Builder Image
FROM golang:1.18-alpine3.15 as builder
# Build deps
RUN apk --no-cache add build-base git
# Setup
WORKDIR /app
COPY . ./
# Build
RUN go get
RUN go build -v -o main

# Run Image
FROM alpine:3.15
# necessary binaries
RUN apk add --no-cache bash curl file 
# Setup
WORKDIR /app
COPY --from=builder /app/main /app/main
# Run
CMD ["/app/main"]
