"""
Dhan historical data fetcher.

Downloads daily OHLCV data from Dhan API and saves as CSV files
for consumption by the AI scoring pipeline.

Usage:
    python3 -m python_ai.data.fetch_dhan --days-back 365
    python3 -m python_ai.data.fetch_dhan --symbols RELIANCE,TCS --days-back 30
    python3 -m python_ai.data.fetch_dhan --symbols-from config/stock_universe.json

API Details:
    - Endpoint: POST https://api.dhan.co/v2/charts/historical
    - Auth: Header access-token (Client-Id is optional)
    - Rate limit: 10 req/sec
    - Max 90 days per request (auto-chunked)
    - Response: arrays of open, high, low, close, volume, timestamp (epoch)
"""

import argparse
import json
import os
import sys
import time
from datetime import date, datetime, timedelta

import pandas as pd
import requests

# Dhan API constants.
DHAN_CHART_URL = "https://api.dhan.co/v2/charts/historical"
MAX_CHUNK_DAYS = 90
RATE_LIMIT_DELAY = 0.5  # seconds between requests (conservative to avoid 429s)
MAX_RETRIES = 5          # max retries on 429 rate limit errors
RETRY_BASE_DELAY = 2.0   # base delay in seconds for exponential backoff


def load_config() -> tuple:
    """Load Dhan credentials from config.json or environment variables.

    Returns (client_id, access_token). client_id may be empty — it's optional.
    """
    client_id = os.environ.get("DHAN_CLIENT_ID", "")
    access_token = os.environ.get("DHAN_ACCESS_TOKEN", "")

    if not access_token:
        config_path = os.environ.get("ALGO_CONFIG_PATH", "config/config.json")
        if os.path.exists(config_path):
            with open(config_path) as f:
                cfg = json.load(f)
            dhan_cfg = cfg.get("broker_config", {}).get("dhan", {})
            client_id = client_id or dhan_cfg.get("client_id", "")
            access_token = access_token or dhan_cfg.get("access_token", "")

    if not access_token:
        print("ERROR: Dhan access_token not found.")
        print("Set DHAN_ACCESS_TOKEN env var,")
        print("or fill access_token in config/config.json")
        sys.exit(1)

    return client_id, access_token


def load_instruments(path: str) -> dict:
    """Load ticker -> securityId mapping from JSON."""
    if not os.path.exists(path):
        print(f"ERROR: Instrument file not found: {path}")
        print("Run: python3 scripts/fetch_instruments.py")
        sys.exit(1)

    with open(path) as f:
        data = json.load(f)
    return data.get("instruments", {})


def fetch_chunk(
    client_id: str,
    access_token: str,
    security_id: str,
    exchange_segment: str,
    instrument: str,
    from_date: date,
    to_date: date,
) -> pd.DataFrame:
    """Fetch a single <=90 day chunk from Dhan historical API.

    Retries with exponential backoff on 429 rate limit errors.
    """
    headers = {
        "access-token": access_token,
        "Content-Type": "application/json",
    }
    if client_id:
        headers["Client-Id"] = client_id
    payload = {
        "securityId": security_id,
        "exchangeSegment": exchange_segment,
        "instrument": instrument,
        "expiryCode": 0,
        "fromDate": from_date.strftime("%Y-%m-%d"),
        "toDate": to_date.strftime("%Y-%m-%d"),
    }

    for attempt in range(MAX_RETRIES + 1):
        resp = requests.post(DHAN_CHART_URL, headers=headers, json=payload, timeout=30)

        if resp.status_code == 401:
            raise ValueError("Authentication failed (401): check credentials or token may have expired")

        if resp.status_code == 429:
            if attempt < MAX_RETRIES:
                wait = RETRY_BASE_DELAY * (2 ** attempt)  # 2s, 4s, 8s, 16s, 32s
                print(f"429 rate limited, retrying in {wait:.0f}s (attempt {attempt + 1}/{MAX_RETRIES})...", end=" ", flush=True)
                time.sleep(wait)
                continue
            else:
                raise ValueError(f"Rate limited (429): exhausted {MAX_RETRIES} retries")

        if resp.status_code != 200:
            resp.raise_for_status()

        break

    data = resp.json()

    if not data.get("timestamp"):
        return pd.DataFrame()

    df = pd.DataFrame({
        "date": pd.to_datetime(data["timestamp"], unit="s"),
        "open": data["open"],
        "high": data["high"],
        "low": data["low"],
        "close": data["close"],
        "volume": data["volume"],
    })
    df["date"] = df["date"].dt.strftime("%Y-%m-%d")
    return df


