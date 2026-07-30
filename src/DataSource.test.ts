jest.mock('rxjs', () => {
  const rxjs = jest.requireActual<typeof import('rxjs')>('rxjs');

  return {
    ...rxjs,
    lastValueFrom: undefined,
  };
});

import { CircularDataFrame, DataQueryResponse, FieldType, toDataFrame } from '@grafana/data';
import { Observable, of } from 'rxjs';

import { DataSource } from './DataSource';
import { VerticaQuery } from './types';

describe('DataSource streaming', () => {
  const target: VerticaQuery = {
    refId: 'A',
    format: 'Time Series',
    queryString: 'SELECT 1',
    queryTemplated: 'SELECT 1',
    streaming: true,
    streamingInterval: 1,
    timeFillEnabled: false,
    timeFillMode: 'null',
    timeFillStaticValue: 0,
  };

  it('reports an empty backend response through the subscriber error path without rejecting', async () => {
    const dataSource = Object.create(DataSource.prototype, {});
    jest.spyOn(dataSource, 'query').mockReturnValue(of({ data: [] }));

    const error = jest.fn();
    const streamInvocation = new Promise<void>((resolve, reject) => {
      new Observable<DataQueryResponse>((subscriber) => {
        dataSource
          .streamData(
            new CircularDataFrame({
              append: 'tail',
              capacity: 1,
            }),
            target,
            true,
            subscriber
          )
          .then(resolve, reject);
      }).subscribe({ error });
    });

    await expect(streamInvocation).resolves.toBeUndefined();

    expect(error).toHaveBeenCalledWith({
      err: 'Query should return at least one time frame',
      message: 'Query should return at least one time frame',
    });
  });

  it('uses the last frame emitted by the backend observable', async () => {
    const dataSource = Object.create(DataSource.prototype, {});
    const firstFrame = toDataFrame({
      fields: [
        { name: 'time', type: FieldType.time, values: [1000] },
        { name: 'value', type: FieldType.number, values: [1] },
      ],
    });
    const lastFrame = toDataFrame({
      fields: [
        { name: 'time', type: FieldType.time, values: [2000] },
        { name: 'value', type: FieldType.number, values: [2] },
      ],
    });
    jest.spyOn(dataSource, 'query').mockReturnValue(of({ data: [firstFrame, lastFrame] }));

    const frame = new CircularDataFrame({
      append: 'tail',
      capacity: 1,
    });
    const streamInvocation = new Promise<void>((resolve, reject) => {
      new Observable<DataQueryResponse>((subscriber) => {
        dataSource.streamData(frame, target, true, subscriber).then(resolve, reject);
      }).subscribe();
    });

    await expect(streamInvocation).resolves.toBeUndefined();

    expect(frame.fields.map((field) => Array.from(field.values))).toEqual([[2000], [2]]);
  });
});
