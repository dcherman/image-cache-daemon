package source

import (
	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
)

func getImageSetFromTemplates(templates []argov1alpha1.Template) map[string]bool {
	imageMap := make(map[string]bool)

	for _, t := range templates {
		for _, ic := range t.InitContainers {
			imageMap[ic.Container.Image] = true
		}

		if t.Script != nil {
			imageMap[t.Script.Image] = true
		}

		if t.Container != nil {
			imageMap[t.Container.Image] = true
		}

		if t.ContainerSet != nil {
			for _, c := range t.ContainerSet.Containers {
				imageMap[c.Image] = true
			}
		}
	}

	return imageMap
}

func setDifference(a map[string]bool, b map[string]bool) []string {
	var results []string

	for key := range a {
		if _, exists := b[key]; !exists {
			results = append(results, key)
		}
	}

	return results
}
