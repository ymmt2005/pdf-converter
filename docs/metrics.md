Metrics
=======

`pdf-converter` exports various metrics for Prometheus.

Since it uses [the default registry](https://pkg.go.dev/github.com/prometheus/client_golang/prometheus#pkg-variables), the metrics include process-related and Go-related metrics.

Other metrics are:

## `pdf_converter_requests_total`

The total number of HTTP requests for the converter.

Labels:

- `status`: HTTP status code

## `pdf_converter_conversion_total`

The total number of file conversions.

Labels:

- `extension`: filename extensions such as `pptx`.

## `pdf_converter_conversion_failed`

The total number of failed file conversions.

Labels:

- `extension`: filename extensions such as `pptx`.

## `pdf_converter_conversion_duration_seconds`

[Histogram][] of conversion latencies in seconds.

Labels:

- `extension`: filename extensions such as `pptx`.

## `pdf_converter_conversion_source_bytes`

[Histogram][] of source data length in bytes.

Labels:

- `extension`: filename extensions such as `pptx`.

## `pdf_converter_conversion_output_bytes`

[Histogram][] of generated data length in bytes.

Labels:

- `extension`: filename extensions such as `pptx`.


[Histogram]: https://www.robustperception.io/how-does-a-prometheus-histogram-work
