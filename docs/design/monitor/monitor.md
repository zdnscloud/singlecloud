# 集群监控
## 概要
整合prometheus-operator，为集群提供全面的监控功能
## 动机和目标
基于官方chart修改后的chart（兼容原有prometheus服务发现功能）进行部署，提供精简后的rest api
## 架构
## 详细设计
### 资源定义
monitor为cluster的子资源，且为单例，即一个集群只能创建一个monitor
```go
type Monitor struct {
	resttypes.Resource  `json:",inline"`
	IngressDomain       string `json:"ingressDomain"`
	StorageClass        string `json:"storageClass"`
	StorageSize         int    `json:"storageSize"`
	PrometheusRetention int    `json:"prometheusRetention"`
	ScrapeInterval      int    `json:"scrapeInterval"`
	AdminPassword       string `json:"adminPassword"`
	RedirectUrl         string `json:"redirectUrl"`
	ApplicationLink     string `json:"applicationLink"`
}
```
* `IngressDomain`:prometheus dashboard ingress域名，若不指定，默认使用集群edge节点ip生成的动态域（域名后缀固定为zc.zdns.cn），例如：`zcloud-monitor-zcloud-local.202.173.9.62.zc.zdns.cn`
* `StorageClass`:默认为`lvm`
* `StorageSize`:单位`Gi`，默认为`10`，prometheus metric数据持久化使用
* `PrometheusRetention`:prometheus数据保留时间，单位`d`，默认`10`
* `ScrapeInterval`:prometheus指标拉取时间间隔，单位`s`，默认`15`
* `AdminPassword`:默认为`zcloud`
* `RedirectUrl`:prometheus dashboard页面跳转链接
* `ApplicationLink`:harbor的helm应用链接，用于查看应用详情及删除应用
### 权限验证
admin用户可创建、查看；普通用户只可查看
### 创建监控
* 校验是否为admin用户
* 根据请求查找对应的cluster，若找不到则报错提示cluster不存在
* 检查该集群是否已存在monitor，若已存在则报错提示Duplicate
* 检查集群是否存在相应的storageClass，若不存在则报错
* 根据请求参数生成相应的prometheus application对象
* 调用helm application创建接口异步创建prometheus应用
> 注：因后台异步创建，故需等待monitor application创建成功后方可正常使用；monitor为系统应用，相应的chart也为系统chart，仅admin可见
### 查看监控
* 根据请求查找cluster，若找不到则返回空
* 查看`zcloud`命名空间下是否存在名为 `zcloud-monitor`的application，若不存在则返回空
* 提取`zcloud-monitor` application的config信息，生成monitor对象返回前端
> 注：因monitor为单例资源，所以查看仅提供list接口（返回的列表中只会有一个monitor对象）
### 删除监控
monitor本身不提供删除接口，删除需通过删除相应的application实现（可以通过application链接完成删除）
## 未来工作
* prometheus-operator crd资源创建从zke中移除，集成至prometheus chart中
* grafana图表调整
