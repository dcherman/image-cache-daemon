module github.com/dcherman/image-cache-daemon

go 1.16

require (
	github.com/argoproj/argo-workflows/v3 v3.1.6
	github.com/benbjohnson/clock v1.1.0
	github.com/google/uuid v1.2.0 // indirect
	github.com/onsi/gomega v1.10.3 // indirect
	github.com/sirupsen/logrus v1.8.1
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20210426230700-d19ff857e887 // indirect
	google.golang.org/genproto v0.0.0-20201110150050-8816d57aaa9a // indirect
	google.golang.org/grpc v1.33.2 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/client-go v0.21.0
	sigs.k8s.io/yaml v1.2.0
)

replace sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
