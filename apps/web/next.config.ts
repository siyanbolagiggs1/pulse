import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  output: "standalone",

  // Proxy /api/* to the Go server — used when NEXT_PUBLIC_API_URL=/api
  // (Codespaces, local dev without Docker, or behind a reverse proxy).
  // When NEXT_PUBLIC_API_URL is an absolute URL the browser calls Go directly
  // and this rewrite is never hit.
  async rewrites() {
    const dest = process.env.INTERNAL_API_URL ?? "http://localhost:5000";
    return [
      {
        source: "/api/:path*",
        destination: `${dest}/api/:path*`,
      },
    ];
  },

  images: {
    remotePatterns: [
      { protocol: "https", hostname: "**.instagram.com" },
      { protocol: "https", hostname: "pbs.twimg.com" },
      { protocol: "https", hostname: "abs.twimg.com" },
    ],
  },
};

export default nextConfig;
