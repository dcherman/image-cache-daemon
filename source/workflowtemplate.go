package source

import (
	"context"
	"time"

	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo/pkg/client/clientset/versioned"
	argoinformers "github.com/argoproj/argo/pkg/client/informers/externalversions"
	"k8s.io/client-go/tools/cache"
)

type WorkflowTemplateSource struct {
	client       argoclientset.Interface
	imageCh      chan string
	resyncPeriod time.Duration
}

func (t *WorkflowTemplateSource) ImageCh() <-chan string {
	return t.imageCh
}

func (WorkflowTemplateSource) Name() string {
	return "WorkflowTemplate"
}

func (t *WorkflowTemplateSource) Run(ctx context.Context) {
	fac := argoinformers.NewSharedInformerFactory(t.client, t.resyncPeriod)
	inf := fac.Argoproj().V1alpha1().WorkflowTemplates().Informer()

	handleWorkflowTemplateChange := func(obj interface{}) {
		tmpl := obj.(*argov1alpha1.WorkflowTemplate)
		emitImagesFromTemplatesToChan(tmpl.Spec.Templates, t.imageCh)
	}

	inf.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: handleWorkflowTemplateChange,
		UpdateFunc: func(_, obj interface{}) {
			handleWorkflowTemplateChange(obj)
		},
	})

	inf.Run(ctx.Done())
}

func NewWorkflowTemplateSource(client argoclientset.Interface, resyncPeriod time.Duration) ImageSource {
	return &WorkflowTemplateSource{
		imageCh:      make(chan string),
		client:       client,
		resyncPeriod: resyncPeriod,
	}
}
