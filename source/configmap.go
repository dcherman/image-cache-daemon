package source

import (
	"context"
	"time"

	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/yaml"
)

const defaultImagesKey = "images"
const imagesKeyAnnotation = "image-cache-daemon/key"

type OptFn func(cms *ConfigMapSource)

func WithLogger(logger *logrus.Logger) OptFn {
	return func(cms *ConfigMapSource) {
		cms.logger = logger
	}
}

func WithConfigMapSelector(selector string) OptFn {
	return func(cms *ConfigMapSource) {
		cms.configmapSelector = selector
	}
}

type ConfigMapSource struct {
	configmapSelector string
	logger            *logrus.Logger
	client            kubernetes.Interface
	imageCh           chan string
	resyncPeriod      time.Duration
}

func (cms *ConfigMapSource) ImageCh() <-chan string {
	return cms.imageCh
}

func (ConfigMapSource) Name() string {
	return "ConfigMap"
}

func (cms ConfigMapSource) handleConfigMapChange(obj interface{}) {
	cm := obj.(*corev1.ConfigMap)

	imagesKey := defaultImagesKey

	if value, ok := cm.Annotations[imagesKeyAnnotation]; ok {
		imagesKey = value
	}

	if imagesStr, ok := cm.Data[imagesKey]; ok {
		var images []string

		if err := yaml.Unmarshal([]byte(imagesStr), &images); err != nil {
			cms.logger.Errorf("failed to unmarshal key %s in configmap %s/%s: %v", imagesKey, cm.Namespace, cm.Name, err)
		} else {
			for _, i := range images {
				cms.imageCh <- i
			}
		}
	}
}

func (cms *ConfigMapSource) Run(ctx context.Context) {
	fac := informers.NewSharedInformerFactoryWithOptions(cms.client, cms.resyncPeriod, informers.WithTweakListOptions(func(lo *v1.ListOptions) {
		lo.LabelSelector = fields.ParseSelectorOrDie(cms.configmapSelector).String()
	}))

	inf := fac.Core().V1().ConfigMaps().Informer()

	inf.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc:    cms.handleConfigMapChange,
		UpdateFunc: skipOldObject(cms.handleConfigMapChange),
	}, cms.resyncPeriod)

	inf.Run(ctx.Done())

	close(cms.imageCh)
}

func NewConfigMapSource(client kubernetes.Interface, resyncPeriod time.Duration, opts ...OptFn) ImageSource {
	cms := &ConfigMapSource{
		imageCh:      make(chan string),
		client:       client,
		logger:       logrus.StandardLogger(),
		resyncPeriod: resyncPeriod,
	}

	for _, fn := range opts {
		fn(cms)
	}

	return cms
}
