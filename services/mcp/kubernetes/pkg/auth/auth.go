package auth

import (
	"crypto/subtle"
	"fmt"
	"net/http"
	"strings"
	"time"
	
	"github.com/dgrijalva/jwt-go"
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	// JWTSecret is the secret key used to sign JWTs
	JWTSecret string
	
	// APIKeys is a map of API key IDs to API keys
	APIKeys map[string]string
	
	// TokenExpiration is the duration for which a token is valid
	TokenExpiration time.Duration
}

// TokenClaims represents JWT claims
type TokenClaims struct {
	jwt.StandardClaims
	UserID   string   `json:"userId"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
}

// NewAuthConfig creates a new authentication configuration
func NewAuthConfig(jwtSecret string, apiKeys map[string]string, tokenExpiration time.Duration) *AuthConfig {
	return &AuthConfig{
		JWTSecret:       jwtSecret,
		APIKeys:         apiKeys,
		TokenExpiration: tokenExpiration,
	}
}

// Authenticator is a middleware for authenticating requests
func (c *AuthConfig) Authenticator(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health endpoint
		if r.URL.Path == "/api/v1/health" {
			next.ServeHTTP(w, r)
			return
		}
		
		// Check for API key authentication
		if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {
			if c.validateAPIKey(apiKey) {
				next.ServeHTTP(w, r)
				return
			}
		}
		
		// Check for JWT authentication
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Unauthorized: Missing authentication", http.StatusUnauthorized)
			return
		}
		
		// Extract token from header
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenStr == authHeader {
			http.Error(w, "Unauthorized: Invalid token format", http.StatusUnauthorized)
			return
		}
		
		// Validate token
		claims, err := c.validateToken(tokenStr)
		if err != nil {
			http.Error(w, fmt.Sprintf("Unauthorized: %s", err.Error()), http.StatusUnauthorized)
			return
		}
		
		// Set claims in context for later use
		ctx := r.Context()
		ctx = ContextWithClaims(ctx, claims)
		
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// validateAPIKey validates an API key
func (c *AuthConfig) validateAPIKey(apiKey string) bool {
	for _, key := range c.APIKeys {
		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(key)) == 1 {
			return true
		}
	}
	return false
}

// validateToken validates a JWT token
func (c *AuthConfig) validateToken(tokenStr string) (*TokenClaims, error) {
	claims := &TokenClaims{}
	
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(c.JWTSecret), nil
	})
	
	if err != nil {
		return nil, err
	}
	
	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	
	return claims, nil
}

// GenerateToken generates a new JWT token
func (c *AuthConfig) GenerateToken(userID, username string, roles []string) (string, error) {
	claims := TokenClaims{
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(c.TokenExpiration).Unix(),
			IssuedAt:  time.Now().Unix(),
			Issuer:    "mcp-server",
		},
		UserID:   userID,
		Username: username,
		Roles:    roles,
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(c.JWTSecret))
}