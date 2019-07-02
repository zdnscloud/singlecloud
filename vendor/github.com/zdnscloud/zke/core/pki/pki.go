package pki

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"
	"path"
	"path/filepath"
	"strings"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/docker/docker/api/types/container"
)

type CertificatePKI struct {
	Certificate    *x509.Certificate        `json:"-"`
	Key            *rsa.PrivateKey          `json:"-"`
	CSR            *x509.CertificateRequest `json:"-"`
	CertificatePEM string                   `json:"certificatePEM"`
	KeyPEM         string                   `json:"keyPEM"`
	CSRPEM         string                   `json:"-"`
	Config         string                   `json:"config"`
	Name           string                   `json:"name"`
	CommonName     string                   `json:"commonName"`
	OUName         string                   `json:"ouName"`
	EnvName        string                   `json:"envName"`
	Path           string                   `json:"path"`
	KeyEnvName     string                   `json:"keyEnvName"`
	KeyPath        string                   `json:"keyPath"`
	ConfigEnvName  string                   `json:"configEnvName"`
	ConfigPath     string                   `json:"configPath"`
}

type GenFunc func(context.Context, map[string]CertificatePKI, types.ZKEConfig, bool) error
type CSRFunc func(context.Context, map[string]CertificatePKI, types.ZKEConfig) error

const (
	etcdRole            = "etcd"
	controlRole         = "controlplane"
	workerRole          = "worker"
	BundleCertContainer = "zke-bundle-cert"
)

func GenerateZKECerts(ctx context.Context, zkeConfig types.ZKEConfig) (map[string]CertificatePKI, error) {
	certs := make(map[string]CertificatePKI)
	// generate ZKE CA certificates
	if err := GenerateZKECACerts(ctx, certs); err != nil {
		return certs, err
	}
	// Generating certificates for kubernetes components
	if err := GenerateZKEServicesCerts(ctx, certs, zkeConfig, false); err != nil {
		return certs, err
	}
	return certs, nil
}

func GenerateZKENodeCerts(ctx context.Context, zkeConfig types.ZKEConfig, nodeAddress string, certBundle map[string]CertificatePKI) map[string]CertificatePKI {
	crtMap := make(map[string]CertificatePKI)
	crtKeys := []string{}
	removeCAKey := true
	for _, node := range zkeConfig.Nodes {
		if node.Address == nodeAddress {
			for _, role := range node.Role {
				switch role {
				case controlRole:
					keys := getControlCertKeys()
					crtKeys = append(crtKeys, keys...)
					removeCAKey = false
				case workerRole:
					keys := getWorkerCertKeys()
					crtKeys = append(crtKeys, keys...)
				case etcdRole:
					keys := getEtcdCertKeys(zkeConfig.Nodes, etcdRole)
					crtKeys = append(crtKeys, keys...)
				}
			}
			break
		}
	}
	for _, key := range crtKeys {
		crtMap[key] = certBundle[key]
	}
	if removeCAKey {
		caCert := crtMap[CACertName]
		caCert.Key = nil
		caCert.KeyEnvName = ""
		caCert.KeyPath = ""
		crtMap[CACertName] = caCert
	}
	return crtMap
}

func RegenerateEtcdCertificate(
	ctx context.Context,
	crtMap map[string]CertificatePKI,
	etcdHost *hosts.Host,
	etcdHosts []*hosts.Host,
	clusterDomain string,
	KubernetesServiceIP net.IP) (map[string]CertificatePKI, error) {

	log.Infof(ctx, "[certificates] Regenerating new etcd-%s certificate and key", etcdHost.InternalAddress)
	caCrt := crtMap[CACertName].Certificate
	caKey := crtMap[CACertName].Key
	etcdAltNames := GetAltNames(etcdHosts, clusterDomain, KubernetesServiceIP, []string{})

	etcdCrt, etcdKey, err := GenerateSignedCertAndKey(caCrt, caKey, true, EtcdCertName, etcdAltNames, nil, nil)
	if err != nil {
		return nil, err
	}
	etcdName := GetEtcdCrtName(etcdHost.InternalAddress)
	crtMap[etcdName] = ToCertObject(etcdName, "", "", etcdCrt, etcdKey, nil)
	log.Infof(ctx, "[certificates] Successfully generated new etcd-%s certificate and key", etcdHost.InternalAddress)
	return crtMap, nil
}

