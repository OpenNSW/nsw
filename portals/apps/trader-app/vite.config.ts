import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react-swc'
import * as path from "node:path";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
  plugins: [react(), tailwindcss()],
  resolve: {
    alias: {
      '@opennsw/ui': path.resolve(__dirname, '../../packages/ui/src'),
      '@opennsw/jsonforms-renderers': path.resolve(__dirname, '../../packages/jsonforms-renderers/src')
    }
  }
})
