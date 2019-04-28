package config

import (
	"github.com/zdnscloud/cement/configure"
)

type ConfigureOwner interface {
	ReloadConfig(*VanguardConf)
}

func ReloadConfig(o interface{}, conf *VanguardConf) {
	if owner, ok := o.(ConfigureOwner); ok {
		owner.ReloadConfig(conf)
	}
}

type VanguardConf struct {
	Path          string                `yaml:"-"`
	Server        ServerConf            `yaml:"server"`
	EnableModules []string              `yaml:"enable_modules"`
	Logger        LoggerConf            `yaml:"logger"`
	Acls          []AclConf             `yaml:"acl"`
	Views         ViewConf              `yaml:"view"`
	Cache         CacheConf             `yaml:"cache"`
	Forwarder     ForwarderConf         `yaml:"forwarder"`
	QuerySource   []QuerySourceInView   `yaml:"query_source"`
	Recursor      []RecursorInView      `yaml:"recursor"`
	Resolver      ResolverConf          `yaml:"resolver"`
	Filter        FilterConf            `yaml:"filter"`
	AAAAFilter    []AAAAFilterInView    `yaml:"aaaa_filter"`
	LocalData     []LocaldataInView     `yaml:"local_data"`
	Hijack        []HijackInView        `yaml:"hijack"`
	SortList      []SortListInView      `yaml:"sort_list"`
	VipDomain     []VipDomainInView     `yaml:"vip_domain"`
	Auth          []AuthZoneInView      `yaml:"auth_zone"`
	Stub          []StubZoneInView      `yaml:"stub_zone"`
	FailForwarder []FailForwarderInView `yaml:"fail_forwarder"`
	DNS64         []DNS64InView         `yaml:"dns64"`
	Kubernetes    Kubernetes            `yaml:"kubernetes"`
}

type ServerConf struct {
	Addrs        []string `yaml:"addr"`
	HttpCmdAddr  string   `yaml:"http_cmd_addr"`
	HandlerCount int      `yaml:"handler_count"`
	EnableTCP    bool     `yaml:"enable_tcp"`
}

type ViewConf struct {
	ViewAcls         []ViewAcl         `yaml:"ip_view_binding,omitempty"`
	ZoneViewBindings []ZoneViewBinding `yaml:"zone_view_binding,omitempty"`
	ViewWeights      []ViewWeight      `yaml:"weight_view_binding,omitempty"`
	viewNames        []string          `yaml:"-"`
}

type ViewAcl struct {
	View         string   `yaml:"view"`
	Acls         []string `yaml:"acls"`
	KeyName      string   `yaml:"key_name"`
	KeySecret    string   `yaml:"key_secret"`
	KeyAlgorithm string   `yaml:"key_algorithm"`
}

type AclConf struct {
	Name     string          `yaml:"name"`
	Networks AclNetworksConf `yaml:"networks"`
}

type AclNetworksConf struct {
	IPs             []string    `yaml:"ips"`
	ValidInterval   []TimeRange `yaml:"valid_time"`
	InvalidInterval []TimeRange `yaml:"invalid_time"`
}

type TimeRange struct {
	Begin string `yaml:"from"`
	End   string `yaml:"to"`
}

type ZoneViewBinding struct {
	Zone string `yaml:"zone"`
	View string `yaml:"view"`
}

type ViewWeight struct {
	View   string `yaml:"view"`
	Weight int    `yaml:"weight"`
}

type CacheConf struct {
	PositiveTtl  uint32 `yaml:"positive_ttl"`
	NegativeTtl  uint32 `yaml:"negative_ttl"`
	MaxCacheSize uint   `yaml:"max_cache_size"`
	ShortAnswer  bool   `yaml:"short_answer"`
	Prefetch     bool   `yaml:"prefetch"`
}

type SortListInView struct {
	View         string   `yaml:"view"`
	SourceIp     string   `yaml:"source_ip"`
	PreferredIps []string `yaml:"preferred_ips"`
}

type ForwardProberConf struct {
	ProbeInterval  uint32 `yaml:"probe_interval"`
	Timeout        uint32 `yaml:"timeout"`
	TimeoutLasting uint32 `yaml:"timeout_lasting"`
}

type ForwardZoneConf struct {
	Name         string   `yaml:"name"`
	ForwardStyle string   `yaml:"forward_style"`
	Forwarders   []string `yaml:"forwarders"`
}

type RecursorInView struct {
	Enable           bool   `yaml:"enable"`
	View             string `yaml:"view"`
	RootHintFile     string `yaml:"root_hint"`
	EdnsSubnetEnable bool   `yaml:"subnet_enable"`
}