func SaveBackupBundleOnHost(ctx context.Context, host *hosts.Host, alpineSystemImage, etcdSnapshotPath string, prsMap map[string]types.PrivateRegistry) error {
	imageCfg := &container.Config{
		Cmd: []string{
			"sh",
			"-c",
			fmt.Sprintf("if [ -d %s ] && [ \"$(ls -A %s)\" ]; then tar czvf %s %s;fi", TempCertPath, TempCertPath, BundleCertPath, TempCertPath),
		},
		Image: alpineSystemImage,
	}
	hostCfg := &container.HostConfig{

		Binds: []string{
			fmt.Sprintf("%s:/etc/kubernetes:z", path.Join(host.PrefixPath, "/etc/kubernetes")),
			fmt.Sprintf("%s:/backup:z", etcdSnapshotPath),
		},
		Privileged: true,
	}
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, BundleCertContainer, host.Address, "certificates", prsMap); err != nil {
		return err
	}
	status, err := docker.WaitForContainer(ctx, host.DClient, host.Address, BundleCertContainer)
	if err != nil {
		return err
	}
	if status != 0 {
		return fmt.Errorf("Failed to run certificate bundle compress, exit status is: %d", status)
	}
	log.Infof(ctx, "[certificates] successfully saved certificate bundle [%s/pki.bundle.tar.gz] on host [%s]", etcdSnapshotPath, host.Address)
	return docker.RemoveContainer(ctx, host.DClient, host.Address, BundleCertContainer)
}

func ExtractBackupBundleOnHost(ctx context.Context, host *hosts.Host, alpineSystemImage, etcdSnapshotPath string, prsMap map[string]types.PrivateRegistry) error {
	imageCfg := &container.Config{
		Cmd: []string{
			"sh",
			"-c",
			fmt.Sprintf(
				"mkdir -p %s; tar xzvf %s -C %s --strip-components %d --exclude %s",
				TempCertPath,
				BundleCertPath,
				TempCertPath,
				len(strings.Split(filepath.Clean(TempCertPath), "/"))-1,
				ClusterStateFile),
		},
		Image: alpineSystemImage,
	}
	hostCfg := &container.HostConfig{

		Binds: []string{
			fmt.Sprintf("%s:/etc/kubernetes:z", path.Join(host.PrefixPath, "/etc/kubernetes")),
			fmt.Sprintf("%s:/backup:z", etcdSnapshotPath),
		},
		Privileged: true,
	}
	if err := docker.DoRunContainer(ctx, host.DClient, imageCfg, hostCfg, BundleCertContainer, host.Address, "certificates", prsMap); err != nil {
		return err
	}
	status, err := docker.WaitForContainer(ctx, host.DClient, host.Address, BundleCertContainer)
	if err != nil {
		return err
	}
	if status != 0 {
		containerErrLog, _, err := docker.GetContainerLogsStdoutStderr(ctx, host.DClient, BundleCertContainer, "5", false)
		if err != nil {
			return err
		}
		// removing the container in case of an error too
		if err := docker.RemoveContainer(ctx, host.DClient, host.Address, BundleCertContainer); err != nil {
			return err
		}
		return fmt.Errorf("Failed to run certificate bundle extract, exit status is: %d, container logs: %s", status, containerErrLog)
	}
	log.Infof(ctx, "[certificates] successfully extracted certificate bundle on host [%s] to backup path [%s]", host.Address, TempCertPath)
	return docker.RemoveContainer(ctx, host.DClient, host.Address, BundleCertContainer)
}
