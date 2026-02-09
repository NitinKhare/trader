"""
Technical indicator calculations for feature engineering.

This module computes standard technical indicators from OHLCV data.
All indicators are calculated offline (not during market hours).
Outputs are deterministic: same input -> same output.
"""

import pandas as pd
import numpy as np


def compute_sma(series: pd.Series, period: int) -> pd.Series:
    """Simple Moving Average."""
    return series.rolling(window=period, min_periods=period).mean()


def compute_ema(series: pd.Series, period: int) -> pd.Series:
    """Exponential Moving Average."""
    return series.ewm(span=period, adjust=False).mean()


def compute_rsi(series: pd.Series, period: int = 14) -> pd.Series:
    """Relative Strength Index (0-100)."""
    delta = series.diff()
    gain = delta.where(delta > 0, 0.0)
    loss = -delta.where(delta < 0, 0.0)

    avg_gain = gain.rolling(window=period, min_periods=period).mean()
    avg_loss = loss.rolling(window=period, min_periods=period).mean()

    rs = avg_gain / avg_loss
    rsi = 100 - (100 / (1 + rs))
    return rsi


def compute_atr(df: pd.DataFrame, period: int = 14) -> pd.Series:
    """Average True Range."""
    high = df["high"]
    low = df["low"]
    close = df["close"]

    tr1 = high - low
    tr2 = (high - close.shift(1)).abs()
    tr3 = (low - close.shift(1)).abs()

    true_range = pd.concat([tr1, tr2, tr3], axis=1).max(axis=1)
    atr = true_range.rolling(window=period, min_periods=period).mean()
    return atr


def compute_bollinger_bands(
    series: pd.Series, period: int = 20, num_std: float = 2.0
) -> tuple[pd.Series, pd.Series, pd.Series]:
    """Bollinger Bands: (upper, middle, lower)."""
    middle = compute_sma(series, period)
    std = series.rolling(window=period, min_periods=period).std()
    upper = middle + (std * num_std)
    lower = middle - (std * num_std)
    return upper, middle, lower


def compute_macd(
    series: pd.Series,
    fast_period: int = 12,
    slow_period: int = 26,
    signal_period: int = 9,
) -> tuple[pd.Series, pd.Series, pd.Series]:
    """MACD: (macd_line, signal_line, histogram)."""
    fast_ema = compute_ema(series, fast_period)
    slow_ema = compute_ema(series, slow_period)
    macd_line = fast_ema - slow_ema
    signal_line = compute_ema(macd_line, signal_period)
    histogram = macd_line - signal_line
    return macd_line, signal_line, histogram


def compute_vwap(df: pd.DataFrame) -> pd.Series:
    """Volume Weighted Average Price (daily reset)."""
    typical_price = (df["high"] + df["low"] + df["close"]) / 3
    cumulative_tp_vol = (typical_price * df["volume"]).cumsum()
    cumulative_vol = df["volume"].cumsum()
    return cumulative_tp_vol / cumulative_vol


def compute_adx(df: pd.DataFrame, period: int = 14) -> pd.Series:
    """Average Directional Index (trend strength 0-100)."""
    high = df["high"]
    low = df["low"]
    close = df["close"]

    plus_dm = high.diff()
    minus_dm = -low.diff()

    plus_dm = plus_dm.where((plus_dm > minus_dm) & (plus_dm > 0), 0.0)
    minus_dm = minus_dm.where((minus_dm > plus_dm) & (minus_dm > 0), 0.0)

    atr = compute_atr(df, period)

    plus_di = 100 * compute_ema(plus_dm, period) / atr
    minus_di = 100 * compute_ema(minus_dm, period) / atr

    dx = 100 * (plus_di - minus_di).abs() / (plus_di + minus_di)
    adx = compute_ema(dx, period)
    return adx


def compute_obv(df: pd.DataFrame) -> pd.Series:
    """On-Balance Volume."""
    direction = np.sign(df["close"].diff())
    obv = (direction * df["volume"]).cumsum()
    return obv


def compute_all_features(df: pd.DataFrame) -> pd.DataFrame:
    """
    Compute all technical features for a stock's OHLCV dataframe.

    Expected columns: date, open, high, low, close, volume
    Returns the dataframe with added feature columns.
    """
    result = df.copy()

    # Moving averages.
    result["sma_20"] = compute_sma(df["close"], 20)
    result["sma_50"] = compute_sma(df["close"], 50)
    result["sma_200"] = compute_sma(df["close"], 200)
    result["ema_9"] = compute_ema(df["close"], 9)
    result["ema_21"] = compute_ema(df["close"], 21)

    # Momentum.
    result["rsi_14"] = compute_rsi(df["close"], 14)

    # Volatility.
    result["atr_14"] = compute_atr(df, 14)
    bb_upper, bb_middle, bb_lower = compute_bollinger_bands(df["close"])
    result["bb_upper"] = bb_upper
    result["bb_middle"] = bb_middle
    result["bb_lower"] = bb_lower

    # Trend.
    macd_line, signal_line, histogram = compute_macd(df["close"])
    result["macd"] = macd_line
    result["macd_signal"] = signal_line
    result["macd_histogram"] = histogram
    result["adx_14"] = compute_adx(df, 14)

    # Volume.
    result["obv"] = compute_obv(df)
    result["volume_sma_20"] = compute_sma(df["volume"].astype(float), 20)

    return result
