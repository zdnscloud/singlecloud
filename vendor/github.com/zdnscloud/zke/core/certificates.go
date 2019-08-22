package core

import (
	"context"

	"github.com/zdnscloud/zke/pkg/log"
)

func SetUpAuthentication(ctx context.Context, kubeCluster, currentCluster *Cluster, fullState *FullState) {
	select {
	case <-ctx.Done():
		log.Infof(ctx, "cluster build has beed canceled")
	default:
		if kubeCluster.AuthnStrategies[AuthnX509Provider] {
			kubeCluster.Certificates = fullState.DesiredState.CertificatesBundle
		}
	}
}
