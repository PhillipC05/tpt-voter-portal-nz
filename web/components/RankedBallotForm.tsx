"use client";

import { useState } from "react";

interface Poll {
  id: string;
  title: string;
  options: string[];
}

interface BallotReceipt {
  receiptToken: string;
  pollId: string;
  choiceIndex: number;
  castAt: string;
}

interface RankedBallotFormProps {
  poll: Poll;
  onVote: (rankings: number[]) => Promise<BallotReceipt>;
}

export default function RankedBallotForm({ poll, onVote }: RankedBallotFormProps) {
  // rankings[i] = index of the (i+1)th preference choice; null = unranked
  const [rankings, setRankings] = useState<(number | null)[]>(
    Array(poll.options.length).fill(null)
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<BallotReceipt | null>(null);

  // Build ranked list from the ranking slots (drop nulls)
  const buildRankings = (): number[] =>
    rankings.filter((r): r is number => r !== null);

  const handleRankChange = (optionIdx: number, rank: number | null) => {
    setRankings((prev) => {
      const next = [...prev];
      // Clear any previous slot that held this rank
      if (rank !== null) {
        for (let i = 0; i < next.length; i++) {
          if (next[i] === rank && i !== optionIdx) next[i] = null;
        }
      }
      next[optionIdx] = rank;
      return next;
    });
  };

  const rankForOption = (optionIdx: number): number | null =>
    rankings[optionIdx];

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const ranked = buildRankings();
    if (ranked.length === 0) {
      setError("Please rank at least one option.");
      return;
    }
    setSubmitting(true);
    setError(null);
    try {
      const rec = await onVote(ranked);
      setReceipt(rec);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Vote failed. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  if (receipt) {
    return (
      <div className="bg-green-50 border border-green-200 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-green-800 mb-2">
          Ranked ballot cast successfully
        </h2>
        <p className="text-green-700 mb-4 text-sm">
          Your preferences have been recorded. Save your receipt token to verify
          your vote in the public audit:
        </p>
        <div className="bg-white rounded border border-green-200 p-3 font-mono text-sm text-gray-800 break-all">
          {receipt.receiptToken}
        </div>
        <p className="text-xs text-gray-500 mt-2">
          Cast at{" "}
          {new Date(receipt.castAt).toLocaleString("en-NZ", {
            day: "numeric",
            month: "short",
            year: "numeric",
            hour: "2-digit",
            minute: "2-digit",
          })}
        </p>
      </div>
    );
  }

  return (
    <form onSubmit={handleSubmit} className="bg-white rounded-lg shadow-sm p-6">
      <h2 className="text-lg font-semibold text-gray-900 mb-1">
        Rank your preferences
      </h2>
      <p className="text-sm text-gray-500 mb-4">
        Assign a rank to each option. 1 = most preferred. You may leave options
        unranked — they will not receive any preference from your ballot.
      </p>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 mb-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <fieldset className="mb-6">
        <legend className="sr-only">Rank each option</legend>
        <div className="space-y-3">
          {poll.options.map((option, optIdx) => {
            const currentRank = rankForOption(optIdx);
            return (
              <div
                key={optIdx}
                className={`flex items-center gap-3 p-4 rounded-lg border-2 transition-colors ${
                  currentRank !== null
                    ? "border-blue-600 bg-blue-50"
                    : "border-gray-200"
                }`}
              >
                <select
                  value={currentRank ?? ""}
                  onChange={(e) =>
                    handleRankChange(
                      optIdx,
                      e.target.value === "" ? null : Number(e.target.value)
                    )
                  }
                  className="w-20 border border-gray-300 rounded-md px-2 py-1 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 bg-white"
                  aria-label={`Rank for ${option}`}
                >
                  <option value="">—</option>
                  {poll.options.map((_, rank) => (
                    <option key={rank} value={rank}>
                      {rank + 1}
                    </option>
                  ))}
                </select>
                <span className="text-gray-900 font-medium flex-1">{option}</span>
                {currentRank !== null && (
                  <span className="text-xs text-blue-700 font-semibold">
                    #{currentRank + 1}
                  </span>
                )}
              </div>
            );
          })}
        </div>
      </fieldset>

      <div className="bg-amber-50 border border-amber-200 rounded-lg p-3 mb-6 text-xs text-amber-800">
        Ranked-choice ballots use Instant Runoff Voting (IRV). Your vote
        transfers to your next preference if your top choice is eliminated.
        Once submitted it cannot be changed.
      </div>

      <button
        type="submit"
        disabled={buildRankings().length === 0 || submitting}
        className="w-full inline-flex justify-center items-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {submitting ? "Submitting ballot..." : "Submit Ranked Ballot"}
      </button>
    </form>
  );
}
