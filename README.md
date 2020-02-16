# frp controller

| Resource | Link |
|:----|:----|
| Azure DevOps Build | [![Build Status](https://dev.azure.com/build4/b4fun%20-%20public/_apis/build/status/b4fun.frpcontroller?branchName=master)](https://dev.azure.com/build4/b4fun%20-%20public/_build/latest?definitionId=1&branchName=master) |
| Reference | [![API](https://godoc.org/github.com/b4fun/frpcontroller?status.svg)](https://godoc.org/github.com/b4fun/frpcontroller/api/v1) |
| Docker Image (latest) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28latest%29)](https://hub.docker.com/r/b4fun/frpcontroller) |
| Docker Image (v20200216) | [![](https://img.shields.io/docker/pulls/b4fun/frpcontroller?label=docker%20pulls%20%28v20200216%29)](https://hub.docker.com/r/b4fun/frpcontroller) |

## Installation

```
$ kubectl apply -f https://raw.githubusercontent.com/b4fun/frpcontroller/master/release/v20200216/install.yaml 
# or, if you're installing from China
$ kubectl apply -f https://raw.githubusercontent.com/b4fun/frpcontroller/master/release/v20200216/install-cn.yaml 
```

## Usage

TODO

## TODO

- improve refresh usage of the controllers
- test
- doc
- add `ServerEndpoint` resource

## Hacking

### Do a release

```
$ RELEASE=v20200216 make release
```

## Change History

### [`v20200216`](https://github.com/b4fun/frpcontroller/releases/tag/v20200216)

- initial version
- `Endpoint` resource
- `Service` resource
- controllers

## LICENSE

MIT