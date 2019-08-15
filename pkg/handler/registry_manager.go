package handler

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/x509"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	registryNameSpace    = "zcloud"
	registryAppName      = "zcloud-registry"
	registryChartName    = "harbor"
	registryChartVersion = "v1.1.1"
	registryTableName    = "global_registry"
	zcloudCaCert         = `-----BEGIN CERTIFICATE-----
	MIIC9TCCAd2gAwIBAgIRAKwOONM//IuAkwur95I+rCIwDQYJKoZIhvcNAQELBQAw
	FDESMBAGA1UEAxMJemNsb3VkLWNhMB4XDTE5MDgxNDAxMzQxOVoXDTM5MDgwOTAx
	MzQxOVowFDESMBAGA1UEAxMJemNsb3VkLWNhMIIBIjANBgkqhkiG9w0BAQEFAAOC
	AQ8AMIIBCgKCAQEAunsoVkGYPiyou6DSlJhH9VII569HLTCJk99r6lx8YqowixO+
	dU3jXk5ZV7PC8ttirBZZqiZ8Gi8XRsZdk7Po2AcppW11vCK9WBSzU1orP8lAJEVF
	7ATyVHATeGwLmpx93RuF8A/TS+fMYjHoxeui/gRWWKJ/IjGLhWgAOfpznTNI49I7
	8WqJZBoWLZ2X4Nopy2Ivr53d7Tqn8dSy9BKJTO2tVUhE0CtSu5Dpqp09/YaugKiI
	MGyopOSRacvA+o7d/lns2sZJqW783Rm9kzLcp0v4PVTSA9tRTgx1JgCz0TLvGqkn
	/vYhdyDls5Q53/afnFFNf8wXkHcfuK9Ny5xTKQIDAQABo0IwQDAOBgNVHQ8BAf8E
	BAMCAqQwHQYDVR0lBBYwFAYIKwYBBQUHAwEGCCsGAQUFBwMCMA8GA1UdEwEB/wQF
	MAMBAf8wDQYJKoZIhvcNAQELBQADggEBAIz66wt6tPA2Nw+uHENLXUnuNkKUlPZt
	jkFWFGN64XKwndsfriqdZoChvMMsmk7SxLtm4PiUmIKHU2ocSABDUumIwj5ZE8XW
	FlaPCIx7uK1Jrt6skmVKCOLDOm4bOTJ03k/BOKAmX2HYQRs55kBZYB2e11iFClEr
	eRJJp8Fv2w59vIBjGgfqG+Q7MChYhGel72lo9mu38Q7a8rOlhKAj4S/ApQ314g3m
	+REkxkwQeW6A8+M/VFUsgnPpBzTsSi6Jw9Pm7ZI165XzB31DI0cYOXj5NbWpf9NE
	Zyxq1BLChDMqGv79cc1HJnL+Hxv+6z4mQkhl0q70RdmXNO9zP7kUbWs=
	-----END CERTIFICATE-----`
	zcloudCaKey = `-----BEGIN RSA PRIVATE KEY-----
	MIIEpAIBAAKCAQEAunsoVkGYPiyou6DSlJhH9VII569HLTCJk99r6lx8YqowixO+
	dU3jXk5ZV7PC8ttirBZZqiZ8Gi8XRsZdk7Po2AcppW11vCK9WBSzU1orP8lAJEVF
	7ATyVHATeGwLmpx93RuF8A/TS+fMYjHoxeui/gRWWKJ/IjGLhWgAOfpznTNI49I7
	8WqJZBoWLZ2X4Nopy2Ivr53d7Tqn8dSy9BKJTO2tVUhE0CtSu5Dpqp09/YaugKiI
	MGyopOSRacvA+o7d/lns2sZJqW783Rm9kzLcp0v4PVTSA9tRTgx1JgCz0TLvGqkn
	/vYhdyDls5Q53/afnFFNf8wXkHcfuK9Ny5xTKQIDAQABAoIBAAf/nlRMzfXkvnwF
	wuKCwZthIGanmvryOQRxsdREkUU+HYTpnOK1K4pw+94KJNN723icIM5uhiYtXOc5
	POxH7DXP4NZqooEmUE7F3Ic3t+EthaXInt4nvCkpAXzJzZmdGrzwIEeStjJsR9Ty
	ZRSQLdaNYxK8LY3O6DgZpODXwDu+0336UOOtDeGWcgaXfCp+s3Elu4tTrFYAzkBn
	YGDbG3TCZy0khFfGCLSfsiS912NI+sCHVnaIE0IASfhb6DljsL5W2VSGR3uqXf1p
	ZCcrGoD90uIXzQVoRDuwi4e5inVUBNBxBOyyU3CRmyAN++YjRXbzTBO45ro35G/G
	NGNWEzECgYEAwJc+QwXw/6FSvvg+CkrCz22IuuUxYz/DBx/7m0l4rlC1nkfINOe/
	d0MAkt4Jrj8w+e5lx07ocB1eMfEkMR/HTJnTS5Igsish/zPw+gj0n+Fkkn0y/x1U
	6tGlPhaaadN6xhC8kRKYb6FHZR06qk5lN2iH+CpeIKIv6QR82g9YHW0CgYEA9+Dz
	J0aM+f4giL9j2v+QNa3LL6meU11P5exiLIAeJyHvY+GKfbZ67nHhDIleF1YdyNV2
	e38zu8FlaG2lagiBP8phUjNH4RVmnPa3gut7vXKLVhgLUabub5jWZHrbTF7whSkV
	5wAbBLgoCH7vSRGZF9uPgDntmtjpQpLOiFv2Yy0CgYAS5UPok4anrg5OSlDb9aXT
	cC3AGIiV8kWSR2MKQ1Uh1S1ckDJmbm5spxhBUKOmgvCtNOSrf2Ryy47YW45ve2y0
	aUs/2OB4Wp8FSPVVstc9cIHLlZkRSrFwMI2D3/fadjNPh4jYuvhVy38TvqBo4TQx
	EYJ1qMJ/dSo6NISDaIn+qQKBgQCs0c01SN7pPOB59tYrzZpBkpXi+SNFg/08lH4u
	AHUFW4eH36uq0hsLO6JoFy3en0/MwecFWz46XS/Siv+U2bEjRHpt0QsARudv8CMp
	x/xRrRawQ7tAhl4euDRhgbZ7nIWckXSPxWcQ90QSCE3UZ8yQ8acvAzRBjZGztJ8C
	OvuhUQKBgQCQQ+o0mR5TSj8GKwjqjPEsEQ/z/8I5t/9X05hYaDKD2vmBnR2ZQgaW
	Qpa42HfuY0En+yTFtFmIMLjv/5Me2PChEG7GE2AjFcJlsQtzJ8WTpUSGAdMSgq9K
	/EVs1RXRTpkSf29WP7oF2gJ30y/rEjfvbxuqZc7A+HoKIs2kac901A==
	-----END RSA PRIVATE KEY-----`
)

