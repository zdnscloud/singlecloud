package core

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strings"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"

	"k8s.io/client-go/util/cert"
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

func GetClusterCertsFromKubernetes(ctx context.Context, kubeCluster *Cluster) (map[string]pki.CertificatePKI, error) {
	log.Infof(ctx, "[certificates] Getting Cluster certificates from Kubernetes")

	certificatesNames := []string{
		pki.CACertName,
		pki.KubeAPICertName,
		pki.KubeNodeCertName,
		pki.KubeProxyCertName,
		pki.KubeControllerCertName,
		pki.KubeSchedulerCertName,
		pki.KubeAdminCertName,
		pki.APIProxyClientCertName,
		pki.RequestHeaderCACertName,
		pki.ServiceAccountTokenKeyName,
	}

	for _, etcdHost := range kubeCluster.EtcdHosts {
		etcdName := pki.GetEtcdCrtName(etcdHost.InternalAddress)
		certificatesNames = append(certificatesNames, etcdName)
	}

	certMap := make(map[string]pki.CertificatePKI)
	for _, certName := range certificatesNames {
		secret, err := k8s.GetSecret(kubeCluster.KubeClient, certName)
		if err != nil && !strings.HasPrefix(certName, "kube-etcd") &&
			!strings.Contains(certName, pki.RequestHeaderCACertName) &&
			!strings.Contains(certName, pki.APIProxyClientCertName) &&
			!strings.Contains(certName, pki.ServiceAccountTokenKeyName) {
			return nil, err
		}
		// If I can't find an etcd, requestheader, or proxy client cert, I will not fail and will create it later.
		if (secret == nil || secret.Data == nil) &&
			(strings.HasPrefix(certName, "kube-etcd") ||
				strings.Contains(certName, pki.RequestHeaderCACertName) ||
				strings.Contains(certName, pki.APIProxyClientCertName) ||
				strings.Contains(certName, pki.ServiceAccountTokenKeyName)) {
			certMap[certName] = pki.CertificatePKI{}
			continue
		}

		secretCert, err := cert.ParseCertsPEM(secret.Data["Certificate"])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse certificate of %s: %v", certName, err)
		}
		secretKey, err := cert.ParsePrivateKeyPEM(secret.Data["Key"])
		if err != nil {
			return nil, fmt.Errorf("Failed to parse private key of %s: %v", certName, err)
		}
		secretConfig := string(secret.Data["Config"])
		if len(secretCert) == 0 || secretKey == nil {
			return nil, fmt.Errorf("certificate or key of %s is not found", certName)
		}
		certificatePEM := string(cert.EncodeCertPEM(secretCert[0]))
		keyPEM := string(cert.EncodePrivateKeyPEM(secretKey.(*rsa.PrivateKey)))

		certMap[certName] = pki.CertificatePKI{
			Certificate:    secretCert[0],
			Key:            secretKey.(*rsa.PrivateKey),
			CertificatePEM: certificatePEM,
			KeyPEM:         keyPEM,
			Config:         secretConfig,
			EnvName:        string(secret.Data["EnvName"]),
			ConfigEnvName:  string(secret.Data["ConfigEnvName"]),
			KeyEnvName:     string(secret.Data["KeyEnvName"]),
			Path:           string(secret.Data["Path"]),
			KeyPath:        string(secret.Data["KeyPath"]),
			ConfigPath:     string(secret.Data["ConfigPath"]),
		}
	}
	// Handle service account token key issue
	kubeAPICert := certMap[pki.KubeAPICertName]
	if certMap[pki.ServiceAccountTokenKeyName].Key == nil {
		log.Infof(ctx, "[certificates] Creating service account token key")
		certMap[pki.ServiceAccountTokenKeyName] = pki.ToCertObject(pki.ServiceAccountTokenKeyName, pki.ServiceAccountTokenKeyName, "", kubeAPICert.Certificate, kubeAPICert.Key, nil)
	}
	log.Infof(ctx, "[certificates] Successfully fetched Cluster certificates from Kubernetes")
	return certMap, nil
}
