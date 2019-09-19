package types

import (
	gorestError "github.com/zdnscloud/gorest/error"
)

var (
	//cluster
	InvalidClusterConfig = gorestError.ErrorCode{"InvalidClusterConfig", 422}
	ConnectClusterFailed = gorestError.ErrorCode{"ConnectClusterFailed", 422}
)
