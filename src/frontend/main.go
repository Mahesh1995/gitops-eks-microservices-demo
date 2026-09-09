// frontend is a tiny Go web service that acts as the UI + API gateway.
// It aggregates data from product-catalog and cart, and renders a minimal HTML page.
//
// Dependencies (addresses) are injected via environment variables so the same binary
// runs unchanged locally, in Docker Compose, and in Kubernetes:
//
//	PRODUCT_CATALOG_ADDR  host:port of the product-catalog service (default localhost:5000)
//	CART_ADDR             host:port of the cart service            (default localhost:3000)
//	PORT                  port to listen on                        (default 8080)
//
// Only the Go standard library is used on purpose — no external modules to fetch — so
// it builds anywhere with zero network access.
package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"sync/atomic"
	"time"
)

var (
	productAddr = env("PRODUCT_CATALOG_ADDR", "localhost:5000")
	cartAddr    = env("CART_ADDR", "localhost:3000")
	port        = env("PORT", "8080")

	// Minimal hand-rolled Prometheus counters (no client library needed).
	reqTotal   int64
	reqErrors  int64
	httpClient = &http.Client{Timeout: 3 * time.Second}
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type product struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	PriceUSD    float64 `json:"price_usd"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/", withMetrics(handleIndex))
	mux.HandleFunc("/healthz", handleHealthz) // liveness: always OK if process is up
	mux.HandleFunc("/readyz", handleReadyz)   // readiness: product-catalog reachable
	mux.HandleFunc("/metrics", handleMetrics)

	addr := ":" + port
	log.Printf("frontend listening on %s (product=%s cart=%s)", addr, productAddr, cartAddr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func withMetrics(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&reqTotal, 1)
		h(w, r)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	products, err := fetchProducts()
	if err != nil {
		atomic.AddInt64(&reqErrors, 1)
		http.Error(w, "product-catalog unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	if err := pageTmpl.Execute(w, products); err != nil {
		log.Printf("template error: %v", err)
	}
}

func fetchProducts() ([]product, error) {
	resp, err := httpClient.Get("http://" + productAddr + "/products")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		Products []product `json:"products"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Products, nil
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	io.WriteString(w, "ok")
}

func handleReadyz(w http.ResponseWriter, _ *http.Request) {
	if _, err := fetchProducts(); err != nil {
		http.Error(w, "not ready: "+err.Error(), http.StatusServiceUnavailable)
		return
	}
	io.WriteString(w, "ready")
}

// handleMetrics emits Prometheus text-format metrics without any client library.
func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP frontend_requests_total Total HTTP requests served.\n")
	fmt.Fprintf(w, "# TYPE frontend_requests_total counter\n")
	fmt.Fprintf(w, "frontend_requests_total %d\n", atomic.LoadInt64(&reqTotal))
	fmt.Fprintf(w, "# HELP frontend_request_errors_total Total upstream errors.\n")
	fmt.Fprintf(w, "# TYPE frontend_request_errors_total counter\n")
	fmt.Fprintf(w, "frontend_request_errors_total %d\n", atomic.LoadInt64(&reqErrors))
}

var pageTmpl = template.Must(template.New("page").Parse(`<!doctype html>
<html lang="en"><head><meta charset="utf-8"><title>GitOps Demo Shop</title>
<style>body{font-family:system-ui,sans-serif;max-width:720px;margin:2rem auto;padding:0 1rem}
h1{font-size:1.4rem}.card{border:1px solid #ddd;border-radius:8px;padding:1rem;margin:.75rem 0}
.price{color:#0a7d33;font-weight:600}</style></head>
<body>
<h1>🛒 GitOps Demo Shop</h1>
<p>Products served by <code>product-catalog</code> via the Go <code>frontend</code>.</p>
{{range .}}<div class="card"><strong>{{.Name}}</strong><br>{{.Description}}<br>
<span class="price">${{printf "%.2f" .PriceUSD}}</span></div>{{else}}<p>No products.</p>{{end}}
</body></html>`))
