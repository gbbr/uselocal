# uselocal

The `uselocal` tool helps you switch between development and production environments for module (and sub-module) repositories by programmatically applying and undoing replace directives.

## Usage

The configuration file is located by default in the current folder where the command is run under the name `.uselocal.yaml`, but can be placed anywhere and set via the `USELOCAL` variable.

An example configuration file would be:
```yaml
targets:
  - ./cmd/otelcontribcol
  - ./exporter/datadogexporter
  - ./processor/datadogprocessor
  - .
replace:
  - from: github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/metrics
    to: ../../DataDog/opentelemetry-mapping-go/pkg/otlp/metrics
  - from: github.com/DataDog/opentelemetry-mapping-go/pkg/otlp/attributes
    to: ../../DataDog/opentelemetry-mapping-go/pkg/otlp/attributes
  - from: github.com/DataDog/datadog-agent/pkg/trace
    to: ../../DataDog/datadog-agent/pkg/trace
```

This file specifies that for the `go.mod` files in the subfolders specified in `targets`, the list of replace directives is applied.

To apply the changes, run `uselocal`. To undo the changes, run `uselocal --drop`.
