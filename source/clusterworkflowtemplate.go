package source

import (
	"context"
	"time"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	argoinformers "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

type ClusterWorkflowTemplateSource struct {
	client       argoclientset.Interface
	imageCh      chan string
	resyncPeriod time.Duration
}

func (t *ClusterWorkflowTemplateSource) ImageCh() <-chan string {
	return t.imageCh
}

func (ClusterWorkflowTemplateSource) Name() string {
	return "ClusterWorkflowTemplate"
}

func (t *ClusterWorkflowTemplateSource) Run(ctx context.Context) {
	fac := argoinformers.NewSharedInformerFactory(t.client, t.resyncPeriod)
	inf := fac.Argoproj().V1alpha1().ClusterWorkflowTemplates().Informer()

	handleWorkflowTemplateChange := func(obj interface{}) {
		tmpl := obj.(*argov1alpha1.ClusterWorkflowTemplate)
		emitImagesFromTemplatesToChan(tmpl.Spec.Templates, t.imageCh)
	}

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    handleWorkflowTemplateChange,
		UpdateFunc: skipOldObject(handleWorkflowTemplateChange),
	})

	inf.Run(ctx.Done())
	close(t.imageCh)
}

func NewClusterWorkflowTemplateSource(client argoclientset.Interface, resyncPeriod time.Duration) ImageSource {
	return &ClusterWorkflowTemplateSource{
		imageCh:      make(chan string),
		client:       client,
		resyncPeriod: resyncPeriod,
	}
}
