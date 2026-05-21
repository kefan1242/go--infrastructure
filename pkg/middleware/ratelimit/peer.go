package ratelimit

import (
	"context"

	"google.golang.org/grpc/peer"
)

// remotePeer reads the client address from the gRPC peer info.
// HTTP paths already cover X-Forwarded-For / X-Real-IP; this is the gRPC fallback.
func remotePeer(ctx context.Context) (string, bool) {
	if p, ok := peer.FromContext(ctx); ok && p.Addr != nil {
		return p.Addr.String(), true
	}
	return "", false
}
