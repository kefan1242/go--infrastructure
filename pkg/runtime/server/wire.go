package server

// No central ProviderSet is exported. Each service wires its own Register
// callbacks and calls NewGRPCServer / NewBizHTTPServer / NewOtherHTTPServer.
// That keeps pkg/runtime/server free of any business dependency: the
// service-specific Register functions become the providers.
