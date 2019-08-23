用户资源申请
======
## 概要
为了管理普通用户对存储资源的使用，新添加自定义资源为userquota，当用户申请资源时，为其自动分配namespace，并且增加对所分配的namespace资源使用的限制。

## 动机和目标
限制普通用户对storage资源创建和使用, 普通用户只能操作自己创建的资源申请纪录，管理员可以管理所有用户的资源申请纪录。

## 使用场景和用例
 * 资源申请纪录状态有：processing、approval、rejection
 * 普通用户可以创建、更新和删除资源申请纪录，且只能更新和删除approval或者rejection状态的纪录
 * 管理员可以审批和删除资源申请纪录，审批操作包括approval和reject，管理员只能审批processing状态的纪录，同时只能删除approval或者rejection状态的纪录

## 详细设计
* 资源类型为usrquota，是顶级资源
* 资源字段

		"resourceFields":{
          "name": {"type": "string"},
          "clusterName": {"type": "string"},
          "namespace": {"type": "string"},
          "userName": {"type": "string"},
          "cpu": {"type": "string"},
          "memory": {"type": "string"},
          "storage": {"type": "string"},
          "requestType": {"type": "enum", "validValues": ["create", "update"]},
          "status": {"type": "enum", "validValues": ["processing", "approval", "rejection"]},
          "purpose": {"type": "string"},
          "requestor": {"type": "string"},
          "telephone": {"type": "string"},
          "rejectionReason": {"type": "string"},
          "responseTimestamp": {"type": "date"}
    	} 

* 支持操作和业务逻辑

  * create 
    * 检查namespace、cpu、memory、storage参数有效性
    * 设置请求类型为create，状态为processing，用户名为当前用户
    * 检查数据库中是否有同名namespace，否则报duplicate error
    * 添加记录到数据库 

  * list
    * 获取所有资源记录
    * 检查用户类型
		* 如果该用户是管理员，则返回全部记录
		* 如果该用户是普通用户，只返回该用户的资源记录

  * get
    * 获取提供的ID的资源记录
    * 检查用户类型
		* 如果该用户是管理员，返回该记录
		* 如果该用户是普通用户，如果获取的资源记录的用户名是该用户才返回

  * delete
    * 检查用户名，如果不是管理员，需要检查是否和该记录的用户名一致
    * 检查记录状态是否是processing，此状态不允许删除操作
    * 如果此条记录的集群名字不是空，需要删除对应的namespace
    * 从数据库删除该记录
    * 如果此条记录的集群名字不是空，更新用户authorizer，即删除该用户与namespace所属关系

  * update
    * 检查namespace、cpu、memory、storage参数有效性
    * 检查用户名和namespace是否和该记录的用户名和namespace一致
    * 检查记录状态是否是processing，此状态不允许普通用户做任何操作
    * 更新数据库中的纪录，请求类型为update

  * approval（action）
    * 检查用户是否为管理员，只有管理员才能做此操作
    * 检查记录状态是否是processing，只有processing状态才能做此操作
    * 检查k8s集群中是否存在用户申请的namespace
		* 如果不存在，则创建namespace和resourcequota， resourcequota名字和namespace一致，如果创建resourcequota失败，需要删除之前创建的namespace
		* 如果存在，则更新该namespace下面的resourcequota 
    * 更新数据库中的纪录，状态变成approval，responseTimestamp为更改时间，如果更新失败，则需要对上一步操作进行回滚
    * 如果是第一次approval，则需要更新用户authorizer，即添加用户与namespace的所属关系
    
  * reject（action）
     * 检查用户是否为管理员，只有管理员才能做此操作
     * 检查记录状态是否是processing，只有processing状态才能做此操作
     * 更新数据库中的纪录，更新rejectionReason，状态变成rejection，responseTimestamp为更改时间


## 未来工作
增加cpu、memory资源限制