FROM golang:1.18.1-alpine as builder

# Set the Current Working Directory inside the container
WORKDIR /app

# Copy go mod and sum files
COPY go.mod go.sum ./

# Download all dependencies. Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download

# Copy the source from the current directory to the Working Directory inside the container
COPY . .

RUN go build -o twdataupdater ./cmd/dataupdater

######## Start a new stage from scratch #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /app/twdataupdater .

ENV APP_MODE=production
EXPOSE 8080

CMD ./twdataupdater
