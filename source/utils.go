package source

import (
	argov1alpha1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
)

func emitImagesFromTemplatesToChan(templates []argov1alpha1.Template, ch chan<- string) {
	images := make(map[string]bool)

	for _, t := range templates {
		if t.Container != nil {
			images[t.Container.Image] = true
		}

		for _, ic := range t.InitContainers {
			images[ic.Container.Image] = true
		}
	}

	for image := range images {
		ch <- image
	}
}
