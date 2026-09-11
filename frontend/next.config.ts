import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Enable standalone output — packages only the files needed to run the app,
  // which drastically reduces the Docker image size (no node_modules bloat).
  // See: https://nextjs.org/docs/pages/api-reference/next-config-js/output
  output: "standalone",
};

export default nextConfig;
