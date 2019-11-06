package handler

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"path"
	"reflect"
	"strings"
	"text/template"

	corev1 "k8s.io/api/core/v1"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	resterror "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/iniconfig"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

const (
	FluentBitConfigMapNamespace    = "zcloud"
	FluentBitConfigMapPrefix       = "efk"
	FluentBitConfigMapSuffix       = "fluent-bit-config"
	FluentBitConfigFileName        = "fluent-bit.conf"
	FluentBitConfigParsersFileName = "parsers.conf"
	ElasticsearchSvcName           = "elasticsearch-master"
	ElasticsearchSvcPort           = "9200"
)

const (
	InstanceConfTemp = `[INPUT]
    Name              tail
    Tag               InstanceName.*
    Path              /var/log/containers/LogName*.log
    Parser            docker
    DB                /var/log/flb_DBPath.db
    Mem_Buf_Limit     5MB
    Skip_Long_Lines   On
    Refresh_Interval  10
[FILTER]
    Name                parser
    Match               InstanceName.*
    Parser              InstanceName
    Key_Name            log
    Reserve_Data        On
[OUTPUT]
    Name            es
    Match           InstanceName.*
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
	return &FluentBitConfigManager{clusters: clusters}
}

func (m *FluentBitConfigManager) Create(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	conf := ctx.Resource.(*types.FluentBitConfig)
	cm, err := getFluentBitConfigMap(cluster.KubeClient)
	if err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create fluent-bit config failed. %s", err.Error()))
	}
	if err := createConfig(cluster.KubeClient, conf, cm); err != nil {
		return nil, resterror.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create fluent-bit config failed. %s", err.Error()))
	}
	conf.SetID(conf.Name)
	return conf, nil
}

func (m *FluentBitConfigManager) Update(ctx *resource.Context) (resource.Resource, *resterror.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterror.NewAPIError(resterror.NotFound, "cluster s doesn't exist")
	}

	conf := ctx.Resource.(*types.FluentBitConfig)
	cm, err := getFluentBitConfigMap(cluster.KubeClient)
	if err != nil {
		return nil, resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update fluent-bit config %s failed. %s", conf.GetID(), err.Error()))
	}
	if err := updateConfig(cluster.KubeClient, conf, cm); err != nil {
		return nil, resterror.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("update fluent-bit config %s failed. %s", conf.GetID(), err.Error()))
	}
	return conf, nil
}

func (m *FluentBitConfigManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}

	cm, err := getFluentBitConfigMap(cluster.KubeClient)
	if err != nil {
		log.Warnf("list fluent-bit config failed %s", err.Error())
	}
	fbConfs, err := getConfigs(cluster.KubeClient, cm)
	if err != nil {
		log.Warnf("list fluent-bit config failed %s", err.Error())
	}
	return fbConfs
}

func (m FluentBitConfigManager) Get(ctx *resource.Context) resource.Resource {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil
	}
	conf := ctx.Resource.(*types.FluentBitConfig)
	cm, err := getFluentBitConfigMap(cluster.KubeClient)
	if err != nil {
		log.Warnf("get fluent-bit config %s failed %s", conf.GetID(), err.Error())
	}
	fbConf, err := getConfig(cluster.KubeClient, conf.GetID(), cm)
	if err != nil {
		log.Warnf("get fluent-bit config %s failed %s", conf.GetID(), err.Error())
	}
	return fbConf
}

func (m FluentBitConfigManager) Delete(ctx *resource.Context) *resterror.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterror.NewAPIError(resterror.NotFound, "cluster doesn't exist")
	}
	conf := ctx.Resource.(*types.FluentBitConfig)
	cm, err := getFluentBitConfigMap(cluster.KubeClient)
	if err != nil {
		return resterror.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete fluent-bit config %s. failed %s", conf.GetID(), err.Error()))
	}
	if err := deleteConfig(cluster.KubeClient, conf.GetID(), cm); err != nil {
		return resterror.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("delete fluent-bit config %s. failed %s", conf.GetID(), err.Error()))
	}
	return nil
}

func getFluentBitConfigMap(cli client.Client) (corev1.ConfigMap, error) {
	cms := corev1.ConfigMapList{}
	if err := cli.List(context.TODO(), &client.ListOptions{Namespace: FluentBitConfigMapNamespace}, &cms); err != nil {
		return corev1.ConfigMap{}, fmt.Errorf("get fluent-bit configmap failed, %v", err)
	}
	for _, cm := range cms.Items {
		if strings.HasPrefix(cm.Name, FluentBitConfigMapPrefix) && strings.HasSuffix(cm.Name, FluentBitConfigMapSuffix) {
			return cm, nil
		}
	}
	return corev1.ConfigMap{}, errors.New("get fluent-bit configmap none")
}

func isFluentBitConfigExist(cli client.Client, conf *types.FluentBitConfig, cm corev1.ConfigMap) bool {
	instances := getInstances(cli, cm)
	if index := slice.SliceIndex(instances, conf.Name); index >= 0 {
		return true
	}
	return false
}

func isFluentBitConfigWorkloadExist(cli client.Client, conf *types.FluentBitConfig, cm corev1.ConfigMap) (bool, error) {
	confs, err := getConfigs(cli, cm)
	if err != nil {
		return true, err
	}
	for _, c := range confs {
		if reflect.DeepEqual(conf.Workload, c.Workload) {
			return true, nil
		}
	}
	return false, nil
}

func createConfig(cli client.Client, conf *types.FluentBitConfig, cm corev1.ConfigMap) error {
	if conf.Name == "" {
		return errors.New("fluent-bit config field name can not be null")
	}
	if isFluentBitConfigExist(cli, conf, cm) {
		return errors.New(fmt.Sprintf("fluent-bit config name %s has already exists", conf.Name))
	}
	exist, err := isFluentBitConfigWorkloadExist(cli, conf, cm)
	if err != nil {
		return fmt.Errorf("check fluent-bit config failed, %v", err)
	}
	if exist {
		return errors.New(fmt.Sprintf("fluent-bit config workload namespace:%s kind:%s name:%s has already exists", conf.Workload.Namespace, conf.Workload.Kind, conf.Workload.Name))
	}

	oldFbConf, ok := cm.Data[FluentBitConfigFileName]
	if ok {
		newFbConf := updateFluentBitFile(oldFbConf, conf.Name, "add")
		cm.Data[FluentBitConfigFileName] = newFbConf
	}

	oldPaConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if ok {
		newPaConf, err := updateParserFile(conf, oldPaConf, "add")
		if err != nil {
			return fmt.Errorf("Gen parsers conf failed, %v", err)
		}
		cm.Data[FluentBitConfigParsersFileName] = newPaConf
	}

	instanceFileName := conf.Name + ".conf"
	cm.Data[instanceFileName] = genInstanceFile(conf)
	return cli.Update(context.TODO(), &cm)
}

func getConfigs(cli client.Client, cm corev1.ConfigMap) ([]*types.FluentBitConfig, error) {
	var confs []*types.FluentBitConfig
	instances := getInstances(cli, cm)
	for _, instance := range instances {
		if instance == "alllog" {
			continue
		}
		conf, err := getConfig(cli, instance, cm)
		if err != nil {
			return confs, err
		}
		confs = append(confs, conf)
	}
	return confs, nil
}

func getConfig(cli client.Client, name string, cm corev1.ConfigMap) (*types.FluentBitConfig, error) {
	var conf types.FluentBitConfig
	conf.Name = name
	conf.SetID(name)
	parserConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if ok {
		regex, timeKey, timeFormat, err := getRegex(parserConf, name)
		if err != nil {
			return &conf, err
		}
		conf.RegExp = regex
		conf.Time_Key = timeKey
		conf.Time_Format = timeFormat
	}
	instanceFileName := name + ".conf"
	instanceConf, ok := cm.Data[instanceFileName]
	if ok {
		namespace, kind, workloadName, err := getDBPath(instanceConf)
		if err != nil {
			return &conf, err
		}
		conf.Workload = types.WorkloadConf{
			Namespace: namespace,
			Kind:      kind,
			Name:      workloadName,
		}
	}
	return &conf, nil
}

func deleteConfig(cli client.Client, name string, cm corev1.ConfigMap) error {
	oldFbConf, ok := cm.Data[FluentBitConfigFileName]
	if ok {
		newFbConf := updateFluentBitFile(oldFbConf, name, "del")
		cm.Data[FluentBitConfigFileName] = newFbConf
	}
	oldPaConf, ok := cm.Data[FluentBitConfigParsersFileName]
	if ok {
		conf, err := getConfig(cli, name, cm)
		if err != nil {
			return fmt.Errorf("get fluent-bit config %s failed, %v", name, err)
		}
		newPaConf, err := updateParserFile(conf, oldPaConf, "del")
		if err != nil {
			return fmt.Errorf("update parsers conf failed, %v", err)
		}
		cm.Data[FluentBitConfigParsersFileName] = newPaConf
	}

	instanceFileName := name + ".conf"
	delete(cm.Data, instanceFileName)
	return cli.Update(context.TODO(), &cm)
}

func updateConfig(cli client.Client, conf *types.FluentBitConfig, cm corev1.ConfigMap) error {
	if conf.Name == "" {
		conf.Name = conf.GetID()
	} else {
		if conf.Name != conf.GetID() {
			return errors.New("fluent-bit config field name not allowed to update")
		}
	}
	oldConf, err := getConfig(cli, conf.Name, cm)
	if err != nil {
		return fmt.Errorf("get config failed, %v", err)
	}
	if oldConf.RegExp != conf.RegExp || oldConf.Time_Key != conf.Time_Key || oldConf.Time_Format != conf.Time_Format {
		oldPaConf, ok := cm.Data[FluentBitConfigParsersFileName]
		if ok {
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
	}
	if !reflect.DeepEqual(oldConf.Workload, conf.Workload) {
		instanceFileName := conf.Name + ".conf"
		delete(cm.Data, instanceFileName)
		cm.Data[instanceFileName] = genInstanceFile(conf)
	}
	return cli.Update(context.TODO(), &cm)
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
		return result, fmt.Errorf("gen parsers conf failed, %v", err)
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
	LogName := conf.Workload.Name + "*_" + conf.Workload.Namespace + "_"
	DBPath := conf.Workload.Namespace + "_" + conf.Workload.Kind + "_" + conf.Workload.Name
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

func getInstances(cli client.Client, cm corev1.ConfigMap) []string {
	var instances []string
	infos := strings.Split(cm.Data[FluentBitConfigFileName], "\n")
	for _, line := range infos {
		if strings.HasPrefix(line, "@INCLUDE") && strings.HasSuffix(line, ".conf") {
			instance := strings.Split(strings.Fields(line)[1], ".")[0]
			instances = append(instances, instance)
		}
	}
	return instances
}

func getRegex(paConf, name string) (string, string, string, error) {
	var regex, timeKey, timeFormat string
	section := "[PARSER]"
	ps := strings.Split(string(paConf), section)
	for _, p := range ps {
		if len(p) == 0 {
			continue
		}
		data := section + p
		conf, err := ForMat(strings.Trim(fmt.Sprint(section), "[]"), data)
		if err != nil {
			return regex, timeKey, timeFormat, err
		}
		n, ok := conf["Name"]
		if ok && n == name {
			regex = conf["Regex"]
			timeKey = conf["Time_Key"]
			timeFormat = conf["Time_Format"]
		}
	}
	return regex, timeKey, timeFormat, nil
}

func getDBPath(instanceConf string) (string, string, string, error) {
	var namespace, kind, name string
	conf, err := ForMat("INPUT", instanceConf)
	if err != nil {
		return namespace, kind, name, err
	}
	dbPtah, ok := conf["DB"]
	if ok {
		ps := strings.Split(path.Base(dbPtah), "_")
		namespace = ps[1]
		kind = ps[2]
		name = strings.Split(ps[3], ".")[0]
	}
	return namespace, kind, name, nil
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
