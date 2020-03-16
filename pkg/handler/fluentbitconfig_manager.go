package handler

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"text/template"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/iniconfig"
	eb "github.com/zdnscloud/singlecloud/pkg/eventbus"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	FluentBitConfigMapPrefix       = "efk"
	FluentBitConfigMapSuffix       = "fluent-bit-config"
	FluentBitConfigFileName        = "fluent-bit.conf"
	FluentBitConfigParsersFileName = "parsers.conf"
	ElasticsearchSvcName           = "elasticsearch-master"
	ElasticsearchSvcPort           = "9200"
)

var (
	ParsersFileNotFoundErr   = errors.New(fmt.Sprintf("%s can not find in fluent-bit configmap", FluentBitConfigParsersFileName))
	FluentBitFileNotFoundErr = errors.New(fmt.Sprintf("%s can not find in fluent-bit configmap", FluentBitConfigFileName))
)

const (
	InstanceConfTemp = `[INPUT]
    Name              tail
    Tag               InstanceName
    Path              /var/log/containers/LogName*.log
    Parser            docker
    DB                /var/log/flb_DBPath.db
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10
[FILTER]
    Name                parser
    Match               InstanceName
    Parser              InstanceName
    Key_Name            log
    Reserve_Data        On
[OUTPUT]
    Name            es
    Match           InstanceName
    Host            ElasticsearchSvcName
    Port            ElasticsearchSvcPort
    Logstash_Format On
    Replace_Dots    On
    Retry_Limit     False
    Logstash_Prefix InstanceName`
	ParserTemp = `
[PARSER]
    Name        {{.Name}}
    Format      regex
    Regex       {{.Regex}}
    {{- if .TimeKey}}
    Time_Key    {{.TimeKey}}
    Time_Format {{.TimeFormat}}
    {{- end}}`
)

type FluentBitConfigManager struct {
	clusters *ClusterManager
}

func newFluentBitConfigManager(clusters *ClusterManager) *FluentBitConfigManager {
	mgr := &FluentBitConfigManager{clusters: clusters}
	go mgr.eventLoop()
	return mgr
}

func (m *FluentBitConfigManager) eventLoop() {
	eventCh := eb.SubscribeResourceEvent(
		types.Namespace{},
		types.Deployment{},
		types.DaemonSet{},
		types.StatefulSet{})
	for {
		event := <-eventCh
		switch e := event.(type) {
		case eb.ResourceDeleteEvent:
			switch r := e.Resource.(type) {
			case *types.Namespace:
				cluster := m.clusters.GetClusterForSubResource(r)
				if cluster == nil {
					log.Warnf("get cluster nil for namespace %s", r.GetID())
					continue
				}
				go deleteNamespaceFluentBitConfig(cluster.GetKubeClient(), r.GetID())
			case *types.Deployment, *types.DaemonSet, *types.StatefulSet:
				namespace := r.GetParent().GetID()
				name := namespace + "_" + r.GetType() + "_" + r.GetID()
				cluster := m.clusters.GetClusterForSubResource(r)
				if cluster == nil {
					log.Warnf("get cluster nil for workload %s", name)
					continue
				}
				go deleteWorkLoadFluentBitConfig(cluster.GetKubeClient(), name)
			}
		}
	}
}

func (m *FluentBitConfigManager) Create(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster s doesn't exist")
	}
	conf := ctx.Resource.(*types.FluentBitConfig)
	replenishConf(ctx, conf)

	cm, err := getFluentBitConfigMap(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found fluent-bit config configmap")
		}
		return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create fluent-bit config failed. %s", err.Error()))
	}
	if err := createConfig(cluster.GetKubeClient(), conf, cm); err != nil {
		return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create fluent-bit config failed. %s", err.Error()))
	}
	conf.SetID(conf.Name)
	return conf, nil
}

func (m *FluentBitConfigManager) Update(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster s doesn't exist")
	}
	conf := ctx.Resource.(*types.FluentBitConfig)
	replenishConf(ctx, conf)

	cm, err := getFluentBitConfigMap(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found fluent-bit config configmap")
		}
		return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update fluent-bit config %s failed. %s", conf.GetID(), err.Error()))
	}
	if err := updateConfig(cluster.GetKubeClient(), conf, cm); err != nil {
		return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update fluent-bit config %s failed. %s", conf.GetID(), err.Error()))
	}
	return conf, nil
}

func (m *FluentBitConfigManager) List(ctx *resource.Context) (interface{}, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Resource.GetParent().GetParent().GetID()
	ownerType := ctx.Resource.GetParent().GetType()
	ownerName := ctx.Resource.GetParent().GetID()
	name := namespace + "_" + ownerType + "_" + ownerName

	cm, err := getFluentBitConfigMap(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found fluent-bit config configmap")
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get fluent-bit configs failed %s", err.Error()))
	}
	fbConf, err := getConfig(cluster.GetKubeClient(), name, cm)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get fluent-bit configs failed %s", err.Error()))
	}
	return []*types.FluentBitConfig{fbConf}, nil
}

