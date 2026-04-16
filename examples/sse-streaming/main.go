package main

import (
	"context"
	"fmt"
	"log"
	"math/rand/v2"
	"net/http"
	"time"

	"github.com/SebastienMelki/sebuf/examples/sse-streaming/api/proto/services"
)

// MarketDataServer implements the generated MarketDataServiceServer interface.
type MarketDataServer struct {
	baseQuotes map[string]*services.Quote
}

func NewMarketDataServer() *MarketDataServer {
	return &MarketDataServer{
		baseQuotes: map[string]*services.Quote{
			"AAPL": {Symbol: "AAPL", Bid: 185.50, Ask: 185.55, Last: 185.52, Volume: 45_000_000},
			"GOOG": {Symbol: "GOOG", Bid: 141.20, Ask: 141.25, Last: 141.22, Volume: 22_000_000},
			"TSLA": {Symbol: "TSLA", Bid: 248.10, Ask: 248.20, Last: 248.15, Volume: 80_000_000},
			"MSFT": {Symbol: "MSFT", Bid: 415.30, Ask: 415.40, Last: 415.35, Volume: 18_000_000},
		},
	}
}

// GetQuote returns a single snapshot quote (standard unary RPC).
func (s *MarketDataServer) GetQuote(_ context.Context, req *services.GetQuoteRequest) (*services.Quote, error) {
	base, ok := s.baseQuotes[req.Symbol]
	if !ok {
		return nil, fmt.Errorf("unknown symbol: %s", req.Symbol)
	}
	return jitter(base), nil
}

// StreamQuotes streams real-time price updates via SSE.
// The generated SSEHandler sets Content-Type: text/event-stream and flushes after each Send.
func (s *MarketDataServer) StreamQuotes(ctx context.Context, req *services.StreamQuotesRequest, sender services.SSESender) error {
	base, ok := s.baseQuotes[req.Symbol]
	if !ok {
		return fmt.Errorf("unknown symbol: %s", req.Symbol)
	}

	log.Printf("[SSE] StreamQuotes started for %s", req.Symbol)
	defer log.Printf("[SSE] StreamQuotes ended for %s", req.Symbol)

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			quote := jitter(base)
			if err := sender.Send(quote); err != nil {
				return err // client disconnected
			}
		}
	}
}

// StreamTrades streams a simulated trade feed via SSE.
func (s *MarketDataServer) StreamTrades(ctx context.Context, req *services.StreamTradesRequest, sender services.SSESender) error {
	symbols := []string{"AAPL", "GOOG", "TSLA", "MSFT"}
	if req.Symbol != "" {
		if _, ok := s.baseQuotes[req.Symbol]; !ok {
			return fmt.Errorf("unknown symbol: %s", req.Symbol)
		}
		symbols = []string{req.Symbol}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 50 // default: stream 50 trades then stop
	}

	log.Printf("[SSE] StreamTrades started (symbols=%v, limit=%d)", symbols, limit)
	defer log.Printf("[SSE] StreamTrades ended")

	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()

	sent := 0
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			sym := symbols[rand.IntN(len(symbols))]
			base := s.baseQuotes[sym]
			trade := &services.Trade{
				Id:        fmt.Sprintf("T%d", time.Now().UnixNano()),
				Symbol:    sym,
				Price:     base.Last + (rand.Float64()-0.5)*2,
				Size:      float64(rand.IntN(500) + 1),
				Side:      []string{"buy", "sell"}[rand.IntN(2)],
				Timestamp: time.Now().UnixMilli(),
			}

			if err := sender.SendWithEvent("trade", trade); err != nil {
				return err
			}
			sent++
			if sent >= limit {
				return nil // stream complete
			}
		}
	}
}

// jitter returns a copy of the quote with small random price movements.
func jitter(base *services.Quote) *services.Quote {
	delta := (rand.Float64() - 0.5) * 0.50
	return &services.Quote{
		Symbol:    base.Symbol,
		Bid:       base.Bid + delta,
		Ask:       base.Ask + delta,
		Last:      base.Last + delta,
		Volume:    base.Volume + int64(rand.IntN(10000)),
		Timestamp: time.Now().UnixMilli(),
	}
}

func main() {
	server := NewMarketDataServer()
	mux := http.NewServeMux()

	if err := services.RegisterMarketDataServiceServer(server, services.WithMux(mux)); err != nil {
		log.Fatal(err)
	}

	fmt.Println("SSE Streaming Demo")
	fmt.Println("==================")
	fmt.Println("Server starting on :8080")
	fmt.Println()
	fmt.Println("Endpoints:")
	fmt.Println("  GET /api/v1/quotes/{symbol}         - Snapshot quote (unary)")
	fmt.Println("  GET /api/v1/quotes/{symbol}/stream   - Stream quotes (SSE)")
	fmt.Println("  GET /api/v1/trades/stream             - Stream trades (SSE)")
	fmt.Println()
	fmt.Println("Try:")
	fmt.Println("  curl http://localhost:8080/api/v1/quotes/AAPL")
	fmt.Println("  curl -N http://localhost:8080/api/v1/quotes/AAPL/stream")
	fmt.Println("  curl -N 'http://localhost:8080/api/v1/trades/stream?symbol=TSLA&limit=10'")

	log.Fatal(http.ListenAndServe(":8080", mux))
}
