# Multi-stage build template for a kris-* service.

FROM golang:1.25-bullseye AS builder

COPY . /src
WORKDIR /src

ENV GOPROXY=https://proxy.golang.org,direct

RUN GOPROXY=$GOPROXY go build -o ./bin/alpha ./cmd/alpha

FROM debian:bullseye-slim AS prod

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    netbase \
    && rm -rf /var/lib/apt/lists/* \
    && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app
EXPOSE 50051
EXPOSE 8080
EXPOSE 8081

CMD ["./alpha"]
