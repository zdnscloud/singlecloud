# 概要
k8s1.7+开始允许用户注册一个自定义资源到集群中，可以通过API来创建、访问、维护这个自定义的对象。利用这个功能，我们在k8s集群中增加3个关于存储的CRD：cluster、iscsi、nfs，然后部署一个operator（资源控制器）pod从API中获取该资源的对象，通过对存储集群组件、CSI和storageclass等资源进行一系列操作，最终完成存储的创建，为用户提供同名storageclass

# 目标
简化用户使用存储的复杂性，用户只需为某种类型的存储指定一个或多个有干净块设备的存储节点或者配置当前已经存在的iscsi target/nfs，即可完成对应的storageclass，供用户通过pvc来申请使用

# 使用场景
目前支持种类型的存储
- cluster
  - lvm
  - cephfs
- iscsi
- nfs

## lvm

### 特点
- 本地存储
- 块存储
- 存储节点上只要有可用的块设备即可在创建时自动加入存储集群
### 局限性
- 没有高可用
- 使用本地存储的pod调度是由pv决定的,pv一旦创建，使用该pv的pod只能调度到某个固定节点
- 与cephfs类型存储之间节点互斥（一个节点只能属于一种类型存储）
### 读写模式
ReadWriteOnce

## cephfs
### 特点
- 网络存储
- 文件系统存储
- 高可用
- 存储节点上只要有可用的块设备即可在创建时自动加入存储集群
### 局限性
- 存储节点的删除会导致数据迁移,整个更新周期可能较长
- 与cephfs类型存储之间节点互斥（一个节点只能属于一种类型存储）
### 读写模式
ReadWriteMany

## iscsi
### 特点
- 网络存储
- 块存储
- 利用现有iscsi target
- 支持用户名密码自动登陆
- 多个节点任意调度
### 局限性
- pod只能在用户选择的存储节点之间调度
### 读写模式
ReadWriteOnce
	
## nfs
### 特点
- 网络存储
- 文件系统存储
- 利用现有nfs
- pvc大小由nfs服务共享路径大小决定
### 读写模式
ReadWriteMany

# 详细设计

## 存储类型注册
pkg/eventhandler/eventhandler.go
```
func New(cli client.Client) *HandlerManager {
	storageCtrl := &Controller{
                stopCh:     stopCh,
                clusterMgr: cluster.New(cli),
                iscsiMgr:   iscsi.New(cli),
                nfsMgr:     nfs.New(cli),
                client:     cli,
        }
}
```
## 监听API
    监听VolumeAttachment是为了给资源对象增加/删除Finalizer属性
    
    监听cluster，调用common完成块设备信息补充，再交由相应类型存储完成

    监听iscsi，调用iscsi模块完成

    监听nfs，调用nfs模块完成
    
    监听Endpoints是针对cephfs需要根据mon的地址更新configmap交给csi使用
pkg/controller/controller.go 
```
ctrl := controller.New("zcloudStorage", c, scm)
ctrl.Watch(&storagev1.Cluster{})
ctrl.Watch(&storagev1.Iscsi{})
ctrl.Watch(&storagev1.Nfs{})
ctrl.Watch(&k8sstorage.VolumeAttachment{})
ctrl.Watch(&corev1.Endpoints{})
ctrl.Start(stopCh, storageCtrl, predicate.NewIgnoreUnchangedUpdate())
```

## 获取节点块设备并更新对象的Status.Config字段
pkg/controller/controller.go 

对于新增节点，通过k8s服务cluster-agent获取
```
func AssembleCreateConfig(cli client.Client, cluster *storagev1.Cluster) (*storagev1.Cluster, error) {
  devstmp, err := GetBlocksFromClusterAgent(cli, h)
	if err := UpdateStorageclusterConfig(cli, cluster.Name, "add", infos); err != nil {}
}
```
对于删除节点，通过k8s对象的Status.Config字段获取
```
func AssembleDeleteConfig(cli client.Client, cluster *storagev1.Cluster) (*storagev1.Cluster, error) {
	storagecluster, err := GetStorage(cli, cluster.Name)
	if err := UpdateStorageclusterConfig(cli, cluster.Name, "del", infos); err != nil {}
}
```
对于更新操作，先对比出新增和删除节点，然后分别调用上面俩个函数
```
func AssembleUpdateConfig(cli client.Client, oldc, newc *storagev1.Cluster) (*storagev1.Cluster, *storagev1.Cluster, error) {
	del, add := HostsDiff(oldc.Spec.Hosts, newc.Spec.Hosts)
	dels, err := AssembleDeleteConfig(cli, delc)
	adds, err := AssembleCreateConfig(cli, addc)
}
```
### lvm说明
目录结构
```
lvm/
├── create.go	  #创建存储
├── delete.go	  #删除存储
├── lvm.go        #主引导文件
├── status.go     #状态更新
├── template.go   #资源模板
├── image.go      #镜像版本
├── update.go	  #更新存储
└── yaml.go 	  #构建yaml
```

