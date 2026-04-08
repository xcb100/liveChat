import path from "node:path";

import vue from "@vitejs/plugin-vue";
import { defineConfig } from "vite";

// createViteConfig 输入为空，输出为 Vite 配置，目的在于把 Vue SPA 构建产物输出到 Go 服务可直接托管的静态目录。
export default defineConfig({
  root: path.resolve(__dirname),
  base: "/static/dist/",
  publicDir: false,
  plugins: [vue()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "src"),
    },
  },
  build: {
    outDir: path.resolve(__dirname, "../static/dist"),
    emptyOutDir: true,
  },
});
