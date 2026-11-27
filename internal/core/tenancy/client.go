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

type toolKey struct{}

// WithTool says which AI tool ctx's request came from (the token's client, SB-6006).
func WithTool(ctx context.Context, tool string) context.Context {
	return context.WithValue(ctx, toolKey{}, tool)
}

// ToolOf is the AI tool ctx's request came from, "" for none.
func ToolOf(ctx context.Context) string {
	t, _ := ctx.Value(toolKey{}).(string)
	return t
}

// ClientOf is the client ctx's request came from; the web app when none says.
func ClientOf(ctx context.Context) string {
	if c, ok := ctx.Value(clientKey{}).(string); ok && c != "" {
		return c
	}
	return ClientWeb
}
