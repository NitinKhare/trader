"use client";

import { useState } from "react";

export type DateRangePreset = "all" | "1y" | "3m" | "1m" | "1w";

interface DateRangeSelectorProps {
  onRangeChange: (from: Date | undefined, to: Date | undefined) => void;
}

export function DateRangeSelector({ onRangeChange }: DateRangeSelectorProps) {
  const [selectedPreset, setSelectedPreset] = useState<DateRangePreset>("1y");

  const handlePresetClick = (preset: DateRangePreset) => {
    setSelectedPreset(preset);

    const now = new Date();
    let from: Date | undefined = undefined;

    switch (preset) {
      case "all":
        // All data (no date filter)
        from = undefined;
        break;
      case "1y":
        // Last 1 year
        from = new Date(now);
        from.setFullYear(from.getFullYear() - 1);
        break;
      case "3m":
        // Last 3 months
        from = new Date(now);
        from.setMonth(from.getMonth() - 3);
        break;
      case "1m":
        // Last 1 month
        from = new Date(now);
        from.setMonth(from.getMonth() - 1);
        break;
      case "1w":
        // Last 1 week
        from = new Date(now);
        from.setDate(from.getDate() - 7);
        break;
    }

    onRangeChange(from, now);
  };

  const presets: { label: string; value: DateRangePreset }[] = [
    { label: "All", value: "all" },
    { label: "1Y", value: "1y" },
    { label: "3M", value: "3m" },
    { label: "1M", value: "1m" },
    { label: "1W", value: "1w" },
  ];

  return (
    <div className="flex gap-2 mb-6">
      {presets.map((preset) => (
        <button
          key={preset.value}
          onClick={() => handlePresetClick(preset.value)}
          className={`px-4 py-2 rounded-lg font-semibold transition-all ${
            selectedPreset === preset.value
              ? "bg-blue-600 text-white shadow-lg"
              : "bg-slate-100 text-slate-700 hover:bg-slate-200"
          }`}
        >
          {preset.label}
        </button>
      ))}
    </div>
  );
}
