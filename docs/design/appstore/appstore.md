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
* application资源是namespace下的子资源
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
            	"readyReplicas": {"type": "int"}
        	 } 
		}
   		
* workloadCount 字段表示该application包含的workload（deployment、daemonset、statfulset）个数，readyWorkloadCount字段表示所有pods都ready的workload个数
* appResources字段表示chart创建的所有资源， 目前支持类型定义在appResource.type的validValues中
* status表示当前application的状态，由于一个chart需要创建的资源可能会很多，所以创建和删除是异步操作，所有状态定义在status的validValues中
* appResource.replicas 表示workload设置的replicas值，appResource.readyReplicas表示workload的pods已经ready的个数。
* 支持的操作及业务逻辑
  * create
    * 检查namespace是否存在
    * 获取chart包信息，如果chart包对应的版本不存在则报错，如果当前用户是普通用户，且chart包是系统chart，则报错
    * 如果当前用户是admin且chart为系统chart，为标记application为系统applicaton
    * 获取cluster version信息
    * 检查web server传过来的参数字段，即通过config.json文件对字段规则进行有效性检查，不满足则报错
    * 获取所有chart文件内容，将web server传过来的参数与文件内容做merge，用于templates中文件生成k8s	 manifest文件内容，并保存到application中
    * 设置application的status为create
    * 使用cluster名字、namespace生成数据库表名，将application信息保存到数据库
    * 开线程创建application所有资源，并返回application信息给web server
    * 创建资源线程
      * 检查使用manifest内容生成的object中是否包含namespace
        * 如果有，检查当前用户是否是admin，如果不是则报错，如果是，则使用该namespace生成appResource的link字段
        * 如果没有，则使用web server传过来的namespace设置object
      * 使用生成的object创建k8s资源，如果创建失败且duplicate，标记此资源是重复创建资源
      * 如果创建资源成功，把支持显示的类型保存到application的appResources中, 如果有workload，将workload个数赋值给workloadCount
      * 如果上述任何步骤报错，则修改数据库中该application的status为failed，如果所有都没有报错，则修改status为succeed，并更新application的appResources字段

  * delete
     * 从数据库中获取application，如果application属于系统application，就返回无权限错误
     * 检查application的status，不能删除status是create或delete的application
     * 更新application的status为delete
     * 开启删除资源线程，应答web server
     * 删除资源线程
       * 检查资源是否是duplicate资源，如果是则跳过
       * 检查由manifest内容生成的object是否有namespace，没有则设置为web server传过来的namespace
       * 使用object删除资源
       * 删除数据库中application纪录
       * 如果上述任何步骤出现错误，则更新数据库的application的status为failed，需要用户再次删除
       
  * list
     * 从数据库中获取这个namespace所有application纪录，如果是系统application纪录，则跳过
     * 如果不是系统application，从k8s获取application的appResources信息
       * 获取某个appResource失败则log错误，跳过该application并获取下一个application，
       * 如果某个appResource是workload，根据replicas和readyReplicas信息，计算appplication的readyWorkloadCount个数
     
  * get
    * 从数据库中获取application纪录，如果是系统application纪录则返回空且log无权限错误
    * 从k8s获取application的appResources信息，如果获取某个appResources失败，则log错误并返回空，如果appResource是workload，根据replicas和readyReplicas信息，计算appplication的readyWorkloadCount个数 
        
## 未来工作
* 支持资源排序创建