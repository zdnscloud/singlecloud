# Zcloud SLB
## 概览
在zcloud中引入lb controller，以支持通过外部负载均衡设备暴露k8s集群中的服务
## 动机和目标
负载均衡在企业用户中被广泛使用，为了兼顾k8s l4负载性能和ingress controller高可用，直接支持外部负载设备的自动化配置是最简单的解决方案，也可以更好的补充zcloud解决方案的整体性

设计目标：集成lb controller、支持lb设备的整机ha部署模式
## 设计
### 集群管理
在集群配置中新增一项可选的高级配置--->LoadBalance
* 配置说明：
```go
type ClusterLoadBalance struct {
	Enable       bool   `json:"enable"`
	MasterServer string `json:"masterServer"`
	BackupServer string `json:"backupServer"`
	User         string `json:"user"`
	Password     string `json:"password"`
}
```
enable: 控制是否启用lb插件
masterServer：主用lb设备api地址（ipv4 address or ipv4 host）（必填）
backupServer：备用lb设备api地址（可选）
user：lb设备用户名（必填）
password：lb设备密码（必填）
* 支持的操作：集群创建、集群编辑
> 暂不支持禁用后自动删除lb controller，需手动删除
### service
增加LoadBalancer类型service
* 配置说明：
loadBalanceVip: service在lb上的虚拟服务地址（ipv4 address）（必填）
loadBalanceMethod：service负载均衡算法（可选），有rr（轮询）、lc（最小连接数）、hash（源ip hash）三种可选，默认是轮询