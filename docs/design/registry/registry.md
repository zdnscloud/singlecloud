# 镜像仓库
## 概要
通过整合harbor，为用户提供私有镜像仓库功能
## 动机和目标
基于harbor官方chart进行部署安装，提供一个参数高度精简后的rest api
## 架构
## 详细设计
### 资源定义
registry为cluster的子资源，且为单例，即一个集群只能创建一个registry
```go
type Registry struct {
	resttypes.Resource `json:",inline"`
	IngressDomain      string `json:"ingressDomain"`
	StorageClass       string `json:"storageClass"`
	StorageSize        int    `json:"storageSize"`
	AdminPassword      string `json:"adminPassword"`
	RedirectUrl        string `json:"redirectUrl"`
	ApplicationLink    string `json:"applicationLink"`
}
```
* `IngressDomain`:harbor ingress服务域名，若不指定，默认使用集群edge节点ip生成的动态域（域名后缀固定为zc.zdns.cn），例如：`zcloud-registry-zcloud-local.202.173.9.62.zc.zdns.cn`
* `StorageClass`:若不指定，默认为`lvm`
* `StorageSize`:单位`Gi`若不指定，默认为`50`
* `AdminPassword`:默认为`zcloud`
* `RedirectUrl`:harbor登录页面跳转链接
* `ApplicationLink`:harbor的helm应用链接，用于查看应用详情及删除应用
### 权限验证
admin用户可创建、查看；普通用户只可查看
### 创建镜像仓库
* 校验是否为admin用户
* 根据请求查找对应的cluster，若找不到则报错提示cluster不存在
* 检查该集群是否已存在registry，若已存在则报错提示Duplicate
* 检查集群是否存在相应的storageClass，若不存在则报错
* 根据请求参数生成相应的harbor application对象
* 调用helm application创建接口异步创建harbor应用
> 注：因regist为后台异步创建，故需等待harbor application创建成功后方可正常使用；registry为系统应用，相应的chart也为系统chart，仅admin可见
### 查看镜像仓库
* 根据请求查找cluster，若找不到则返回空
* 查看`zcloud`命名空间下是否存在名为 `zcloud-registry`的application，若不存在则返回空
* 提取`zcloud-registry` application的config信息，生成registry对象返回前端
> 注：因registry为单例资源，所以查看仅提供list接口（返回的列表中只会有一个registry对象）
### 删除镜像仓库
registry本身不提供删除接口，删除需通过删除相应的application实现（可以通过registry的application链接完成删除）
## 未来工作
* harbor用户体系与singlecloud用户体系集成
