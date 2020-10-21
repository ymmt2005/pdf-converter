Convert API
===========

This document describes the HTTP API of `pdf-converter`.

## `GET /metrics`

This returns Prometheus metrics.

## `POST /convert`

This takes [`multipart/form-data`](https://developer.mozilla.org/en-US/docs/Web/HTTP/Methods/POST) input with the following parameters.

- `file`

    This is the parameter for the file to be converted.
    The filename must be given with `Content-Disposition` header.

Expected HTTP responses are:

- 100 Continue

    This may be returned if the client sent `Expect: 100-continue` in the
    request header and the other headers, especially `Content-Length`,
    are valid to be processed.

- 200 OK

    Conversion succeeded.  The converted data are returned in the body.
    `Content-Type` header value will be `application/pdf`.

- 400 Bad Request

    The request was bad.

- 404 Not Found

    The requested API does not exist.

- 405 Method Not Allowed

    The request method is not allowed for the API.

- 411 Length Required

    `Content-Length` header was missing.

- 413 Request Entity Too Large

    The request body was too large.

- 415 Unsupported Media Type

    The `file` (actually, the filename) is not supported for conversion.

- 429 Too Many Requests

    There were too many requests beyond the server capacity.

- 5xx

    Various server errors may happen.
