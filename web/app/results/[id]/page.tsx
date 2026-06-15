"use client";

import { useEffect, useState } from "react";
import { useParams, useSearchParams } from "next/navigation";
import ResultsChart from "@/components/ResultsChart";
import AuditProofDisplay from "@/components/AuditProofDisplay";

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
    id: string;
    title: string;
    description: string;
    options: string[];
    status: string;
    closesAt: string;
  };
  counts: Record<string, number>;
  totalVotes: number;
  auditRoot: string;
  computedAt: string;
  irvResult?: IRVResult;
}

interface VerifyResult {
  verified: boolean;
  entry: {
    receiptToken: string;
    choiceIndex: number;
    commitment: string;
    castAt: string;
  } | null;
}

export default function ResultsPage() {
  const params = useParams<{ id: string }>();
  const searchParams = useSearchParams();
  const pollId = params.id;
  const receiptParam = searchParams.get("receipt");

  const [tally, setTally] = useState<Tally | null>(null);
  const [verifyResult, setVerifyResult] = useState<VerifyResult | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [tab, setTab] = useState<"results" | "audit">("results");

  useEffect(() => {
    const fetchData = async () => {
      try {
        const res = await fetch(`/polls/${pollId}/results`);
        if (!res.ok) throw new Error("Failed to load results");
        setTally(await res.json());

        if (receiptParam) {
          const vRes = await fetch(
            `/polls/${pollId}/verify?receipt=${encodeURIComponent(receiptParam)}`
          );
          if (vRes.ok) setVerifyResult(await vRes.json());
        }
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [pollId, receiptParam]);

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-12">
        <p className="text-gray-600">Loading results...</p>
      </div>
    );
  }

  if (error || !tally) {
    return (
      <div className="max-w-4xl mx-auto px-4 py-12">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error || "Results not found."}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 py-12">
      <div className="mb-2">
        <a href="/polls" className="text-sm text-blue-600 hover:text-blue-800">
          ← Active Polls
        </a>
      </div>

      <h1 className="text-3xl font-bold text-gray-900 mt-4 mb-1">
        {tally.poll.title}
      </h1>
      <p className="text-sm text-gray-500 mb-8">
        {tally.totalVotes} vote{tally.totalVotes === 1 ? "" : "s"} &middot;
        Last computed{" "}
        {new Date(tally.computedAt).toLocaleDateString("en-NZ", {
          day: "numeric",
          month: "short",
          year: "numeric",
        })}
      </p>

      {verifyResult !== null && (
        <div
          className={`rounded-lg p-4 mb-6 ${
            verifyResult.verified
              ? "bg-green-50 border border-green-200 text-green-800"
              : "bg-red-50 border border-red-200 text-red-700"
          }`}
        >
          {verifyResult.verified
            ? "Your vote has been verified — it appears in the public ballot list."
            : "Receipt not found in this poll's ballot list."}
        </div>
      )}

      <div className="flex border-b border-gray-200 mb-6">
        <button
          onClick={() => setTab("results")}
          className={`px-4 py-2 text-sm font-medium border-b-2 ${
            tab === "results"
              ? "border-blue-600 text-blue-600"
              : "border-transparent text-gray-500 hover:text-gray-700"
          }`}
        >
          Results
        </button>
        <button
          onClick={() => setTab("audit")}
          className={`px-4 py-2 text-sm font-medium border-b-2 ${
            tab === "audit"
              ? "border-blue-600 text-blue-600"
              : "border-transparent text-gray-500 hover:text-gray-700"
          }`}
        >
          Audit Proof
        </button>
      </div>

      {tab === "results" ? (
        <ResultsChart tally={tally} pollId={pollId} />
      ) : (
        <AuditProofDisplay
          pollId={pollId}
          auditRoot={tally.auditRoot}
          pollOptions={tally.poll.options}
        />
      )}
    </div>
  );
}
