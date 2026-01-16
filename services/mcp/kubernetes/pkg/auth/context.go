package auth

import (
	"context"
)

type contextKey string

const (
	claimsContextKey contextKey = "claims"
)

// ContextWithClaims adds token claims to the context
func ContextWithClaims(ctx context.Context, claims *TokenClaims) context.Context {
	return context.WithValue(ctx, claimsContextKey, claims)
}

// ClaimsFromContext extracts token claims from the context
func ClaimsFromContext(ctx context.Context) (*TokenClaims, bool) {
	claims, ok := ctx.Value(claimsContextKey).(*TokenClaims)
	return claims, ok
}

// HasRole checks if the claims in the context have a specific role
func HasRole(ctx context.Context, role string) bool {
	claims, ok := ClaimsFromContext(ctx)
	if !ok {
		return false
	}
	
	for _, r := range claims.Roles {
		if r == role {
			return true
		}
	}
	return false
}