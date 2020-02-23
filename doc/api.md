# API

## Meta

| | |
|:---|:---|
| API Group | `frp.go.build4.fun` |
| Current Version | `v1` |

## `Endpoint`

Endpoint resource describes an frp server endpoint (`frps.ini`).

| spec field | type | description |
|:------:|:---:|:----------|
| `addr` | `string` | the address of the remote endpoint, **required**  |
| `port` | `int32` | the port of the remote endpoint, **required**  |
| `token` | `string` | the token to connect to the remote endpoint, **required**  |

## `Service`

Service resource describes & selects local pods to expose (`frpc.ini`).

| spec field | type | description |
|:------:|:---:|:----------|
| `endpoint` | `string` | name of the endpoint to use, **required**  |
| `selector` | `map[string]string` | pods selector, same as `corev1/Service#selector`, **required**  |
| `serviceLabels` | `map[string]string` | extra labels to set for the generated service object, defaults to empty |
| `ports` | `[]ServciePort` | list of ports to expose |


## `ServicePort`

ServicePort resource describes a service port to expose to frp server.

| spec field | type | description |
|:------:|:---:|:----------|
| `name` | `string` | name of the port, must be `DNS_LABEL` format, **required** |
| `protocol` | `ServiceProtocol` | protocol to use, values: `TCP` / `UDP`, **required** |
| `localPort` | `int32` | local port to expose (`corev1/Service.ports.TargetPort`) |
| `remotePort` | `int32` | report port to use (`corev1/Service.ports.Port`) |