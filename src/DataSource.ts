import {
  CircularDataFrame,
  DataFrame,
  DataQueryRequest,
  DataQueryResponse,
  DataSourceInstanceSettings,
  Field,
  LoadingState,
  MetricFindValue,
  ScopedVars,
} from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { VerticaDataSourceOptions, VerticaQuery, defaultQuery } from './types';
import { merge, Observable, Subscriber } from 'rxjs';
import { defaults } from 'lodash';
import { switchMap as switchMap$ } from 'rxjs/operators';

export class DataSource extends DataSourceWithBackend<VerticaQuery, VerticaDataSourceOptions> {
  templateSrv;

  constructor(instanceSettings: DataSourceInstanceSettings<VerticaDataSourceOptions>) {
    super(instanceSettings);
    this.templateSrv = getTemplateSrv();
    this.annotations = {};
  }

  applyTemplateVariables(query: VerticaQuery, scopedVars: ScopedVars): VerticaQuery {
    query.queryTemplated = this.templateSrv.replace(query.queryString, scopedVars);
    return query;
  }

  async metricFindQuery(query: string, options?: any) {
    const findVal: MetricFindValue[] = [];
    if (!query) {
      return Promise.resolve(findVal);
    }
    const response = await super
      .query({
        targets: [
          {
            format: 'Table',
            queryString: query,
            queryTemplated: query,
            streaming: false,
            timeFillEnabled: false,
            refId: 'Var',
          },
        ],
      } as any)
      .toPromise();

    if (response === undefined) {
      return findVal;
    }

    if (response.error) {
      throw new Error(response.error.message);
    }
    console.log(response);
    const data = response.data[0] as DataFrame;
    const textField = (data.length > 0 ? data.fields.find((f) => f.name === '_text') : undefined) as Field;
    const valueField = (data.length > 0 ? data.fields.find((f) => f.name === '_value') : undefined) as Field;

    switch (true) {
      case data.fields.length > 3:
        throw new Error(
          `Received more than two (${data.fields.length}) fields: ${data.fields.map((x) => x.name).join(',')}`
        );
        break;
      case textField === undefined && data.length > 0:
        throw new Error(
          `Variable query must have at least one column with name '_text' fields: ${data.fields
            .map((x) => x.name)
            .join(',')}`
        );
        break;
    }

    if (textField !== undefined) {
      for (let i = 0; i < data.fields[0].values.length; i++) {
        const metricValue: MetricFindValue = {
          text: '',
          value: undefined,
        } as MetricFindValue;
        metricValue.text = textField?.values.toArray()[i];
        if (valueField === undefined) {
          metricValue.value = textField?.values.toArray()[i];
        } else {
          metricValue.value = valueField?.values.toArray()[i];
        }

        findVal.push(metricValue);
      }
    }

    return findVal;
  }

  query(options: DataQueryRequest<VerticaQuery>): Observable<DataQueryResponse> {
    /*
     *Grafana panels can have multiple queries
     *here if a panel has multiple queries we and not all of them have refresh , we throw an error forcing all the queries to have refresh enabled
     */
    const queriesWithStream = options.targets.filter((target) => target.streaming === true).length;

    /*
     *Below switch statement run 3 conditions , if all queries are streaming then handel streaming
     *if none of the queries are streaming that use the default query option
     *if partial queries are streaming then throw and error
     */
    switch (true) {
      case queriesWithStream > 0 && queriesWithStream !== options.targets.length: {
        return new Observable<DataQueryResponse>((subscriber) => {
          subscriber.error(`All queries in a panel need to be stream enabled, if using streaming. Total Queries: ${
            options.targets.length
          } Queries with streaming: ${options.targets.filter((target) => target.streaming === true).length}
          `);
        });
      }
      case queriesWithStream > 0 && queriesWithStream === options.targets.length: {
        /*
         *We loop over all the queries in a panel and create stream for them
         */
        const streams = options.targets.map((target) => {
          /*
           *If no queries are defined then use the default query
           */
          const query = defaults(target, defaultQuery);
          return new Observable<DataQueryResponse>((subscriber) => {
            /*
             *Creating a circular frame
             *capacity is set to maxDataPoints from the panel option
             */
            const frame = new CircularDataFrame({
              append: 'tail',
              capacity: options.maxDataPoints ? options.maxDataPoints : 1000,
            });
            frame.refId = query.refId;

            /*
             *Stream data takes the circular frame, add fields from the backend query, if updateFields is true , then new fields are added to the frame.
             *reason doing this , in sql we do not know how many time frames might be returned.
             *so on the first run of the data we update the fields of the frame
             */
            this.streamData(frame, target, true, subscriber);

            /*
             *Based on the interval input by this below function will fire the query to the backend and append the data to the frame.
             */
            const intervalId = setInterval(async () => {
              await this.streamData(frame, target, false, subscriber);
            }, query.streamingInterval * 1000);
            return () => {
              clearInterval(intervalId);
            };
          });
        });
        return merge(...streams);
      }
      default: {
        return super.query(options);
      }
    }
  }

  async streamData(
    frame: CircularDataFrame,
    target: VerticaQuery,
    firstRun: boolean,
    subscriber: Subscriber<DataQueryResponse>
  ) {
    /*
     *Values is the variable that holds the data
     */
    const values: any[] = [],
      data: DataFrame | undefined = await this.query({
        targets: [
          {
            queryString: target.queryString,
            queryTemplated: target.queryTemplated,
            format: target.format,
            refId: target.refId,
          },
        ],
      } as DataQueryRequest<VerticaQuery>)
        .pipe(
          switchMap$((response) => {
            if (response.error) {
              subscriber.error({
                err: response.error,
                message: response.error.message,
              });
            }
            return response.data;
          })
        )
        .toPromise();

    /*
     *For now all streaming queries need to have a time field.
     *Might remove this based on community feedback.
     *If this plugin receives any :P
     */
    if (data && data.fields.find((f: Field) => f.type === 'time')) {
      data.fields.forEach((f) => {
        if (firstRun) {
          frame.addField({
            name: f.name,
            type: f.type,
            config: f.config,
            display: f.display,
            labels: f.labels,
            state: f.state,
          });
        } else {
          const fieldValues = f.values.toArray();

          /*
           *Push field values into an array to mimic rows
           */
          values.push(fieldValues[fieldValues.length - 1]);
        }
      });
      if (!firstRun) {
        /*
         *Append rows to the frame
         */
        frame.appendRow(values);
      }

      /*
       *For the first run add all values to the frame
       */
      if (firstRun) {
        const timeField = data.fields.find((f: Field) => f.type === 'time');
        if (timeField) {
          const timeEntries = timeField.values.toArray();
          for (let i = 0; i < timeEntries.length; i++) {
            const val = data.fields.map((f) => f.values.toArray()[i]);
            frame.appendRow(val);
          }
        }
      }
    } else {
      /*
       *Throw any error if there is no time field
       */
      subscriber.error({
        err: 'Query should return at least one time frame',
        message: 'Query should return at least one time frame',
      });
    }

    /*
     *Call the next on subscribe to feed data to visualization
     */
    subscriber.next({
      data: [frame],
      key: frame.refId,
      state: LoadingState.Streaming,
    });
  }
}
