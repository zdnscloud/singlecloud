package main

import "github.com/zdnscloud/singlecloud/pkg/alarm"

func main() {
	alarm.NewAlarm().
		Namespace("zcloudxxxx").
		Kind("pod").
		Name("csi-cephfsplugin-4h7qc").
		Message("MountVolume.SetUp failed for volume ceph-csi-config : configmaps ceph-csi-config not found").
		Reason("FailedMount").
		Publish()
}
