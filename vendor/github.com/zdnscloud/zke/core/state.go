package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/zdnscloud/zke/core/pki"
	"github.com/zdnscloud/zke/pkg/hosts"
	"github.com/zdnscloud/zke/pkg/k8s"
	"github.com/zdnscloud/zke/pkg/log"
	"github.com/zdnscloud/zke/types"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	"k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	stateFileExt = ".zkestate"
	certDirExt   = "_certs"
)

type FullState struct {
	DesiredState State `json:"desiredState,omitempty"`
	CurrentState State `json:"currentState,omitempty"`
}

type State struct {
	ZKEConfig          *types.ZKEConfig              `json:"zkeConfig,omitempty"`
	CertificatesBundle map[string]pki.CertificatePKI `json:"certificatesBundle,omitempty"`
}

func (c *Cluster) UpdateClusterCurrentState(ctx context.Context, fullState *FullState) error {
	fullState.CurrentState.ZKEConfig = c.ZKEConfig.DeepCopy()
	fullState.CurrentState.CertificatesBundle = c.Certificates
	return fullState.WriteStateFile(ctx, pki.StateFileName)
}

func (c *Cluster) UpdateClusterCurrentStateForRest(ctx context.Context, fullState *FullState) (*FullState, error) {
	fullState.CurrentState.ZKEConfig = c.ZKEConfig.DeepCopy()
	fullState.CurrentState.CertificatesBundle = c.Certificates
	return fullState, nil
}

func (c *Cluster) GetClusterState(ctx context.Context, fullState *FullState) (*Cluster, error) {
	var err error
	if fullState.CurrentState.ZKEConfig == nil {
		return nil, nil
	}

	currentCluster, err := InitClusterObject(ctx, fullState.CurrentState.ZKEConfig)
	if err != nil {
		return nil, err
	}
	currentCluster.Certificates = fullState.CurrentState.CertificatesBundle

	// resetup dialers
	dialerOptions := hosts.GetDialerOptions(c.DockerDialerFactory, c.K8sWrapTransport)
	if err := currentCluster.SetupDialers(ctx, dialerOptions); err != nil {
		return nil, err
	}
	return currentCluster, nil
}

func SaveFullStateToKubernetes(ctx context.Context, kubeCluster *Cluster, fullState *FullState) error {

	log.Infof(ctx, "[state] Saving full cluster state to Kubernetes")
	stateFile, err := json.Marshal(*fullState)
	if err != nil {
		return err
	}
	timeout := make(chan bool, 1)
	go func() {
		for {
			_, err := k8s.UpdateConfigMap(kubeCluster.KubeClient, stateFile, FullStateConfigMapName)
			if err != nil {
				time.Sleep(time.Second * 5)
				continue
			}
			log.Infof(ctx, "[state] Successfully Saved full cluster state to Kubernetes ConfigMap: %s", StateConfigMapName)
			timeout <- true
			break
		}
	}()
	select {
	case <-timeout:
		return nil
	case <-time.After(time.Second * UpdateStateTimeout):
		return fmt.Errorf("[state] Timeout waiting for kubernetes to be ready")
	}
}

func GetStateFromKubernetes(ctx context.Context, kubeCluster *Cluster) (*Cluster, error) {
	log.Infof(ctx, "[state] Fetching cluster state from Kubernetes")

	var err error
	var cfgMap *v1.ConfigMap
	var currentCluster Cluster
	timeout := make(chan bool, 1)
	go func() {
		for {
			cfgMap, err = k8s.GetConfigMap(kubeCluster.KubeClient, StateConfigMapName)
			if err != nil {
				time.Sleep(time.Second * 5)
				continue
			}
			log.Infof(ctx, "[state] Successfully Fetched cluster state to Kubernetes ConfigMap: %s", StateConfigMapName)
			timeout <- true
			break
		}
	}()
	select {
	case <-timeout:
		clusterData := cfgMap.Data[StateConfigMapName]
		err := yaml.Unmarshal([]byte(clusterData), &currentCluster)
		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshal cluster data")
		}
		return &currentCluster, nil
	case <-time.After(time.Second * GetStateTimeout):
		log.Infof(ctx, "Timed out waiting for kubernetes cluster to get state")
		return nil, fmt.Errorf("Timeout waiting for kubernetes cluster to get state")
	}
}

func GetK8sVersion(k8sClient *kubernetes.Clientset) (string, error) {
	discoveryClient := k8sClient.DiscoveryClient
	logrus.Debugf("[version] Getting Kubernetes server version..")
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return "", fmt.Errorf("Failed to get Kubernetes server version: %v", err)
	}
	return fmt.Sprintf("%#v", *serverVersion), nil
}

