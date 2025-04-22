package main

import (
	"log"
	"os"
	"strings"

	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/golang-jwt/jwt/v5"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Services map[string]struct {
		BaseURL string `yaml:"base_url"`
	} `yaml:"services"`
}

var (
	jwtSecret     []byte
	serviceConfig Config
)

func loadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &serviceConfig)
}

func jwtMiddleware(c *fiber.Ctx) error {
	tokenStr := c.Get("Authorization")
	if !strings.HasPrefix(tokenStr, "Bearer ") {
		return c.Status(fiber.StatusUnauthorized).SendString("missing or invalid token")
	}
	tokenStr = strings.TrimPrefix(tokenStr, "Bearer ")

	_, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return jwtSecret, nil
	})
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).SendString("unauthorized")
	}
	return c.Next()
}

func proxyHandler(c *fiber.Ctx) error {
	service := c.Params("service")
	svc, ok := serviceConfig.Services[service]
	if !ok {
		return c.Status(fiber.StatusBadGateway).SendString("unknown service")
	}

	targetURL := svc.BaseURL + "/" + c.Params("*")
	agent := fiber.AcquireAgent()
	defer fiber.ReleaseAgent(agent)
	req := agent.Request()

	req.Header.SetMethod(c.Method())
	req.SetRequestURI(targetURL)
	req.SetBodyRaw(c.Body())

	// Copy headers except auth
	c.Request().Header.VisitAll(func(k, v []byte) {
		if string(k) != "Authorization" {
			req.Header.SetBytesKV(k, v)
		}
	})

	var resp *fiber.Response
	err := agent.Do(req, resp)
	if err != nil {
		return c.Status(fiber.StatusBadGateway).SendString("proxy error")
	}

	c.Response().SetStatusCode(resp.StatusCode())
	c.Response().SetBodyStream(resp.BodyStream(), -1)
	return nil
}

func main() {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		log.Fatal("JWT_SECRET not set")
	}
	jwtSecret = []byte(secret)

	if err := loadConfig("/etc/passgo/config.yaml"); err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	app := fiber.New()
	app.Use(logger.New())
	app.Use("/proxy/:service/*", jwtMiddleware, proxyHandler)

	log.Println("Pass Go running on :8080")
	log.Fatal(app.Listen(":8080"))
}
