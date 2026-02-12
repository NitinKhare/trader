"""
Dynamic symbol to security ID resolver using Dhan API.

This module provides utilities to look up Dhan security IDs by symbol
using the official Dhan instrument master CSV or API.

Usage:
    from python_ai.data.symbol_resolver import SymbolResolver

    resolver = SymbolResolver()
    sec_id = resolver.get_security_id("HAL")  # Returns "2303"
    sec_id = resolver.get_security_id("RELIANCE")  # Returns "2885"
"""

import json
import os
import sys
from typing import Dict, Optional

import pandas as pd
import requests


class SymbolResolver:
    """Resolves trading symbols to Dhan security IDs."""

    DHAN_SCRIP_MASTER_URL = "https://images.dhan.co/api-data/api-scrip-master.csv"
    DHAN_SCRIP_MASTER_DETAILED = "https://images.dhan.co/api-data/api-scrip-master-detailed.csv"

    def __init__(self, cache_path: Optional[str] = None, use_detailed: bool = False):
        """Initialize the resolver with optional caching.

        Args:
            cache_path: Path to cache the instrument master CSV. If None, no caching.
            use_detailed: Use detailed version of master CSV for more info.
        """
        self.cache_path = cache_path
        self.use_detailed = use_detailed
        self.url = self.DHAN_SCRIP_MASTER_DETAILED if use_detailed else self.DHAN_SCRIP_MASTER_URL
        self.instruments_df = None
        self.security_map = {}

    def load_instruments(self, force_refresh: bool = False) -> pd.DataFrame:
        """Load instrument master data from cache or Dhan API.

        Args:
            force_refresh: Force reload from Dhan API even if cached.

        Returns:
            DataFrame with instrument data.
        """
        # Try cache first
        if self.instruments_df is not None and not force_refresh:
            return self.instruments_df

        if self.cache_path and os.path.exists(self.cache_path) and not force_refresh:
            print(f"Loading instruments from cache: {self.cache_path}")
            self.instruments_df = pd.read_csv(self.cache_path)
            return self.instruments_df

        # Fetch from Dhan API
        print(f"Fetching instruments from {self.url}...")
        resp = requests.get(self.url, timeout=60)
        resp.raise_for_status()

        self.instruments_df = pd.read_csv(
            pd.io.common.StringIO(resp.text),
            low_memory=False
        )

        # Cache if path provided
        if self.cache_path:
            os.makedirs(os.path.dirname(self.cache_path) or ".", exist_ok=True)
            self.instruments_df.to_csv(self.cache_path, index=False)
            print(f"Cached instruments to: {self.cache_path}")

        return self.instruments_df

    def get_security_id(
        self,
        symbol: str,
        exchange: str = "NSE",
        segment: str = "E",
        force_refresh: bool = False
    ) -> Optional[str]:
        """Get security ID for a given symbol.

        Args:
            symbol: Trading symbol (e.g., "HAL", "RELIANCE", "NIFTY50")
            exchange: Exchange code (e.g., "NSE", "BSE", "MCX")
            segment: Segment code (e.g., "E" for equity, "D" for derivatives)
            force_refresh: Force reload from API

        Returns:
            Security ID as string, or None if not found.
        """
        df = self.load_instruments(force_refresh=force_refresh)

        # Normalize symbol
        symbol = symbol.upper().strip()

        # Try different column combinations
        for sym_col in ["SEM_TRADING_SYMBOL", "SEM_CUSTOM_SYMBOL", "SYMBOL_NAME"]:
            if sym_col not in df.columns:
                continue

            # Exact match
            matches = df[
                (df[sym_col].str.upper() == symbol) &
                (df.get("SEM_EXM_EXCH_ID", df.get("SEM_EXCH_ID")) == exchange)
            ]

            if not matches.empty:
                if "SEM_SMST_SECURITY_ID" in matches.columns:
                    return str(int(matches.iloc[0]["SEM_SMST_SECURITY_ID"]))
                elif "SEM_SECURITY_ID" in matches.columns:
                    return str(int(matches.iloc[0]["SEM_SECURITY_ID"]))

        # Try with -EQ suffix for equity
        if segment == "E":
            symbol_with_eq = f"{symbol}-EQ"
            for sym_col in ["SEM_TRADING_SYMBOL", "SEM_CUSTOM_SYMBOL"]:
                if sym_col not in df.columns:
                    continue

                matches = df[
                    (df[sym_col].str.upper() == symbol_with_eq) &
                    (df.get("SEM_EXM_EXCH_ID", df.get("SEM_EXCH_ID")) == exchange)
                ]

                if not matches.empty:
                    if "SEM_SMST_SECURITY_ID" in matches.columns:
                        return str(int(matches.iloc[0]["SEM_SMST_SECURITY_ID"]))
                    elif "SEM_SECURITY_ID" in matches.columns:
                        return str(int(matches.iloc[0]["SEM_SECURITY_ID"]))

        print(f"Warning: Security ID not found for {symbol}")
        return None

    def get_multiple_security_ids(
        self,
        symbols: list,
        exchange: str = "NSE",
        segment: str = "E",
        force_refresh: bool = False
    ) -> Dict[str, Optional[str]]:
        """Get security IDs for multiple symbols.

        Args:
            symbols: List of trading symbols
            exchange: Exchange code
            segment: Segment code
            force_refresh: Force reload from API

        Returns:
            Dictionary mapping symbols to security IDs (None if not found).
        """
        result = {}
        for symbol in symbols:
            result[symbol] = self.get_security_id(
                symbol,
                exchange=exchange,
                segment=segment,
                force_refresh=force_refresh
            )
        return result

    def build_config(
        self,
        symbols: list,
        output_path: str,
        exchange: str = "NSE",
        segment: str = "E",
        force_refresh: bool = False
    ) -> Dict:
        """Build a Dhan instruments config file for given symbols.

        Args:
            symbols: List of trading symbols
            output_path: Path to save config JSON
            exchange: Exchange code
            segment: Segment code
            force_refresh: Force reload from API

        Returns:
            Dictionary with config data.
        """
        print(f"Building instruments config for {len(symbols)} symbols...")

        security_map = self.get_multiple_security_ids(
            symbols,
            exchange=exchange,
            segment=segment,
            force_refresh=force_refresh
        )

        config = {
            "description": "NSE ticker to Dhan securityId mapping",
            "exchange_segment": f"{exchange}_EQ" if segment == "E" else exchange,
            "updated": pd.Timestamp.now().strftime("%Y-%m-%d"),
            "instruments": {
                symbol: sec_id
                for symbol, sec_id in security_map.items()
                if sec_id is not None
            }
        }

        # Write config
        os.makedirs(os.path.dirname(output_path) or ".", exist_ok=True)
        with open(output_path, "w") as f:
            json.dump(config, f, indent=2)

        success = sum(1 for v in security_map.values() if v is not None)
        missing = [s for s, sec_id in security_map.items() if sec_id is None]

        print(f"\nMapped {success}/{len(symbols)} symbols -> {output_path}")
        if missing:
            print(f"Missing ({len(missing)}): {', '.join(missing)}")

        return config


