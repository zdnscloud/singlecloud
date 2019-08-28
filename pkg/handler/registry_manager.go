package handler

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/charts"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/storage"

	"github.com/zdnscloud/cement/x509"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
)

const (
	registryNameSpace        = "zcloud"
	registryAppName          = "zcloud-registry"
	registryChartName        = "harbor"
	registryChartVersion     = "v1.1.1"
	registryTableName        = "global_registry"
	registryAppStorageClass  = "lvm"
	registryAppStorageSize   = "50Gi"
	registryAppAdminPassword = "zcloud"
	zcloudCaCertB64          = `LS0tLS1CRUdJTiBDRVJUSUZJQ0FURS0tLS0tCk1JSUM5VENDQWQyZ0F3SUJBZ0lSQUt3T09OTS8vSXVBa3d1cjk1SStyQ0l3RFFZSktvWklodmNOQVFFTEJRQXcKRkRFU01CQUdBMVVFQXhNSmVtTnNiM1ZrTFdOaE1CNFhEVEU1TURneE5EQXhNelF4T1ZvWERUTTVNRGd3T1RBeApNelF4T1Zvd0ZERVNNQkFHQTFVRUF4TUplbU5zYjNWa0xXTmhNSUlCSWpBTkJna3Foa2lHOXcwQkFRRUZBQU9DCkFROEFNSUlCQ2dLQ0FRRUF1bnNvVmtHWVBpeW91NkRTbEpoSDlWSUk1NjlITFRDSms5OXI2bHg4WXFvd2l4TysKZFUzalhrNVpWN1BDOHR0aXJCWlpxaVo4R2k4WFJzWmRrN1BvMkFjcHBXMTF2Q0s5V0JTelUxb3JQOGxBSkVWRgo3QVR5VkhBVGVHd0xtcHg5M1J1RjhBL1RTK2ZNWWpIb3hldWkvZ1JXV0tKL0lqR0xoV2dBT2Zwem5UTkk0OUk3CjhXcUpaQm9XTFoyWDROb3B5Mkl2cjUzZDdUcW44ZFN5OUJLSlRPMnRWVWhFMEN0U3U1RHBxcDA5L1lhdWdLaUkKTUd5b3BPU1JhY3ZBK283ZC9sbnMyc1pKcVc3ODNSbTlrekxjcDB2NFBWVFNBOXRSVGd4MUpnQ3owVEx2R3FrbgovdlloZHlEbHM1UTUzL2FmbkZGTmY4d1hrSGNmdUs5Tnk1eFRLUUlEQVFBQm8wSXdRREFPQmdOVkhROEJBZjhFCkJBTUNBcVF3SFFZRFZSMGxCQll3RkFZSUt3WUJCUVVIQXdFR0NDc0dBUVVGQndNQ01BOEdBMVVkRXdFQi93UUYKTUFNQkFmOHdEUVlKS29aSWh2Y05BUUVMQlFBRGdnRUJBSXo2Nnd0NnRQQTJOdyt1SEVOTFhVbnVOa0tVbFBadApqa0ZXRkdONjRYS3duZHNmcmlxZFpvQ2h2TU1zbWs3U3hMdG00UGlVbUlLSFUyb2NTQUJEVXVtSXdqNVpFOFhXCkZsYVBDSXg3dUsxSnJ0NnNrbVZLQ09MRE9tNGJPVEowM2svQk9LQW1YMkhZUVJzNTVrQlpZQjJlMTFpRkNsRXIKZVJKSnA4RnYydzU5dklCakdnZnFHK1E3TUNoWWhHZWw3MmxvOW11MzhRN2E4ck9saEtBajRTL0FwUTMxNGczbQorUkVreGt3UWVXNkE4K00vVkZVc2duUHBCelRzU2k2Snc5UG03WkkxNjVYekIzMURJMGNZT1hqNU5iV3BmOU5FClp5eHExQkxDaERNcUd2NzljYzFISm5MK0h4dis2ejRtUWtobDBxNzBSZG1YTk85elA3a1ViV3M9Ci0tLS0tRU5EIENFUlRJRklDQVRFLS0tLS0K`
	zcloudCaKeyB64           = `LS0tLS1CRUdJTiBSU0EgUFJJVkFURSBLRVktLS0tLQpNSUlFcEFJQkFBS0NBUUVBdW5zb1ZrR1lQaXlvdTZEU2xKaEg5VklJNTY5SExUQ0prOTlyNmx4OFlxb3dpeE8rCmRVM2pYazVaVjdQQzh0dGlyQlpacWlaOEdpOFhSc1pkazdQbzJBY3BwVzExdkNLOVdCU3pVMW9yUDhsQUpFVkYKN0FUeVZIQVRlR3dMbXB4OTNSdUY4QS9UUytmTVlqSG94ZXVpL2dSV1dLSi9JakdMaFdnQU9mcHpuVE5JNDlJNwo4V3FKWkJvV0xaMlg0Tm9weTJJdnI1M2Q3VHFuOGRTeTlCS0pUTzJ0VlVoRTBDdFN1NURwcXAwOS9ZYXVnS2lJCk1HeW9wT1NSYWN2QStvN2QvbG5zMnNaSnFXNzgzUm05a3pMY3AwdjRQVlRTQTl0UlRneDFKZ0N6MFRMdkdxa24KL3ZZaGR5RGxzNVE1My9hZm5GRk5mOHdYa0hjZnVLOU55NXhUS1FJREFRQUJBb0lCQUFmL25sUk16Zlhrdm53Rgp3dUtDd1p0aElHYW5tdnJ5T1FSeHNkUkVrVVUrSFlUcG5PSzFLNHB3Kzk0S0pOTjcyM2ljSU01dWhpWXRYT2M1ClBPeEg3RFhQNE5acW9vRW1VRTdGM0ljM3QrRXRoYVhJbnQ0bnZDa3BBWHpKelptZEdyendJRWVTdGpKc1I5VHkKWlJTUUxkYU5ZeEs4TFkzTzZEZ1pwT0RYd0R1KzAzMzZVT090RGVHV2NnYVhmQ3ArczNFbHU0dFRyRllBemtCbgpZR0RiRzNUQ1p5MGtoRmZHQ0xTZnNpUzkxMk5JK3NDSFZuYUlFMElBU2ZoYjZEbGpzTDVXMlZTR1IzdXFYZjFwClpDY3JHb0Q5MHVJWHpRVm9SRHV3aTRlNWluVlVCTkJ4Qk95eVUzQ1JteUFOKytZalJYYnpUQk80NXJvMzVHL0cKTkdOV0V6RUNnWUVBd0pjK1F3WHcvNkZTdnZnK0NrckN6MjJJdXVVeFl6L0RCeC83bTBsNHJsQzFua2ZJTk9lLwpkME1Ba3Q0SnJqOHcrZTVseDA3b2NCMWVNZkVrTVIvSFRKblRTNUlnc2lzaC96UHcrZ2owbitGa2tuMHkveDFVCjZ0R2xQaGFhYWRONnhoQzhrUktZYjZGSFpSMDZxazVsTjJpSCtDcGVJS0l2NlFSODJnOVlIVzBDZ1lFQTkrRHoKSjBhTStmNGdpTDlqMnYrUU5hM0xMNm1lVTExUDVleGlMSUFlSnlIdlkrR0tmYlo2N25IaERJbGVGMVlkeU5WMgplMzh6dThGbGFHMmxhZ2lCUDhwaFVqTkg0UlZtblBhM2d1dDd2WEtMVmhnTFVhYnViNWpXWkhyYlRGN3doU2tWCjV3QWJCTGdvQ0g3dlNSR1pGOXVQZ0RudG10anBRcExPaUZ2Mll5MENnWUFTNVVQb2s0YW5yZzVPU2xEYjlhWFQKY0MzQUdJaVY4a1dTUjJNS1ExVWgxUzFja0RKbWJtNXNweGhCVUtPbWd2Q3ROT1NyZjJSeXk0N1lXNDV2ZTJ5MAphVXMvMk9CNFdwOEZTUFZWc3RjOWNJSExsWmtSU3JGd01JMkQzL2ZhZGpOUGg0all1dmhWeTM4VHZxQm80VFF4CkVZSjFxTUovZFNvNk5JU0RhSW4rcVFLQmdRQ3MwYzAxU043cFBPQjU5dFlyelpwQmtwWGkrU05GZy8wOGxINHUKQUhVRlc0ZUgzNnVxMGhzTE82Sm9GeTNlbjAvTXdlY0ZXejQ2WFMvU2l2K1UyYkVqUkhwdDBRc0FSdWR2OENNcAp4L3hSclJhd1E3dEFobDRldURSaGdiWjduSVdja1hTUHhXY1E5MFFTQ0UzVVo4eVE4YWN2QXpSQmpaR3p0SjhDCk92dWhVUUtCZ1FDUVErbzBtUjVUU2o4R0t3anFqUEVzRVEvei84STV0LzlYMDVoWWFES0Qydm1CblIyWlFnYVcKUXBhNDJIZnVZMEVuK3lURnRGbUlNTGp2LzVNZTJQQ2hFRzdHRTJBakZjSmxzUXR6SjhXVHBVU0dBZE1TZ3E5SwovRVZzMVJYUlRwa1NmMjlXUDdvRjJnSjMweS9yRWpmdmJ4dXFaYzdBK0hvS0lzMmthYzkwMUE9PQotLS0tLUVORCBSU0EgUFJJVkFURSBLRVktLS0tLQo=`
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

	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	existRegistry, err := getApplicationFromDB(m.clusters.GetDB(), genAppTableName(cluster.Name, registryNameSpace), registryAppName)
	if err != nil && err != storage.ErrNotFoundResource {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("get registry application from db failed %s", err.Error()))
	}

	if existRegistry != nil {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("cluster %s registry has exist", cluster.Name))
	}

	requiredStorageClass := monitorAppStorageClass
	if r.StorageClass != "" {
		requiredStorageClass = r.StorageClass
	}

	if !isStorageClassExist(cluster.KubeClient, requiredStorageClass) {
		return nil, resttypes.NewAPIError(resttypes.PermissionDenied, fmt.Sprintf("%s storageclass does't exist in cluster %s", requiredStorageClass, cluster.Name))
	}

	app, err := genRegistryApplication(cluster.KubeClient, r, cluster.Name)
	if err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, err.Error())
	}

	app.SetID(app.Name)
	if err := m.apps.create(ctx, cluster, registryNameSpace, app); err != nil {
		return nil, resttypes.NewAPIError(resttypes.ServerError, fmt.Sprintf("create registry application failed %s", err.Error()))
	}
	r.SetID(registryAppName)
	r.SetCreationTimestamp(time.Now())
	r.ApplicationLink = genRegistryAppLink(ctx, cluster.Name)
	return r, nil
}

