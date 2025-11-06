FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git make

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o forwarder ./cmd/forwarder

FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

COPY --from=builder /app/forwarder .
COPY --from=builder /app/config.yaml.example ./config.yaml.example

RUN mkdir -p /root/config

EXPOSE 8080

CMD ["./forwarder"]
