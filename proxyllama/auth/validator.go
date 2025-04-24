package auth

import (
	"context"
	"log"
	"net/http"
	"proxyllama/config"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// contextKey is a custom type for context keys to avoid collisions
type contextKey string

// Key for user ID in context
const UserIDKey contextKey = "user_id"

func NewValidator(ctx context.Context, jwksUri string) keyfunc.Keyfunc {
	k, err := keyfunc.NewDefaultCtx(ctx, []string{jwksUri}) // Context is used to end the refresh goroutine.
	if err != nil {
		log.Fatalf("Failed to create a keyfunc.Keyfunc from the server's URL.\nError: %s", err)
	}
	return k
}

func WithAuth(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	conf := config.GetConfig()

	k := NewValidator(c.UserContext(), conf.Auth.JWKSURI)

	if authHeader == "" {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	token, err := jwt.Parse(tokenStr, k.Keyfunc)
	if err != nil {
		log.Printf("Failed to parse token: %v", err)
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	if !token.Valid {
		log.Printf("Invalid token")
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	claims := token.Claims.(jwt.MapClaims)
	userID, ok := claims["sub"].(string)
	if !ok {
		return c.Status(http.StatusUnauthorized).JSON(fiber.Map{
			"error": "Unauthorized",
		})
	}
	ctx := context.WithValue(c.Context(), UserIDKey, userID)
	c.SetUserContext(ctx)
	c.Locals("user_id", userID)
	c.Set("X-User-ID", userID)

	// Generate and set request ID
	rid := uuid.New().String()
	c.Set("X-Request-ID", rid)
	c.Locals("request_id", rid)
	log.Printf("Request ID: %s, User ID: %s", rid, userID)
	return c.Next()
}
