# syntax=docker/dockerfile:1
#
# Build from the **repo root** (the go.mod `replace pkg => ../pkg` needs it):
#
#   docker build -f kris-beta/Dockerfile -t kris/beta:dev .

FROM golang:1.25-bookworm AS builder

WORKDIR /src
COPY . .

ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILDTIME=unknown

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOFLAGS=-trimpath

WORKDIR /src/kris-beta
RUN go build \
      -ldflags "-s -w \
                -X main.Name=kris-beta \
                -X main.Version=${VERSION} \
                -X main.Commit=${COMMIT} \
                -X main.BuildTime=${BUILDTIME}" \
      -o /out/beta ./cmd/beta

FROM gcr.io/distroless/static-debian12:nonroot AS prod

COPY --from=builder /out/beta /app/beta

USER nonroot:nonroot
WORKDIR /app

EXPOSE 8082
EXPOSE 8083

ENTRYPOINT ["/app/beta"]
