应用商店
======
## 概要
通过整合helm功能，添加自定义配置文件，定制化chart包，实现快速部署第三方chart包
## 使用场景，用例
* chart分为系统级别chart和普通chart两种，系统级别chart只有admin用户可以查看，普通chart每个用户都可以查看
* 由系统chart创建的application为系统application

## 详细设计
### chart资源
* chart是顶级资源
* 所有chart包都放在一个目录下，每个chart包文件夹名字和包名字一致，chart包文件夹下面是各个版本文件夹，版本文件夹下面就是创建chart资源的所有文件，如vanguard的chart包版本为0.1.0的文件为：

		ls  vanguard/v0.1.0/
		Chart.yaml  config  templates  values.yaml

* Chart.yaml文件中包含chart的基本信息，如app的版本，chart名字和版本，chart的关键字keywords，如果keywords中包含zcloud-system，表示该chart被zcloud认定为系统chart
* 其中config文件下的config.json文件是提供给web server，说明这个chart可以配置的参数有哪些，格式如下：

		{ 
			"label": "replicaCount", 
			"jsonKey": "deployment.replicas", 
			"type": "int", 
			"required": "false", 
			"description": "vanguard deployment replicas", 
			"default": 1,
		}

* config.json文件字段说明
  * jsonKey就是所需要传json对象的key值
  * required是表明这个字段是否是必传的, 当设置为true时，如果传过来的值为空就会报错
  * 当某个字段required为true且有类型检查规则，就会对这个字段进行有效性检查，目前仅支持三种字段类型的检查：int、string和enum，且一种类型只支持一种规则的配置
    * int 类型支持min、max，min和max必须同时配置或同时不配置 
    * string 类型支持minLen、maxLen，minLen和maxLen必须同时配置且同时不配置 
    * enum 类型必须配置validValues

* 资源字段，所有字段定义在resourceFields，如果字段类型为复合类型，子类型定义在subResources

		"resourceFields": {
        	"name": {"type": "string"},
        	"description": {"type": "string"},
        	"icon": {"type": "string"},
        	"versions": {"type": "array", "elemType": "chartVersion"}
        }
  		
		"subResources":{
        	"chartVersion": {
           		 "version": {"type": "string"},
           		 "config": {"type": "array", "elemType": "json"}
        	}   
    	}

* 支持的操作及业务逻辑
	* list
	  * 检查chart包目录是否存在，不存在则报错
	  * 获取所有chart信息及各个版本, 且过滤掉系统chart包
	 
	* get
	  * 获取提供chart包名字的包信息及所有版本
	  * 如果chart包是系统chart，则返回空且log "no valid chart versions"，不然则直接返回chart信息

### application资源
* application资源是namespace下的子资源，对应k8s为Application CRD资源
* 资源字段, 所有字段定义在resourceFields，如果字段类型为复合类型，子类型定义在subResources
		
		"resourceFields": {
        	"name": {"type": "string"},
        	"chartName": {"type": "string"},
        	"chartVersion": {"type": "string"},
        	"chartIcon": {"type": "string"},
        	"workloadCount": {"type": "int"},
        	"readyWorkloadCount": {"type": "int"},
        	"appResources": {"type": "array", "elemType": "appResource"},
        	"status": {"type": "enum", "validValues": ["create", "delete", "succeed", "failed"]}
    	},  

    	"subResources": {
        	"appResource": {
            	"name": {"type": "string"},
            	"type": {"type": "enum", "validValues": ["deployment", "daemonset", "statefulset", "configmap", "secret", "service", "ingress", "cronjob", "job"]},
            	"link": {"type": "string"}
            	"replicas": {"type": "int"},
            	"readyReplicas": {"type": "int"},
            	"creationTimestamp": {"type": "date"},
            	"exists": {"type": "bool"}
        	 } 
		}
   		
