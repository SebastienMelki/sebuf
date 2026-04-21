import { MarketDataServiceClient } from "./generated/proto/services/market_data_client.ts";

async function main() {
  const client = new MarketDataServiceClient("http://localhost:8080");

  console.log("SSE Streaming — TypeScript Client Demo");
  console.log("=======================================\n");

  // 1. Unary RPC: get a single quote snapshot.
  console.log("--- GetQuote (unary) ---");
  const quote = await client.getQuote({ symbol: "AAPL" });
  console.log(`  ${quote.symbol}  bid=${quote.bid}  ask=${quote.ask}  last=${quote.last}  vol=${quote.volume}`);

  // 2. SSE stream: real-time price updates.
  console.log("\n--- StreamQuotes (SSE, 5 events) ---");
  let count = 0;
  for await (const q of client.streamQuotes({ symbol: "TSLA" })) {
    console.log(`  ${q.symbol}  bid=${q.bid}  ask=${q.ask}  last=${q.last}  vol=${q.volume}`);
    count++;
    if (count >= 5) break;
  }
  console.log(`Received ${count} quote events`);

  // 3. SSE stream with query params: filtered trade feed.
  console.log("\n--- StreamTrades (SSE, symbol=GOOG, limit=5) ---");
  for await (const trade of client.streamTrades({ symbol: "GOOG", limit: 5 })) {
    console.log(`  trade ${trade.id}: ${trade.symbol} ${trade.price} x${trade.size} (${trade.side})`);
  }

  // 4. SSE stream with AbortController for cancellation.
  console.log("\n--- StreamQuotes with AbortController (cancel after 3 events) ---");
  const controller = new AbortController();
  let abortCount = 0;
  try {
    for await (const q of client.streamQuotes({ symbol: "MSFT" }, { signal: controller.signal })) {
      console.log(`  ${q.symbol}  last=${q.last}`);
      abortCount++;
      if (abortCount >= 3) {
        controller.abort();
      }
    }
  } catch (e) {
    if (e instanceof Error && e.name === "AbortError") {
      console.log(`  Stream aborted after ${abortCount} events`);
    } else {
      throw e;
    }
  }

  console.log("\n=== TypeScript client demo complete ===");
}

main()
  .catch(console.error)
  .finally(() => process.exit(0));
