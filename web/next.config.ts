import type { NextConfig } from "next";

const allowedDevOrigins = (process.env.ALLOWED_DEV_ORIGINS || "127.0.0.1,localhost,38.246.253.179")
  .split(",")
  .map((origin) => origin.trim())
  .filter(Boolean);

const nextConfig: NextConfig = {
  allowedDevOrigins,
  async headers() {
    const noindex = { key: 'X-Robots-Tag', value: 'noindex, nofollow' }
    const preview = process.env.SEO_NOINDEX === 'true' || Boolean(process.env.VERCEL_ENV && process.env.VERCEL_ENV !== 'production')
    return [
      ...(preview ? [{ source: '/:path*', headers: [noindex] }] : []),
      { source: '/api/:path*', headers: [noindex] },
      { source: '/health', headers: [noindex] },
    ]
  },
};

export default nextConfig;
