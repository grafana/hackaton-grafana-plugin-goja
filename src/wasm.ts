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
}

async function gojaDefine(deps: string[], runner: () => { plugin: PluginResolver }) {
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
  console.log('Running plugin code');
  //@ts-ignore
  const result = runner.apply(null, resolvedDeps);
  console.log('Creating plugin instance');
  const plugin = new result.plugin.DataSourceClass({});
  const constant = Math.floor(Math.random() * 100);
  const query = `Test query with constant ${constant}`;
  console.log(`Querying with constant ${constant} and text: ${query}`);
  const queryResult = await plugin.query({
    targets: [
      {
        constant: constant,
        queryText: query,
        refId: 'A',
      },
    ],
    range: {
      from: new Date(),
      to: new Date(),
    },
  });
  console.log('Query result: ', JSON.stringify(queryResult, null, 2));
}

console.log('Running actual plugin code');
