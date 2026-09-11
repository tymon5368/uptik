import {defineConfig} from 'vite'
import {svelte} from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import {paraglide} from '@inlang/paraglide-vite'
import path from 'node:path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [
    tailwindcss(),
    paraglide({
      project: './project.inlang',
      outdir: './src/lib/paraglide'
    }),
    svelte()
  ],
  resolve: {
    alias: {
      '$lib': path.resolve(import.meta.dirname, './src/lib')
    }
  }
})

