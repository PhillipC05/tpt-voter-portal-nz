"use client";

import { useEffect, useState } from "react";

interface CountdownTimerProps {
  closesAt: string;
}

function formatDuration(ms: number): string {
  if (ms <= 0) return "Closed";
  const totalSeconds = Math.floor(ms / 1000);
  const days = Math.floor(totalSeconds / 86400);
  const hours = Math.floor((totalSeconds % 86400) / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;

  if (days > 0) return `${days}d ${hours}h ${minutes}m`;
  if (hours > 0) return `${hours}h ${minutes}m ${seconds}s`;
  return `${minutes}m ${seconds}s`;
}

export default function CountdownTimer({ closesAt }: CountdownTimerProps) {
  const closeTime = new Date(closesAt).getTime();
  const [remaining, setRemaining] = useState(closeTime - Date.now());

  useEffect(() => {
    const id = setInterval(() => {
      const r = closeTime - Date.now();
      setRemaining(r);
      if (r <= 0) clearInterval(id);
    }, 1000);
    return () => clearInterval(id);
  }, [closeTime]);

  const isUrgent = remaining > 0 && remaining < 3600_000; // < 1 hour

  return (
    <span
      className={`tabular-nums font-mono text-sm ${
        remaining <= 0
          ? "text-gray-400"
          : isUrgent
          ? "text-red-600 font-semibold"
          : "text-gray-600"
      }`}
      aria-live="polite"
    >
      {remaining <= 0 ? "Voting closed" : `Closes in ${formatDuration(remaining)}`}
    </span>
  );
}
