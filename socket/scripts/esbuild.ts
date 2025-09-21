import { buildSync } from 'esbuild';

buildSync({
  bundle: true,
  entryPoints: ['src/index.ts'],
  outfile: 'main.mjs',
  inject: ['scripts/cjs-shim.ts'],
  format: 'esm',
  minify: true,
  platform: 'node',
});
