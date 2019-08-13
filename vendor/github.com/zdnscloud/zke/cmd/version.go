package cmd

import (
	"fmt"

	"github.com/zdnscloud/zke/core"
	"github.com/zdnscloud/zke/types"
)

func validateConfigVersion(zkeConfig *types.ZKEConfig) error {
	if zkeConfig.ConfigVersion != core.DefaultConfigVersion {
		return fmt.Errorf("config version not match[new version is %s, and current config file version is %s], please execut config command to generate new config", core.DefaultConfigVersion, zkeConfig.ConfigVersion)
	}
	return nil
}