func (m FluentBitConfigManager) Get(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	conf := ctx.Resource.(*types.FluentBitConfig)
	replenishConf(ctx, conf)

	cm, err := getFluentBitConfigMap(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found fluent-bit config configmap")
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get fluent-bit config %s failed %s", conf.GetID(), err.Error()))
	}
	fbConf, err := getConfig(cluster.GetKubeClient(), conf.GetID(), cm)
	if err != nil {
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get fluent-bit config %s failed %s", conf.GetID(), err.Error()))
	}
	return fbConf, nil
}

func (m FluentBitConfigManager) Delete(ctx *resource.Context) *resterr.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}
	conf := ctx.Resource.(*types.FluentBitConfig)
	cm, err := getFluentBitConfigMap(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resterr.NewAPIError(resterr.NotFound, "no found fluent-bit config configmap")
		}
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("delete fluent-bit config %s. failed %s", conf.GetID(), err.Error()))
	}
	if err := deleteConfig(cluster.GetKubeClient(), conf.GetID(), cm); err != nil {
		return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("delete fluent-bit config %s. failed %s", conf.GetID(), err.Error()))
	}
	return nil
}

func replenishConf(ctx *resource.Context, conf *types.FluentBitConfig) {
	conf.Namespace = ctx.Resource.GetParent().GetParent().GetID()
	conf.OwnerType = ctx.Resource.GetParent().GetType()
	conf.OwnerName = ctx.Resource.GetParent().GetID()
}

func getFluentBitConfigMap(cli client.Client) (*corev1.ConfigMap, error) {
	cms := corev1.ConfigMapList{}
	if err := cli.List(context.TODO(), &client.ListOptions{Namespace: ZCloudNamespace}, &cms); err != nil {
		return nil, fmt.Errorf("get fluent-bit configmap failed, %v", err)
	}
	for _, cm := range cms.Items {
		if strings.HasPrefix(cm.Name, FluentBitConfigMapPrefix) && strings.HasSuffix(cm.Name, FluentBitConfigMapSuffix) {
			return &cm, nil
		}
	}
	return nil, errors.New("get fluent-bit configmap none, please check if EFK has been successfully deployed")
}

func isFluentBitConfigExist(cli client.Client, conf *types.FluentBitConfig, cm *corev1.ConfigMap) bool {
	instances := getInstances(cli, cm)
	if index := slice.SliceIndex(instances, conf.Name); index >= 0 {
		return true
	}
	return false
}

func createConfig(cli client.Client, conf *types.FluentBitConfig, cm *corev1.ConfigMap) error {
	if conf.RegExp == "" {
		return errors.New(fmt.Sprintf("regexp can not be null"))
	}
	conf.Name = conf.Namespace + "_" + conf.OwnerType + "_" + conf.OwnerName
	if isFluentBitConfigExist(cli, conf, cm) {
		return errors.New(fmt.Sprintf("fluent-bit config for %s: %s in namespace %s has already exists", conf.OwnerType, conf.OwnerName, conf.Namespace))
	}

	oldFbConf, ok := cm.Data[FluentBitConfigFileName]
	if !ok {
		return FluentBitFileNotFoundErr
	}
	cm.Data[FluentBitConfigFileName] = updateFluentBitFile(oldFbConf, conf.Name, "add")

	oldPaConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if !ok {
		return ParsersFileNotFoundErr
	}
	newPaConf, err := updateParserFile(conf, oldPaConf, "add")
	if err != nil {
		return fmt.Errorf("update %s in fluent-bit configmap failed, %v", FluentBitConfigParsersFileName, err)
	}
	cm.Data[FluentBitConfigParsersFileName] = newPaConf

	instanceFileName := conf.Name + ".conf"
	cm.Data[instanceFileName] = genInstanceFile(conf)
	return cli.Update(context.TODO(), cm)
}

func getConfig(cli client.Client, name string, cm *corev1.ConfigMap) (*types.FluentBitConfig, error) {
	parserConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if !ok {
		return nil, ParsersFileNotFoundErr
	}
	confs, err := genFluentBitConfigs(parserConf)
	if err != nil {
		return nil, err
	}
	conf := &types.FluentBitConfig{
		Name: name,
	}
	c, ok := confs[name]
	if ok {
		conf.RegExp = c.RegExp
		conf.Time_Key = c.Time_Key
		conf.Time_Format = c.Time_Format
	}
	conf.SetID(name)
	return conf, nil
}

