// checkout orchestrates the other services to turn a user's cart into an order.
// It demonstrates service-to-service calls (the pattern the production reference uses
// with gRPC) using plain REST:
//
//	POST /checkout/{user}  ->  reads the cart, looks up each product's price,
//	                           computes a total, and returns an order summary.
//
// Dependencies are injected via env vars so the binary is environment-agnostic:
//
//	CART_ADDR             host:port of the cart service            (default localhost:3000)
//	PRODUCT_CATALOG_ADDR  host:port of the product-catalog service (default localhost:5000)
//	PORT                  port to listen on                        (default 4000)
//
// Standard library only — builds with no network access.
package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"
)

var (
	cartAddr    = env("CART_ADDR", "localhost:3000")
	productAddr = env("PRODUCT_CATALOG_ADDR", "localhost:5000")
	port        = env("PORT", "4000")

	ordersPlaced int64
	client       = &http.Client{Timeout: 3 * time.Second}
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

type cartItem struct {
	ProductID string `json:"product_id"`
	Qty       int    `json:"qty"`
}

type product struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	PriceUSD float64 `json:"price_usd"`
}

type orderLine struct {
	ProductID  string  `json:"product_id"`
	Name       string  `json:"name"`
	Qty        int     `json:"qty"`
	UnitPrice  float64 `json:"unit_price_usd"`
	LineTotal  float64 `json:"line_total_usd"`
}

type order struct {
	User       string      `json:"user"`
	Lines      []orderLine `json:"lines"`
	TotalUSD   float64     `json:"total_usd"`
	PlacedAtMS int64       `json:"placed_at_ms"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/checkout/", handleCheckout) // POST /checkout/{user}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { io.WriteString(w, "ok") })
	mux.HandleFunc("/readyz", handleReadyz)
	mux.HandleFunc("/metrics", handleMetrics)

	addr := ":" + port
	log.Printf("checkout listening on %s (cart=%s product=%s)", addr, cartAddr, productAddr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func handleCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "use POST", http.StatusMethodNotAllowed)
		return
	}
	user := strings.TrimPrefix(r.URL.Path, "/checkout/")
	if user == "" {
		http.Error(w, "missing user in path: POST /checkout/{user}", http.StatusBadRequest)
		return
	}

	items, err := getCart(user)
	if err != nil {
		http.Error(w, "cart unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}
	if len(items) == 0 {
		http.Error(w, "cart is empty", http.StatusBadRequest)
		return
	}

	ord := order{User: user, PlacedAtMS: time.Now().UnixMilli()}
	for _, it := range items {
		p, err := getProduct(it.ProductID)
		if err != nil {
			http.Error(w, "product lookup failed for "+it.ProductID+": "+err.Error(), http.StatusBadGateway)
			return
		}
		line := orderLine{
			ProductID: it.ProductID,
			Name:      p.Name,
			Qty:       it.Qty,
			UnitPrice: p.PriceUSD,
			LineTotal: p.PriceUSD * float64(it.Qty),
		}
		ord.Lines = append(ord.Lines, line)
		ord.TotalUSD += line.LineTotal
	}
	ord.TotalUSD = round2(ord.TotalUSD)
	atomic.AddInt64(&ordersPlaced, 1)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(ord)
}

func getCart(user string) ([]cartItem, error) {
	resp, err := client.Get("http://" + cartAddr + "/cart/" + user)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}
	var out struct {
		Items []cartItem `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Items, nil
}

func getProduct(id string) (product, error) {
	var p product
	resp, err := client.Get("http://" + productAddr + "/products/" + id)
	if err != nil {
		return p, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return p, fmt.Errorf("status %d", resp.StatusCode)
	}
	err = json.NewDecoder(resp.Body).Decode(&p)
	return p, err
}

func handleReadyz(w http.ResponseWriter, _ *http.Request) {
	// Ready when both dependencies answer their health checks.
	for name, addr := range map[string]string{"cart": cartAddr, "product-catalog": productAddr} {
		resp, err := client.Get("http://" + addr + "/healthz")
		if err != nil || resp.StatusCode != http.StatusOK {
			http.Error(w, "dependency not ready: "+name, http.StatusServiceUnavailable)
			return
		}
		resp.Body.Close()
	}
	io.WriteString(w, "ready")
}

func handleMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP checkout_orders_placed_total Total orders placed.\n")
	fmt.Fprintf(w, "# TYPE checkout_orders_placed_total counter\n")
	fmt.Fprintf(w, "checkout_orders_placed_total %d\n", atomic.LoadInt64(&ordersPlaced))
}

func round2(f float64) float64 {
	return float64(int64(f*100+0.5)) / 100
}
