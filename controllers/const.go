package controllers

import frpv1 "github.com/b4fun/frpcontroller/api/v1"

var (
	apiGVStr = frpv1.GroupVersion.String()
)

const (
	KindEndpoint = "Endpoint"
	KindService  = "Service"

	frpsFileName = "frps.ini"
	frpcFileName = "frpc.ini"
)
