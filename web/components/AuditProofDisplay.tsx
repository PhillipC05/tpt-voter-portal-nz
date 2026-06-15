"use client";

import { useEffect, useState, useCallback } from "react";

interface AuditEntry {
  receiptToken: string;
  choiceIndex: number;
  rankings?: number[];
  commitment: string;
  castAt: string;
}

interface AuditProof {
  pollId: string;
  entries: AuditEntry[];
  auditRoot: string;
  total: number;
  offset: number;
  limit: number;
}

interface AuditProofDisplayProps {
  pollId: string;
  auditRoot: string;
  pollOptions: string[];
}

const PAGE_SIZE = 100;

export default function AuditProofDisplay({
  pollId,
  auditRoot,
  pollOptions,
}: AuditProofDisplayProps) {
  const [proof, setProof] = useState<AuditProof | null>(null);
  const [allEntries, setAllEntries] = useState<AuditEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [searchToken, setSearchToken] = useState("");
  const [searchResult, setSearchResult] = useState<AuditEntry | null | "not-found">(null);

  const fetchPage = useCallback(async (offset: number, append: boolean) => {
    try {
      const res = await fetch(
        `/polls/${pollId}/audit?offset=${offset}&limit=${PAGE_SIZE}`
      );
      if (!res.ok) throw new Error("Failed to load audit proof");
      const data: AuditProof = await res.json();
      setProof(data);
      setAllEntries((prev) => (append ? [...prev, ...data.entries] : data.entries));
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Failed to load audit");
    }
  }, [pollId]);

  useEffect(() => {
    fetchPage(0, false).finally(() => setLoading(false));
  }, [fetchPage]);

  const handleLoadMore = async () => {
    if (!proof) return;
    setLoadingMore(true);
    await fetchPage(allEntries.length, true);
    setLoadingMore(false);
  };

  const handleSearch = () => {
    if (!searchToken.trim()) return;
    const found = allEntries.find((e) => e.receiptToken === searchToken.trim());
    setSearchResult(found ?? "not-found");
  };

  const choiceLabel = (entry: AuditEntry): string => {
    if (entry.rankings && entry.rankings.length > 0) {
      return entry.rankings
        .map((idx, rank) => `${rank + 1}. ${pollOptions[idx] ?? idx}`)
        .join(" → ");
    }
    return pollOptions[entry.choiceIndex] ?? String(entry.choiceIndex);
  };

  const handleExportCSV = () => {
    const header = "receiptToken,choiceIndex,commitment,castAt\n";
    const rows = allEntries
      .map(
        (e) =>
          `${e.receiptToken},${e.choiceIndex},"${e.commitment}",${e.castAt}`
      )
      .join("\n");
    download("audit.csv", "text/csv", header + rows);
  };

  const handleExportJSON = () => {
    download(
      "audit.json",
      "application/json",
      JSON.stringify(allEntries, null, 2)
    );
  };

  const download = (filename: string, mime: string, content: string) => {
    const url = URL.createObjectURL(new Blob([content], { type: mime }));
    const a = document.createElement("a");
    a.href = url;
    a.download = filename;
    a.click();
    URL.revokeObjectURL(url);
  };

  if (loading) return <p className="text-gray-600">Loading audit proof...</p>;
  if (error || !proof) {
    return (
      <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
        {error || "Audit proof not available."}
      </div>
    );
  }

  const hasMore = allEntries.length < proof.total;

  return (
    <div>
      {/* How-to box */}
      <div className="bg-blue-50 border border-blue-200 rounded-lg p-4 mb-6">
        <h3 className="text-sm font-semibold text-blue-800 mb-1">
          How to verify independently
        </h3>
        <ol className="text-xs text-blue-700 space-y-1 list-decimal list-inside">
          <li>Download all {proof.total} commitment hashes (use JSON export below).</li>
          <li>Sort them lexicographically (standard string sort).</li>
          <li>Concatenate them in order and compute SHA-256.</li>
          <li>The result must match the Audit Root shown on the Results tab.</li>
        </ol>
      </div>

      {/* Export buttons */}
      <div className="flex gap-2 mb-4">
        <button
          onClick={handleExportCSV}
          className="px-3 py-1.5 bg-white border border-gray-300 rounded-md text-xs font-medium text-gray-700 hover:bg-gray-50"
        >
          Export CSV
        </button>
        <button
          onClick={handleExportJSON}
          className="px-3 py-1.5 bg-white border border-gray-300 rounded-md text-xs font-medium text-gray-700 hover:bg-gray-50"
        >
          Export JSON
        </button>
        <span className="text-xs text-gray-400 self-center ml-auto">
          Showing {allEntries.length} of {proof.total} ballots
        </span>
      </div>

      {/* Receipt search */}
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
            Receipt not found in loaded ballots.{" "}
            {hasMore && "Try loading all ballots first."}
          </p>
        )}
        {searchResult && searchResult !== "not-found" && (
          <div className="mt-3 bg-green-50 border border-green-200 rounded-lg p-3 text-sm">
            <p className="text-green-800 font-medium mb-1">Vote verified.</p>
            <p className="text-green-700 text-xs">
              Choice: <strong>{choiceLabel(searchResult)}</strong>
            </p>
            <p className="text-green-700 text-xs mt-1">
              Commitment:{" "}
              <span className="font-mono">{searchResult.commitment}</span>
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

      {/* Ballot table */}
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
              {allEntries.map((entry) => (
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
                  <td className="px-4 py-2 text-gray-700 max-w-xs">
                    {choiceLabel(entry)}
                  </td>
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
          {allEntries.length === 0 && (
            <p className="p-6 text-center text-gray-500 text-sm">
              No ballots cast yet.
            </p>
          )}
        </div>

        {hasMore && (
          <div className="p-4 border-t border-gray-100 text-center">
            <button
              onClick={handleLoadMore}
              disabled={loadingMore}
              className="px-4 py-2 text-sm text-blue-700 font-medium hover:text-blue-900 disabled:opacity-50"
            >
              {loadingMore
                ? "Loading..."
                : `Load more (${proof.total - allEntries.length} remaining)`}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
