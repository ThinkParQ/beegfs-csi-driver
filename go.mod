module github.com/netapp/beegfs-csi-driver

go 1.12

require (
	github.com/container-storage-interface/spec v1.3.0
	github.com/golang/glog v0.0.0-20160126235308-23def4e6c14b
	github.com/golang/protobuf v1.4.3 // indirect
	github.com/kubernetes-csi/csi-lib-utils v0.9.0
	github.com/kubernetes-csi/csi-test v1.1.1
	github.com/onsi/ginkgo v1.14.2 // indirect
	github.com/onsi/gomega v1.10.4 // indirect
	github.com/pkg/errors v0.9.1
	github.com/smartystreets/goconvey v1.6.4 // indirect
	github.com/spf13/afero v1.5.1
	go.uber.org/multierr v1.6.0
	golang.org/x/net v0.0.0-20210119194325-5f4716e94777
	golang.org/x/sys v0.0.0-20210124154548-22da62e12c0c
	golang.org/x/text v0.3.5 // indirect
	google.golang.org/genproto v0.0.0-20210126160654-44e461bb6506 // indirect
	google.golang.org/grpc v1.35.0
	gopkg.in/ini.v1 v1.62.0
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/klog/v2 v2.4.0
	k8s.io/utils v0.0.0-20200912215256-4140de9c8800
)
