/** @type {import('next').NextConfig} */
const nextConfig = {
  output: "standalone",

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
