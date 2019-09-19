package types

import (
	resterror "github.com/zdnscloud/gorest/error"
)

var (
	//cluster
	InvalidClusterConfig = resterror.ErrorCode{"InvalidClusterConfig", 422}
	ConnectClusterFailed = resterror.ErrorCode{"ConnectClusterFailed", 422}
)
