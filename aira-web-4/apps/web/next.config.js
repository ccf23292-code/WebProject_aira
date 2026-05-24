const path = require('path'); // 1. 必须引入 path 模块

/** @type {import('next').NextConfig} */
const nextConfig = {
  turbopack: {
    // 2. 将 root 指向 monorepo 的根目录 (当前目录向上退两级)
    root: path.resolve(__dirname, '../../'),
  },
  transpilePackages: ['@aira/shared'],
};

module.exports = nextConfig;