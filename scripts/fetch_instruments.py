"""
Fetch Dhan instrument master and build the symbol mapping file.

Downloads Dhan's instrument master CSV, filters for NSE equity,
and maps ticker symbols from the stock universe to Dhan securityIds.

Usage:
    python scripts/fetch_instruments.py
    python scripts/fetch_instruments.py --universe config/stock_universe.json --output config/dhan_instruments.json
"""

import argparse
import io
import json

import pandas as pd
import requests

DHAN_SCRIP_MASTER_URL = "https://images.dhan.co/api-data/api-scrip-master.csv"


def main():
    parser = argparse.ArgumentParser(description="Build Dhan instrument mapping")
    parser.add_argument("--universe", default="config/stock_universe.json")
    parser.add_argument("--output", default="config/dhan_instruments.json")
    args = parser.parse_args()

    # Load stock universe.
    with open(args.universe) as f:
        universe = json.load(f)
    symbols = set(universe.get("symbols", []))
    symbols.add("NIFTY50")  # For regime detection.

    # Download instrument master.
    print(f"Downloading Dhan instrument master from {DHAN_SCRIP_MASTER_URL}...")
    resp = requests.get(DHAN_SCRIP_MASTER_URL, timeout=60)
    resp.raise_for_status()

    df = pd.read_csv(io.StringIO(resp.text))
    print(f"Total instruments: {len(df)}")
    print(f"Columns: {list(df.columns)}")

    # Filter for NSE equity segment.
    # Column names may vary — try common Dhan master formats.
    if "SEM_EXM_EXCH_ID" in df.columns:
        nse_eq = df[df["SEM_EXM_EXCH_ID"] == "NSE"]
        if "SEM_SEGMENT" in df.columns:
            nse_eq = nse_eq[nse_eq["SEM_SEGMENT"] == "E"]
        symbol_col = "SEM_TRADING_SYMBOL"
        custom_col = "SEM_CUSTOM_SYMBOL"
        id_col = "SEM_SMST_SECURITY_ID"
    elif "SEM_EXCH_ID" in df.columns:
        nse_eq = df[df["SEM_EXCH_ID"] == "NSE"]
        symbol_col = "SEM_TRADING_SYMBOL"
        custom_col = "SEM_CUSTOM_SYMBOL" if "SEM_CUSTOM_SYMBOL" in df.columns else symbol_col
        id_col = "SEM_SECURITY_ID" if "SEM_SECURITY_ID" in df.columns else "SEM_SMST_SECURITY_ID"
    else:
        # Fallback: print columns and exit.
        print("ERROR: Unexpected column format. Columns found:")
        for col in df.columns:
            print(f"  - {col}")
        print("\nPlease update this script to match the actual column names.")
        return

    print(f"NSE equity instruments: {len(nse_eq)}")

    # Build mapping.
    instruments = {}
    missing = []
    for symbol in sorted(symbols):
        matches = nse_eq[nse_eq[symbol_col] == symbol]
        if matches.empty:
            # Try with -EQ suffix.
            matches = nse_eq[nse_eq[symbol_col] == f"{symbol}-EQ"]
        if matches.empty and custom_col != symbol_col:
            matches = nse_eq[nse_eq[custom_col] == symbol]
        if matches.empty:
            # Try partial match.
            matches = nse_eq[nse_eq[symbol_col].str.startswith(symbol, na=False)]

        if not matches.empty:
            sec_id = str(int(matches.iloc[0][id_col]))
            instruments[symbol] = sec_id
        else:
            missing.append(symbol)

    # Write output.
    output = {
        "description": "NSE ticker to Dhan securityId mapping",
        "exchange_segment": "NSE_EQ",
        "updated": str(pd.Timestamp.now().date()),
        "instruments": instruments,
    }

    with open(args.output, "w") as f:
        json.dump(output, f, indent=2)

    print(f"\nMapped {len(instruments)}/{len(symbols)} symbols -> {args.output}")
    if missing:
        print(f"Missing ({len(missing)}): {', '.join(missing)}")
        print("NOTE: Manually add missing securityIds or check symbol names in the master CSV.")


if __name__ == "__main__":
    main()
