# 资源
## blockdevice
### 接口
list
### 原理
1. 通过zcloud-proxy向集群内的cluster-agent发出blockdevices的请求，获取主机上所有块设备信息
2. 去除角色是controlplane、etcd的主机
3. 去除没有块设备的主机
4. 判断块设备

    -- 如果主机不是存储节点
       
       -- 如果设备有分区/文件系统/挂载点，则标记usedby为"other"（将会被过滤）
       -- 否则标记usedby为空
       
    -- 如果主机是存储节点，则标记usedby为其对应的存储类型（lvm|cephfs）


       
## storagecluster

### 接口
- get：  返回存储详细信息（类型、节点信息、总容量、状态、节点容量和状态信息）
- list： 返回存储预览信息（类型、节点数、总容量、状态）
- create
- delete
- update

### 原理

#### List
获取CRD cluster信息返回
#### Get
获取CRD cluster信息后再补充以下信息

- 通过zcloud-proxy向集群内的cluster-agent发出storages/storagetype的请求，获取pv的信息
- 根据CRD cluster的信息计算节点容量和状态

#### Create
补充crd的name与storagetype一致
#### Update
如果是lvm存储类型，判断cluster的Finalizer里是否包含更新操作中要删除的节点，如果包含则更新失败
#### Delete
判断cluster的Finalizer是否为空，如果不为空则删除失败
