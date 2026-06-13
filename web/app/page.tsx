"use client";

export default function HomePage() {
  return (
    <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div className="py-20 text-center">
        <h1 className="text-5xl font-bold text-gray-900 mb-6">
          Secure, Verified Local Body Polling
        </h1>
        <p className="text-xl text-gray-600 max-w-2xl mx-auto mb-10">
          Vote in local body polls using your RealMe Verified identity. Your
          vote is anonymous, counted once, and publicly auditable — you can
          verify it was included without revealing how you voted.
        </p>
        <div className="flex justify-center gap-4 flex-wrap">
          <a
            href="/polls"
            className="inline-flex items-center px-8 py-3 border border-transparent text-base font-medium rounded-md shadow-sm text-white bg-blue-700 hover:bg-blue-800"
          >
            View Active Polls
          </a>
          <a
            href="/register"
            className="inline-flex items-center px-8 py-3 border border-gray-300 text-base font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50"
          >
            Register as a Voter
          </a>
        </div>
      </div>

      <div className="py-16">
        <h2 className="text-3xl font-bold text-gray-900 text-center mb-12">
          How It Works
        </h2>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          <div className="bg-white rounded-lg shadow-sm p-8">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center mb-4">
              <span className="text-2xl font-bold text-blue-700">1</span>
            </div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              Verify Your Identity
            </h3>
            <p className="text-gray-600">
              Sign in with your RealMe Verified identity — the same system used
              by NZ government agencies. Your verified name and date of birth
              confirm your eligibility.
            </p>
          </div>
          <div className="bg-white rounded-lg shadow-sm p-8">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center mb-4">
              <span className="text-2xl font-bold text-blue-700">2</span>
            </div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              Register &amp; Vote
            </h3>
            <p className="text-gray-600">
              Register once to join the roll, then cast your ballot in any open
              poll. The system enforces one vote per person per poll — your
              identity is never stored in the ballot record.
            </p>
          </div>
          <div className="bg-white rounded-lg shadow-sm p-8">
            <div className="w-12 h-12 bg-blue-100 rounded-lg flex items-center justify-center mb-4">
              <span className="text-2xl font-bold text-blue-700">3</span>
            </div>
            <h3 className="text-lg font-semibold text-gray-900 mb-2">
              Verify Your Vote
            </h3>
            <p className="text-gray-600">
              After voting you receive a receipt token. Use it at any time to
              confirm your ballot appears in the public audit list — without
              revealing your choice to anyone else.
            </p>
          </div>
        </div>
      </div>

      <div className="py-16 border-t border-gray-200">
        <div className="max-w-3xl mx-auto text-center">
          <h2 className="text-2xl font-bold text-gray-900 mb-4">
            Privacy &amp; Security
          </h2>
          <p className="text-gray-600 mb-6">
            Your RealMe Federated Login Token (FLT) is hashed before storage
            and is never recorded alongside your vote. Each poll uses a unique
            cryptographic salt so voter tokens cannot be linked across polls.
            All ballot commitments are published for independent verification.
          </p>
          <div className="flex justify-center gap-8 text-sm text-gray-500 flex-wrap">
            <span>RealMe Verified Identity</span>
            <span>Privacy Act 2020 Compliant</span>
            <span>Electoral Act 1993</span>
            <span>Publicly Auditable Results</span>
          </div>
        </div>
      </div>
    </div>
  );
}
