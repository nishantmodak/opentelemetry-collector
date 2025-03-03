# Debug Exporter

| Status                   |                       |
| ------------------------ |-----------------------|
| Stability                | [Development]         |
| Supported pipeline types | traces, metrics, logs |
| Distributions            | [core], [contrib]     |

Exports data to the console via zap.Logger.

Supported pipeline types: traces, metrics, logs.

See also the [Troubleshooting][troubleshooting_docs] document for examples on using this exporter.

[troubleshooting_docs]: ../../docs/troubleshooting.md

## Getting Started

The following settings are optional:

- `verbosity` (default = `normal`): the verbosity of the logging export
  (detailed|normal|basic). When set to `detailed`, pipeline data is verbosely
  logged.
- `sampling_initial` (default = `2`): number of messages initially logged each
  second.
- `sampling_thereafter` (default = `500`): sampling rate after the initial
  messages are logged (every Mth message is logged). Refer to [Zap
  docs](https://godoc.org/go.uber.org/zap/zapcore#NewSampler) for more details.
  on how sampling parameters impact number of messages.

Example configuration:

```yaml
exporters:
  debug:
    verbosity: detailed
    sampling_initial: 5
    sampling_thereafter: 200
```

## Verbosity levels

The following subsections describe the output from the exporter depending on the configured verbosity level - `basic`, `normal` and `detailed`.
The default verbosity level is `normal`.

### Basic verbosity

With `verbosity: basic`, the exporter's behavior is currently the same as with `verbosity: normal`.
See below for more details.

### Normal verbosity

With `verbosity: normal`, the exporter outputs a single-line summary of received data with a total count of telemetry records for every batch of received logs, metrics or traces.

Here's an example output:

```console
2023-11-10T22:49:03.510-0600    info    TracesExporter  {"kind": "exporter", "data_type": "traces", "name": "debug", "resource spans": 1, "spans": 2}
```

### Detailed verbosity

With `verbosity: detailed`, the exporter outputs all details of every telemetry record, typically writing multiple lines for every telemetry record.

Here's an example output:

```console
2023-11-10T22:49:03.510-0600    info    TracesExporter  {"kind": "exporter", "data_type": "traces", "name": "debug", "resource spans": 1, "spans": 2}
2023-11-10T22:49:03.510-0600    info    ResourceSpans #0
Resource SchemaURL: https://opentelemetry.io/schemas/1.4.0
Resource attributes:
     -> service.name: Str(telemetrygen)
ScopeSpans #0
ScopeSpans SchemaURL:
InstrumentationScope telemetrygen
Span #0
    Trace ID       : 3bde5d3ee82303571bba6e1136781fe4
    Parent ID      : 5e9dcf9bac4acc1f
    ID             : 2cf3ef2899aba35c
    Name           : okey-dokey
    Kind           : Server
    Start time     : 2023-11-11 04:49:03.509369393 +0000 UTC
    End time       : 2023-11-11 04:49:03.50949377 +0000 UTC
    Status code    : Unset
    Status message :
Attributes:
     -> net.peer.ip: Str(1.2.3.4)
     -> peer.service: Str(telemetrygen-client)
Span #1
    Trace ID       : 3bde5d3ee82303571bba6e1136781fe4
    Parent ID      :
    ID             : 5e9dcf9bac4acc1f
    Name           : lets-go
    Kind           : Client
    Start time     : 2023-11-11 04:49:03.50935117 +0000 UTC
    End time       : 2023-11-11 04:49:03.50949377 +0000 UTC
    Status code    : Unset
    Status message :
Attributes:
     -> net.peer.ip: Str(1.2.3.4)
     -> peer.service: Str(telemetrygen-server)
        {"kind": "exporter", "data_type": "traces", "name": "debug"}
```

[contrib]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol-contrib
[core]: https://github.com/open-telemetry/opentelemetry-collector-releases/tree/main/distributions/otelcol
[Development]: https://github.com/open-telemetry/opentelemetry-collector#in-development
