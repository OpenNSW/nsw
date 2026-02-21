import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import dts from 'vite-plugin-dts'
import * as path from "node:path";
import tailwindcss from "@tailwindcss/vite";

// https://vite.dev/config/
export default defineConfig({
    plugins: [
        react(),
        tailwindcss(),
        dts({
            include: ['src'],
            tsconfigPath: './tsconfig.app.json',
            rollupTypes: true,
        }),
    ],
    build: {
        lib: {
            entry: path.resolve(__dirname, 'src/index.ts'),
            name: 'ui',
            formats: ['es', 'cjs'],
            fileName: (format) => `ui.${format}.js`
        },
        rollupOptions: {
            external: [
                'react',
                'react-dom',
                'react/jsx-runtime',
                '@radix-ui/themes',
                '@radix-ui/react-icons',
            ]
        }
    }
})
