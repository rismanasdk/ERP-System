FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
ENV CGO_ENABLED=0 GOOS=linux
RUN go build -o /usr/local/bin/backend ./cmd/api

FROM scratch AS runtime
COPY --from=builder /usr/local/bin/backend /backend
EXPOSE 8080
CMD ["/backend"]