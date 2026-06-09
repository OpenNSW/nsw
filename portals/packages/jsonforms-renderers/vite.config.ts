import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import dts from 'vite-plugin-dts'
import * as path from 'node:path'
import tailwindcss from '@tailwindcss/vite'

// https://vite.dev/config/
// `vite build` builds the library; `vite` (serve) runs the dev playground in dev/.
export default defineConfig(({ command }) => ({
  // In serve mode, treat dev/ as the app root so dev/index.html is the entry.
  ...(command === 'serve' ? { root: 'dev' } : {}),
  plugins: [
    react(),
    tailwindcss(),
    // Type declarations are only needed for the library build.
    ...(command === 'build'
      ? [
          dts({
            include: ['src'],
            tsconfigPath: './tsconfig.app.json',
            rollupTypes: true,
          }),
        ]
      : []),
  ],
  build: {
    lib: {
      entry: path.resolve(__dirname, 'src/index.ts'),
      name: 'renderers',
      formats: ['es', 'cjs'],
      fileName: (format) => `renderers.${format}.js`,
    },
    rollupOptions: {
      external: [
        'react',
        'react-dom',
        'react/jsx-runtime',
        '@jsonforms/core',
        '@jsonforms/react',
        '@opennsw/ui',
        '@radix-ui/themes',
        '@radix-ui/react-icons',
      ],
    },
  },
}))
