/** @type {import('next').NextConfig} */
import fs from 'node:fs';
import path from 'node:path';
import dotenv from 'dotenv';

const parentEnvPath = path.resolve(process.cwd(), '../.env');
if (fs.existsSync(parentEnvPath)) {
  const parentEnv = dotenv.parse(fs.readFileSync(parentEnvPath));
  for (const [key, value] of Object.entries(parentEnv)) {
    process.env[key] = value;
  }
}

const nextConfig = {
  reactStrictMode: true,
  output: 'export',
  images: {
    unoptimized: true
  },
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || '',
    API_URL: process.env.NEXT_PUBLIC_API_URL || ''
  }
};

export default nextConfig;
