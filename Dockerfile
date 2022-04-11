FROM golang:1.17 as builder
WORKDIR /app
COPY . .
RUN go build -o flow-rosetta ./cmd/flow-rosetta-server/main.go

FROM alpine
COPY --from=builder /app/flow-rosetta /app/flow-rosetta

RUN chmod +x /app/flow-rosetta
WORKDIR /app

EXPOSE 8080

CMD ["./rosetta-flow", "-c", "access-001.mainnet16.nodes.onflow.org:9000", "-p", "8080"]