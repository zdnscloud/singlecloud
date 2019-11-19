# 集群管理
## 概要
集群管理功能为singlecloud提供了对于kubernetes集群的基本管理功能，有效降低了管理员管理kubernetes集群的难度。
## 动机和目标
提供一个标准的rest api可以用来管理zcloud kubernetes集群，提供集群的创建、删除、增删节点等功能，该api参数要尽可能简洁，只要求安装kubernetes集群必须参数
## 架构
集群管理部分主要分为五个子模块：
* clusterManager：维护集群列表并接收集群操作的api请求
* fsm：集群状态机，维护集群状态、集群event定义及相关事件callback定义
* zke：提供集群创建及更新、删除方法
* validation：提供集群配置检查方法
* db：提供集群信息的持久化支持

因集群状态和集群相关的操作种类较多，故采用状态机的方式来管理一个集群的状态，集群状态转移图如下所示：
```
+---------------------------------------------------------------------------+
|                                                                           |
|                                                                           |
|                                                                           v
|             +---------+                                            +------+------+
|             |         |      loadDB                   delete       |             |
|             |  Init   +-------------------+     +----------------->+   Deleting  +----+
|             |         |                   |     |                  |             |    |
|             +-----+---+                   |     |                  +------+------+    |
|                   |                       |     |                         ^           |
|                   +                       |     |                         +           |
|                create                     |     |                      delete         |
| delete            +                       |     |                         +           |
|                   v                       v     |                         |           |
|             +-----+-----+             +---+-----+---+getInfoFailed +------+------+    |
|             |           |createSucceed|             +------------->+             |    |
|             |  Creating +------------>+  Running    |              | Unreachable |    |
|             |           |             |             +<-------------+             |    |deleteCompleted
|             +-+-------+-+             +--+------+---+getInfoSucceed+-------------+    |
|               |       ^                  |      ^                                     |
|   creatFailed |       |                  |      |                                     |
|   createCanceled      +              update     |updateCompleted                      |
|               |  continuteCreate         |      |updateCanceled                       |
|               |       |                  |      |                                     |
|               v       |                  v      |                                     |
|             +-+-------+--+            +--+------+---+              +-------------+    |
|             |            |            |             |              |             |    |
+-------------+CreateFailed|            |   Updating  |              |   Deleted   +<---+
              |            |            |             |              |             |
              +------------+            +-------------+              +-------------+

```
## 详细设计
### 集群状态
* Running：运行中
* Unreachable：集群apiserver不可达，与apiserver连接恢复后会变为Running状态
* Creating：正在创建中
* Updating：更新中（增删节点、修改集群ssh配置及zcloud server地址配置）
* CreateFailed：创建失败
* Deleting：删除中
* Deleted：已删除（内部状态）
* Init：初始（内部状态）
### 集群操作
* create：创建集群，用于创建一个新kubernetes集群
* update：更新集群，可在CreateFailed及Running状态下执行，当在CreateFailed状态下更新，会向集群状态机发送continuteCreate事件；在Running状态下则发送Update事件
* delete：删除集群，会清理集群所有节点上的容器和数据存储
* cancel：取消操作，可在Creating和Updating状态下执行，执行完成后分别会向集群状态机发送createCanceled和updateCanceled事件，表明已取消完成
### 权限验证
集群的创建、更新、删除、导入、获取集群kubeconfig接口只允许admin用户调用，普通用户只有对集群的查看权限
### 配置校验
* 创建集群
    * 集群配置参数是否合法：集群名称（非空）、ssh用户名（非空）、ssh私钥（非空）、zcloud server地址（ipv4 host，即ip:port格式）、集群域名后缀（dns name）
    * 节点参数校验：name（rfc1123subdomain即全小写，只能有"-"和"."两个特殊符号）、address（ipv4地址）
    * 重复节点校验：地址或名称相同
    * 节点角色校验：controlplane和worker互斥，etcd和edge不能单独存在，需依赖于controlplane或worker，即一个节点必须为controlplane或者worker节点）
    * 节点数量校验：一个集群至少需要包含一个主节点、一个etcd节点和一个工作节点
    * 节点列表中是否包含了singlecloud server，若节点中包含singlecloud server，则不允许创建