func (m *RegistryManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	rs := []*types.Registry{}

	app, err := getApplicationFromDB(m.clusters.GetDB(), genAppTableName(cluster.Name, registryNameSpace), registryAppName)
	if err != nil {
		return rs
	}

	r, err := genRegistryFromApp(ctx, cluster.Name, app)
	if err != nil {
		return rs
	}

	rs = append(rs, r)
	return rs
}

func genRegistryApplication(cli client.Client, r *types.Registry, clusterName string) (*types.Application, error) {
	config, err := genRegistryConfigs(cli, r, clusterName)
	if err != nil {
		return nil, err
	}
	return &types.Application{
		Name:         registryAppName,
		ChartName:    registryChartName,
		ChartVersion: registryChartVersion,
		Configs:      config,
		SystemChart:  true,
	}, nil
}

func genRegistryConfigs(cli client.Client, r *types.Registry, clusterName string) ([]byte, error) {
	if len(r.IngressDomain) == 0 {
		firstEdgeNodeIP := getFirstEdgeNodeAddress(cli)
		if len(firstEdgeNodeIP) == 0 {
			return nil, fmt.Errorf("can not find edge node for this cluster")
		}
		r.IngressDomain = registryAppName + "-" + registryNameSpace + "-" + clusterName + "." + firstEdgeNodeIP + "." + zcloudDynamicalDnsPrefix
	}
	r.RedirectUrl = "https://" + r.IngressDomain

	caCrtBytes, _ := base64.StdEncoding.DecodeString(zcloudCaCertB64)
	caKeyBytes, _ := base64.StdEncoding.DecodeString(zcloudCaKeyB64)
	ca := x509.Certificate{
		Cert: string(caCrtBytes),
		Key:  string(caKeyBytes),
	}

	tls, err := x509.GenerateSignedCertificate(r.IngressDomain, nil, []interface{}{r.IngressDomain}, 7300, ca)
	if err != nil {
		return nil, err
	}

	harbor := charts.Harbor{
		Ingress: charts.HarborIngress{
			Core:   r.IngressDomain,
			CaCrt:  ca.Cert,
			TlsCrt: tls.Cert,
			TlsKey: tls.Key,
		},
		Persistence: charts.HarborPersistence{
			StorageClass: registryAppStorageClass,
			StorageSize:  registryAppStorageSize,
		},
		AdminPassword: registryAppAdminPassword,
		ExternalURL:   "https://" + r.IngressDomain,
	}

	if len(r.StorageClass) > 0 {
		harbor.Persistence.StorageClass = r.StorageClass
	}
	if r.StorageSize > 0 {
		harbor.Persistence.StorageSize = strconv.Itoa(r.StorageSize) + "Gi"
	}
	if len(r.AdminPassword) > 0 {
		harbor.AdminPassword = r.AdminPassword
	}
	content, err := json.Marshal(&harbor)
	if err != nil {
		return nil, err
	}
	return content, nil
}

func genRegistryFromApp(ctx *resttypes.Context, cluster string, app *types.Application) (*types.Registry, error) {
	h := charts.Harbor{}
	if err := json.Unmarshal(app.Configs, &h); err != nil {
		return nil, err
	}
	r := types.Registry{
		IngressDomain:   h.Ingress.Core,
		StorageClass:    h.Persistence.StorageClass,
		RedirectUrl:     "https://" + h.Ingress.Core,
		ApplicationLink: genRegistryAppLink(ctx, cluster),
	}
	r.SetID(app.Name)
	r.CreationTimestamp = app.CreationTimestamp
	return &r, nil
}

func genRegistryAppLink(ctx *resttypes.Context, clusterName string) string {
	return genUrlPrefix(ctx, clusterName) + "/" + registryNameSpace + "/applications/" + registryAppName
}
