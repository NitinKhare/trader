"use client";

import { RegimeResponse } from "@/types/api";
import { formatPercent } from "@/utils/formatting";

interface RegimeIndicatorProps {
  regime: RegimeResponse;
}

const REGIME_CONFIG: Record<string, { color: string; bg: string; icon: string }> = {
  TRENDING: { color: "text-green-700", bg: "bg-green-50 border-green-200", icon: "📈" },
  RANGING: { color: "text-amber-700", bg: "bg-amber-50 border-amber-200", icon: "↔️" },
  HIGH_VOLATILITY: { color: "text-red-700", bg: "bg-red-50 border-red-200", icon: "⚡" },
  LOW_VOLATILITY: { color: "text-blue-700", bg: "bg-blue-50 border-blue-200", icon: "😴" },
};

const AI_REGIME_CONFIG: Record<string, { color: string; bg: string; icon: string }> = {
  BULL: { color: "text-green-700", bg: "bg-green-50 border-green-200", icon: "🐂" },
  SIDEWAYS: { color: "text-amber-700", bg: "bg-amber-50 border-amber-200", icon: "➡️" },
  BEAR: { color: "text-red-700", bg: "bg-red-50 border-red-200", icon: "🐻" },
};

export default function RegimeIndicator({ regime }: RegimeIndicatorProps) {
  const detected = REGIME_CONFIG[regime.detected_regime] || REGIME_CONFIG.RANGING;
  const ai = AI_REGIME_CONFIG[regime.ai_regime] || AI_REGIME_CONFIG.SIDEWAYS;

  return (
    <div className="bg-white rounded-2xl shadow-md border border-slate-100 p-6">
      <h3 className="text-lg font-black text-slate-900 mb-6">Market Regime</h3>

      <div className="grid grid-cols-2 gap-4 mb-6">
        {/* Detected Regime */}
        <div className={`rounded-xl border-2 p-4 ${detected.bg}`}>
          <p className="text-xs font-bold text-slate-500 uppercase mb-2">Technical Regime</p>
          <div className="flex items-center gap-2">
            <span className="text-2xl">{detected.icon}</span>
            <span className={`text-lg font-black ${detected.color}`}>
              {regime.detected_regime.replace("_", " ")}
            </span>
          </div>
        </div>

        {/* AI Regime */}
        <div className={`rounded-xl border-2 p-4 ${ai.bg}`}>
          <p className="text-xs font-bold text-slate-500 uppercase mb-2">AI Regime</p>
          <div className="flex items-center gap-2">
            <span className="text-2xl">{ai.icon}</span>
            <span className={`text-lg font-black ${ai.color}`}>{regime.ai_regime}</span>
          </div>
        </div>
      </div>

      {/* Metrics */}
      <div className="grid grid-cols-2 gap-4">
        <div className="bg-slate-50 rounded-xl p-4">
          <p className="text-xs font-bold text-slate-500 uppercase mb-1">ADX</p>
          <p className="text-xl font-black text-slate-900">{regime.adx.toFixed(1)}</p>
          <p className="text-xs text-slate-500 mt-1">
            {regime.adx > 25 ? "Strong trend" : "Weak trend"}
          </p>
        </div>
        <div className="bg-slate-50 rounded-xl p-4">
          <p className="text-xs font-bold text-slate-500 uppercase mb-1">ATR Percentile</p>
          <p className="text-xl font-black text-slate-900">
            {formatPercent(regime.atr_percentile * 100, 0)}
          </p>
          <p className="text-xs text-slate-500 mt-1">
            {regime.atr_percentile > 0.7 ? "High volatility" : "Normal"}
          </p>
        </div>
        <div className="bg-slate-50 rounded-xl p-4">
          <p className="text-xs font-bold text-slate-500 uppercase mb-1">Volatility</p>
          <p className="text-xl font-black text-slate-900">
            {formatPercent(regime.volatility * 100, 1)}
          </p>
        </div>
        <div className="bg-slate-50 rounded-xl p-4">
          <p className="text-xs font-bold text-slate-500 uppercase mb-1">Confidence</p>
          <p className="text-xl font-black text-slate-900">
            {formatPercent(regime.confidence * 100, 0)}
          </p>
          <div className="mt-2 h-2 bg-slate-200 rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all duration-500"
              style={{ width: `${regime.confidence * 100}%` }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
