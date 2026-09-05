// Vite executes this config in Node, while the application tsconfig contains
// only browser types. Keep the config dependency-free for the existing setup.
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
  },
});
