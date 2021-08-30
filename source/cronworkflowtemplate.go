package source

import (
	"time"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	argoclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	argoinformers "github.com/argoproj/argo-workflows/v3/pkg/client/informers/externalversions"
)

func NewCronWorkflowTemplateSource(client argoclientset.Interface, resyncPeriod time.Duration) ImageSource {
	fac := argoinformers.NewSharedInformerFactory(client, resyncPeriod)

	return NewArgoTemplateSource(&ArgoTemplateSourceOpts{
		sourceName: "CronWorkflow",
		informer:   fac.Argoproj().V1alpha1().CronWorkflows().Informer(),
		extractTemplatesFromObject: func(obj interface{}) []argov1alpha1.Template {
			tmpl := obj.(*argov1alpha1.CronWorkflow)
			return tmpl.Spec.WorkflowSpec.Templates
		},
		client:       client,
		resyncPeriod: resyncPeriod,
	})
}
