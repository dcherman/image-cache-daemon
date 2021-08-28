package source

import (
	"context"
	"time"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	argoinformers "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

type CronWorkflowTemplateSource struct {
	client       argoclientset.Interface
	imageCh      chan string
	resyncPeriod time.Duration
}

func (t *CronWorkflowTemplateSource) ImageCh() <-chan string {
	return t.imageCh
}

func (CronWorkflowTemplateSource) Name() string {
	return "CronWorkflowTemplateSource"
}

func (t *CronWorkflowTemplateSource) Run(ctx context.Context) {
	fac := argoinformers.NewSharedInformerFactory(t.client, t.resyncPeriod)
	inf := fac.Argoproj().V1alpha1().CronWorkflows().Informer()

	handleWorkflowTemplateChange := func(obj interface{}) {
		tmpl := obj.(*argov1alpha1.CronWorkflow)
		emitImagesFromTemplatesToChan(tmpl.Spec.WorkflowSpec.Templates, t.imageCh)
	}

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    handleWorkflowTemplateChange,
		UpdateFunc: skipOldObject(handleWorkflowTemplateChange),
	})

	inf.Run(ctx.Done())
	close(t.imageCh)
}

func NewCronWorkflowTemplateSource(client argoclientset.Interface, resyncPeriod time.Duration) ImageSource {
	return &CronWorkflowTemplateSource{
		imageCh:      make(chan string),
		client:       client,
		resyncPeriod: resyncPeriod,
	}
}