* 更新集群
    * 集群配置参数是否合法：集群名称（非空）、ssh用户名（非空）、zcloud server地址（ipv4 host，即ip:port格式）、集群域名后缀（dns name）
    * 节点参数校验：name（rfc1123subdomain即全小写，只能有"-"和"."两个特殊符号）、address（ipv4地址）
    * 重复节点校验：地址或名称相同
    * 节点角色校验：controlplane和worker互斥，etcd和edge不能单独存在，需依赖于controlplane或worker，即一个节点必须为controlplane或者worker节点）
    * 节点数量校验：一个集群至少需要包含一个主节点、一个etcd节点和一个工作节点
    * 节点列表中是否包含了singlecloud server，若节点中包含singlecloud server，则不允许创建
    * 检查是否删除了不允许被删除的节点：contronplane和etcd节点只允许添加不允许删除、存储节点不允许删除
    > 注意：因安全考虑，集群的get接口返回时会将集群的ssh私钥参数置空，若更新请求中的ssh私钥不为空则用新的私钥覆盖原有私钥，若为空则忽略沿用原有私钥
### 创建集群逻辑
* 检查是否存在同名集群
* 配置校验
* 创建cluster对象（状态为Creating），并将集群配置保存至db（此时db中的集群的Created属性为false，即集群未创建完成）
* 调用zke接口异步创建集群（zke up）
    * 若创建成功，则向集群状态机发送创建成功事件，集群状态机执行callback更新db中该集群配置（添加集群证书配置）和db中集群的Created属性为true同时更新集群状态为Running，通过singlecloud pubsub向其他模块广播集群创建事件
    * 若创建失败，则向集群状态机发送创建失败事件，集群状态机更新内存中该集群状态为CreateFailed
    * 若创建过程中收到集群取消请求，则zke安装线程会退出，并向集群状态机createCanceled事件，集群状态变为CreateFailed
### 更新集群逻辑
* 检查是否存在该集群
* 检查集群是否可以更新（与集群状态相关，Running和CreateFailed下的集群可以更新）
* 配置校验
* 更新集群的节点配置、ssh配置、zcloud server地址配置（网络和转发dns配置会忽略），将集群配置保存至db，根据db中该集群的Created属性，若集群已创建完成，则发送update事件；若集群未创建完成，则发送continueCreate事件
* 调用zke接口异步更新集群（zke up）
    * 若发送的是update事件，无论更新成功与否或收到集群取消请求，更新完成后均发送updateCompleted事件，集群状态机执行callback更新db中集群信息，同时更新集群状态至Running
    * 若发送的是continueCreate事件：
        * 若创建成功，则向集群状态机发送创建成功事件，集群状态机执行callback更新db中该集群配置（添加集群证书配置）和db中集群的Created属性为true同时更新集群状态为Running，通过singlecloud pubsub向其他模块广播集群创建事件
        * 若创建失败，则向集群状态机发送创建失败事件，集群状态机更新内存中该集群状态为CreateFailed
        * 若创建过程中收到集群取消请求，则zke安装线程会退出，并向集群状态机createCanceled事件，集群状态变为CreateFailed
### 删除集群逻辑
* 检查集群是否存在
* 检查当前集群是否可删除（与集群状态相关，Creating和Updating以及Deleting状态下的集群不允许删除）
* 向集群状态机发送集群删除事件，并同步通过pubsub向其他模块广播集群删除事件
* 回收集群informer对象
* 异步调用zke执行集群清理，无论清理成功与否，执行结束后向集群状态机发送deleteCompleted事件，状态机执行callback将集群从列表和db中删除
### DB加载
singlecloud启动时会读取db中的集群信息，并将其加载至内存中，加载过程为同步加载，一个集群的加载流程如下：
* 从db中读取集群的信息（集群名称、Created属性、配置及kubeconfig）
* 根据读取的集群信息创建集群对象
    * 若Created为true，则执行集群初始化（创建集群kubeClient）
        * 若初始化成功则添加至集群列表
        * 若初始化失败，则继续下一个集群
        > 某集群初始化失败会记录日志
    * 若Created为false，则修改集群状态为CreateFailed，并添加至集群列表中
### 集群导入逻辑
zcloud支持导入通过zke创建的集群
* 读取请求中的集群信息
* 检查是否存在同名集群
* 创建cluster对象（状态为Running），初始化该集群
    * 若初始化成功，则将该集群信息保存至DB，并通过pubsub向其他模块广播集群创建事件
    * 若初始化失败，则返回错误信息
> 导入的集群暂不支持节点增删操作
### 获取集群kubeconfig
* 检查是否为admin用户，仅admin用户可调用获取集群kubeconfig接口
* 检查是否存在该集群
* 检查db中集群证书是否存在请求的用户名（默认为kube-admin）
* 从db中读取该集群的kubeconfig配置，返回
> kubeconfig为集群的子资源
## 未来工作
* 集群安装或更新进度实时反馈
* 安装日志优化，更便于理解
* 安装前环境检测
* 集群备份恢复功能
* 调用zke更新集群和清理集群时的错误信息展示
* 支持导入集群的更新操作
