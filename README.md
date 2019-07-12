#####这是一个是一个自定义的k8s cni插件，主要用来固定IP地址，负责IP地址分配

######(1) 配置文件格式
```json
{
	"ipam": {
		"type": "host-remote",
		"ipam_server": "http://xx.xx.xx.xx:9090"
	}
}
```

######(2) 文件说明
main.go: 插件主文件

host_remote_*_test.go: 测试文件

