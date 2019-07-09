package cmd

import (
	"fmt"

	"github.com/zdnscloud/zke/types"
)

const defaultConfigVersion = "v1.0.9"

func validateConfigVersion(zkeConfig *types.ZKEConfig) error {
	if zkeConfig.ConfigVersion != defaultConfigVersion {
		return fmt.Errorf("config version not match[new version is %s, and current config file version is %s], please execut config command to generate new config", defaultConfigVersion, zkeConfig.ConfigVersion)
	}
	return nil
}
