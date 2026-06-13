"use client";

import { useEffect, useState } from "react";

interface Poll {
  id: string;
  title: string;
  description: string;
  options: string[];
  status: string;
  opensAt: string;
  closesAt: string;
}

export default function PollsPage() {
  const [polls, setPolls] = useState<Poll[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchPolls = async () => {
      try {
        const res = await fetch("/polls");
        if (!res.ok) throw new Error("Failed to load polls");
        const data = await res.json();
        setPolls(data.items || []);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "An error occurred");
      } finally {
        setLoading(false);
      }
    };
    fetchPolls();
  }, []);

  if (loading) {
    return (
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <p className="text-gray-600">Loading polls...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 text-red-700">
          {error}
        </div>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-12">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-3xl font-bold text-gray-900">Active Polls</h1>
          <p className="mt-1 text-gray-600">
            {polls.length} poll{polls.length === 1 ? "" : "s"} open for voting
          </p>
        </div>
      </div>

      {polls.length === 0 ? (
        <div className="bg-white rounded-lg shadow-sm p-12 text-center">
          <h2 className="text-xl font-semibold text-gray-900 mb-2">
            No active polls
          </h2>
          <p className="text-gray-600">
            There are no polls open for voting at the moment. Check back later.
          </p>
        </div>
      ) : (
        <div className="space-y-4">
          {polls.map((poll) => (
            <a
              key={poll.id}
              href={`/polls/${poll.id}`}
              className="block bg-white rounded-lg shadow-sm p-6 hover:shadow-md transition-shadow"
            >
              <div className="flex items-start justify-between">
                <div className="flex-1">
                  <h2 className="text-lg font-semibold text-gray-900">
                    {poll.title}
                  </h2>
                  {poll.description && (
                    <p className="text-sm text-gray-600 mt-1 line-clamp-2">
                      {poll.description}
                    </p>
                  )}
                  <p className="text-sm text-gray-500 mt-2">
                    {poll.options.length} options &middot; closes{" "}
                    {new Date(poll.closesAt).toLocaleDateString("en-NZ", {
                      day: "numeric",
                      month: "short",
                      year: "numeric",
                    })}
                  </p>
                </div>
                <span className="ml-4 inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Open
                </span>
              </div>
            </a>
          ))}
        </div>
      )}
    </div>
  );
}
