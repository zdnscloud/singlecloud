# WorkFlow
## 概要
新提供两个资源WorkFlow和WorkFlowTask，提供支持基础`构建-部署`的CICD功能

## 动机和目标
实际使用中k8s基本都会结合CICD系统来进行使用，所以singlecloud需要能够提供基本的`构建-部署`CICD流程支持，以提高使用友好性。

## 前提
1. 项目git仓库，且仓库顶级目录下需要有`Dockerfile`文件
2. 自建镜像仓库的用户名密码或dockerhub用户密码

## 详细设计
WorkFlow为namespace的子资源，可以理解为工作流的配置信息
WorkFlowTask为WorkFlow的子资源，可以理解为WorkFlow的运行实例
### WorkFlow
* 定义
```
type WorkFlow struct {
	resource.ResourceBase `json:",inline"`
	Name                  string             `json:"name" rest:"required=true,isDomain=true,description=immutable"`
	Git                   GitInfo            `json:"git" rest:"required=true"`
	Image                 ImageInfo          `json:"image" rest:"required=true"`
	AutoDeploy            bool               `json:"autoDeploy"`
	Deploy                Deployment         `json:"deploy"`
	SubTasks              []WorkFlowSubTask  `json:"subTasks" rest:"description=readonly"`
	Status                WorkFlowTaskStatus `json:"status" rest:"description=readonly"`
}
```
主要包含名称、Git信息、Image信息、自动部署开关、部署配置、状态信息（subtasks和status）几部分
Git信息：仓库URL、用户名（可选）、密码（可选）
Image信息：Image名称（类似zdnscloud/singlecloud，注意不能有http或https前缀）、镜像仓库用户名（必填）、镜像仓库密码（必填）
部署配置：基本与singlecloud deployment一致，支持挂载pvc，加入服务网络等功能
状态信息：包含subtasks（子task的状态，目前子task最多有`build`和`deploy`）和status（该workflow的整体状态）
>workflow的subtasks和status信息定义为该workflow下最近一个workflowtask的subtasks和status，若一个workflow没有任何workflowtask，则没有状态信息
* 支持的操作：create、update、get、list、delete、emptytask（即清空该workflow下所有task，为action接口）
* 业务逻辑：
    * create
        * 检查workflow同名deployment是否已存在，若已存在，报错，不允许创建
        * 创建workflow需要的docker secret（用于build后推送镜像）和git secret（只有在填写了git用户名和密码的情况下才会创建）
        > 因workflow的secret和workflow非一一对应关系，所以secret采用workflow名称作为固定前缀，后缀随机字符串的id，通过label的方式和workflow进行关联，label key为:`workflow.zdns.cn/id`
        * 创建workflow对应的serviceaccount（和workflow同名），并将该serviceaccount添加至cicd对应的clusterrolebinding中
        > serviceaccount有两个作用：1.和docker secret、git secret相关联，提供拉取git代码和推送镜像时的身份认证；2.借用cicd的全局clusterrole权限，在k8s中apply deployment和对应的pvc，此处考虑到简化实现复杂度，所有workflow共用同一个预置的clusterrole和clusterrolebinding
        * 创建workflow对应的pipelineresource（tekton crd资源，和workflow同名），并将workflow资源json序列化后保存至该pipelineresour的annotation中
    * update
        * 更新workflow对应的docker secret和git secret
        * 更新对应的pipelineresource
    * get
        * 根据请求的id向k8s get同名pipelineresource，若存在，则获取pipelineresource annotation中的workflow json内容，并反序列化为workflow对象
        * 查找pipelineresouce annotation中latest-task-id是否为空，若不为空，根据id获取最近一次workflowtask，并将workflowtask的subtasks和status赋予该workflow；否则，直接返回
    * list
        * list k8s中pipelineresource资源
        * 同get逻辑，得到workflow，返回
    * delete
        * 删除该workflow下所有的workflowtask
        * 删除同名pipelineresource
        * 从cicd的clusterrolebinding中移除该workflow的serviceaccount
        * 删除同名serviceaccount
        * 通过指定label的方式，list得到k8s中该workflow的secret，并删除
        
    * emptytask
        * 更新同名pipelineresource，将其lastest-task-id置空
        * 根据workflow名称list workflowtasks
        * 遍历删除该workflow下所有workflowtask
### WorkFlowTask
* 定义
```
type WorkFlowTask struct {
	resource.ResourceBase `json:",inline"`
	ImageTag              string             `json:"imageTag" rest:"required=true"`
	SubTasks              []WorkFlowSubTask  `json:"subTasks" rest:"description=readonly"`
	Status                WorkFlowTaskStatus `json:"status" rest:"description=readonly"`
}
```
ImageTak：指定的镜像tag，必填
SubTasks：子task名称及其状态
Status：workflowtask总体状态，有`Pending`、`Running`、`Failed`、`Succeeded`几种可能取值
支持的操作：create、list、get、openlog（即查看task日志，为websocket接口）
* 业务逻辑
    * create
        * get父资源workflow，若workflow不存在，则直接报错返回；否则继续创建
        * 获取父资源worlflow下所有workflowtask，若存在Pending或Running状态的task，则报错，不允许创建
        * 遍历父资源workflow的Deploy中Container，若有image名称包含该workflow的Image.Name，则替换为Image.Name:ImageTag（tag为该workflowtask的ImageTag参数），并生成相应的yaml
        * 生成需要的pipelinerun对象，并创建，pipelinerun的id同workflow secret，为前缀固定，后缀随机生成
        * 更新父资源workflow同名pipelineresource的lastes-task-id annotation
        > pipelinerun通过label的方式与workflow相关联
    * get
        * 向k8s中get同名pipelinerun，若不存在，返回；否则继续
        * 从pipelinerun中提取相应字段，生成workflowtask并返回
    * list
        * 根据workflow名称list namespace下该workflow所有的pipelinerun
        * 同上get，将pipelinerun转为workflowtask并返回
    * openlog
        * 先get请求的workflowtask是否存在，若不存在，返回，并记录日志；否则继续
        * step2: 根据该workflowtask subtasks获取其所有未读的container，若未读container列表长度为0，则表示已读取完毕，直接返回
        * step3: 遍历读取所有container日志，发送至websocket，完成后返回至step2

## 未来工作
* 主体逻辑改为crd实现
* 支持自定义subtask
* 支持与github联动，通过commit触发构建及部署