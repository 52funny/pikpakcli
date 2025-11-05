FROM golang:1.21-alpine AS builder

RUN apk add --no-cache git
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
	go build -ldflags "-s -w" -o /usr/local/bin/pikpakcli ./main.go

FROM alpine:3.18
RUN apk add --no-cache ca-certificates
COPY --from=builder /usr/local/bin/pikpakcli /usr/local/bin/pikpakcli
WORKDIR /root

ENTRYPOINT ["/usr/local/bin/pikpakcli"]

