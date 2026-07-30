import type { Configuration, ExternalItemFunctionData } from 'webpack';

type ExternalsType = Configuration['externals'];

export const externals: ExternalsType = [
  { 'amd-module': 'module' },
  'lodash',
  'jquery',
  'moment',
  'slate',
  'emotion',
  '@emotion/react',
  '@emotion/css',
  'prismjs',
  'slate-plain-serializer',
  '@grafana/slate-react',
  'react',
  'react/jsx-runtime',
  'react/jsx-dev-runtime',
  'react-dom',
  'react-redux',
  'redux',
  'rxjs',
  'i18next',
  'react-router',
  'd3',
  'angular',
  /^@grafana\/ui/i,
  /^@grafana\/runtime/i,
  /^@grafana\/data/i,
  ({ request }: ExternalItemFunctionData, callback: (error?: Error, result?: string) => void) => {
    const prefix = 'grafana/';
    if (request?.startsWith(prefix)) {
      return callback(undefined, request.slice(prefix.length));
    }

    callback();
  },
];
