package source_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/dcherman/image-cache-daemon/source"
	"github.com/sirupsen/logrus"
	"github.com/sirupsen/logrus/hooks/test"
)

func marshalOrPanic(obj interface{}) string {
	marshaled, err := json.Marshal(obj)

	if err != nil {
		panic(err)
	}

	return string(marshaled)
}

func Test_ConfigMapSource_Defaults(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
		},
		Data: map[string]string{
			"images": marshalOrPanic([]string{"alpine", "debian"}),
		},
	}

	nonParticipatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-2",
			Namespace: "default",
		},
		Data: map[string]string{
			"images": marshalOrPanic([]string{"ubuntu"}),
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap, &nonParticipatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithConfigMapSelector("app.kubernetes.io/part-of=image-cache-daemon"))

	go src.Run(ctx)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian"})
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_Modify(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
		},
		Data: map[string]string{
			"images": marshalOrPanic([]string{"alpine", "debian"}),
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithConfigMapSelector("app.kubernetes.io/part-of=image-cache-daemon"))

	go src.Run(ctx)

	configmapSource := src.(*source.ConfigMapSource)

	for !configmapSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	participatingConfigMap.Data["images"] = marshalOrPanic([]string{"ubuntu", "debian"})
	_, err := fakeClient.CoreV1().ConfigMaps("default").Update(ctx, &participatingConfigMap, metav1.UpdateOptions{})
	assert.NoError(t, err)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian", "ubuntu"})
	assert.ElementsMatch(t, []string{"debian", "ubuntu"}, src.Images())
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_Modify_Bad_Into_Good(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
		},
		Data: map[string]string{
			"images": "][",
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithConfigMapSelector("app.kubernetes.io/part-of=image-cache-daemon"))

	go src.Run(ctx)

	configmapSource := src.(*source.ConfigMapSource)

	for !configmapSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	participatingConfigMap.Data["images"] = marshalOrPanic([]string{"ubuntu", "debian"})
	_, err := fakeClient.CoreV1().ConfigMaps("default").Update(ctx, &participatingConfigMap, metav1.UpdateOptions{})
	assert.NoError(t, err)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"ubuntu", "debian"})
	assert.ElementsMatch(t, []string{"ubuntu", "debian"}, src.Images())
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_Modify_Good_Into_Bad(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
		},
		Data: map[string]string{
			"images": marshalOrPanic([]string{"ubuntu", "debian"}),
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithConfigMapSelector("app.kubernetes.io/part-of=image-cache-daemon"))

	go src.Run(ctx)

	configmapSource := src.(*source.ConfigMapSource)

	for !configmapSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	participatingConfigMap.Data["images"] = "]["
	_, err := fakeClient.CoreV1().ConfigMaps("default").Update(ctx, &participatingConfigMap, metav1.UpdateOptions{})
	assert.NoError(t, err)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"ubuntu", "debian"})
	assert.ElementsMatch(t, []string{"ubuntu", "debian"}, src.Images())
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_Delete(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
		},
		Data: map[string]string{
			"images": marshalOrPanic([]string{"alpine", "debian"}),
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithConfigMapSelector("app.kubernetes.io/part-of=image-cache-daemon"))

	go src.Run(ctx)

	configmapSource := src.(*source.ConfigMapSource)

	for !configmapSource.HasSynced() {
		time.Sleep(time.Millisecond * 10)
	}

	err := fakeClient.CoreV1().ConfigMaps("default").Delete(ctx, "configmap-1", metav1.DeleteOptions{})
	assert.NoError(t, err)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"alpine", "debian"})
	assert.ElementsMatch(t, []string{}, src.Images())
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_AlternateKey(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)

	t.Cleanup(cancel)

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
			Annotations: map[string]string{
				"image-cache-daemon/key": "foobar",
			},
		},
		Data: map[string]string{
			"images": marshalOrPanic([]string{"alpine", "debian"}),
			"foobar": marshalOrPanic([]string{"ubuntu", "centos"}),
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithConfigMapSelector("app.kubernetes.io/part-of=image-cache-daemon"))

	go src.Run(ctx)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.ElementsMatch(t, received, []string{"ubuntu", "centos"})
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_Bad_Input(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*1)
	t.Cleanup(cancel)

	logger, hook := test.NewNullLogger()

	participatingConfigMap := corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configmap-1",
			Namespace: "default",
			Labels: map[string]string{
				"app.kubernetes.io/part-of": "image-cache-daemon",
			},
		},
		Data: map[string]string{
			"images": "][",
		},
	}

	fakeClient := fake.NewSimpleClientset(&participatingConfigMap)
	src := source.NewConfigMapSource(fakeClient, time.Minute*15, source.WithLogger(logger))

	go src.Run(ctx)

	var received []string

	for image := range src.ImageCh() {
		received = append(received, image)
	}

	assert.Equal(t, logrus.ErrorLevel, hook.LastEntry().Level)

	assert.Len(t, received, 0)
	assert.Len(t, src.ImageCh(), 0)
}

func Test_ConfigMapSource_Name(t *testing.T) {
	fakeClient := fake.NewSimpleClientset()
	src := source.NewConfigMapSource(fakeClient, time.Minute*15)
	assert.Equal(t, "ConfigMap", src.Name())
}
