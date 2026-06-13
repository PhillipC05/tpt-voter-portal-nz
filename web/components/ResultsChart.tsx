"use client";

interface Tally {
  poll: {
    options: string[];
  };
  counts: Record<string, number>;
  totalVotes: number;
  auditRoot: string;
}

interface ResultsChartProps {
  tally: Tally;
}

export default function ResultsChart({ tally }: ResultsChartProps) {
  const { poll, counts, totalVotes, auditRoot } = tally;

  const results = poll.options.map((option, idx) => ({
    option,
    count: counts[String(idx)] ?? 0,
    pct: totalVotes > 0 ? ((counts[String(idx)] ?? 0) / totalVotes) * 100 : 0,
  }));

  const winner =
    totalVotes > 0
      ? results.reduce((a, b) => (a.count >= b.count ? a : b))
      : null;

  return (
    <div>
      {totalVotes === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-8 text-center text-gray-500">
          No votes have been cast yet.
        </div>
      ) : (
        <div className="space-y-4">
          {results.map((r, idx) => (
            <div key={idx} className="bg-white rounded-lg shadow-sm p-5">
              <div className="flex items-center justify-between mb-2">
                <div className="flex items-center gap-2">
                  <span className="text-base font-medium text-gray-900">
                    {r.option}
                  </span>
                  {winner && r.option === winner.option && totalVotes > 0 && (
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
          ))}

          <div className="text-sm text-gray-500 text-right mt-2">
            {totalVotes} total vote{totalVotes === 1 ? "" : "s"}
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
