// Command api is the demo shop's backend for play.anectico.com.
//
// It is a small but genuinely instrumented service: every request is traced
// through the Anectico Go SDK's Gin integration, the browser's identity rides in
// on W3C baggage, and failures are captured as errors. The checkout behind the
// new-checkout flag (checkout_v2.go) carries a deliberate bug, so the demo
// always has a real issue that only some people hit.
//
// Env:
//
//	ANECTICO_ENDPOINT     ingestion endpoint, e.g. http://localhost:4318      [required]
//	ANECTICO_API_KEY      native_ingest key                                  [required]
//	ANECTICO_CAPTURE_KEY  server_capture key, for product events             [required]
//	DEMO_RELEASE          release name stamped on telemetry                  [default "demo-shop@dev"]
//	DEMO_ADDR             listen address                                     [default ":8090"]
//	DEMO_STATIC_DIR       built storefront to serve from the same origin     [optional]
//
// The storefront reads its public browser configuration from GET /api/config:
//
//	DEMO_BROWSER_ENDPOINT     ingestion endpoint as the browser reaches it   [default ANECTICO_ENDPOINT]
//	DEMO_BROWSER_INGEST_KEY   native_ingest key for the browser              [required]
//	DEMO_BROWSER_CAPTURE_KEY  browser_capture key                            [required]
//	DEMO_BROWSER_REPLAY_KEY   key with only replay:write                     [optional]
//	DEMO_BROWSER_FLAGS_KEY    key with only flags:read                       [optional]
//	DEMO_APP_ENDPOINT         Anectico app endpoint for flag decisions       [required with the flags key]
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	anectico "github.com/anectico/anectico/sdks/go"
	anecticogin "github.com/anectico/anectico/sdks/go/instrumentation/gin"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	endpoint, ingestKey, captureKey := os.Getenv("ANECTICO_ENDPOINT"), os.Getenv("ANECTICO_API_KEY"), os.Getenv("ANECTICO_CAPTURE_KEY")
	if endpoint == "" || ingestKey == "" || captureKey == "" {
		return errors.New("ANECTICO_ENDPOINT, ANECTICO_API_KEY and ANECTICO_CAPTURE_KEY are required")
	}
	release := envOr("DEMO_RELEASE", "demo-shop@dev")

	client, err := anectico.New(
		anectico.WithEndpoint(endpoint),
		anectico.WithProtocol("http"),
		anectico.WithAPIKey(ingestKey),
		anectico.WithCaptureAPIKey(captureKey),
		anectico.WithServiceName("shop-api"),
		anectico.WithServiceVersion(release),
		anectico.WithEnvironment("demo"),
		anectico.WithErrorRelease(release),
		anectico.WithErrorEnvironment("demo"),
		anectico.WithAttachStackTrace(true),
	)
	if err != nil {
		return fmt.Errorf("anectico client: %w", err)
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start anectico: %w", err)
	}

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	// The storefront is served from this origin and the demo has no other
	// callers, so the browser's identity baggage can be trusted here.
	router.Use(anecticogin.MiddlewareWithOptions(client, anecticogin.Options{
		TrustIncomingIdentity: true,
		SkipPaths:             []string{"/api/health"},
	}))
	shop := &shop{client: client}
	router.GET("/api/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })
	browser := browserConfig{
		Endpoint:    envOr("DEMO_BROWSER_ENDPOINT", endpoint),
		AppEndpoint: os.Getenv("DEMO_APP_ENDPOINT"),
		Release:     release,
		Keys: browserKeys{
			Ingest: os.Getenv("DEMO_BROWSER_INGEST_KEY"), Capture: os.Getenv("DEMO_BROWSER_CAPTURE_KEY"),
			Replay: os.Getenv("DEMO_BROWSER_REPLAY_KEY"), Flags: os.Getenv("DEMO_BROWSER_FLAGS_KEY"),
		},
	}
	if browser.Keys.Ingest == "" || browser.Keys.Capture == "" {
		return errors.New("DEMO_BROWSER_INGEST_KEY and DEMO_BROWSER_CAPTURE_KEY are required")
	}
	if browser.Keys.Flags != "" && browser.AppEndpoint == "" {
		return errors.New("DEMO_APP_ENDPOINT is required with DEMO_BROWSER_FLAGS_KEY")
	}
	router.GET("/api/config", func(c *gin.Context) { c.JSON(http.StatusOK, browser) })
	router.GET("/api/products", shop.products)
	router.POST("/api/checkout", shop.checkout(checkoutV1))
	router.POST("/api/v2/checkout", shop.checkout(checkoutV2))
	if dir := os.Getenv("DEMO_STATIC_DIR"); dir != "" {
		router.NoRoute(staticSite(dir))
	}

	server := &http.Server{Addr: envOr("DEMO_ADDR", ":8090"), Handler: router, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.WithoutCancel(ctx), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdown); err != nil {
			log.Printf("shutdown: %v", err)
		}
		if err := client.Stop(shutdown); err != nil {
			log.Printf("flush telemetry: %v", err)
		}
	}()
	log.Printf("demo shop api listening on %s", server.Addr)
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// browserConfig is the storefront's public configuration. Every key in it is a
// public, single-purpose browser credential.
type browserConfig struct {
	Endpoint    string      `json:"endpoint"`
	AppEndpoint string      `json:"app_endpoint,omitempty"`
	Release     string      `json:"release"`
	Keys        browserKeys `json:"keys"`
}

type browserKeys struct {
	Ingest  string `json:"ingest"`
	Capture string `json:"capture"`
	Replay  string `json:"replay,omitempty"`
	Flags   string `json:"flags,omitempty"`
}

type shop struct {
	client anectico.Client
}

func (s *shop) products(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"products": catalog})
}

func (s *shop) checkout(price func(CheckoutRequest, string) (Order, error)) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CheckoutRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid checkout request"})
			return
		}
		ctx := c.Request.Context()
		order, err := price(req, uuid.NewString())
		if err != nil {
			s.client.Logger().WarnContext(ctx, "checkout refused", "reason", err.Error())
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		s.client.Logger().InfoContext(ctx, "order placed",
			"order_id", order.ID, "total_cents", order.TotalCents, "discount_code", req.DiscountCode)
		c.JSON(http.StatusOK, order)
	}
}

// staticSite serves the built storefront, falling back to index.html for
// client-side routes. http.Dir cleans the path and refuses to leave dir, so a
// request such as /../../etc/passwd can never read outside the build.
func staticSite(dir string) gin.HandlerFunc {
	files := http.Dir(dir)
	fileServer := http.FileServer(files)
	return func(c *gin.Context) {
		if f, err := files.Open(path.Clean("/" + c.Request.URL.Path)); err == nil {
			info, statErr := f.Stat()
			_ = f.Close()
			if statErr == nil && !info.IsDir() {
				fileServer.ServeHTTP(c.Writer, c.Request)
				return
			}
		}
		c.File(filepath.Join(dir, "index.html"))
	}
}

func envOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
