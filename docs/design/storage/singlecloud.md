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


       
## storage

### 接口
- get：  返回存储详细信息（类型、节点信息、总容量、状态、节点容量和状态信息）
- list： 返回存储预览信息（类型、节点数、总容量、状态）
- create
- delete
- update

### 原理
由于storage资源不是k8s标准资源，因此将其存储在db后再根据不同type进行响应的操作

#### 注册存储handle

目前支持4中存储类型
```
type StorageHandle interface {
        GetType() types.StorageType
        GetStorage(cli client.Client, name string) (*types.Storage, error)
        GetStorageDetail(cluster *zke.Cluster, name string) (*types.Storage, error)
        Delete(cli client.Client, name string) error
        Create(cluster *zke.Cluster, storage *types.Storage) error
        Update(cluster *zke.Cluster, storage *types.Storage) error
}
```
```
	storageHandles: []StorageHandle{
                 &LvmManager{},
                 &CephFsManager{},
                 &IscsiManager{},
                 &NfsManager{}},
        }
```

#### List
从db中获取所有storage及其type，根据type调用不同的存储handle返回预览信息
#### Get
从db中获取所有storage及其type，根据type调用不同的存储handle返回详细

- 通过zcloud-proxy向集群内的cluster-agent发出storages/storageclass的请求，获取pv的信息
- 根据CRD 的信息计算节点容量和状态

#### Create
根据type调用不同的存储handle进行创建
#### Update
根据type调用不同的存储handle进行更新
> nfs 类型不支持update
#### Delete
根据type调用不同的存储handle进行删除

判断cluster的Finalizer是否为空，如果不为空则删除失败
