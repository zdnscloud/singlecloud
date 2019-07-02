package hosts

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/zdnscloud/zke/pkg/docker"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/pkg/util"

	"github.com/docker/docker/client"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh"
)

const (
	DockerAPIVersion = "1.24"
)

func (h *Host) TunnelUp(ctx context.Context, dialerFactory DialerFactory, clusterPrefixPath string, clusterVersion string) error {
	if h.DClient != nil {
		return nil
	}
	log.Infof(ctx, "[dialer] Setup tunnel for host [%s]", h.Address)
	httpClient, err := h.newHTTPClient(dialerFactory)
	if err != nil {
		return fmt.Errorf("Can't establish dialer connection: %v", err)
	}
	// set Docker client
	logrus.Debugf("Connecting to Docker API for host [%s]", h.Address)
	h.DClient, err = client.NewClient("unix:///var/run/docker.sock", DockerAPIVersion, httpClient, nil)
	if err != nil {
		return fmt.Errorf("Can't initiate NewClient: %v", err)
	}
	if err := checkDockerVersion(ctx, h, clusterVersion); err != nil {
		return err
	}
	h.PrefixPath = clusterPrefixPath
	return nil
}

func checkDockerVersion(ctx context.Context, h *Host, clusterVersion string) error {
	info, err := h.DClient.Info(ctx)
	if err != nil {
		return fmt.Errorf("Can't retrieve Docker Info: %v", err)
	}
	logrus.Debugf("Docker Info found: %#v", info)
	h.DockerInfo = info
	K8sSemVer, err := util.StrToSemVer(clusterVersion)
	if err != nil {
		return fmt.Errorf("Error while parsing cluster version [%s]: %v", clusterVersion, err)
	}
	K8sVersion := fmt.Sprintf("%d.%d", K8sSemVer.Major, K8sSemVer.Minor)
	isvalid, err := docker.IsSupportedDockerVersion(info, K8sVersion)
	if err != nil {
		return fmt.Errorf("Error while determining supported Docker version [%s]: %v", info.ServerVersion, err)
	}

	if !isvalid && !h.IgnoreDockerVersion {
		return fmt.Errorf("Unsupported Docker version found [%s], supported versions are %v", info.ServerVersion, docker.K8sDockerVersions[K8sVersion])
	} else if !isvalid {
		log.Warnf(ctx, "Unsupported Docker version found [%s], supported versions are %v", info.ServerVersion, docker.K8sDockerVersions[K8sVersion])
	}
	return nil
}

func parsePrivateKey(keyBuff string) (ssh.Signer, error) {
	return ssh.ParsePrivateKey([]byte(keyBuff))
}

func GetSSHConfig(username, sshPrivateKeyString string) (*ssh.ClientConfig, error) {
	config := &ssh.ClientConfig{
		User:            username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	signer, err := parsePrivateKey(sshPrivateKeyString)
	if err != nil {
		return config, err
	}

	config.Auth = append(config.Auth, ssh.PublicKeys(signer))

	return config, nil
}

func PrivateKeyPath(sshKeyPath string) (string, error) {
	if sshKeyPath[:2] == "~/" {
		sshKeyPath = filepath.Join(userHome(), sshKeyPath[2:])
	}
	buff, err := ioutil.ReadFile(sshKeyPath)
	if err != nil {
		return "", fmt.Errorf("Error while reading SSH key file: %v", err)
	}
	return string(buff), nil
}

func CertificatePath(sshCertPath string) (string, error) {
	if sshCertPath[:2] == "~/" {
		sshCertPath = filepath.Join(userHome(), sshCertPath[2:])
	}
	buff, err := ioutil.ReadFile(sshCertPath)
	if err != nil {
		return "", fmt.Errorf("Error while reading SSH certificate file: %v", err)
	}
	return string(buff), nil
}

func userHome() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	homeDrive := os.Getenv("HOMEDRIVE")
	homePath := os.Getenv("HOMEPATH")
	if homeDrive != "" && homePath != "" {
		return homeDrive + homePath
	}
	return os.Getenv("USERPROFILE")
}
