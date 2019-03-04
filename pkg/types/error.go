package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

var (
	//cluster
	InvalidClusterConfig = resttypes.ErrorCode{"InvalidClusterConfig", 422}
	ConnectClusterFailed = resttypes.ErrorCode{"ConnectClusterFailed", 422}

	//node
)