def main():
    """CLI tool to lookup security IDs by symbol."""
    import argparse

    parser = argparse.ArgumentParser(description="Look up Dhan security IDs by symbol")
    parser.add_argument("symbols", nargs="*", help="Symbol(s) to look up (e.g., HAL RELIANCE)")
    parser.add_argument("--symbol-file", help="JSON file with symbols list")
    parser.add_argument("--exchange", default="NSE", help="Exchange code (NSE, BSE, MCX)")
    parser.add_argument("--segment", default="E", help="Segment (E for equity)")
    parser.add_argument("--cache", default="./market_data/instrument_cache.csv",
                        help="Cache file for instrument master")
    parser.add_argument("--output", help="Output config file path")
    parser.add_argument("--refresh", action="store_true", help="Force refresh from Dhan API")
    args = parser.parse_args()

    resolver = SymbolResolver(cache_path=args.cache)

    # Determine symbols to look up
    symbols = list(args.symbols) if args.symbols else []

    if args.symbol_file:
        if args.symbol_file.endswith(".json"):
            with open(args.symbol_file) as f:
                data = json.load(f)
                symbols.extend(data.get("symbols", []))
        else:
            # Assume it's stock_universe.json
            with open(args.symbol_file) as f:
                data = json.load(f)
                symbols.extend(data.get("symbols", []))

    if not symbols:
        print("ERROR: No symbols provided. Use positional args or --symbol-file")
        sys.exit(1)

    # Look up security IDs
    security_ids = resolver.get_multiple_security_ids(
        symbols,
        exchange=args.exchange,
        segment=args.segment,
        force_refresh=args.refresh
    )

    # Output results
    print("\n=== Symbol to Security ID Mapping ===")
    for symbol, sec_id in security_ids.items():
        status = "✓" if sec_id else "✗"
        print(f"{status} {symbol:20} -> {sec_id or 'NOT FOUND'}")

    # Write config if requested
    if args.output:
        symbols_found = [s for s, sec_id in security_ids.items() if sec_id]
        resolver.build_config(
            symbols_found,
            args.output,
            exchange=args.exchange,
            segment=args.segment,
            force_refresh=args.refresh
        )


if __name__ == "__main__":
    main()
