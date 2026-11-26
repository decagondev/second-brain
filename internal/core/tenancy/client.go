package tenancy

import "context"

// Clients an ask can come from (SB-5305): the web app, a chat app, an MCP client or a
// read-only API token.
const (
	ClientWeb   = "web"
	ClientChat  = "chat"
	ClientMCP   = "mcp"
	ClientToken = "token"
)

type clientKey struct{}

// WithClient says which client ctx's request came from.
func WithClient(ctx context.Context, client string) context.Context {
	return context.WithValue(ctx, clientKey{}, client)
}

// ClientOf is the client ctx's request came from; the web app when none says.
func ClientOf(ctx context.Context) string {
	if c, ok := ctx.Value(clientKey{}).(string); ok && c != "" {
		return c
	}
	return ClientWeb
}
