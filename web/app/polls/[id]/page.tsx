"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import BallotForm from "@/components/BallotForm";

interface Poll {
  id: string;
  title: string;
  description: string;
  options: string[];
  status: string;
  opensAt: string;
  closesAt: string;
}

interface BallotReceipt {
  receiptToken: string;
  pollId: string;
  choiceIndex: number;
  castAt: string;
}

export default function PollPage() {
  const params = useParams<{ id: string }>();
  const pollId = params.id;

  const [poll, setPoll] = useState<Poll | null>(null);
  const [receipt, setReceipt] = useState<BallotReceipt | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchData = async () => {
      try {
        const [pollRes, receiptRes] = await Promise.allSettled([
          fetch(`/polls/${pollId}`),
          fetch(`/polls/${pollId}/my-receipt`),
        ]);

        if (pollRes.status === "fulfilled" && pollRes.value.ok) {
          setPoll(await pollRes.value.json());
        } else if (pollRes.status === "fulfilled" && !pollRes.value.ok) {
          setError("Poll not found.");
        }

        if (
          receiptRes.status === "fulfilled" &&
          receiptRes.value.status === 200
        ) {
          setReceipt(await receiptRes.value.json());
        }
      } catch {
        setError("Failed to load poll.");
      } finally {
        setLoading(false);
      }
    };
    fetchData();
  }, [pollId]);

  const handleVote = async (choiceIndex: number): Promise<BallotReceipt> => {
    const res = await fetch(`/polls/${pollId}/vote`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ choiceIndex }),
    });

    if (res.status === 401)
      throw new Error("Please sign in with RealMe to vote.");
    if (res.status === 403)
      throw new Error(
        "You must register as a voter before casting a ballot. Visit /register first."
      );
    if (!res.ok) {
      const data = await res.json();
      throw new Error(data.error || "Vote failed");
    }

    const rec: BallotReceipt = await res.json();
    setReceipt(rec);
    return rec;
  };

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <p className="text-gray-600">Loading poll...</p>
      </div>
    );
  }

  if (error || !poll) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error || "Poll not found."}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 py-12">
      <div className="mb-2">
        <a href="/polls" className="text-sm text-blue-600 hover:text-blue-800">
          ← Active Polls
        </a>
      </div>
      <h1 className="text-3xl font-bold text-gray-900 mt-4 mb-2">
        {poll.title}
      </h1>
      {poll.description && (
        <p className="text-gray-600 mb-6">{poll.description}</p>
      )}

      <div className="text-sm text-gray-500 mb-8">
        Closes{" "}
        {new Date(poll.closesAt).toLocaleDateString("en-NZ", {
          day: "numeric",
          month: "long",
          year: "numeric",
          hour: "2-digit",
          minute: "2-digit",
        })}
      </div>

      {receipt ? (
        <div className="bg-green-50 border border-green-200 rounded-lg p-6">
          <h2 className="text-lg font-semibold text-green-800 mb-2">
            You have voted
          </h2>
          <p className="text-green-700 mb-4">
            Your ballot has been recorded. Keep your receipt token to verify
            your vote in the public audit.
          </p>
          <div className="bg-white rounded border border-green-200 p-3 font-mono text-sm text-gray-800 break-all">
            {receipt.receiptToken}
          </div>
          <div className="mt-4 flex gap-3 flex-wrap">
            <a
              href={`/results/${poll.id}`}
              className="text-sm text-blue-600 hover:text-blue-800 font-medium"
            >
              View Results →
            </a>
            <a
              href={`/results/${poll.id}?receipt=${receipt.receiptToken}`}
              className="text-sm text-blue-600 hover:text-blue-800 font-medium"
            >
              Verify Your Vote →
            </a>
          </div>
        </div>
      ) : poll.status !== "open" ? (
        <div className="bg-gray-50 border border-gray-200 rounded-lg p-6 text-gray-600">
          This poll is no longer open for voting.{" "}
          <a
            href={`/results/${poll.id}`}
            className="text-blue-600 hover:text-blue-800"
          >
            View results →
          </a>
        </div>
      ) : (
        <BallotForm poll={poll} onVote={handleVote} />
      )}
    </div>
  );
}
