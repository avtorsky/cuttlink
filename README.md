# cuttlink

[About](#about) /
[Changelog](#changelog) /
[Deploy](#deploy) /
[Testing](#testing) /
[Healthcheck](#healthcheck)

[![CI](https://github.com/avtorsky/cuttlink/actions/workflows/shortenertest.yml/badge.svg?branch=iter7)](https://github.com/avtorsky/cuttlink/actions/workflows/shortenertest.yml)

## About
URL shortener service

## Changelog
Release 20230122:
feat(./internal/server/gzip.go): sprint2 iter8 gzip compression config

Release 20230121:
feat(./cmd/shortener/main.go): sprint2 iter7 cli flags config
docs(./README.md): add deploy, testing && healthcheck specifications

Release 20230119:
feat(./internal/config): sprint2 iter6 file storage config

Release 20230117:
feat(./internal/config): sprint2 iter5 env config

Release 20230116:
feat(./internal/server): sprint2 iter4 /api/shorten endpoint serialization setup && unit tests coverage
test(./internal/server): /api/shorten endpoint unit tests coverage

Release 20221224:
* refactor(./internal/server): iter3 add Gin framework for routing && cover with unit tests

Release 20221224:
* test(./internal/server): iter2 server.go unit tests setup

Release 20221222:
* feat(./cmd/shortener): compiled binary for iter1 && autotests fixes

## Deploy

Clone repository and compile binary

```bash
git clone https://github.com/avtorsky/cuttlink.git
cd cmd/shortener
go build -o cuttlink main.go
```

Define settings using CLI flags and init server

```bash
./cuttlink --help        
Usage of ./cuttlink:
  -a string
    	define server address (default ":8080")
  -b string
    	define base URL (default "http://localhost:8080")
  -f string
    	define file storage path (default "kv_store.txt")

./cuttlink
```

## Testing

Run unit test from root directory:

```bash
GIN_MODE=release go test -v ./internal/server -run ^TestServer
```

## Healthcheck

Basic endpoints healthcheck

```bash
curl -X POST http://localhost:8080/api/shorten \
    -H 'Content-Type: application/json' \
    -d '{"url": "https://explorer.avtorskydeployed.online/"}'

{"result":"http://localhost:8080/2"}
```

```bash
curl -sI -X GET -L http://localhost:8080/2

HTTP/1.1 307 Temporary Redirect
Content-Type: text/html; charset=utf-8
Location: https://explorer.avtorskydeployed.online/
```
