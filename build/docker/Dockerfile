FROM golang:1.23.1-alpine3.20 as builder

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor

RUN go mod verify
RUN go mod tidy -v

COPY . .

RUN CGO_ENABLED=0 GOARCH=amd64 GOOS=linux \
    go build -mod=vendor -ldflags "-s -w -extldflags '-static'" \
    -o ./main ./cmd/main.go
RUN apk add upx
RUN upx ./main

FROM alpine:3.20 AS runner
ENV MIGRATIONS_PATH=.

COPY ./internal/repo/db/migration ./
COPY --from=builder /app/main ./
RUN apk add --no-cache curl

EXPOSE 8080 8085 50050

ENTRYPOINT ["/main"]
