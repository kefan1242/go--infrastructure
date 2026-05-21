# syntax=docker/dockerfile:1
#
# Build from the **repo root** (the go.mod `replace pkg => ../pkg` needs it):
#
#   docker build -f kris-gamma/Dockerfile -t kris/gamma:dev .

FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDTIME=unknown

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS=-trimpath

WORKDIR /src/kris-gamma
RUN go build \
      -ldflags "-s -w \
                -X main.Name=kris-gamma \
                -X main.Version=${VERSION} \
                -X main.Commit=${COMMIT} \
                -X main.BuildTime=${BUILDTIME}" \
      -o /out/gamma ./cmd/gamma

FROM gcr.io/distroless/static-debian12:nonroot AS prod

COPY --from=builder /out/gamma /app/gamma

USER nonroot:nonroot
WORKDIR /app

EXPOSE 50053
EXPOSE 8085

ENTRYPOINT ["/app/gamma"]