- 创建  
  1. 给节点增加labels
  2. 部署Lvmd并等待其全部运行
  3. 初始化磁盘
        1. 检查Volume Group是否已经存在
        2. 检查磁盘是否有分区和文件系统，如果有则强制格式化磁盘
        3. 创建Physical volume
        4. 创建Volume Group
    >
    >  注：如果创建Volume Group之前不存在，则直接vgcreate。如果已经存在，则进行vgextend
  4. 部署CSI并等待其全部运行
  5. 部署storageclass
  6. gorouting循环检查lvm的运行及磁盘空间并更新cluster状态（频率60秒）   
  
- 更新  
  1. 对比更新前后的配置，确定删除的主机、增加的主机
  2. 对上面2种配置进行分别处理
  > 
  > 如果删除前只有一块磁盘组成Volume Group，则直接vgremove。如果是有多块磁盘组成Volume Group，则进行vgreduce
  
  > 如果有Pod正在使用这个Volume Group，则Volume Group的操作将会失败
     
- 删除  
  1. 删除storageclass
  2. 删除CSI
  3. 格式化磁盘
  4. 删除Lvmd
  5. 删除节点的labels
  
  
### ceph说明
[ceph简介](https://github.com/zdnscloud/immense/blob/master/docs/ceph.md)

目录结构
```
├── ceph
│   ├── ceph.go     #主引导文件
│   ├── client      #包装ceph命令
│   ├── config      #为ceph集群创建configmap,secret,headless-service
│   ├── create.go   #创建存储集群
│   ├── delete.go   #删除存储集群
│   ├── fscsi       #CSI相关
│   ├── global      #全局变量配置
│   ├── mds         #ceph组件mds
│   ├── mgr         #ceph组件mgr         
│   ├── mon         #ceph组件mon
│   ├── osd         #ceph组件osd
│   ├── status      #更新存储集群状态
│   ├── update.go   #更惨存储集群配置
│   ├── util        #常用工具函数
│   └── zap         #初始化磁盘
```
- 创建  
  1. 给节点增加labels
  2. 创建ceph集群
     1. 获取k8s集群Pod地址段
     2. 随机生成adminkey, monkey，并根据crd的uuid作为ceph的集群id
     3. 根据磁盘个数设置副本数（默认为2）
     4. 根据前面3步的配置
		1. 创建configmap保存ceph集群配置文件，用于后面启动ceph组件挂载使用
		2. 创建无头服务，用于后面ceph组件连接mon
		3. 创建secret，保存账户和密钥，用于后面storageclass使用
		4. 创建serviceaccount，用于后面部署ceph组件用
	
     5. 保存ceph集群配置到本地
     6. 启动mon并等待其全部运行
     7. 启动mgr并等待其全部运行
     8. 启动osd并等待其全部运行
     9. 启动mds并等待其全部运行
  3. 部署CSI并等待其全部运行
  4. 启动2个gorouting
     - 循环检查ceph集群中是否有异常的osd，如果有就remove，等待集群数据恢复
     - 循环检查ceph的运行及磁盘空间并更新cluster状态（频率60秒）    
- 更新  
  1. 对比更新前后的配置，确定删除的主机、增加的主机
  2. 实际上就是增加/删除osd组件Pod
  3. 删除后增加labels
- 删除  
  1. 删除CSI
  2. 删除Ceph集群
     1. 删除mds
     2. 删除osd（后调用zap对磁盘进行清理）
     3. 删除mgr
     4. 删除mon
     5. 删除本地ceph配置文件
  3. 删除configmap,secret,service
  3. 删除节点的labels

### iscsi说明
目录结构
```
iscsi/
├── create.go   #创建存储
├── delete.go   #删除存储
├── image.go    #镜像版本
├── iscsi.go    #主引导文件
├── status.go   #状态更新
├── template.go #资源模板
├── util.go     #常用工具函数
└── yaml.go     #构建yaml
```
- 创建
  1. 给节点增加labels
  2. 部署init，完成iscsi target的登陆挂载，以及pv、vg的创建
  3. 部署lvmd并等待其全部运行
  4. 部署CSI并等待其全部运行
  5. 部署storageclass
  6. gorouting循环检查iscsi的运行及磁盘空间并更新cluster状态（频率60秒）

- 更新
  1. 对比更新前后的配置，确定删除的主机、增加的主机
  2. 对上面2种配置进行分别镜像labels的增删，相应的组件会自动部署

- 删除
  1. 删除storageclass
  2. 删除CSI
  4. 删除Lvmd
  5. 删除节点的labels
  
### nfs说明
目录结构
```
iscsi/
├── create.go   #创建存储
├── delete.go   #删除存储
├── image.go    #镜像版本
├── nfs.go      #主引导文件
├── status.go   #状态更新
├── template.go #资源模板
├── util.go     #常用工具函数
└── yaml.go     #构建yaml
```
- 创建
  1. 部署provisioner
  2. 部署storageclass
  3. gorouting循环检查nfs的运行及磁盘空间并更新cluster状态（频率60秒）

- 更新
  暂不支持

- 删除
  1. 删除storageclass
  2. 删除provisioner
  
# 未来工作
- iscsi支持PVC的扩容和快照（CSI）
- 在lvmd上只用精简卷thinpool
- Ceph版本升级
- 存储节点运行时自动发现和扩容
- 存储节点删除时数据可靠性保障

