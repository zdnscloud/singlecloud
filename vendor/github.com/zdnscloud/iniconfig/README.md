# 说明
该软件包实现了一种ini配置文件解析器语言

配置文件由多个部分组成，以“ [section] ”标题开头，后跟“ name value ”条目

> 前后的空格已从值中删除

例如：
```
    [INPUT]
        Name             tail
        Path             /var/log/containers/*.log
        Parser           docker
        Tag              kube.*
        Refresh_Interval 5
        Mem_Buf_Limit    5MB
        Skip_Long_Lines  On

    [FILTER]
        Name   kubernetes
        Match  kube.*

    [OUTPUT]
        Name  es
        Match *
        Host  elasticsearch-master
        Port  9200
        Logstash_Format On
        Retry_Limit False
```
# 使用

## 读取文件
```
  file := "./config.ini"
  cfg, err := iniconfig.ReadDefault(file)
```
## 读取字符串
```
  cfg := iniconfig.NewDefault()
  cfg.Read(bufio.NewReader(strings.NewReader(str)))
```
## 构造map
```
flag := "INPUT"
if cfg.HasSection(flag) {
    section, err := cfg.SectionOptions(flag)
    if err == nil {
        for _, v := range section {
            options, err := cfg.String(flag, v)
            if err == nil {
                TOPIC[v] = options
            }
        }
    }
}
fmt.Println(TOPIC)
```

# 另外
参考的https://github.com/larspensjo/config

原项目有以下几个点不符合读取fluent-bit配置文件格式
-   支持的分隔符为“:”或者“=”，没有“ ”
-   value部分内容不支持空格
-   只支持读取文件，不支持读取字符串

因此对代码做了部分修改
