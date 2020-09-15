# Start from golang base image
FROM golang:alpine as builder

ENV GO111MODULE=on

# Git is required for fetching the dependencies.
RUN apk update && apk add --no-cache git

# All these steps will be cached
RUN mkdir /auth_s
WORKDIR /auth_s
COPY . .

# Download all dependencies. Dependencies will be cached if the go.mod and the go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the working Directory inside the container
# COPY . .

# Build the Go app
#RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .
RUN go build -o main .

# Start a new stage from scratch
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage.
# Observe we also copied the .env file
COPY --from=builder /auth_s/main .
COPY --from=builder /auth_s/.env .

# Or we can set env variables by Docker ENV command
# as well if not defined in docker-compose

#ENV HTTP_PORT=8899
#ENV DB_HOST=127.0.0.1
#ENV DB_DRIVER=postgres
#ENV DB_USER=postgres
#ENV DB_PASSWORD=1234
#ENV DB_NAME=postgres
#ENV DB_PORT=5432

ENTRYPOINT ./main