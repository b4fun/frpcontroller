# frp controller

| Resource | Link |
|:----|:----|
| Azure DevOps Build | [![Build Status](https://dev.azure.com/build4/b4fun%20-%20public/_apis/build/status/b4fun.frpcontroller?branchName=master)](https://dev.azure.com/build4/b4fun%20-%20public/_build/latest?definitionId=1&branchName=master) |
| Reference | [![API](https://godoc.org/github.com/b4fun/frpcontroller?status.svg)](https://godoc.org/github.com/b4fun/frpcontroller/api/v1) |
| Docker Image (latest) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28latest%29)](https://hub.docker.com/r/b4fun/frpcontroller) |
| Docker Image (v20200308) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28v20200308%29)](https://hub.docker.com/r/b4fun/frpcontroller) |

## Usage

| What to do | Doc to read |
|:-----------|:------------|
| Quick start | [Get Start](./docs/get-start.md)
| Find the API | [API](./docs/api.md)

## TODO

- improve refresh usage of the controllers
- add `ServerEndpoint` resource

## Hacking

### Run e2e test (in local)

1. Setup a test cluster (e.g. by using [kind][kind])
2. `make test-local`

[kind]: https://github.com/kubernetes-sigs/kind

### Make release

```
$ export RELEASE=v20200216
$ make release
$ make docker-build
$ make docker-push
$ export RELEASE=latest
$ make release
$ make docker-build
$ make docker-push
```

## Change History

### [`v20200308`](https://github.com/b4fun/frpcontroller/releases/tag/v20200308)

- implement endpoint/service state update logic
- add e2e tests

### [`v20200222`](https://github.com/b4fun/frpcontroller/releases/tag/v20200222)

- fix #1: add service port name validation
- fix #2: add `serviceLabels` field to decorate created service labels

### [`v20200216`](https://github.com/b4fun/frpcontroller/releases/tag/v20200216)

- initial version
- `Endpoint` resource
- `Service` resource
- controllers

## LICENSE

MIT

---

a [@b4fun][@b4fun] project

[@b4fun]: https://www.build4.fun