func deleteConfig(cli client.Client, name string, cm *corev1.ConfigMap) error {
	conf, err := getConfig(cli, name, cm)
	if err != nil {
		return fmt.Errorf("get fluent-bit config %s failed, %v", name, err)
	}
	oldFbConf, ok := cm.Data[FluentBitConfigFileName]
	if !ok {
		return FluentBitFileNotFoundErr
	}
	cm.Data[FluentBitConfigFileName] = updateFluentBitFile(oldFbConf, name, "del")

	oldPaConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if !ok {
		return ParsersFileNotFoundErr
	}
	newPaConf, err := updateParserFile(conf, oldPaConf, "del")
	if err != nil {
		return fmt.Errorf("update parsers conf failed, %v", err)
	}
	cm.Data[FluentBitConfigParsersFileName] = newPaConf

	instanceFileName := name + ".conf"
	delete(cm.Data, instanceFileName)
	return cli.Update(context.TODO(), cm)
}

func updateConfig(cli client.Client, conf *types.FluentBitConfig, cm *corev1.ConfigMap) error {
	if conf.RegExp == "" {
		return errors.New(fmt.Sprintf("regexp can not be null"))
	}
	if conf.Name == "" {
		conf.Name = conf.GetID()
	} else {
		if conf.Name != conf.GetID() {
			return errors.New("fluent-bit config field name not allowed to update")
		}
	}
	oldConf, err := getConfig(cli, conf.Name, cm)
	if err != nil {
		return fmt.Errorf("get fluent-bit config failed, %v", err)
	}
	if oldConf.RegExp != conf.RegExp || oldConf.Time_Key != conf.Time_Key || oldConf.Time_Format != conf.Time_Format {
		oldPaConf, ok := cm.Data[FluentBitConfigParsersFileName]
		if !ok {
			return errors.New(fmt.Sprintf("%s can not find in fluent-bit config", FluentBitConfigParsersFileName))
		}
		afterPaConf, err := updateParserFile(oldConf, oldPaConf, "del")
		if err != nil {
			return fmt.Errorf("gen parsers conf failed, %v", err)
		}
		newPaConf, err := updateParserFile(conf, afterPaConf, "add")
		if err != nil {
			return fmt.Errorf("gen parsers conf failed, %v", err)
		}
		cm.Data[FluentBitConfigParsersFileName] = newPaConf
	}
	return cli.Update(context.TODO(), cm)
}

func updateFluentBitFile(oldFbConf, name, action string) string {
	var result string
	include := "@INCLUDE " + name + ".conf" + "\n"
	switch action {
	case "add":
		buf := bytes.NewBufferString(oldFbConf)
		buf.WriteString(include)
		result = buf.String()
	case "del":
		result = strings.Replace(oldFbConf, include, "", -1)
	}
	return result
}

func updateParserFile(conf *types.FluentBitConfig, oldPaConf, action string) (string, error) {
	var result string
	newPaConf, err := genParserConf(conf)
	if err != nil {
		return "", fmt.Errorf("gen parsers conf failed, %v", err)
	}
	switch action {
	case "add":
		buf := bytes.NewBufferString(oldPaConf)
		strs := strings.Split(newPaConf, "\n")
		for _, str := range strs {
			buf.WriteString(strings.TrimRight(str, " ") + "\n")
		}
		result = buf.String()
	case "del":
		result = strings.Replace(oldPaConf, newPaConf+"\n", "", -1)
	}
	return result, nil
}

func genInstanceFile(conf *types.FluentBitConfig) string {
	LogName := conf.OwnerName + "*_" + conf.Namespace + "_"
	DBPath := conf.Namespace + "_" + conf.OwnerType + "_" + conf.OwnerName
	instanceConf := InstanceConfTemp
	instanceConf = strings.Replace(instanceConf, "InstanceName", conf.Name, -1)
	instanceConf = strings.Replace(instanceConf, "LogName", LogName, -1)
	instanceConf = strings.Replace(instanceConf, "DBPath", DBPath, -1)
	instanceConf = strings.Replace(instanceConf, "ElasticsearchSvcName", ElasticsearchSvcName, -1)
	instanceConf = strings.Replace(instanceConf, "ElasticsearchSvcPort", ElasticsearchSvcPort, -1)
	return instanceConf
}

func genParserConf(conf *types.FluentBitConfig) (string, error) {
	cfg := map[string]interface{}{
		"Name":       conf.Name,
		"Regex":      conf.RegExp,
		"TimeKey":    conf.Time_Key,
		"TimeFormat": conf.Time_Format,
	}
	return CompileTemplateFromMap(ParserTemp, cfg)
}

