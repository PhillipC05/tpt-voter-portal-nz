/** @type {import('next').NextConfig} */
const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

const nextConfig = {
  async rewrites() {
    return [
      { source: "/polls/:path*", destination: `${API_URL}/polls/:path*` },
      { source: "/register", destination: `${API_URL}/register` },
      { source: "/register/:path*", destination: `${API_URL}/register/:path*` },
      { source: "/auth/:path*", destination: `${API_URL}/auth/:path*` },
      { source: "/health", destination: `${API_URL}/health` },
    ];
  },
};

module.exports = nextConfig;