* workloadCount 字段表示该application包含的workload（deployment、daemonset、statfulset）个数，readyWorkloadCount字段表示所有pods都ready的workload个数
* appResources字段表示chart创建的所有资源， 目前支持类型定义在appResource.type的validValues中
* status表示当前application的状态，由于一个chart需要创建的资源可能会很多，所以创建和删除是异步操作，所有状态定义在status的validValues中
* appResource.replicas 表示workload设置的replicas值，appResource.readyReplicas表示workload的pods已经ready的个数。
* appResource.exists表示资源是否可以从k8s获取到
* 支持的操作及业务逻辑
  * create
    * 检查namespace是否存在
    * 检查是否存在同名的application crd
    * 获取chart包信息
      * 如果chart包对应的版本不存在则报错
      * 如果chart包是系统chart，且把该chart为普通chart创建application则报错
    * 如果chart为系统chart，为标记application为系统applicaton
    * 获取cluster version信息
    * 检查web server传过来的参数字段，即通过config.json文件对字段规则进行有效性检查，不满足则报错
    * 获取所有chart文件内容，将web server传过来的参数与文件内容做merge，用于templates中文件生成k8s	 manifest和crdmanifest文件内容，并保存到application中
    * 调用k8s创建接口创建application crd资源，application crd资源被创建，application operator就会掌管application包含的子资源的创建，并更新application状态
    * 当application operator收到application创建event会做如下操作：
      * 更新application.status.state 为 create
      * 如果application.spec.crdManifests有值则优先创建所需要的crd
      * 为application添加finalizer
      * 根据application.spec.manifests创建application的子资源
        * 检查使用manifest内容生成的object中是否包含namespace
          * 如果有，检查创建application的用户是否是admin，如果不是则报错
          * 如果没有，则object将使用application的namespace
        * 如果资源是workload，检查是否设置了加入网格，如果设置了，则为资源添加服务网格所需要的annotations
        * 使用生成的object创建k8s资源，如果创建失败且duplicate，标记此资源是重复创建资源
      * 如果上述任何步骤报错，则修改该application.status.state为failed，并发布相应的event
      * 如果所有都没有报错，则修改application.status.state为succeed，并更新application.status.appResources字段，并根据workload个数对application.status.workloadCount进行赋值
      * 保存application与application子资源中支持显示资源的对应关系（后面简称对应关系）
    * 当application operator收到创建某个application的子资源的event，或者workload的更新event，且更新的是readyReplicas，会进行如下操作：
      * 通过对应关系查找application信息，找不到则结束
      * 从集群获取对应的application crd资源，不存在则报错
      * 通过资源的replicas、readyReplicas、creationTimestramp对application.status.appResources对应的资源进行更新
      * 如果资源是workload，通过replicas、readyReplicas的值，判断该workload是否是ready状态，即replicas<=readyReaplicas
        * 如果ready，且application.status.appResources对应的资源状态不是ready，则app.status.readyWorkloadCount 加一
        * 如果不ready，且application.status.appResources对应的资源状态是ready，则app.status.readyWorkloadCount 减一
        
  * delete
     * 检查application crd资源是否存在，不存在则报错
     * 然后调用k8s删除接口，删除application crd资源，由于该crd资源在创建时设置了finalizer，所以当k8s收到删除命令，会设置application crd资源的删除时间，后面application operator会接管后续删除操作
     * 当application operator收到application的更新操作event，且更新的是删除时间，则进行以下操作：
       * 根据application.spec.manifests删除所有子资源
         * 检查资源是否是duplicate资源，如果是则跳过
         * 检查由manifest内容生成的object是否有namespace，没有则设置为application的namespace
         * 对于支持显示的资源，如果没有在对应关系中找到，则直接进行下一个资源的删除
         * 调用k8s删除接口使用object删除资源
       * 如果上述任何步骤出现错误，则更新application.status.state为failed
     * 当application operator收到资源的删除操作的event，则进行以下操作：
       * 通过对应关系查找application信息，如果没找到则结束
       * 从集群获取application crd资源，不存在则报错
       * 如果资源是workload，且application.status.appResources中对应的资源是ready，则application.status.readyReplcas 减一
       * 更新application.status.appResources
       * 从对应关系中删除该资源
       * 如果对应关系中，application对应的子资源全部被删除，则删除application crd资源的finalizer，并删除对应关系中application的信息
       
  * list
     * 获取所有application crd资源，如果是系统application，则跳过
     * 遍历application.status.appResources, 如果exists为true，则根据该资源的namespace、type、name生成link
     * 如果application有删除时间，则设置singlecloud的application.status为delete
     
  * get
    * 获取application crd资源，如果是系统application纪录则返回空且log无权限错误
    * 遍历application.status.appResources, 如果exists为true，则根据该资源的namespace、type、name生成link
    * 如果application crd资源有删除时间，则设置singlecloud的application.status为delete
        
## 未来工作
* 支持资源排序创建
