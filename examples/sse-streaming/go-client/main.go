package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/SebastienMelki/sebuf/examples/sse-streaming/api/proto/services"
)

func main() {
	client := services.NewMarketDataServiceClient("http://localhost:8080")

	fmt.Println("SSE Streaming — Go Client Demo")
	fmt.Println("===============================")

	// 1. Unary RPC: get a single quote snapshot.
	fmt.Println("\n--- GetQuote (unary) ---")
	quote, err := client.GetQuote(context.Background(), &services.GetQuoteRequest{Symbol: "AAPL"})
	if err != nil {
		log.Fatalf("GetQuote: %v", err)
	}
	printQuote(quote)

	// 2. SSE stream: real-time price updates.
	fmt.Println("\n--- StreamQuotes (SSE, 5 events) ---")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stream, err := client.StreamQuotes(ctx, &services.StreamQuotesRequest{Symbol: "TSLA"})
	if err != nil {
		log.Fatalf("StreamQuotes: %v", err)
	}
	defer stream.Close()

	event := &services.Quote{}
	count := 0
	for stream.Next(event) {
		printQuote(event)
		count++
		if count >= 5 {
			break
		}
	}
	if err := stream.Err(); err != nil {
		log.Printf("StreamQuotes error: %v", err)
	}
	fmt.Printf("Received %d quote events\n", count)

	// 3. SSE stream with query params: filtered trade feed.
	fmt.Println("\n--- StreamTrades (SSE, symbol=GOOG, limit=5) ---")
	tradeStream, err := client.StreamTrades(context.Background(), &services.StreamTradesRequest{
		Symbol: "GOOG",
		Limit:  5,
	})
	if err != nil {
		log.Fatalf("StreamTrades: %v", err)
	}
	defer tradeStream.Close()

	trade := &services.Trade{}
	for tradeStream.Next(trade) {
		fmt.Printf("  trade %s: %s %.2f x%.0f (%s)\n",
			trade.Id, trade.Symbol, trade.Price, trade.Size, trade.Side)
	}
	if err := tradeStream.Err(); err != nil {
		log.Printf("StreamTrades error: %v", err)
	}

	fmt.Println("\n=== Go client demo complete ===")
}

func printQuote(q *services.Quote) {
	fmt.Printf("  %s  bid=%.2f  ask=%.2f  last=%.2f  vol=%d\n",
		q.Symbol, q.Bid, q.Ask, q.Last, q.Volume)
}
