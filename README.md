# rollouts-plugin-metric-coralogix

[![Go Report Card](https://goreportcard.com/badge/github.com/argoproj-labs/rollouts-plugin-metric-coralogix)](https://goreportcard.com/report/github.com/argoproj-labs/rollouts-plugin-metric-coralogix)
[![GitHub license](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/argoproj-labs/rollouts-plugin-metric-coralogix/blob/master/LICENSE)

The `rollouts-plugin-metric-coralogix` is a Coralogix Data Prime plugin designed for use with the Argo Rollouts plugin system. This plugin enables the integration of Coralogix metrics into Argo Rollouts, allowing for advanced metric analysis and monitoring during application rollouts.

> [!IMPORTANT]
> The Coralogix Metric Plugin relies on Data Prime query results to function correctly. Data Prime queries in Coralogix allow for the computation of metrics, such as averages, sums, and counts, over a set of logs. This plugin specifically requires the presence of an aggregation block in the query results to operate. If the query results do not contain an aggregation block, the plugin will be unable to process the data and will not function as intended. Therefore, it is essential to ensure that all queries used with this plugin include the necessary aggregation blocks to enable accurate metric analysis and monitoring.

## Features

- **Metric Integration:** Seamlessly integrates Coralogix metrics with Argo Rollouts.

- **Custom Queries:** Supports custom Coralogix Data Prime queries for flexible metric retrieval.

- **Error Handling:** Robust error handling to ensure reliable metric collection.

- **Debugging Support:** Provides options for building debug versions and attaching debuggers.

## Build & Debug

To build the plugin, use the following commands:

### Release Build

```bash
make build-rollouts-plugin-metric-coralogix
```

### Debug Build

```bash
make build-rollouts-plugin-metric-coralogix-debug
```

### Attaching a debugger to debug build

If using goland you can attach a debugger to the debug build by following the directions https://www.jetbrains.com/help/go/attach-to-running-go-processes-with-debugger.html

You can also do this with many other debuggers as well. Including cli debuggers like delve.

## Using a Metric Plugin

There are two methods of installing and using an argo rollouts plugin. The first method is to mount up the plugin executable
into the rollouts controller container. The second method is to use a HTTP(S) server to host the plugin executable.

### Mounting the plugin executable into the rollouts controller container

There are a few ways to mount the plugin executable into the rollouts controller container. Some of these will depend on your
particular infrastructure. Here are a few methods:

- Using an init container to download the plugin executable
- Using a Kubernetes volume mount with a shared volume such as NFS, EBS, etc.
- Building the plugin into the rollouts controller container

Then you can use setup the configmap to point to the plugin executable. Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-rollouts-config
data:
  plugins: |-
    metrics:
    - name: "argoproj-labs/coralogix-metric-plugin" # name of the plugin uses the name to find this configuration, it must match the name required by the plugin
      location: "file://./my-custom-plugin" # supports http(s):// urls and file://
```

### Using a HTTP(S) server to host the plugin executable

Argo Rollouts supports downloading the plugin executable from a HTTP(S) server. To use this method, you will need to
configure the controller via the `argo-rollouts-config` configmaps `pluginLocation` to an http(s) url. Example:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: argo-rollouts-config
data:
  plugins: |-
    metrics:
    - name: "argoproj-labs/coralogix-metric-plugin" # name of the plugin uses the name to find this configuration, it must match the name required by the plugin
      location: "https://github.com/argoproj-labs/rollouts-plugin-metric-coralogix/releases/download/v0.0.1/rollouts-plugin-metric-coralogix-linux-amd64" # supports http(s):// urls and file://
      sha256: "08f588b1c799a37bbe8d0fc74cc1b1492dd70b2c" #optional sha256 checksum of the plugin executable
```

### Sample Analysis Template

The `successCondition` checks that the success ratio of HTTP requests (excluding 500, 502, and 503 status codes) is above a certain threshold.

> [!TIP]
> Note: The fields `successCondition`, `container.image.tag`, and HTTP status codes can be configured based on your needs. For example, you might want to monitor different versions or different HTTP status codes.

An example for this sample plugin below.

```yaml
apiVersion: argoproj.io/v1alpha1
kind: AnalysisTemplate
metadata:
  name: success-rate
spec:
  args:
    - name: service-name
  metrics:
    - name: success-rate
      interval: 10s
      successCondition: result >= 0.95  # 95% success rate threshold
      failureLimit: 2
      count: 3
      provider:
        plugin:
          argoproj-labs/coralogix-metric-plugin:
            address: https://api.coralogix.com
            apiKey: your-api-key
            applicationName: your-app-name
            subsystemName: your-subsystem
            queryTier: TIER_FREQUENT_SEARCH  # Optional: TIER_FREQUENT_SEARCH (default) or TIER_ARCHIVE
            query: |
              source logs | 
              filter $d.kubernetes['container.image.tag'] == 'v3.0.44-rc28' | 
              groupby true 
              aggregate count_if($d.log_processed.http_status != null && $d.log_processed.http_status:number != 500 && $d.log_processed.http_status:number != 502 && $d.log_processed.http_status:number != 503) as $d.count_success, 
              count_if($d.log_processed.http_status != null) as $d.count_all | 
              create $d.ratio from $d.count_success / $d.count_all
```

### Configuration Parameters

| Parameter | Description | Required | Default |
|-----------|-------------|----------|---------|
| `address` | The Coralogix API endpoint URL | Yes | - |
| `apiKey` | Your Coralogix API key | Yes | - |
| `query` | The Data Prime query to execute | Yes | - |
| `queryTier` | The query execution tier: `TIER_FREQUENT_SEARCH` or `TIER_ARCHIVE` | No | `TIER_FREQUENT_SEARCH` |

The `queryTier` parameter allows you to specify which tier to use for query execution:
- `TIER_FREQUENT_SEARCH`: For querying recent data (default)
- `TIER_ARCHIVE`: For querying archived/historical data

### Sample Analysis Result

Coralogix Data Prime query should response like below:

```json
{
  "_expr0": true,
  "count_all": 411671,
  "count_success": 411671,
  "ratio": 1
}
```

## Credit

The development of this plugin was inspired by the [Argo Rollouts Prometheus Metric Plugin](https://github.com/argoproj-labs/rollouts-plugin-metric-sample-prometheus). Leveraging the knowledge and design principles from the Prometheus plugin, this Coralogix Metric Plugin was created to provide similar functionality for Coralogix metrics. The foundational concepts and architecture were adapted to suit the specific requirements and capabilities of Coralogix Data Prime, ensuring seamless integration and reliable performance within the Argo Rollouts ecosystem.

## Building Custom Argo Rollouts Image

You can build a custom Argo Rollouts image that includes this plugin by creating a Dockerfile that extends the official Argo Rollouts image. Here's an example:

To build the image:

```bash
# Build for AMD64 (even on ARM machines)
docker buildx build --platform linux/amd64 -t your-registry/argo-rollouts-coralogix:latest .
```

After building, you can use this custom image in your Kubernetes deployment by updating the Argo Rollouts controller deployment

