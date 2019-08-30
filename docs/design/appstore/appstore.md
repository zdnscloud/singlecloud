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
  * required是表明这个字段是否是必传的
    * 当设置为true时，如果传过来的值为空就会报错
    * 当设置为false时，如果没有传值，则使用default配置的值
  * 当type时enum时，传值必须包含在validValues中

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
	  * 获取所有chart信息及各个版本
	    * 如果当前用户是admin，则可以返回所有chart包 
	    * 如果是普通用户， 则过滤掉系统chart包
	 
	* get
	  * 获取提供chart包名字的包信息及所有版本
	  * 如果当前用户是普通用户，且chart包是系统chart，则报错，不然则直接返回chart信息

### application资源
* application资源是namespace下的子资源
* 资源字段, 所有字段定义在resourceFields，如果字段类型为复合类型，子类型定义在subResources
		
		"resourceFields": {
        	"name": {"type": "string"},
        	"chartName": {"type": "string"},
        	"chartVersion": {"type": "string"},
        	"chartIcon": {"type": "string"},
        	"appResources": {"type": "array", "elemType": "appResource"},
        	"status": {"type": "enum", "validValues": ["create", "delete", "succeed", "failed"]}
    	},  

    	"subResources": {
        	"appResource": {
            	"name": {"type": "string"},
            	"type": {"type": "enum", "validValues": ["deployment", "daemonset", "statefulset", "configmap", "secret", "service", "ingress", "cronjob", "job"]},
            	"link": {"type": "string"}
        	 } 
		}
   		
* appResources字段表示chart创建的所有资源， 目前支持类型定义在appResources的validValues中
* status表示当前application的状态，由于一个chart需要创建的资源可能会很多，所以创建和删除是异步操作，所有状态定义在status的validValues中
* 支持的操作及业务逻辑
  * create
    * 检查namespace是否存在
    * 检查提供的chart包对应的版本是否存在
    * 获取chart包信息，如果当前用户是普通用户，且chart包是系统chart，则报错
    * 如果当前用户是admin且chart为系统chart，为标记application为系统applicaton
    * 获取所有chart文件内容，将web server传过来的参数与文件内容做merge，用于templates中文件生成k8s	 manifest文件内容，并保存到application中
    * 设置application的status为create
    * 使用cluster名字、namespace生成数据库表名，将application信息保存到数据库
    * 开线程创建application所有资源，并返回application信息给web server
    * 创建资源线程
      * 检查使用manifest内容生成的object中是否包含namespace
        * 如果有，检查当前用户是否是admin，如果不是则报错，如果是，则使用该namespace生成appResource的link字段
        * 如果没有，则使用web server传过来的namespace设置object
      * 使用生成的object创建k8s资源，如果创建失败且duplicate，标记此资源是重复创建资源
      * 如果创建资源成功，把支持显示的类型保存到application的appResources中
      * 如果上述任何步骤报错，则修改数据库中该application的status为failed，如果所有都没有报错，则修改status为succeed，并更新application的appResources字段

  * delete
     * 从数据库中获取application，如果当前用户不是admin，且application属于系统application，就返回无权限错误
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
     * 从数据库中获取这个namespace所有application纪录
     * 如果当前用户是admin，则返回所有application纪录
     * 如果是普通用户，只返回非系统application纪录
     
## 未来工作
* 支持CRD
* 支持资源排序创建