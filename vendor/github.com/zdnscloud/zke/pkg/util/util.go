package util

import (
	"fmt"
	"os"
	"strings"

	"github.com/zdnscloud/zke/types"

	"github.com/coreos/go-semver/semver"
)

const (
	WorkerThreads = 50
)

func StrToSemVer(version string) (*semver.Version, error) {
	v, err := semver.NewVersion(strings.TrimPrefix(version, "v"))
	if err != nil {
		return nil, err
	}
	return v, nil
}

// UniqueStringSlice - Input slice, retrun slice with unique elements. Will not maintain order.
func UniqueStringSlice(elements []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range elements {
		if !encountered[elements[v]] {
			encountered[elements[v]] = true
			result = append(result, elements[v])
		}
	}
	return result
}

func IsSymlink(file string) (bool, error) {
	f, err := os.Lstat(file)
	if err != nil {
		return false, err
	}
	if f.Mode()&os.ModeSymlink != 0 {
		return true, nil
	}
	return false, nil
}

// ValidateVersion - Return error if version is not valid
// Is version major.minor >= oldest major.minor supported
// Is version in the AllK8sVersions list
// Is version not in the "bad" list
func ValidateVersion(version string) error {
	// Create target version and current versions list
	targetVersion, err := StrToSemVer(version)
	if err != nil {
		return fmt.Errorf("%s is not valid semver", version)
	}
	currentVersionsList := []*semver.Version{}
	for _, ver := range types.K8sVersionsCurrent {
		v, err := StrToSemVer(ver)
		if err != nil {
			return fmt.Errorf("%s in Current Versions list is not valid semver", ver)
		}

		currentVersionsList = append(currentVersionsList, v)
	}

	// Make sure Target version is greater than or equal to oldest major.minor supported.
	semver.Sort(currentVersionsList)
	if targetVersion.Major < currentVersionsList[0].Major {
		return fmt.Errorf("%s is an unsupported Kubernetes version - see 'zke config --system-images --all' for versions supported with this release", version)
	}
	if targetVersion.Major == currentVersionsList[0].Major {
		if targetVersion.Minor < currentVersionsList[0].Minor {
			return fmt.Errorf("%s is an unsupported Kubernetes version - see 'zke config --system-images --all' for versions supported with this release", version)
		}
	}
	// Make sure Target version is in the AllK8sVersions list.
	_, ok := types.AllK8sVersions[version]
	if !ok {
		return fmt.Errorf("%s is an unsupported Kubernetes version - see 'zke config --system-images --all' for versions supported with this release", version)
	}
	// Make sure Target version is not "bad".
	_, ok = types.K8sBadVersions[version]
	if ok {
		return fmt.Errorf("%s is an unsupported Kubernetes version - see 'zke config --system-images --all' for versions supported with this release", version)
	}

	return nil
}
