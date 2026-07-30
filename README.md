![Release](https://github.com/rajsameer/vertica-datasource/workflows/Release/badge.svg)
# Vertica Grafana Data Source
Grafana plugin for Vertica DB.   
New data source that communicates with Vertica using the Vertica golang driver. [http://github.com/vertica/vertica-sql-go].    
This plugin is a backend data source plugin.

## Using the plugin

### Maintained fork installation (recommended)

The maintained `v2.0.10` archive is intentionally unsigned and is for internal
deployment. Download `rajsameer-vertica-datasource-2.0.10.zip` and its `.sha256`
file from this fork's `v2.0.10` GitHub release. Verify the checksum from the
download directory, then extract the ZIP into Grafana's plugin directory. In
Grafana's configuration, allow this exact plugin ID:

```ini
[plugins]
allow_loading_unsigned_plugins = rajsameer-vertica-datasource
```

Restart Grafana after installing or replacing the plugin. Unsigned loading is a
Grafana administrator decision; this repository does not claim a live Grafana
or Vertica smoke test has been run.

### Upstream Grafana catalog installation

The Grafana CLI command installs the upstream/catalog distribution. It does not
install the maintained unsigned artifact built by this fork:

```bash
grafana-cli plugins install rajsameer-vertica-datasource
```

Use the catalog command only when the upstream distribution is intended.

### Creating data source connection
1. Add Data source.
![](src/img/vertica-ds-conf.png)
- **Name**: Data source name
- **Host**: Ip and port of vertica data base , example: *vertica-ip:vertica-port*
- **Database**: Database name
- **User**: User name of vertica database.   
  **Note**: Use a user name with less privileges. This data source does not prevent user from executing DELETE or DROP commands.
- **Password**: password for vertica Database.
- **SSL Mode**: This states how the plugin will connect to the database.Options supported are as below:   
        1. "none"
        2. "server"
        3. "server-string". 
- **Use Prepared Statement**: If unchecked, query arguments will be interpolated into the query on the client side. If checked, query arguments will be bound on the server.
- **Use Connection Load balancing**: If checked the query will be distributed to vertica nodes.
- **Set Max Open Connection**, **Ideal Connections** and **Max connection ideal time**
2. Save and test the data source.
To test the connectivity "select version()" query is executed against the database.

### Querying data.
Queried data is returned to Grafana in data-frame format.   
To lean more about data-frames please refer. https://grafana.com/docs/grafana/latest/developers/plugins/data-frames/#data-frames

- **Time series queries**   
Query type's supported are *Time Series* and *Table*. Query Type can be changed using the drop down in the query editor.   
Example: Time Series Query    
~~~~sql
SELECT 
  time_slice(end_time, $__interval_ms, 'ms', 'end') as time , 
  node_name,
  avg(average_cpu_usage_percent)
FROM 
  v_monitor.cpu_usage 
WHERE 
  end_time > TO_TIMESTAMP($__from/1000) and end_time < TO_TIMESTAMP($__to/1000)
GROUP BY 1, 2
ORDER BY 1 asc
~~~~
![](src/img/vertica-query-time-series.png)
No macros are used in the data source. Instead grafana global variables are used.   
Example: "time_slice(end_time, $__interval_ms, 'ms', 'end') as time" following statement helps the quey to honor the interval of visualization. $__interval_ms is a grafana global variables.   
Time filter:   
Example: "end_time > TO_TIMESTAMP($__from/1000) and end_time < TO_TIMESTAMP($__to/1000)"  this convert the the global $__from and $__to variables from grafana, to a timestamp format for vertica.   


Table Query
~~~~sql 
SELECT 
  time_slice(end_time, $__interval_ms, 'ms', 'end') as time , 
  node_name,
  avg(average_cpu_usage_percent)
FROM 
  v_monitor.cpu_usage 
WHERE 
  end_time > TO_TIMESTAMP($__from/1000) and end_time < TO_TIMESTAMP($__to/1000)
GROUP BY 1, 2
ORDER BY 1 asc
~~~~
Choose Query Type as *Table*    
![](src/img/vertica-query-table.png)

### Variables:
Variables can be easily defined as sql queries, the only restriction is query should return at least one column with "**_text**" name.
If the query has two columns "**_text**" and "**_value**", **_text** would be used as the display value and **_value** would be applied to filter.

Example:

Query:
~~~~sql 
select distinct node_name as '_text' from v_monitor.cpu_usage 
~~~~
![](src/img/vertica-var-example.png)

Usage:
Grafana [Advanced formatting options]https://grafana.com/docs/grafana/latest/variables/advanced-variable-format-options/.
In this example we create a multi select variable of node name , use ${node:sqlstring} for template in the query. 
![](src/img/vertica-var-usage.png)

## Annotations

Annotations are supported from grafana 7.2+   
![](src/img/vertica-annotaions-usage.png)

To use annotations, write any query which will return time, timeEnd, text, title and tags column as shown in the image.   

## Streaming (new) (beta)
Added support for streaming
![](src/img/vertica-streaming.gif)

Example Query
```SQL
  SELECT 
    end_time as time , 
    avg(average_cpu_usage_percent)
  FROM 
   v_monitor.cpu_usage 
  WHERE 
   end_time > TIMESTAMPADD(MINUTE, -1 , CURRENT_TIMESTAMP)
  GROUP BY 1
  ORDER BY 1 asc
```
This will get the latest data from the data base and keep appending the samples.

## Time gap filling (new) (beta)
SQL data can return data which do not have sample for the entire time range , e.g. you could have gaps in the data.    
This feature provide three modes which will add the missing time rows with either null , static, previous values.   
Two modes are **static** , **null** and **previous**.


## SQL syntax highlighting (new) (beta)
SQL syntax highlighting added using CodeMirror library. In future would add auto complete and formatting.

## Development

Prerequisite
 1. Node.js 24
 2. Go 1.26.5
 3. Yarn 1.22.22

Install
```BASH
yarn install --frozen-lockfile
```
Build
 1. **Frontend**
    ```BASH
    yarn build
    ```
 2. **Backend**
    ```BASH
    go test ./...
    go run github.com/magefile/mage@v1.15.0 -v buildAll
    ```

## Legacy local Compose setup

The checked-in `docker-compose.yml` is a legacy development aid. It uses
mutable or legacy images and does not configure Grafana to load this unsigned
plugin, so it is not suitable for release-candidate smoke testing.

Test a release candidate only in a designated disposable Grafana and Vertica
environment that allows the unsigned `rajsameer-vertica-datasource` plugin.
Install the exact candidate ZIP there, then verify plugin loading, saved
datasource configuration, health, representative table and time-series
queries, and returned field and timestamp behavior.
