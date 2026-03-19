FROM golang:1.24-alpine AS builder
WORKDIR /src
COPY . .
WORKDIR /src/examples/krakend-gateway
RUN CGO_ENABLED=0 go build -o /backend .

FROM alpine:3.21
COPY --from=builder /backend /backend
EXPOSE 8080
ENTRYPOINT ["/backend"]
