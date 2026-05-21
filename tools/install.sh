#!/usr/bin/env bash
# Pinned codegen toolchain installer.
#
# Usage:
#   bash tools/install.sh        # or `make tools-install`
#
# Design notes:
# - Each `go install <pkg>@<version>` runs independently so that one tool's
#   transitive deps (e.g. genproto) don't drag the others down.
# - Versions are centralized here; bump in one place when upgrading.
# - Per-service `make init` (uses @latest) still works, but `make tools-install`
#   pins everyone to the same version.
set -euo pipefail

PROTOC_GEN_GO_VERSION="v1.36.11"
PROTOC_GEN_GO_GRPC_VERSION="v1.6.2"
KRATOS_VERSION="v2.8.0"
PROTOC_GEN_GO_HTTP_VERSION="v2.8.0"
PROTOC_GEN_GO_ERRORS_VERSION="v2.8.0"
PROTOC_GEN_OPENAPI_VERSION="v0.7.0"
PROTOC_GEN_VALIDATE_VERSION="v1.1.0"
WIRE_VERSION="v0.6.0"
GOFUMPT_VERSION="v0.7.0"
GOIMPORTS_VERSION="latest"      # x/tools moves fast; not pinned
GOLANGCI_LINT_VERSION="v1.61.0"
MOCKGEN_VERSION="v0.6.0"
BUF_VERSION="v1.47.2"
GOLANG_MIGRATE_VERSION="v4.18.1"
GOVULNCHECK_VERSION="latest"

install_one() {
  local pkg=$1
  echo "  -> $pkg"
  go install "$pkg" 2>&1 | grep -v "^go: downloading" || true
}

echo "==> Installing pinned codegen tools (cache dir: $(go env GOPATH)/bin)"

install_one "google.golang.org/protobuf/cmd/protoc-gen-go@${PROTOC_GEN_GO_VERSION}"
install_one "google.golang.org/grpc/cmd/protoc-gen-go-grpc@${PROTOC_GEN_GO_GRPC_VERSION}"
install_one "github.com/go-kratos/kratos/cmd/kratos/v2@${KRATOS_VERSION}"
install_one "github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@${PROTOC_GEN_GO_HTTP_VERSION}"
install_one "github.com/go-kratos/kratos/cmd/protoc-gen-go-errors/v2@${PROTOC_GEN_GO_ERRORS_VERSION}"
install_one "github.com/google/gnostic/cmd/protoc-gen-openapi@${PROTOC_GEN_OPENAPI_VERSION}"
install_one "github.com/envoyproxy/protoc-gen-validate@${PROTOC_GEN_VALIDATE_VERSION}"
install_one "github.com/google/wire/cmd/wire@${WIRE_VERSION}"
install_one "mvdan.cc/gofumpt@${GOFUMPT_VERSION}"
install_one "golang.org/x/tools/cmd/goimports@${GOIMPORTS_VERSION}"
install_one "github.com/golangci/golangci-lint/cmd/golangci-lint@${GOLANGCI_LINT_VERSION}"
install_one "go.uber.org/mock/mockgen@${MOCKGEN_VERSION}"
install_one "github.com/bufbuild/buf/cmd/buf@${BUF_VERSION}"
install_one "github.com/golang-migrate/migrate/v4/cmd/migrate@${GOLANG_MIGRATE_VERSION}"
install_one "golang.org/x/vuln/cmd/govulncheck@${GOVULNCHECK_VERSION}"

echo ""
echo "==> Done. Tools installed at $(go env GOPATH)/bin"
echo "    Verify any one:  $(go env GOPATH)/bin/wire --help"