func getInstances(cli client.Client, cm *corev1.ConfigMap) []string {
	var instances []string
	for _, line := range strings.Split(cm.Data[FluentBitConfigFileName], "\n") {
		if strings.HasPrefix(line, "@INCLUDE") && strings.HasSuffix(line, ".conf") {
			instances = append(instances, strings.Split(strings.Fields(line)[1], ".")[0])
		}
	}
	return instances
}

func genFluentBitConfigs(paConf string) (map[string]*types.FluentBitConfig, error) {
	confs := make(map[string]*types.FluentBitConfig)
	section := "[PARSER]"
	ps := strings.Split(string(paConf), section)
	for _, p := range ps {
		if len(p) == 0 {
			continue
		}
		conf, err := ForMat(strings.Trim(fmt.Sprint(section), "[]"), section+p)
		if err != nil {
			return nil, fmt.Errorf("parse iniconfig failed. %v", err)
		}
		if name, ok := conf["Name"]; ok {
			confs[name] = &types.FluentBitConfig{
				RegExp:      conf["Regex"],
				Time_Key:    conf["Time_Key"],
				Time_Format: conf["Time_Format"],
			}
		}
	}
	return confs, nil
}

func ForMat(section, data string) (map[string]string, error) {
	TOPIC := make(map[string]string)
	cfg := iniconfig.NewDefault()
	if err := cfg.Read(bufio.NewReader(strings.NewReader(data))); err != nil {
		return TOPIC, fmt.Errorf("Iniconfig read string failed, %v", err)
	}
	if !cfg.HasSection(section) {
		return TOPIC, errors.New(fmt.Sprintf("Iniconfig can not find section %s", section))
	}
	iniconf, err := cfg.SectionOptions(section)
	if err != nil {
		return TOPIC, fmt.Errorf("Iniconfig section options failed, %v", err)
	}
	for _, v := range iniconf {
		options, err := cfg.String(section, v)
		if err != nil {
			return TOPIC, fmt.Errorf("Iniconfig string failed, %v", err)
		}
		TOPIC[v] = options
	}
	return TOPIC, nil
}

func CompileTemplateFromMap(tmplt string, configMap interface{}) (string, error) {
	out := new(bytes.Buffer)
	t := template.Must(template.New("compiled_template").Parse(tmplt))
	if err := t.Execute(out, configMap); err != nil {
		return "", fmt.Errorf("CompileTemplate failed, %v", err)
	}
	return out.String(), nil
}

func getFluentBitConfigsAndConfigMap(cli client.Client) (map[string]*types.FluentBitConfig, *corev1.ConfigMap, error) {
	cm, err := getFluentBitConfigMap(cli)
	if err != nil {
		return nil, nil, err
	}
	parserConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if !ok {
		return nil, nil, ParsersFileNotFoundErr
	}
	confs, err := genFluentBitConfigs(parserConf)
	if err != nil {
		return nil, nil, err
	}
	return confs, cm, nil
}

func deleteAllFluentBitConfig(cli client.Client) {
	confs, cm, err := getFluentBitConfigsAndConfigMap(cli)
	if err != nil {
		log.Warnf("get fluent-bit config and configmap failed: %s", err.Error())
		return
	}
	namespaces, err := getNamespaces(cli)
	if err != nil {
		log.Warnf("list namespace failed: %s", err.Error())
		return
	}
	for _, namespace := range namespaces.Items {
		if err := DelFBCForNamespace(cli, namespace.Name, confs, cm); err != nil {
			log.Warnf("delete fluent-bit config for namespace %s failed: %s", namespace.Name, err.Error())
		}
	}
	return
}

func deleteNamespaceFluentBitConfig(cli client.Client, namespace string) {
	confs, cm, err := getFluentBitConfigsAndConfigMap(cli)
	if err != nil {
		log.Warnf("get fluent-bit config and configmap failed: %s", err.Error())
		return
	}
	if err := DelFBCForNamespace(cli, namespace, confs, cm); err != nil {
		log.Warnf("delete fluent-bit config for namespace %s failed: %s", namespace, err.Error())
	}
	return
}

func DelFBCForNamespace(cli client.Client, namespace string, confs map[string]*types.FluentBitConfig, cm *corev1.ConfigMap) error {
	for name := range confs {
		if strings.HasPrefix(name, namespace+"_") {
			if err := deleteConfig(cli, name, cm); err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteWorkLoadFluentBitConfig(cli client.Client, name string) {
	cm, err := getFluentBitConfigMap(cli)
	if err != nil {
		log.Warnf("get fluent-bit config and configmap failed: %s", err.Error())
		return
	}
	if err := deleteConfig(cli, name, cm); err != nil {
		log.Warnf("delete fluent-bit config %s failed: %s", name, err.Error())
	}
	return
}
