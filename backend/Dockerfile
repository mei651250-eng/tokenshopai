# 多阶段构建
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o tokenhub ./cmd/server/

# 运行阶段
FROM alpine:3.18
RUN apk add --no-cache ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/tokenhub .

EXPOSE 8080
CMD ["./tokenhub"]
