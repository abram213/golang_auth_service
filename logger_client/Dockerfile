# Start from golang base image
FROM golang:alpine as builder

ENV GO111MODULE=on

# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

RUN mkdir /logger_c
WORKDIR /logger_c
COPY . .

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed
RUN go mod download

# Build the go app
RUN go build -o main .

# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the pre-built binary file and .env from the previous stage.
COPY --from=builder /logger_c/main .
COPY --from=builder /logger_c/.env .

ENTRYPOINT ./main