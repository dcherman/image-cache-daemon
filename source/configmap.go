package source

import (
	"context"
	"fmt"
	"sync"
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

func getImagesFromConfigMap(obj interface{}) (map[string]bool, error) {
	cm, ok := obj.(*corev1.ConfigMap)

	if !ok {
		return nil, fmt.Errorf("could not cast input to corev1.ConfigMap")
	}

	imagesKey := defaultImagesKey

	if value, ok := cm.Annotations[imagesKeyAnnotation]; ok {
		imagesKey = value
	}

	imageMap := make(map[string]bool)

	if imagesStr, ok := cm.Data[imagesKey]; ok {
		var images []string

		if err := yaml.Unmarshal([]byte(imagesStr), &images); err != nil {
			return nil, fmt.Errorf("failed to unmarshal key %s in configmap %s/%s: %v", imagesKey, cm.Namespace, cm.Name, err)
		}

		for _, i := range images {
			imageMap[i] = true
		}
	}

	return imageMap, nil
}

type ConfigMapSource struct {
	configmapSelector string
	logger            *logrus.Logger
	client            kubernetes.Interface
	imageCh           chan string
	imageMap          map[string]bool
	informer          cache.SharedIndexInformer
	images            []string
	resyncPeriod      time.Duration
	lock              sync.RWMutex
}

func (cms *ConfigMapSource) ImageCh() <-chan string {
	return cms.imageCh
}

func (cms *ConfigMapSource) Images() []string {
	cms.lock.RLock()
	defer cms.lock.RUnlock()

	return cms.images
}

func (*ConfigMapSource) Name() string {
	return "ConfigMap"
}

func (cms *ConfigMapSource) updateImagesFromInformer() {
	cms.imageMap = cms.getImagesFromInformer()

	var images []string

	for key := range cms.imageMap {
		images = append(images, key)
	}

	cms.images = images
}

func (cms *ConfigMapSource) getImagesFromInformer() map[string]bool {
	imageMap := make(map[string]bool)

	if cms.informer != nil {
		indexer := cms.informer.GetIndexer()

		for _, key := range indexer.ListKeys() {
			value, exists, err := indexer.GetByKey(key)

			if !exists {
				logrus.Warnf("key %s did not exist in indexer", key)
			} else if err != nil {
				logrus.Errorf("failed to retrieve key %s from indexer: %v", key, err)
			} else {
				images, err := getImagesFromConfigMap(value)

				if err != nil {
					cms.logger.Errorf("failed to get images from configmap: %v", err)
					continue
				}

				for image := range images {
					imageMap[image] = true
				}
			}
		}
	}

	return imageMap
}

func (cms *ConfigMapSource) Run(ctx context.Context) {
	cms.informer.AddEventHandlerWithResyncPeriod(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			imageMap, err := getImagesFromConfigMap(obj)

			if err != nil {
				cms.logger.Errorf("failed to get images from configmap: %v", err)
				return
			}

			cms.lock.Lock()
			defer cms.lock.Unlock()

			for key := range imageMap {
				if _, exists := cms.imageMap[key]; !exists {
					cms.imageMap[key] = true
					cms.images = append(cms.images, key)
					cms.imageCh <- key
				}
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			currentImages, err := getImagesFromConfigMap(newObj)

			if err != nil {
				cms.logger.Errorf("failed to get images from configmap: %v", err)
				return
			}

			previousImages, err := getImagesFromConfigMap(oldObj)

			if err != nil {
				cms.logger.Errorf("failed to get images from configmap: %v", err)
				cms.logger.Warn("skipping deletion detection, could not parse prior images from configmap")
				previousImages = currentImages
			}

			deletedImages := setDifference(previousImages, currentImages)

			cms.lock.Lock()
			defer cms.lock.Unlock()

			newImages := setDifference(currentImages, cms.imageMap)

			for _, image := range newImages {
				cms.imageMap[image] = true
				cms.images = append(cms.images, image)
				cms.imageCh <- image
			}

			if len(deletedImages) > 0 {
				cms.updateImagesFromInformer()
			}
		},
		DeleteFunc: func(obj interface{}) {
			cms.lock.Lock()
			defer cms.lock.Unlock()

			cms.updateImagesFromInformer()
		},
	}, cms.resyncPeriod)

	cms.informer.Run(ctx.Done())

	close(cms.imageCh)
}

func (cms *ConfigMapSource) HasSynced() bool {
	if cms.informer != nil {
		return cms.informer.HasSynced()
	}

	return false
}

func NewConfigMapSource(client kubernetes.Interface, resyncPeriod time.Duration, opts ...OptFn) ImageSource {
	cms := &ConfigMapSource{
		imageCh:      make(chan string),
		imageMap:     make(map[string]bool),
		images:       make([]string, 0),
		client:       client,
		logger:       logrus.StandardLogger(),
		lock:         sync.RWMutex{},
		resyncPeriod: resyncPeriod,
	}

	for _, fn := range opts {
		fn(cms)
	}

	fac := informers.NewSharedInformerFactoryWithOptions(client, resyncPeriod, informers.WithTweakListOptions(func(lo *v1.ListOptions) {
		lo.LabelSelector = fields.ParseSelectorOrDie(cms.configmapSelector).String()
	}))

	cms.informer = fac.Core().V1().ConfigMaps().Informer()

	return cms
}
