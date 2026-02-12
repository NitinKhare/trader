#!/usr/bin/env python3
"""
Dynamically fetch Dhan security IDs for stocks in your universe.

This script uses Dhan's official instrument master CSV to automatically
resolve trading symbols to security IDs. It caches the instrument master
for faster subsequent runs.

Usage:
    # Generate instruments config from stock universe
    python scripts/update_instruments_dynamic.py

    # Force refresh from Dhan API
    python scripts/update_instruments_dynamic.py --refresh

    # Look up specific symbols
    python scripts/update_instruments_dynamic.py --symbols HAL RELIANCE TCS

    # Add new stocks to the universe
    python scripts/update_instruments_dynamic.py --add-stocks BANKBARODA,POWERGRID --output-universe new_universe.json
"""

import argparse
import json
import sys
from pathlib import Path

# Add parent directory to path
sys.path.insert(0, str(Path(__file__).parent.parent))

from python_ai.data.symbol_resolver import SymbolResolver


def main():
    parser = argparse.ArgumentParser(
        description="Dynamically fetch Dhan security IDs for your stock universe"
    )
    parser.add_argument(
        "--universe",
        default="config/stock_universe.json",
        help="Stock universe JSON file"
    )
    parser.add_argument(
        "--output",
        default="config/dhan_instruments.json",
        help="Output instruments config file"
    )
    parser.add_argument(
        "--cache",
        default="./market_data/instrument_cache.csv",
        help="Cache file for Dhan instrument master"
    )
    parser.add_argument(
        "--refresh",
        action="store_true",
        help="Force refresh instrument master from Dhan API"
    )
    parser.add_argument(
        "--symbols",
        nargs="+",
        help="Specific symbols to look up (bypasses universe file)"
    )
    parser.add_argument(
        "--add-stocks",
        help="Comma-separated symbols to add to universe"
    )
    parser.add_argument(
        "--output-universe",
        help="Output path for updated universe JSON (with --add-stocks)"
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Show detailed output"
    )

    args = parser.parse_args()

    resolver = SymbolResolver(cache_path=args.cache)

    # Determine symbols to process
    symbols = []

    if args.symbols:
        symbols = args.symbols
    elif args.add_stocks:
        symbols = [s.strip().upper() for s in args.add_stocks.split(",")]
    else:
        # Load from universe
        if not Path(args.universe).exists():
            print(f"ERROR: Universe file not found: {args.universe}")
            sys.exit(1)

        with open(args.universe) as f:
            universe = json.load(f)
        symbols = universe.get("symbols", [])

    print(f"{'='*60}")
    print(f"🔍 Resolving {len(symbols)} symbols to Dhan security IDs")
    print(f"{'='*60}")

    # Resolve security IDs
    security_ids = resolver.get_multiple_security_ids(
        symbols,
        force_refresh=args.refresh
    )

    # Print results
    found = 0
    missing = []

    print("\n📊 Symbol Resolution Results:")
    print("-" * 60)

    for symbol, sec_id in sorted(security_ids.items()):
        if sec_id:
            print(f"✅ {symbol:20} -> {sec_id}")
            found += 1
        else:
            print(f"❌ {symbol:20} -> NOT FOUND")
            missing.append(symbol)

    print("-" * 60)
    print(f"Total: {found}/{len(symbols)} symbols resolved")

    if missing:
        print(f"\n⚠️  Missing symbols ({len(missing)}):")
        for s in missing:
            print(f"   - {s}")
        print("\nTip: Check symbol names at https://www.nseindia.com/")

    # Save instruments config
    if not args.add_stocks:
        symbols_found = [s for s, sec_id in security_ids.items() if sec_id]
        config = resolver.build_config(
            symbols_found,
            args.output,
            force_refresh=args.refresh
        )
        print(f"\n💾 Config saved to: {args.output}")

    # Update universe if adding stocks
    if args.add_stocks and args.output_universe:
        print(f"\n📝 Adding new stocks to universe...")

        with open(args.universe) as f:
            universe = json.load(f)

        # Add new stocks
        new_symbols = [s for s in symbols if security_ids.get(s)]

        # Avoid duplicates
        existing = set(universe.get("symbols", []))
        symbols_to_add = [s for s in new_symbols if s not in existing]

        print(f"   Adding {len(symbols_to_add)} new symbols")

        universe["symbols"].extend(symbols_to_add)
        universe["updated"] = resolver.load_instruments().iloc[0] if len(resolver.load_instruments()) > 0 else "2026-02-13"

        with open(args.output_universe, "w") as f:
            json.dump(universe, f, indent=2)

        print(f"   Updated universe saved to: {args.output_universe}")

    print(f"\n✨ Done! Your instruments are ready for trading.")


if __name__ == "__main__":
    main()
