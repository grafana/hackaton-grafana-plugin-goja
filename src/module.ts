import { DataSourcePlugin } from '@grafana/data';
import { DataSource } from './datasource';
import { ConfigEditor } from './components/ConfigEditor';
import { QueryEditor } from './components/QueryEditor';
import { MyQuery, MyDataSourceOptions } from './types';
import { BEDataSourceProxy } from 'be-proxy-datasource';

let definedPlugin: any;

console.log('Inside module');

if (typeof window !== 'undefined') {
  console.log('In browser environment');
  definedPlugin = new DataSourcePlugin<BEDataSourceProxy, MyQuery, MyDataSourceOptions>(BEDataSourceProxy);
} else {
  console.log('Not in browser environment');
  definedPlugin = new DataSourcePlugin<DataSource, MyQuery, MyDataSourceOptions>(DataSource);
}

definedPlugin.setConfigEditor(ConfigEditor).setQueryEditor(QueryEditor);

export const plugin = definedPlugin;
