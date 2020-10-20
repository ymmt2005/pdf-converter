![CI](https://github.com/ymmt2005/pdf-converter/workflows/CI/badge.svg)

pdf-converter
=============

This directory contains a Dockerfile to convert files to PDF using [LibreOffice][].

## Usage

### Run conversion server

```console
$ docker run -d --rm -p 127.0.0.1:8080:8080/tcp --tmpfs /tmp ghcr.io/ymmt2005/pdf-converter
```

See [API.md](API.md) for the HTTP API.

### Run LibreOffice directly

To convert `$HOME/work/foo.pptx` directly,

```console
$ docker run -it --user $(id -u) --rm --tmpfs /tmp -v $HOME/work:/home/libre \
    ghcr.io/ymmt2005/pdf-converter soffice --headless --convert-to pdf /home/libre/foo.pptx
```

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
