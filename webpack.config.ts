//@ts-nocheck
import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import grafanaConfig from './.config/webpack/webpack.config';
import { execSync } from 'child_process';
import fs from 'fs';

class InjectWasmPlugin {
  apply(compiler) {
    compiler.hooks.afterEmit.tapAsync('InjectWasmPlugin', (compilation, callback) => {
      const wasmOutput = execSync('tsc ./src/wasm.ts --outDir dist');
      const wasmJs = fs.readFileSync('dist/wasm.js', 'utf8');
      const bundlePath = compilation.outputOptions.path + '/module.js';
      const bundleJs = fs.readFileSync(bundlePath, 'utf8');
      fs.writeFileSync(bundlePath, wasmJs + bundleJs);
      callback();
    });
  }
}

const config = async (env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  return merge(baseConfig, {
    //@ts-ignore
    plugins: [...baseConfig.plugins, new InjectWasmPlugin()],
  });
};

export default config;
