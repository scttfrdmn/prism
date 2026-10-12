package course

import "context"

// callerKey is the context key carrying the acting user's ID, used for course authorization and
// audit. An unexported struct type can't collide with keys set by other packages (staticcheck
// SA1029), unlike the bare "caller_id" string this replaces.
type callerKey struct{}

// WithCaller returns a copy of ctx carrying the acting user's ID.
func WithCaller(ctx context.Context, callerID string) context.Context {
	return context.WithValue(ctx, callerKey{}, callerID)
}

// CallerFrom returns the acting user's ID carried by ctx, or "" if none was set.
func CallerFrom(ctx context.Context) string {
	id, _ := ctx.Value(callerKey{}).(string)
	return id
}
