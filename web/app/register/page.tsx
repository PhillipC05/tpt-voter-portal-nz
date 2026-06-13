"use client";

import { useEffect, useState } from "react";

interface RegistrationStatus {
  registered: boolean;
  eligible: boolean;
  status?: string;
  registeredAt?: string;
}

export default function RegisterPage() {
  const [regStatus, setRegStatus] = useState<RegistrationStatus | null>(null);
  const [loading, setLoading] = useState(true);
  const [registering, setRegistering] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  useEffect(() => {
    const fetchStatus = async () => {
      try {
        const res = await fetch("/register/status");
        if (res.status === 401) {
          setRegStatus(null);
          return;
        }
        if (!res.ok) throw new Error("Failed to load status");
        const data = await res.json();
        setRegStatus(data);
      } catch {
        // Not signed in or error — handled below
      } finally {
        setLoading(false);
      }
    };
    fetchStatus();
  }, []);

  const handleRegister = async () => {
    setRegistering(true);
    setError(null);
    try {
      const res = await fetch("/register", { method: "POST" });
      if (res.status === 401) {
        setError("Please sign in with RealMe Verified to register.");
        return;
      }
      if (res.status === 403) {
        setError(
          "RealMe Verified identity is required. Please sign in with a verified account."
        );
        return;
      }
      if (!res.ok) {
        const data = await res.json();
        setError(data.error || "Registration failed");
        return;
      }
      setSuccess(true);
      setRegStatus({ registered: true, eligible: true });
    } catch {
      setError("Network error. Please try again.");
    } finally {
      setRegistering(false);
    }
  };

  if (loading) {
    return (
      <div className="max-w-2xl mx-auto px-4 py-12">
        <p className="text-gray-600">Loading...</p>
      </div>
    );
  }

  return (
    <div className="max-w-2xl mx-auto px-4 sm:px-6 py-12">
      <h1 className="text-3xl font-bold text-gray-900 mb-2">Voter Registration</h1>
      <p className="text-gray-600 mb-8">
        Register once to join the voter roll and participate in all open polls.
        RealMe Verified identity is required.
      </p>

      {success && (
        <div className="bg-green-50 border border-green-200 rounded-lg p-4 mb-6 text-green-800">
          You are registered and eligible to vote.
        </div>
      )}

      {error && (
        <div className="bg-red-50 border border-red-200 rounded-lg p-4 mb-6 text-red-700">
          {error}
        </div>
      )}

      {regStatus?.registered ? (
        <div className="bg-white rounded-lg shadow-sm p-8">
          <div className="flex items-center gap-3 mb-4">
            <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-green-100 text-green-800">
              Registered
            </span>
            {regStatus.eligible && (
              <span className="inline-flex items-center px-3 py-1 rounded-full text-sm font-medium bg-blue-100 text-blue-800">
                Eligible to Vote
              </span>
            )}
          </div>
          <p className="text-gray-700 mb-2">
            You are on the voter roll and may cast ballots in any open poll.
          </p>
          {regStatus.registeredAt && (
            <p className="text-sm text-gray-500">
              Registered:{" "}
              {new Date(regStatus.registeredAt).toLocaleDateString("en-NZ", {
                day: "numeric",
                month: "long",
                year: "numeric",
              })}
            </p>
          )}
          <div className="mt-6">
            <a
              href="/polls"
              className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800"
            >
              View Active Polls
            </a>
          </div>
        </div>
      ) : (
        <div className="bg-white rounded-lg shadow-sm p-8">
          <h2 className="text-lg font-semibold text-gray-900 mb-4">
            Join the Voter Roll
          </h2>
          <p className="text-gray-600 mb-6">
            To register, you must be signed in with a RealMe Verified identity.
            Your personal details are verified by the Department of Internal
            Affairs but are not stored by this service.
          </p>

          <div className="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-6 text-sm text-amber-800">
            <strong>Privacy notice:</strong> Only a cryptographic hash of your
            RealMe FLT (Federated Login Token) is stored. Your name, date of
            birth, and address are not recorded. This is in accordance with the
            Privacy Act 2020.
          </div>

          {regStatus === null ? (
            <a
              href="/auth/login"
              className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800"
            >
              Sign In with RealMe to Register
            </a>
          ) : (
            <button
              onClick={handleRegister}
              disabled={registering}
              className="inline-flex items-center px-6 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800 disabled:opacity-50"
            >
              {registering ? "Registering..." : "Register as a Voter"}
            </button>
          )}
        </div>
      )}
    </div>
  );
}
