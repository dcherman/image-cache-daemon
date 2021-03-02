module github.com/dcherman/image-cache-daemon

go 1.16

require (
	github.com/argoproj/argo v0.0.0-20201121022849-310e099f8252
	github.com/benbjohnson/clock v1.1.0
	github.com/imdario/mergo v0.3.11 // indirect
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.1.3
	github.com/stretchr/testify v1.7.0
	golang.org/x/sys v0.0.0-20210113181707-4bcb84eeeb78 // indirect
	golang.org/x/time v0.0.0-20201208040808-7e3f01d25324 // indirect
	google.golang.org/appengine v1.6.7 // indirect
	k8s.io/api v0.17.17
	k8s.io/apimachinery v0.17.17
	k8s.io/client-go v0.17.17
)

replace sigs.k8s.io/controller-tools => sigs.k8s.io/controller-tools v0.2.9
