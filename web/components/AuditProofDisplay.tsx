"use client";

import { useEffect, useState } from "react";

interface AuditEntry {
  receiptToken: string;
  choiceIndex: number;
  commitment: string;
  castAt: string;
}

interface AuditProof {
  pollId: string;
  entries: AuditEntry[];
  auditRoot: string;
  total: number;
}

interface AuditProofDisplayProps {
  pollId: string;
  auditRoot: string;
}

export default function AuditProofDisplay({
  pollId,
  auditRoot,
}: AuditProofDisplayProps) {
  const [proof, setProof] = useState<AuditProof | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [searchToken, setSearchToken] = useState("");
  const [searchResult, setSearchResult] = useState<AuditEntry | null | "not-found">(null);

  useEffect(() => {
    const fetchProof = async () => {
      try {
        const res = await fetch(`/polls/${pollId}/audit`);
        if (!res.ok) throw new Error("Failed to load audit proof");
        setProof(await res.json());
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Failed to load audit");
      } finally {
        setLoading(false);
      }
    };
    fetchProof();
  }, [pollId]);

  const handleSearch = () => {
    if (!proof || !searchToken.trim()) return;
    const found = proof.entries.find(
      (e) => e.receiptToken === searchToken.trim()
    );
    setSearchResult(found ?? "not-found");
  };

  if (loading) {
    return <p className="text-gray-600">Loading audit proof...</p>;
  }
  if (error || !proof) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
        {error || "Audit proof not available."}
      </div>
    );
  }

  return (
    <div>
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
        <h3 className="text-sm font-semibold text-blue-800 mb-1">
          How to verify independently
        </h3>
        <ol className="text-xs text-blue-700 space-y-1 list-decimal list-inside">
          <li>Download all {proof.total} commitment hashes from this list.</li>
          <li>Sort them lexicographically (standard string sort).</li>
          <li>Concatenate them in order and compute SHA-256.</li>
          <li>The result must match the Audit Root shown on the Results tab.</li>
        </ol>
      </div>

      <div className="bg-white rounded-lg shadow-sm p-5 mb-6">
        <h3 className="text-sm font-semibold text-gray-700 mb-3">
          Verify Your Receipt Token
        </h3>
        <div className="flex gap-2">
          <input
            type="text"
            placeholder="Paste your receipt token..."
            value={searchToken}
            onChange={(e) => {
              setSearchToken(e.target.value);
              setSearchResult(null);
            }}
            className="flex-1 border border-gray-300 rounded-md px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-blue-500"
          />
          <button
            onClick={handleSearch}
            className="px-4 py-2 bg-blue-700 text-white text-sm font-medium rounded-md hover:bg-blue-800"
          >
            Verify
          </button>
        </div>
        {searchResult === "not-found" && (
          <p className="mt-2 text-sm text-red-600">
            Receipt not found in this poll&apos;s audit list.
          </p>
        )}
        {searchResult && searchResult !== "not-found" && (
          <div className="mt-3 bg-green-50 border border-green-200 rounded-lg p-3 text-sm">
            <p className="text-green-800 font-medium mb-1">Vote verified.</p>
            <p className="text-green-700 text-xs">
              Commitment: <span className="font-mono">{searchResult.commitment}</span>
            </p>
            <p className="text-green-700 text-xs mt-1">
              Cast:{" "}
              {new Date(searchResult.castAt).toLocaleString("en-NZ", {
                day: "numeric",
                month: "short",
                year: "numeric",
                hour: "2-digit",
                minute: "2-digit",
              })}
            </p>
          </div>
        )}
      </div>

      <div className="bg-white rounded-lg shadow-sm">
        <div className="p-4 border-b border-gray-100 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-gray-700">
            Public Ballot List ({proof.total} entries)
          </h3>
          <span className="text-xs text-gray-400">
            Voter identities are not included
          </span>
        </div>
        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-gray-100 text-xs">
            <thead className="bg-gray-50">
              <tr>
                <th className="px-4 py-2 text-left text-gray-500 font-medium">
                  Receipt Token
                </th>
                <th className="px-4 py-2 text-left text-gray-500 font-medium">
                  Choice
                </th>
                <th className="px-4 py-2 text-left text-gray-500 font-medium">
                  Commitment
                </th>
                <th className="px-4 py-2 text-left text-gray-500 font-medium">
                  Cast At
                </th>
              </tr>
            </thead>
            <tbody className="divide-y divide-gray-50">
              {proof.entries.map((entry) => (
                <tr
                  key={entry.receiptToken}
                  className={
                    searchResult &&
                    searchResult !== "not-found" &&
                    searchResult.receiptToken === entry.receiptToken
                      ? "bg-green-50"
                      : "hover:bg-gray-50"
                  }
                >
                  <td className="px-4 py-2 font-mono text-gray-600 max-w-xs truncate">
                    {entry.receiptToken}
                  </td>
                  <td className="px-4 py-2 text-gray-700">{entry.choiceIndex}</td>
                  <td className="px-4 py-2 font-mono text-gray-500 max-w-xs truncate">
                    {entry.commitment}
                  </td>
                  <td className="px-4 py-2 text-gray-500 whitespace-nowrap">
                    {new Date(entry.castAt).toLocaleDateString("en-NZ")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {proof.entries.length === 0 && (
            <p className="p-6 text-center text-gray-500 text-sm">
              No ballots cast yet.
            </p>
          )}
        </div>
      </div>
    </div>
  );
}
