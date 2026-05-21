# syntax=docker/dockerfile:1
#
# kris-alpha image. Build from the **repository root**:
#
#   docker build -f kris-alpha/Dockerfile \
#     --build-arg VERSION=$(git describe --tags --always) \
#     --build-arg COMMIT=$(git rev-parse HEAD) \
#     --build-arg BUILDTIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
#     -t kris/alpha:dev .
#
# The build context must include `pkg/` because kris-alpha's go.mod uses a
# `replace github.com/kris/go-infrastructure/pkg => ../pkg` directive.
#
# Runtime base: distroless static — no shell, no package manager, runs as
# nonroot. Static binary because CGO is off; mongo/redis/mysql drivers are
# pure Go.

FROM golang:1.26-bookworm AS builder

WORKDIR /src
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDTIME=unknown

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS=-trimpath

WORKDIR /src/kris-alpha
RUN go build \
      -ldflags "-s -w \
                -X main.Name=kris-alpha \
                -X main.Version=${VERSION} \
                -X main.Commit=${COMMIT} \
                -X main.BuildTime=${BUILDTIME}" \
      -o /out/alpha ./cmd/alpha

FROM gcr.io/distroless/static-debian12:nonroot AS prod

COPY --from=builder /out/alpha /app/alpha

USER nonroot:nonroot
WORKDIR /app

EXPOSE 50051
EXPOSE 8080
EXPOSE 8081

ENTRYPOINT ["/app/alpha"]