type ForwardZoneInView struct {
	View        string            `yaml:"view"`
	QuerySource string            `yaml:"query_source"`
	Zones       []ForwardZoneConf `yaml:"zones"`
}

type ForwarderConf struct {
	ForwardZones []ForwardZoneInView `yaml:"forward_zone_for_view,omitempty"`
	Prober       ForwardProberConf   `yaml:"probe_setting"`
}

type ResolverConf struct {
	CheckCnameIndirect bool `yaml:"check_cname_indirect"`
}

type QuerySourceInView struct {
	View    string `yaml:"view"`
	Address string `yaml:"addr"`
}

type LoggerConf struct {
	Querylog   QuerylogConf   `yaml:"query_log"`
	GeneralLog GeneralLogConf `yaml:"general_log"`
}

type QuerylogConf struct {
	Path      string `yaml:"querylog_file"`
	FileSize  int    `yaml:"size_in_byte"`
	Versions  int    `yaml:"number_of_files"`
	Extension bool   `yaml:"qlog_extension"`
}

type GeneralLogConf struct {
	Enable   bool   `yaml:"enable"`
	Path     string `yaml:"log_file"`
	FileSize int    `yaml:"size_in_byte"`
	Versions int    `yaml:"number_of_files"`
	Level    string `yaml:"level"`
}

type FilterConf struct {
	DropSrvFailed   bool                        `yaml:"drop_server_failed"`
	DomainNameLimit []DomainNameRateLimitInView `yaml:"domain_name_limit_for_view,omitempty"`
	NetworkLimit    []NetworkRateLimit          `yaml:"network_limit,omitempty"`
}

type NetworkRateLimit struct {
	Network string `yaml:"network"`
	Limit   uint32 `yaml:"limit"`
}

type DomainNameRateLimitInView struct {
	View            string                `yaml:"view"`
	DomainNameLimit []DomainNameRateLimit `yaml:"domain_name_limit"`
}

type DomainNameRateLimit struct {
	Name  string `yaml:"name"`
	Limit uint32 `yaml:"limit"`
}

type VipDomainInView struct {
	View       string   `yaml:"view"`
	Domains    []string `yaml:"domain"`
	DefaultTtl int      `yaml:"default_ttl"`
}

type AuthZoneInView struct {
	View  string         `yaml:"view"`
	Zones []AuthZoneConf `yaml:"zones"`
}

type StubZoneInView struct {
	View  string         `yaml:"view"`
	Zones []StubZoneConf `yaml:"zones"`
}

type FailForwarderInView struct {
	View      string `yaml:"view"`
	Forwarder string `yaml:"forwarder"`
}

type DNS64InView struct {
	View            string   `yaml:"view"`
	PreAndPostfixes []string `yaml:"pre_and_postfixes"`
}

type AuthZoneConf struct {
	Name    string   `yaml:"name"`
	File    string   `yaml:"file"`
	Masters []string `yaml:"masters"`
}

type StubZoneConf struct {
	Name    string   `yaml:"name"`
	Masters []string `yaml:"masters"`
}

type LocaldataInView struct {
	View      string   `yaml:"view"`
	NXDomain  []string `yaml:"nxdomain"`
	NXRRset   []string `yaml:"nxrrset"`
	Exception []string `yaml:"exception"`
	Redirect  []string `yaml:"redirect"`
}

type HijackInView struct {
	View     string   `yaml:"view"`
	Redirect []string `yaml:"redirect"`
}

type AAAAFilterInView struct {
	View string   `yaml:"view"`
	Acls []string `yaml:"acls"`
}

type Kubernetes struct {
	ClusterDNSServer      string `yaml:"cluster_dns_server"`
	ClusterDomain         string `yaml:"cluster_domain"`
	ClusterCIDR           string `yaml:"cluster_cidr"`
	ClusterServiceIPRange string `yaml:"cluster_service_ip_range"`
}

func (vc *ViewConf) GetViewWeight() map[string]int {
	viewWeight := make(map[string]int)
	for _, vp := range vc.ViewWeights {
		viewWeight[vp.View] = vp.Weight
	}
	return viewWeight
}

func (vc *ViewConf) GetViews() []string {
	return vc.viewNames
}

func (conf *VanguardConf) Reload() error {
	var newConf VanguardConf
	if err := configure.Load(&newConf, conf.Path); err != nil {
		return err
	}
	newConf.Path = conf.Path
	*conf = newConf

	return nil
}

func LoadConfig(path string) (*VanguardConf, error) {
	var conf VanguardConf
	conf.Path = path
	if err := conf.Reload(); err != nil {
		return nil, err
	}

	return &conf, nil
}
