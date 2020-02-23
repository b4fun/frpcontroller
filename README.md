# frp controller

| Resource | Link |
|:----|:----|
| Azure DevOps Build | [![Build Status](https://dev.azure.com/build4/b4fun%20-%20public/_apis/build/status/b4fun.frpcontroller?branchName=master)](https://dev.azure.com/build4/b4fun%20-%20public/_build/latest?definitionId=1&branchName=master) |
| Reference | [![API](https://godoc.org/github.com/b4fun/frpcontroller?status.svg)](https://godoc.org/github.com/b4fun/frpcontroller/api/v1) |
| Docker Image (latest) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28latest%29)](https://hub.docker.com/r/b4fun/frpcontroller) |
| Docker Image (v20200222) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28v20200222%29)](https://hub.docker.com/r/b4fun/frpcontroller) |
| Docker Image (v20200216) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28v20200216%29)](https://hub.docker.com/r/b4fun/frpcontroller) |

## Usage

| What to do | Doc to read |
|:-----------|:------------|
| Quick start | [Get Start](./docs/get-start.md)
| Find the API | [API](./docs/api.md)

## TODO

- improve refresh usage of the controllers
- test
- add `ServerEndpoint` resource

## Hacking

### Do a release

```
$ RELEASE=v20200216 make release
$ RELEASE=v20200216 make build-docker
$ RELEASE=latest make release
$ RELEASE=latest make build-docker
```

## Change History

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