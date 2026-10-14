import {defineConfig} from 'vite';
import react from '@vitejs/plugin-react';
import {resolve} from 'path';
export default defineConfig({plugins:[react()],server:{port:5173,proxy:{'/api':'http://localhost:8080'}},build:{outDir:resolve(import.meta.dirname,'../internal/app/web'),emptyOutDir:true,rollupOptions:{output:{entryFileNames:'assets/app.js',chunkFileNames:'assets/[name].js',assetFileNames:a=>a.name?.endsWith('.css')?'assets/app.css':'assets/[name][extname]'}}}})
