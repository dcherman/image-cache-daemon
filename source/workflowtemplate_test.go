package source_test

import (
	"context"
	"testing"
	"time"

	argov1alpha1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	fake "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned/fake"
	"github.com/dcherman/image-cache-daemon/source"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func Test_WorkflowTemplateSource_Containers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	workflowTemplate := argov1alpha1.WorkflowTemplate{
		Spec: argov1alpha1.WorkflowTemplateSpec{
			WorkflowSpec: argov1alpha1.WorkflowSpec{
				Templates: []argov1alpha1.Template{
					{
						Container: &v1.Container{
							Image: "alpine",
						},
					},
					{
						Container: &v1.Container{
							Image: "debian",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(&workflowTemplate)
	src := source.NewWorkflowTemplateSource(fakeClient, time.Minute*15)

	go src.Run(ctx)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian"})
	assert.Len(t, src.ImageCh(), 0)
	assert.ElementsMatch(t, src.Images(), []string{"alpine", "debian"})
}

func Test_WorkflowTemplateSource_Modify(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	workflowTemplate := argov1alpha1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: argov1alpha1.WorkflowTemplateSpec{
			WorkflowSpec: argov1alpha1.WorkflowSpec{
				Templates: []argov1alpha1.Template{
					{
						Container: &v1.Container{
							Image: "alpine",
						},
					},
					{
						Container: &v1.Container{
							Image: "debian",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(&workflowTemplate)
	src := source.NewWorkflowTemplateSource(fakeClient, time.Minute*15)

	go src.Run(ctx)

	argoSource := src.(*source.ArgoTemplateSource)

	for !argoSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	workflowTemplate.Spec.WorkflowSpec.Templates[0].Container.Image = "ubuntu"

	_, err := fakeClient.ArgoprojV1alpha1().WorkflowTemplates("default").Update(ctx, &workflowTemplate, metav1.UpdateOptions{})
	assert.NoError(t, err)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian", "ubuntu"})
	assert.Len(t, src.ImageCh(), 0)
	assert.ElementsMatch(t, src.Images(), []string{"ubuntu", "debian"})
}

func Test_WorkflowTemplateSource_Delete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	workflowTemplate := argov1alpha1.WorkflowTemplate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: argov1alpha1.WorkflowTemplateSpec{
			WorkflowSpec: argov1alpha1.WorkflowSpec{
				Templates: []argov1alpha1.Template{
					{
						Container: &v1.Container{
							Image: "alpine",
						},
					},
					{
						Container: &v1.Container{
							Image: "debian",
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(&workflowTemplate)
	src := source.NewWorkflowTemplateSource(fakeClient, time.Minute*15)

	go src.Run(ctx)

	argoSource := src.(*source.ArgoTemplateSource)

	for !argoSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	err := fakeClient.ArgoprojV1alpha1().WorkflowTemplates("default").Delete(ctx, "test", metav1.DeleteOptions{})
	assert.NoError(t, err)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian"})
	assert.Len(t, src.ImageCh(), 0)
	assert.ElementsMatch(t, src.Images(), []string{})
}

func Test_WorkflowTemplateSource_Scripts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	workflowTemplate := argov1alpha1.WorkflowTemplate{
		Spec: argov1alpha1.WorkflowTemplateSpec{
			WorkflowSpec: argov1alpha1.WorkflowSpec{
				Templates: []argov1alpha1.Template{
					{
						Script: &argov1alpha1.ScriptTemplate{
							Container: v1.Container{
								Image: "alpine",
							},
						},
					},
					{
						Script: &argov1alpha1.ScriptTemplate{
							Container: v1.Container{
								Image: "debian",
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(&workflowTemplate)
	src := source.NewWorkflowTemplateSource(fakeClient, time.Minute*15)

	go src.Run(ctx)

	argoSource := src.(*source.ArgoTemplateSource)

	for !argoSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian"})
	assert.Len(t, src.ImageCh(), 0)
	assert.ElementsMatch(t, src.Images(), []string{"alpine", "debian"})
}

func Test_WorkflowTemplateSource_InitContainers(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	workflowTemplate := argov1alpha1.WorkflowTemplate{
		Spec: argov1alpha1.WorkflowTemplateSpec{
			WorkflowSpec: argov1alpha1.WorkflowSpec{
				Templates: []argov1alpha1.Template{
					{
						InitContainers: []argov1alpha1.UserContainer{
							{
								Container: v1.Container{
									Image: "alpine",
								},
							},
						},
						Script: &argov1alpha1.ScriptTemplate{
							Container: v1.Container{
								Image: "debian",
							},
						},
					},
				},
			},
		},
	}

	fakeClient := fake.NewSimpleClientset(&workflowTemplate)
	src := source.NewWorkflowTemplateSource(fakeClient, time.Minute*15)

	go src.Run(ctx)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian"})
	assert.Len(t, src.ImageCh(), 0)
	assert.ElementsMatch(t, src.Images(), []string{"alpine", "debian"})
}

func Test_WorkflowTemplateSource_Name(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	src := source.NewWorkflowTemplateSource(fakeClient, time.Minute*15)

	assert.Equal(t, "WorkflowTemplate", src.Name())
}
