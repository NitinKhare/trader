#!/usr/bin/env python3
"""
Backfill AI scores for historical dates to enable longer backtests.

This script runs the AI scoring pipeline on multiple historical dates,
generating the AI output data needed for backtesting.

Usage:
    # Generate scores for the last 30 days
    python scripts/backfill_ai_scores.py --days-back 30

    # Generate scores for a specific date range
    python scripts/backfill_ai_scores.py --from-date 2026-01-01 --to-date 2026-02-13

    # Generate scores for specific dates only
    python scripts/backfill_ai_scores.py --dates 2026-02-08,2026-02-09,2026-02-10
"""

import argparse
import json
import os
import subprocess
import sys
from datetime import date, datetime, timedelta
from pathlib import Path


def get_trading_dates(
    from_date: date,
    to_date: date,
    market_data_dir: str = "./market_data"
) -> list[date]:
    """Get list of trading dates with available market data.

    A date is a trading date if market data exists for that date.
    """
    # Load a sample stock to get available dates
    sample_file = os.path.join(market_data_dir, "RELIANCE.csv")
    if not os.path.exists(sample_file):
        print(f"ERROR: No market data found in {market_data_dir}")
        return []

    trading_dates = []
    with open(sample_file) as f:
        next(f)  # Skip header
        for line in f:
            try:
                date_str = line.split(",")[0]
                row_date = datetime.strptime(date_str, "%Y-%m-%d").date()
                if from_date <= row_date <= to_date:
                    trading_dates.append(row_date)
            except (ValueError, IndexError):
                continue

    return sorted(set(trading_dates))


def get_existing_scores(output_dir: str = "./ai_outputs") -> set[date]:
    """Get set of dates that already have AI scores."""
    existing = set()
    if os.path.exists(output_dir):
        for item in os.listdir(output_dir):
            try:
                scores_file = os.path.join(output_dir, item, "stock_scores.json")
                if os.path.exists(scores_file):
                    existing.add(datetime.strptime(item, "%Y-%m-%d").date())
            except ValueError:
                continue
    return existing


def run_scoring(
    scoring_date: date,
    data_dir: str = "./market_data",
    output_dir: str = "./ai_outputs",
    universe_file: str = "./config/stock_universe.json"
) -> bool:
    """Run AI scoring for a specific date."""
    date_str = scoring_date.strftime("%Y-%m-%d")

    cmd = [
        "python3", "-m", "python_ai.run_scoring",
        "--date", date_str,
        "--data-dir", data_dir,
        "--output-dir", output_dir,
        "--universe-file", universe_file
    ]

    try:
        result = subprocess.run(cmd, capture_output=True, text=True, timeout=60)
        if result.returncode == 0:
            print(f"✅ {date_str} - Scored successfully")
            return True
        else:
            print(f"❌ {date_str} - Error: {result.stderr[:100]}")
            return False
    except subprocess.TimeoutExpired:
        print(f"❌ {date_str} - Timeout (>60s)")
        return False
    except Exception as e:
        print(f"❌ {date_str} - Exception: {str(e)}")
        return False


def main():
    parser = argparse.ArgumentParser(
        description="Backfill AI scores for historical dates"
    )
    parser.add_argument(
        "--days-back",
        type=int,
        default=30,
        help="Generate scores for last N days (default: 30)"
    )
    parser.add_argument(
        "--from-date",
        type=str,
        help="Start date (YYYY-MM-DD)"
    )
    parser.add_argument(
        "--to-date",
        type=str,
        help="End date (YYYY-MM-DD)"
    )
    parser.add_argument(
        "--dates",
        type=str,
        help="Comma-separated dates (YYYY-MM-DD,YYYY-MM-DD,...)"
    )
    parser.add_argument(
        "--data-dir",
        default="./market_data",
        help="Directory with market data CSV files"
    )
    parser.add_argument(
        "--output-dir",
        default="./ai_outputs",
        help="Output directory for AI scores"
    )
    parser.add_argument(
        "--universe-file",
        default="./config/stock_universe.json",
        help="Stock universe JSON file"
    )
    parser.add_argument(
        "--skip-existing",
        action="store_true",
        help="Skip dates that already have scores"
    )
    parser.add_argument(
        "--dry-run",
        action="store_true",
        help="Show what would be done without running"
    )
    parser.add_argument(
        "--verbose",
        action="store_true",
        help="Show detailed output"
    )

    args = parser.parse_args()

    # Determine dates to process
    dates_to_process = []

    if args.dates:
        # Specific dates
        try:
            dates_to_process = [
                datetime.strptime(d.strip(), "%Y-%m-%d").date()
                for d in args.dates.split(",")
            ]
        except ValueError as e:
            print(f"ERROR: Invalid date format: {e}")
            sys.exit(1)

    elif args.from_date and args.to_date:
        # Date range
        try:
            from_date = datetime.strptime(args.from_date, "%Y-%m-%d").date()
            to_date = datetime.strptime(args.to_date, "%Y-%m-%d").date()
        except ValueError as e:
            print(f"ERROR: Invalid date format: {e}")
            sys.exit(1)
        dates_to_process = get_trading_dates(from_date, to_date, args.data_dir)

    else:
        # Last N days
        to_date = date.today()
        from_date = to_date - timedelta(days=args.days_back)
        dates_to_process = get_trading_dates(from_date, to_date, args.data_dir)

    if not dates_to_process:
        print("ERROR: No trading dates found with market data")
        sys.exit(1)

    print(f"📊 Backfilling AI scores")
    print(f"━" * 60)
    print(f"Dates to process: {len(dates_to_process)}")
    print(f"Date range: {min(dates_to_process)} to {max(dates_to_process)}")
    print(f"Output directory: {args.output_dir}")

    # Get existing scores
    if args.skip_existing:
        existing = get_existing_scores(args.output_dir)
        dates_to_process = [d for d in dates_to_process if d not in existing]
        print(f"Skipping {len(get_existing_scores(args.output_dir))} existing scores")
        print(f"Dates to generate: {len(dates_to_process)}")

    if args.dry_run:
        print(f"\n[DRY RUN] Would process:")
        for d in dates_to_process:
            print(f"  - {d}")
        return

    # Process dates
    print(f"\n🔄 Processing {len(dates_to_process)} dates...")
    print(f"━" * 60)

    success_count = 0
    failed_dates = []

    for i, scoring_date in enumerate(dates_to_process, 1):
        print(f"[{i}/{len(dates_to_process)}]", end=" ")

        if run_scoring(
            scoring_date,
            data_dir=args.data_dir,
            output_dir=args.output_dir,
            universe_file=args.universe_file
        ):
            success_count += 1
        else:
            failed_dates.append(scoring_date)

    # Summary
    print(f"\n{'━' * 60}")
    print(f"✅ Success: {success_count}/{len(dates_to_process)}")

    if failed_dates:
        print(f"❌ Failed ({len(failed_dates)}):")
        for d in failed_dates[:5]:
            print(f"   - {d}")
        if len(failed_dates) > 5:
            print(f"   ... and {len(failed_dates) - 5} more")

    print(f"\n📈 AI scores backfilled successfully!")
    print(f"\n💡 Next: Run backtest to see results across {success_count} days")
    print(f"   $ go run ./cmd/engine --mode backtest")


if __name__ == "__main__":
    main()
