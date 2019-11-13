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
    	Metrics               []MetricSpec                   
    	Status                HorizontalPodAutoscalerStatus  
	}

	type MetricSpec struct {
    	Type         MetricSourceType  
    	MetricName   string            
    	ResourceName ResourceName      
    	TargetType   MetricTargetType  
    	MetricValue  `json:",inline"`
	}
	
	type MetricValue struct {
    	AverageValue       string  
    	AverageUtilization int     
	}
	
## 配置
	
* ScaleTargetKind支持deployment和statefulset，表明HPA对哪一种workload进行配置，即通过ScaleTargetKind和ScaleTargetName字段定位一个workload
* MinReplicas和MaxReplicas表示这个workload的最小和最大replicas值
* Metrics为HPA指标参数数值，MetricSpec.Type支持Resource 和 Pods两种类型的指标，每种指标的值的类型通过MetricSpec.TargetType指定，支持Utilization和AverageValue。
	* Resource类型通过指定MetricSpec.ResourceName表示以哪种资源为指标，MetricSpec.ResourceName支持cpu和memory，MetricSpec.TargetType支持Utilization和AverageValue
	* Pods是以自定义metric为指标，通过指定MetricName表示以哪一个自定义metric为指标，只支持AverageValue
* 无论HPA如何通过Metrics来动态调整workload的pods个数，所调整的pods个数都不会超过MinReplicas和MaxReplicas的范围。

## 配置范例

	"name": "hpa-vg1-vanguard",
    "scaleTargetKind": "deployment",
    "scaleTargetName": "vg1-vanguard",
    "minReplicas": 1,
    "maxReplicas": 5,
    "metrics":[
    	{
    		"type": "Pods",
    		"metricName": "zdns_vanguard_qps",
    		"targetType": "AverageValue",
    		"averageValue": "2000"
    	},
    	{
    		"type": "Resource",
    		"ResourceName": "cpu",
    		"targetType": "Utilization",
    		"averageUtilization": 50
    	},
    	{
    		"type": "Resource",
    		"ResourceName": "memory",
    		"targetType": "AverageValue",
    		"averageValue": "256Mi"
    	}
    ]