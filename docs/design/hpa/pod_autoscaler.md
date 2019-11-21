# Pod AutoScaler
# 概要
通过配置HorizontalPodAutoscaler（简称HPA）资源，以资源（包含cpu和memory）或者自定义metric为指标，对workload的pod个数进行自动扩缩

# 详细设计
## HPA资源
HPA为namespace下的子资源

	type HorizontalPodAutoscaler struct {
    	Name                  string                        
    	ScaleTargetKind       ScaleTargetKind               
    	ScaleTargetName       string
    	MinReplicas           int
    	MaxReplicas           int                              
    	ResourceMetrics       []ResourceMetricSpec
    	CustomMetrics         []CustomMetricSpec                   
    	Status                HorizontalPodAutoscalerStatus  
	}

	type ResourceMetricSpec struct {
    	ResourceName       ResourceName      
    	TargetType         MetricTargetType  
    	AverageValue       string  
    	AverageUtilization int
	}
	
	type CustomMetricSpec struct {
    	MetricName   string            
    	AverageValue string  
	}
	
## 配置
	
* ScaleTargetKind支持deployment和statefulset，表明HPA对哪一种workload进行配置，即通过ScaleTargetKind和ScaleTargetName字段定位一个workload
* MinReplicas和MaxReplicas表示这个workload的最小和最大replicas值
* ResourceMetrics为HPA基础资源指标的配置，通过指定ResourceName表示以哪种资源为指标，ResourceName支持cpu和memory，TargetType支持Utilization和AverageValue
* CustomMetrics是以自定义metric为指标，通过指定MetricName表示以哪一个自定义metric为指标，只支持AverageValue
* 无论HPA如何动态调整workload的pods个数，所调整的pods个数都不会超过MinReplicas和MaxReplicas的范围。

## 配置范例

	"name": "hpa-vg1-vanguard",
    "scaleTargetKind": "deployment",
    "scaleTargetName": "vg1-vanguard",
    "minReplicas": 1,
    "maxReplicas": 5,
    "customMetrics":[
    	{
    		"metricName": "zdns_vanguard_qps",
    		"averageValue": "2000"
    	}
    ]
    "resourceMetrics": [
    	{
    		"ResourceName": "cpu",
    		"targetType": "Utilization",
    		"averageUtilization": 50
    	},
    	{
    		"ResourceName": "memory",
    		"targetType": "AverageValue",
    		"averageValue": "256Mi"
    	}
    ]
    
## 期待副本数计算
* 获取workload所有pods的metrics，每个pod的metrics为pod中所有container的metrics总量
* 计算指标值及使用比率
  * 指标类型是平均值
    * 当前平均值currentAverage ＝ metrics累加总量／pods个数
    * 使用比率为 usageRatio = currentAverage / targetAverage
  * 指标类型是平均利用率
    * 计算请求总值，即累加每个container的resources.requests对应的资源请求量
    * 当前平均利用率currentUtilization ＝ meitrics累加总量 ／ 请求总量 
    * 使用比率 usageRatio ＝ currentUtilization ／ targetUtilization
* 期待副本数计算
  * 对 1 - usageRatio 取绝对值
    * 如果绝对值小于0.1，表明改变不大，则期待副本数desiredReplicas = currentReplicas
    * 如果大于0.1，期待副本数desiredReplicas = usageRatio * pods个数
  * 如果workload在进行缩容，那么距离上次变更的时间差必须大于controller manager 参数 horizontal-pod-autoscaler-downscale-stabilization配置的时间，才可以改变desiredReplicas， 否则还是使用原来的desiredReplicas 
  * 由于k8s限制pods增长过快，设置了最大增长量scaleUpLimit，即在当前副本数*2 与 4 中取最大值
    * 如果hpa.MaxReplicas大于scaleUpLimit，那么此次允许的最大副本数maximumAllowedReplicas为scaleUpLimit
    * 如果hpa.MaxReplicas小于scaleUpLimit，那么此次允许的最大副本数maximumAllowedReplicas为hpa.MaxReplicas
  * 如果desiredReplicas的值不在在hpa.MinReplicas与maximumAllowedReplicas范围内
    * 如果desiredReplicas小于hpa.MinReplicas，则 desiredReplicas ＝ hpa.MinReplicas
    * 如果desiredReplicas大于maximumAllowedReplicas，则 desiredReplicas ＝ maximumAllowedReplicas
  * 最终计算出来的期待副本数为desiredReplicas
