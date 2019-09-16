package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

var (
	//cluster
	InvalidClusterConfig = resttypes.ErrorCode{"InvalidClusterConfig", 422}
	ConnectClusterFailed = resttypes.ErrorCode{"ConnectClusterFailed", 422}
)
