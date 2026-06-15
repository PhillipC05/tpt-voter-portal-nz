"use client";

import { useEffect, useRef, useState } from "react";

interface IRVRound {
  counts: Record<string, number>;
  eliminated: number[];
  totalActive: number;
}

interface IRVResult {
  winner: number | null;
  rounds: IRVRound[];
}

interface Tally {
  poll: {
    options: string[];
  };
  counts: Record<string, number>;
  totalVotes: number;
  auditRoot: string;
  irvResult?: IRVResult;
}

interface ResultsChartProps {
  tally: Tally;
  pollId: string;
}

export default function ResultsChart({ tally: initialTally, pollId }: ResultsChartProps) {
  const [tally, setTally] = useState<Tally>(initialTally);
  const [live, setLive] = useState(false);
  const lastFetchRef = useRef(0);

  useEffect(() => {
    setTally(initialTally);
  }, [initialTally]);

  useEffect(() => {
    const es = new EventSource(`/polls/${pollId}/live-results`);

    es.onopen = () => setLive(true);
    es.onerror = () => setLive(false);

    es.onmessage = () => {
      const now = Date.now();
      if (now - lastFetchRef.current < 2000) return;
      lastFetchRef.current = now;
      fetch(`/polls/${pollId}/results`)
        .then((r) => (r.ok ? r.json() : null))
        .then((data: Tally | null) => {
          if (data) setTally(data);
        })
        .catch(() => {});
    };

    return () => es.close();
  }, [pollId]);

  const { poll, counts, totalVotes, auditRoot, irvResult } = tally;

  const results = poll.options.map((option, idx) => ({
    option,
    idx,
    count: counts[String(idx)] ?? 0,
    pct: totalVotes > 0 ? ((counts[String(idx)] ?? 0) / totalVotes) * 100 : 0,
  }));

  const fptp_leader =
    !irvResult && totalVotes > 0
      ? results.reduce((a, b) => (a.count >= b.count ? a : b))
      : null;

  return (
    <div>
      <div className="flex items-center justify-between mb-4">
        <span className="text-sm text-gray-500">
          {totalVotes} total vote{totalVotes === 1 ? "" : "s"}
        </span>
        {live && (
          <span className="flex items-center gap-1.5 text-xs text-green-700 font-medium">
            <span className="w-2 h-2 bg-green-500 rounded-full animate-pulse" />
            Live
          </span>
        )}
      </div>

      {totalVotes === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-8 text-center text-gray-500">
          No votes have been cast yet.
        </div>
      ) : (
        <div className="space-y-4">
          {results.map((r) => {
            const isIRVWinner = irvResult?.winner === r.idx;
            const isFPTPLeader = fptp_leader?.idx === r.idx;
            return (
              <div key={r.idx} className="bg-white rounded-lg shadow-sm p-5">
                <div className="flex items-center justify-between mb-2">
                  <div className="flex items-center gap-2">
                    <span className="text-base font-medium text-gray-900">
                      {r.option}
                    </span>
                    {isIRVWinner && (
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                        Winner (IRV)
                      </span>
                    )}
                    {isFPTPLeader && (
                      <span className="inline-flex items-center px-2 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                        Leading
                      </span>
                    )}
                  </div>
                  <span className="text-base font-semibold text-gray-700">
                    {r.count} ({r.pct.toFixed(1)}%)
                  </span>
                </div>
                <div className="w-full bg-gray-100 rounded-full h-3">
                  <div
                    className="bg-blue-600 h-3 rounded-full transition-all duration-500"
                    style={{ width: `${r.pct}%` }}
                  />
                </div>
              </div>
            );
          })}
        </div>
      )}

      {irvResult && irvResult.rounds.length > 0 && (
        <div className="mt-6">
          <h3 className="text-sm font-semibold text-gray-700 mb-3">
            IRV Elimination Rounds
          </h3>
          <div className="space-y-2">
            {irvResult.rounds.map((round, ri) => (
              <div key={ri} className="bg-white rounded-lg shadow-sm p-4 text-sm">
                <div className="font-medium text-gray-700 mb-2">Round {ri + 1}</div>
                <div className="grid grid-cols-2 gap-x-6 gap-y-1 text-xs text-gray-600 mb-2">
                  {poll.options.map((opt, oi) => {
                    if (round.eliminated.includes(oi)) return null;
                    return (
                      <div key={oi} className="flex justify-between">
                        <span>{opt}</span>
                        <span className="font-semibold tabular-nums">
                          {round.counts[String(oi)] ?? 0}
                        </span>
                      </div>
                    );
                  })}
                </div>
                {round.eliminated.length > 0 && (
                  <div className="text-xs text-red-600">
                    Eliminated:{" "}
                    {round.eliminated.map((i) => poll.options[i]).join(", ")}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="mt-8 bg-gray-50 border border-gray-200 rounded-lg p-4">
        <h3 className="text-sm font-semibold text-gray-700 mb-1">Audit Root</h3>
        <p className="text-xs text-gray-500 mb-2">
          SHA-256 of all ballot commitments (lexicographically sorted). Recompute
          from the Audit Proof tab to independently verify this tally.
        </p>
        <div className="font-mono text-xs text-gray-600 break-all">{auditRoot}</div>
      </div>
    </div>
  );
}
