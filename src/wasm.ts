console.log('Running module.js. Setting up mocks');
// @grafana/data
const data = {
  DataSourceApi: class {
    getDefaultQuery() {
      return {};
    }
    filterQuery() {
      return true;
    }
    query() {
      return Promise.resolve({ data: [] });
    }
    testDatasource() {
      return Promise.resolve({ status: 'success' });
    }
  },
  createDataFrame: (options: any) => ({
    refId: options.refId,
    fields: options.fields,
    length: options.fields[0].values.length,
  }),
  FieldType: {
    time: 'time',
    number: 'number',
    string: 'string',
    boolean: 'boolean',
  },
  DataSourcePlugin: class {
    DataSourceClass: any;
    constructor(datasourceClass: any) {
      this.DataSourceClass = datasourceClass;
    }
    setConfigEditor(component: any) {
      return this;
    }
    setQueryEditor(component: any) {
      return this;
    }
  },
};

// @grafana/runtime
const runtime = {
  DataSourceWithBackend: class { },
  getBackendSrv: () => ({
    fetch: (options: any) => ({
      pipe: () => ({ subscribe: () => { } }),
      toPromise: () => Promise.resolve({ status: 200 }),
    }),
  }),
  isFetchError: (error: any) => error instanceof Error,
};

// @grafana/ui
const ui = {
  Input: (props: any) => null,
  SecretInput: (props: any) => null,
  InlineField: (props: any) => null,
  Stack: (props: any) => null,
};

// rxjs
const rxjs = {
  lastValueFrom: (input: any) => Promise.resolve(input),
};

const allProxy = new Proxy(() => null, {
  get: () => allProxy,
  set: () => true,
});

const dependencies = {
  '@grafana/data': data,
  '@grafana/runtime': runtime,
  '@grafana/ui': ui,
  react: allProxy,
};

interface PluginResolver {
  DataSourceClass: any;
}

if (typeof window !== 'undefined' && (window as any).define) {
  // eslint-disable-next-line no-console
  console.log('Inside browser. Skipping node environment');
} else {
  console.log('not in browser environment');

  console.log('Setup web server');

  //setup web server in port 8080

  const http = require('http');

  const server = http.createServer(function (request, response) {
    if (request.url === '/query') {
      let body = '';

      request.on('data', (chunk) => {
        body += chunk.toString();
      });

      request.on('end', async () => {
        try {
          const jsonBody = JSON.parse(body);
          response.writeHead(200, { 'Content-Type': 'application/json' });
          const queryResult = await runQuery(jsonBody);
          response.end(JSON.stringify(queryResult));
        } catch (error) {
          response.writeHead(400, { 'Content-Type': 'application/json' });
          response.end(JSON.stringify({ status: 'error', message: 'Invalid JSON' + body }));
        }
      });
      return;
    }
  });

  server.listen(8080);
  console.log('Server started on port 8080');

  let pluginInstance: any | undefined;

  async function define(deps: string[], runner: () => { plugin: PluginResolver }) {
    console.log('inside define function execution??');
    const resolvedDeps = deps.map((dep) => {
      // if (dep === 'rxjs') {
      //   return require('rxjs');
      // }
      if (dep in dependencies) {
        return dependencies[dep as keyof typeof dependencies] as any;
      }
      return allProxy;
    });
    // console.log('Running plugin code');
    //@ts-ignore
    const result = runner.apply(null, resolvedDeps);
    console.log('Creating plugin instance');
    const plugin = new result.plugin.DataSourceClass({});
    pluginInstance = plugin;
    console.log('Plugin instance created');
    return 'OK from define';
  }

  async function runQuery(query: any) {
    if (!pluginInstance) {
      console.log('No plugin instance');
      throw new Error('No plugin instance');
    }
    const queryResult = await pluginInstance.query(query);
    console.log('executed query and got result: ', JSON.stringify(queryResult, null, 2));
    return queryResult;
  }

  console.log('Running actual plugin code');
}
