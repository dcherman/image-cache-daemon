package source

import (
	"context"
	"sync"
	"time"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/tools/cache"
)

type ArgoTemplateSourceOpts struct {
	sourceName                 string
	extractTemplatesFromObject func(obj interface{}) []argov1alpha1.Template
	informer                   cache.SharedIndexInformer
	resyncPeriod               time.Duration
	client                     argoclientset.Interface
}

func NewArgoTemplateSource(opts *ArgoTemplateSourceOpts) ImageSource {
	return &ArgoTemplateSource{
		sourceName:                 opts.sourceName,
		informer:                   opts.informer,
		lock:                       sync.RWMutex{},
		imageMap:                   make(map[string]bool),
		images:                     make([]string, 0),
		extractTemplatesFromObject: opts.extractTemplatesFromObject,
		imageCh:                    make(chan string),
		client:                     opts.client,
		resyncPeriod:               opts.resyncPeriod,
	}
}

type ArgoTemplateSource struct {
	sourceName                 string
	extractTemplatesFromObject func(obj interface{}) []argov1alpha1.Template
	client                     argoclientset.Interface
	imageCh                    chan string
	resyncPeriod               time.Duration

	informer cache.SharedIndexInformer
	imageMap map[string]bool
	images   []string
	lock     sync.RWMutex
}

func (t *ArgoTemplateSource) ImageCh() <-chan string {
	return t.imageCh
}

func (t *ArgoTemplateSource) Images() []string {
	t.lock.RLock()
	defer t.lock.RUnlock()

	return t.images
}

func (ats *ArgoTemplateSource) Name() string {
	return ats.sourceName
}

func (t *ArgoTemplateSource) updateImagesFromInformer() {
	t.imageMap = t.getImagesFromInformer()

	var images []string

	for key := range t.imageMap {
		images = append(images, key)
	}

	t.images = images
}

func (t *ArgoTemplateSource) getImagesFromInformer() map[string]bool {
	imageMap := make(map[string]bool)

	if t.informer != nil {

		indexer := t.informer.GetIndexer()

		for _, key := range indexer.ListKeys() {
			value, exists, err := indexer.GetByKey(key)

			if !exists {
				logrus.Warnf("key %s did not exist in indexer", key)
			} else if err != nil {
				logrus.Errorf("failed to retrieve key %s from indexer: %v", key, err)
			} else {
				for image := range getImageSetFromTemplates(t.extractTemplatesFromObject(value)) {
					imageMap[image] = true
				}
			}
		}
	}

	return imageMap
}

func (t *ArgoTemplateSource) HasSynced() bool {
	if t.informer != nil {
		return t.informer.HasSynced()
	}

	return false
}

func (t *ArgoTemplateSource) Run(ctx context.Context) {
	t.informer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			images := getImageSetFromTemplates(t.extractTemplatesFromObject(obj))

			t.lock.Lock()
			defer t.lock.Unlock()

			newImages := setDifference(images, t.imageMap)

			for _, image := range newImages {
				t.imageMap[image] = true
				t.images = append(t.images, image)
				t.imageCh <- image
			}
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			previousImages := getImageSetFromTemplates(t.extractTemplatesFromObject(oldObj))
			currentImages := getImageSetFromTemplates(t.extractTemplatesFromObject(newObj))

			deletedImages := setDifference(previousImages, currentImages)

			t.lock.Lock()
			defer t.lock.Unlock()

			newImages := setDifference(currentImages, t.imageMap)

			for _, image := range newImages {
				t.imageMap[image] = true
				t.images = append(t.images, image)
				t.imageCh <- image
			}

			if len(deletedImages) > 0 {
				t.updateImagesFromInformer()
			}
		},
		DeleteFunc: func(_ interface{}) {
			t.lock.Lock()
			defer t.lock.Unlock()

			t.updateImagesFromInformer()
		},
	})

	t.informer.Run(ctx.Done())
	close(t.imageCh)
}
