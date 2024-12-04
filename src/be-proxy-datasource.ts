import { CoreApp, DataSourceInstanceSettings } from '@grafana/data';

import { MyQuery, MyDataSourceOptions } from './types';
import { DataSourceWithBackend } from '@grafana/runtime';

export class BEDataSourceProxy extends DataSourceWithBackend<MyQuery, MyDataSourceOptions> {
  constructor(instanceSettings: DataSourceInstanceSettings<MyDataSourceOptions>) {
    super(instanceSettings);
    console.log('Init plugin with backend');
  }

  getDefaultQuery(_: CoreApp): Partial<MyQuery> {
    return { queryText: 'test', constant: 6.5 };
  }
}