func RebuildState(ctx context.Context, zkeConfig *types.ZKEConfig, oldState *FullState) (*FullState, error) {
	newState := &FullState{
		DesiredState: State{
			ZKEConfig: zkeConfig.DeepCopy(),
		},
	}

	// Rebuilding the certificates of the desired state
	if oldState.DesiredState.CertificatesBundle == nil {
		// Get the certificate Bundle
		certBundle, err := pki.GenerateZKECerts(ctx, *zkeConfig)
		if err != nil {
			return nil, fmt.Errorf("Failed to generate certificate bundle: %v", err)
		}
		newState.DesiredState.CertificatesBundle = certBundle
	} else {
		// Regenerating etcd certificates for any new etcd nodes
		pkiCertBundle := oldState.DesiredState.CertificatesBundle
		if err := pki.GenerateZKEServicesCerts(ctx, pkiCertBundle, *zkeConfig, false); err != nil {
			return nil, err
		}
		newState.DesiredState.CertificatesBundle = pkiCertBundle
	}
	newState.CurrentState = oldState.CurrentState
	return newState, nil
}

func (s *FullState) WriteStateFile(ctx context.Context, statePath string) error {
	stateFile, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to Marshal state object: %v", err)
	}
	logrus.Debugf("Writing state file: %s", stateFile)
	if err := ioutil.WriteFile(statePath, stateFile, 0640); err != nil {
		return fmt.Errorf("Failed to write state file: %v", err)
	}
	log.Infof(ctx, "Successfully Deployed state file at [%s]", statePath)
	return nil
}

func GetStateFilePath(configPath, configDir string) string {
	if configPath == "" {
		configPath = pki.ClusterConfig
	}
	baseDir := filepath.Dir(configPath)
	if len(configDir) > 0 {
		baseDir = filepath.Dir(configDir)
	}
	fileName := filepath.Base(configPath)
	baseDir += "/"
	fullPath := fmt.Sprintf("%s%s", baseDir, fileName)
	trimmedName := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	return trimmedName + stateFileExt
}

func GetCertificateDirPath(configPath, configDir string) string {
	if configPath == "" {
		configPath = pki.ClusterConfig
	}
	baseDir := filepath.Dir(configPath)
	if len(configDir) > 0 {
		baseDir = filepath.Dir(configDir)
	}
	fileName := filepath.Base(configPath)
	baseDir += "/"
	fullPath := fmt.Sprintf("%s%s", baseDir, fileName)
	trimmedName := strings.TrimSuffix(fullPath, filepath.Ext(fullPath))
	return trimmedName + certDirExt
}

func ReadStateFile(ctx context.Context, statePath string) (*FullState, error) {
	zkeFullState := &FullState{}
	fp, err := filepath.Abs(statePath)
	if err != nil {
		return zkeFullState, fmt.Errorf("failed to lookup current directory name: %v", err)
	}
	file, err := os.Open(fp)
	if err != nil {
		return zkeFullState, fmt.Errorf("Can not find ZKE state file: %v", err)
	}
	defer file.Close()
	buf, err := ioutil.ReadAll(file)
	if err != nil {
		return zkeFullState, fmt.Errorf("failed to read state file: %v", err)
	}
	if err := json.Unmarshal(buf, zkeFullState); err != nil {
		return zkeFullState, fmt.Errorf("failed to unmarshal the state file: %v", err)
	}
	zkeFullState.DesiredState.CertificatesBundle = pki.TransformPEMToObject(zkeFullState.DesiredState.CertificatesBundle)
	zkeFullState.CurrentState.CertificatesBundle = pki.TransformPEMToObject(zkeFullState.CurrentState.CertificatesBundle)
	return zkeFullState, nil
}

func ReadStateJson(ctx context.Context, stateJson string) (*FullState, error) {
	zkeFullState := &FullState{}
	if len(stateJson) == 0 {
		return zkeFullState, nil
	}
	if err := json.Unmarshal([]byte(stateJson), zkeFullState); err != nil {
		return zkeFullState, fmt.Errorf("failed to unmarshal the state file: %v", err)
	}
	zkeFullState.DesiredState.CertificatesBundle = pki.TransformPEMToObject(zkeFullState.DesiredState.CertificatesBundle)
	zkeFullState.CurrentState.CertificatesBundle = pki.TransformPEMToObject(zkeFullState.CurrentState.CertificatesBundle)
	return zkeFullState, nil
}

func removeStateFile(ctx context.Context, statePath string) {
	log.Infof(ctx, "Removing state file: %s", statePath)
	if err := os.Remove(statePath); err != nil {
		logrus.Warningf("Failed to remove state file: %v", err)
		return
	}
	log.Infof(ctx, "State file removed successfully")
}
