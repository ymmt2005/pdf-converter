![CI](https://github.com/ymmt2005/pdf-converter/workflows/CI/badge.svg)

pdf-converter
=============

This directory contains a Dockerfile to convert files to PDF using [LibreOffice][].

## Image registries

- [ghcr.io/ymmt2005/pdf-converter](https://github.com/users/ymmt2005/packages/container/package/pdf-converter)
- [quay.io/ymmt2005/pdf-converter](https://quay.io/repository/ymmt2005/pdf-converter?tag=latest&tab=tags)

## Usage

### Run conversion server

```console
$ docker run -d --rm -p 127.0.0.1:8080:8080/tcp --tmpfs /tmp ghcr.io/ymmt2005/pdf-converter
```

Try it using `curl`:

```console
$ curl -o result.pdf -F file=@/path/to/your.pptx http://localhost:8080/convert
```

See [docs/api.md](docs/api.md) for the HTTP API.

### Run LibreOffice directly

To convert `$HOME/work/foo.pptx` directly,

```console
$ sudo chown -R 10000:10000 $HOME/work
$ docker run -it --rm --tmpfs /tmp -v $HOME/work:/home/libre \
    ghcr.io/ymmt2005/pdf-converter soffice --headless --convert-to pdf /home/libre/foo.pptx
```

## Metrics

`pdf-converter` exports metrics for Prometheus at `/metrics` endpoint.

Read [docs/metrics.md](docs/metrics.md) for details.

## `pdf-converter` command reference

Run HTTP server to convert Office files to PDF

```
Usage:
  pdf-converter WORKDIR [flags]

Flags:
  -h, --help                        help for pdf-converter
      --listen string               bind address of the HTTP server (default ":8080")
      --logfile string              Log filename
      --logformat string            Log format [plain,logfmt,json]
      --loglevel string             Log level [critical,error,warning,info,debug]
      --max-convert-time duration   the maximum time allowed for conversion (default 3m0s)
      --max-length int              the maximum length of the uploaded contents (default 1073741824)
      --max-parallel int            the maximum parallel conversions.  0 for unlimited parallelism
```

[LibreOffice]: https://www.libreoffice.org/