def fetch_symbol(
    client_id: str,
    access_token: str,
    security_id: str,
    symbol: str,
    from_date: date,
    to_date: date,
) -> pd.DataFrame:
    """Fetch full date range with 90-day chunking and rate limiting."""
    # Determine exchange segment and instrument type.
    if symbol == "NIFTY50":
        exchange_segment = "IDX_I"
        instrument = "INDEX"
    else:
        exchange_segment = "NSE_EQ"
        instrument = "EQUITY"

    all_chunks = []
    chunk_start = from_date

    while chunk_start <= to_date:
        chunk_end = min(chunk_start + timedelta(days=MAX_CHUNK_DAYS - 1), to_date)

        df = fetch_chunk(
            client_id, access_token, security_id,
            exchange_segment, instrument,
            chunk_start, chunk_end,
        )
        if not df.empty:
            all_chunks.append(df)

        time.sleep(RATE_LIMIT_DELAY)
        chunk_start = chunk_end + timedelta(days=1)

    if not all_chunks:
        return pd.DataFrame()

    result = pd.concat(all_chunks, ignore_index=True)
    result = result.drop_duplicates(subset=["date"]).sort_values("date").reset_index(drop=True)
    return result


def save_csv(df: pd.DataFrame, output_dir: str, symbol: str) -> str:
    """Save DataFrame as CSV, merging with existing data if present."""
    os.makedirs(output_dir, exist_ok=True)
    filepath = os.path.join(output_dir, f"{symbol}.csv")

    if os.path.exists(filepath):
        try:
            existing = pd.read_csv(filepath)
            existing["date"] = pd.to_datetime(existing["date"]).dt.strftime("%Y-%m-%d")
            df = pd.concat([existing, df], ignore_index=True)
            df = df.drop_duplicates(subset=["date"]).sort_values("date").reset_index(drop=True)
        except Exception as e:
            print(f"  WARN: could not merge existing {filepath}: {e}")

    df.to_csv(filepath, index=False)
    return filepath


def main():
    parser = argparse.ArgumentParser(description="Fetch historical data from Dhan API")
    parser.add_argument("--symbols-from", default="config/stock_universe.json",
                        help="Path to stock universe JSON file")
    parser.add_argument("--instruments", default="config/dhan_instruments.json",
                        help="Path to Dhan instruments mapping JSON")
    parser.add_argument("--output-dir", default="./market_data",
                        help="Output directory for CSV files")
    parser.add_argument("--days-back", type=int, default=365,
                        help="Number of days of history to fetch")
    parser.add_argument("--symbols", type=str, default="",
                        help="Comma-separated symbols to fetch (overrides --symbols-from)")
    args = parser.parse_args()

    # Load credentials and instruments.
    client_id, access_token = load_config()
    instruments = load_instruments(args.instruments)

    # Determine symbol list.
    if args.symbols:
        symbols = [s.strip() for s in args.symbols.split(",")]
    else:
        with open(args.symbols_from) as f:
            universe = json.load(f)
        symbols = universe.get("symbols", [])

    # Always include NIFTY50 for regime detection.
    if "NIFTY50" not in symbols:
        symbols.append("NIFTY50")

    to_date = date.today()
    from_date = to_date - timedelta(days=args.days_back)

    print(f"=== Dhan Data Fetch ===")
    print(f"Symbols: {len(symbols)}")
    print(f"Range: {from_date} to {to_date} ({args.days_back} days)")
    print(f"Output: {args.output_dir}")
    print()

    success = 0
    failed = []

    for i, symbol in enumerate(symbols, 1):
        sec_id = instruments.get(symbol)
        if not sec_id:
            print(f"  [{i}/{len(symbols)}] SKIP {symbol}: no securityId in instruments file")
            failed.append(symbol)
            continue

        print(f"  [{i}/{len(symbols)}] {symbol} (secId={sec_id})...", end=" ", flush=True)
        try:
            df = fetch_symbol(client_id, access_token, sec_id, symbol, from_date, to_date)
            if df.empty:
                print("no data returned")
                failed.append(symbol)
                continue
            path = save_csv(df, args.output_dir, symbol)
            print(f"{len(df)} candles -> {path}")
            success += 1
        except Exception as e:
            print(f"ERROR: {e}")
            failed.append(symbol)

        # Pause between symbols to avoid sustained rate limiting.
        if i < len(symbols):
            time.sleep(1.0)

    print(f"\n=== Done: {success}/{len(symbols)} symbols fetched ===")
    if failed:
        print(f"Failed ({len(failed)}): {', '.join(failed)}")


if __name__ == "__main__":
    main()
