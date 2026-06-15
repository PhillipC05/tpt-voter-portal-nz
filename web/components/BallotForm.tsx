"use client";

import { useState } from "react";
import { QRCode } from "@tpt-nz/ui-shared";

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

interface BallotFormProps {
  poll: Poll;
  onVote: (choiceIndex: number) => Promise<BallotReceipt>;
}

export default function BallotForm({ poll, onVote }: BallotFormProps) {
  const [selected, setSelected] = useState<number | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [receipt, setReceipt] = useState<BallotReceipt | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (selected === null) return;
    setSubmitting(true);
    setError(null);
    try {
      const rec = await onVote(selected);
      setReceipt(rec);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Vote failed. Please try again.");
    } finally {
      setSubmitting(false);
    }
  };

  if (receipt) {
    const verifyUrl = `${typeof window !== "undefined" ? window.location.origin : ""}/results/${poll.id}?receipt=${encodeURIComponent(receipt.receiptToken)}`;
    return (
      <div className="bg-green-50 border border-green-200 rounded-lg p-6">
        <h2 className="text-lg font-semibold text-green-800 mb-2">
          Ballot cast successfully
        </h2>
        <p className="text-green-700 mb-1">
          Your vote for{" "}
          <strong>&ldquo;{poll.options[receipt.choiceIndex]}&rdquo;</strong> has
          been recorded.
        </p>
        <p className="text-sm text-green-700 mb-4">
          Save your receipt token to verify your vote in the public audit:
        </p>
        <div className="bg-white rounded border border-green-200 p-3 font-mono text-sm text-gray-800 break-all mb-4">
          {receipt.receiptToken}
        </div>
        <div className="flex flex-col items-center gap-2 mb-4">
          <QRCode
            value={verifyUrl}
            size={160}
            label="Scan to verify your vote"
            errorCorrectionLevel="M"
          />
        </div>
        <p className="text-xs text-gray-500">
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
      <h2 className="text-lg font-semibold text-gray-900 mb-4">
        Cast your ballot
      </h2>

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-3 mb-4 text-sm text-red-700">
          {error}
        </div>
      )}

      <fieldset className="mb-6">
        <legend className="sr-only">Choose your option</legend>
        <div className="space-y-3">
          {poll.options.map((option, idx) => (
            <label
              key={idx}
              className={`flex items-center gap-3 p-4 rounded-lg border-2 cursor-pointer transition-colors ${
                selected === idx
                  ? "border-blue-600 bg-blue-50"
                  : "border-gray-200 hover:border-gray-300"
              }`}
            >
              <input
                type="radio"
                name="choice"
                value={idx}
                checked={selected === idx}
                onChange={() => setSelected(idx)}
                className="h-4 w-4 text-blue-600 border-gray-300 focus:ring-blue-500"
              />
              <span className="text-gray-900 font-medium">{option}</span>
            </label>
          ))}
        </div>
      </fieldset>

      <div className="bg-amber-50 border border-amber-200 rounded-lg p-3 mb-6 text-xs text-amber-800">
        Your vote is anonymous. Once submitted it cannot be changed. You will
        receive a receipt token to verify your vote in the public audit.
      </div>

      <button
        type="submit"
        disabled={selected === null || submitting}
        className="w-full inline-flex justify-center items-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800 disabled:opacity-50 disabled:cursor-not-allowed"
      >
        {submitting ? "Submitting ballot..." : "Submit Ballot"}
      </button>
    </form>
  );
}