type RegistryManager struct {
	api.DefaultHandler
	clusters *ClusterManager
	apps     *ApplicationManager
}

func newRegistryManager(clusterMgr *ClusterManager, appMgr *ApplicationManager) *RegistryManager {
	return &RegistryManager{
		clusters: clusterMgr,
		apps:     appMgr,
	}
}

func (m *RegistryManager) Create(ctx *resttypes.Context, yaml []byte) (interface{}, *resttypes.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can create registry")
	}
	r := ctx.Object.(*types.Registry)
	cluster := m.clusters.GetClusterByName(r.Cluster)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}
	app, err := genRegistryApplication(r)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}
	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster, registryNameSpace, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create registry application failed %s", err.Error()))
	}
	r.SetID(registryAppName)
	r.SetCreationTimestamp(time.Now())
	if err := m.addToDB(r); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("add registry to db failed %s", err.Error()))
	}
	return r, nil
}

func (m *RegistryManager) List(ctx *resttypes.Context) interface{} {
	rs := []*types.Registry{}
	r, err := m.getFromDB()
	if err != nil {
		return rs
	}
	rs = append(rs, r)
	return rs
}

func (m *RegistryManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resttypes.NewAPIError(resttypes.PermissionDenied, "only admin can delete registry")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	app, err := updateApplicationStatusFromDB(m.clusters.GetDB(), registryNameSpace, registryAppName, types.AppStatusDelete)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound,
				fmt.Sprintf("registry application %s doesn't exist", registryAppName))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed,
				fmt.Sprintf("delete registry application %s failed: %s", registryAppName, err.Error()))
		}
	}
	if err := m.deleteFromDB(); err != nil {
		return resttypes.NewAPIError(types.ConnectClusterFailed,
			fmt.Sprintf("delete registry from db failed: %s", err.Error()))
	}
	go deleteApplication(m.clusters.GetDB(), cluster.KubeClient, genAppTableName(cluster.Name, registryNameSpace), registryNameSpace, app)
	return nil
}

func genRegistryApplication(r *types.Registry) (*types.Application, error) {
	config, err := genRegistryConfigs(r)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         registryAppName,
		ChartName:    registryChartName,
		ChartVersion: registryChartVersion,
		Configs:      config,
	}, nil
}

func genRegistryConfigs(r *types.Registry) ([]byte, error) {
	ca := x509.Certificate{
		Cert: zcloudCaCert,
		Key:  zcloudCaKey,
	}

	tls, err := x509.GenerateSignedCertificate(r.IngressDomain, nil, []interface{}{r.IngressDomain}, 7300, ca)
	if err != nil {
		return nil, err
	}
	harbor := charts.Harbor{
		IngressDomain:       r.IngressDomain,
		StorageClass:        r.StorageClass,
		RegistryStorageSize: strconv.Itoa(r.StorageSize) + "Gi",
		AdminPassword:       r.AdminPassword,
		CaCert:              zcloudCaCert,
		TlsCert:             tls.Cert,
		TlsKey:              tls.Key,
		ExternalURL:         "https://" + r.IngressDomain,
	}
	content, err := json.Marshal(&harbor)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func (m *RegistryManager) addToDB(r *types.Registry) error {
	value, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("marshal registry %s failed: %s", registryAppName, err.Error())
	}

	tx, err := BeginTableTransaction(m.clusters.GetDB(), registryTableName)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Add(registryAppName, value); err != nil {
		return err
	}
	return tx.Commit()
}

func (m *RegistryManager) getFromDB() (*types.Registry, error) {
	r := &types.Registry{}
	tx, err := BeginTableTransaction(m.clusters.GetDB(), registryTableName)
	if err != nil {
		return r, err
	}
	defer tx.Commit()

	value, err := tx.Get(registryAppName)
	if err != nil {
		return r, err
	}

	err = json.Unmarshal(value, r)
	return r, err
}

func (m *RegistryManager) deleteFromDB() error {
	tx, err := BeginTableTransaction(m.clusters.GetDB(), registryTableName)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if err := tx.Delete(registryAppName); err != nil {
		return err
	}
	return tx.Commit()
}
