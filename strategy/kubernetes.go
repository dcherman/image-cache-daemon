package strategy

import (
	"context"
	"fmt"
	"time"

	coreapiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/sirupsen/logrus"
)

type KubernetesPodPullStrategy struct {
	OwnerReference v1.OwnerReference
	Client         kubernetes.Interface

	WardenImage string

	NodeName  string
	Namespace string
	PodName   string

	imagePullErrorCh   chan string
	imagePullSuccessCh chan string
}

type KubernetesPodPullStrategyOpts struct {
	OwnerReference v1.OwnerReference
	Client         kubernetes.Interface

	WardenImage string

	NodeName  string
	Namespace string
	PodName   string
}

func NewKubernetesPodPullStrategy(opts *KubernetesPodPullStrategyOpts) *KubernetesPodPullStrategy {
	return &KubernetesPodPullStrategy{
		OwnerReference: opts.OwnerReference,
		Client:         opts.Client,
		NodeName:       opts.NodeName,
		Namespace:      opts.Namespace,
		PodName:        opts.PodName,
		WardenImage:    opts.WardenImage,

		imagePullErrorCh:   make(chan string),
		imagePullSuccessCh: make(chan string),
	}
}

func (kpps *KubernetesPodPullStrategy) ImagePullSuccessCh() <-chan string {
	return kpps.imagePullSuccessCh
}

func (kpps *KubernetesPodPullStrategy) ImagePullErrorCh() <-chan string {
	return kpps.imagePullErrorCh
}

func podMatchesOwnerReference(pod *coreapiv1.Pod, ownerRef v1.OwnerReference) bool {
	for _, or := range pod.OwnerReferences {
		if or.APIVersion == ownerRef.APIVersion && or.Kind == ownerRef.Kind && or.Name == ownerRef.Name && or.UID == ownerRef.UID {
			return true
		}
	}

	return false
}

func podImagePullError(pod *coreapiv1.Pod) error {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == "main" {
			if cs.State.Waiting != nil && cs.State.Waiting.Reason == "ErrImagePull" {
				return fmt.Errorf(cs.State.Waiting.Message)
			}
		}
	}

	return nil
}

func podImagePullSucceeded(pod *coreapiv1.Pod) bool {
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.Name == "main" {
			if cs.State.Terminated != nil && cs.State.Terminated.ExitCode == 0 {
				return true
			}
		}
	}

	return false
}

func getImageFromPod(pod *coreapiv1.Pod) string {
	for _, c := range pod.Spec.Containers {
		if c.Name == "main" {
			return c.Image
		}
	}

	return ""
}

func (kpps *KubernetesPodPullStrategy) cleanupPod(ctx context.Context, pod *coreapiv1.Pod) error {
	return kpps.Client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, v1.DeleteOptions{})
}

func (kpps *KubernetesPodPullStrategy) handlePodEvent(ctx context.Context, pod *coreapiv1.Pod) {
	image := getImageFromPod(pod)
	l := logrus.WithFields(logrus.Fields{
		"image": image,
		"pod":   pod.Name,
		"node":  kpps.NodeName,
	})

	if err := podImagePullError(pod); err != nil {
		l.Errorf("image pull failed: %v", err)
		kpps.imagePullErrorCh <- image
	} else if podImagePullSucceeded(pod) {
		l.Info("image pull succeeded")
		kpps.imagePullSuccessCh <- image
	} else {
		l.Fatal("pull neither failed nor succeeded, this should be impossible")
	}

	if err := kpps.cleanupPod(ctx, pod); err != nil {
		l.Errorf("failed to delete pod: %v", err)
	} else {
		l.Infof("pod deleted")
	}
}

func (kpps *KubernetesPodPullStrategy) MonitorPods(ctx context.Context) {
	f := informers.NewSharedInformerFactoryWithOptions(kpps.Client, time.Minute*20, informers.WithNamespace(kpps.Namespace), informers.WithTweakListOptions(func(lo *v1.ListOptions) {
		lo.LabelSelector = fields.SelectorFromSet(fields.Set{
			"part-of": "image-cache-daemon",
		}).String()

	}))

	informer := f.Core().V1().Pods().Informer()

	informer.AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: func(obj interface{}) bool {
			pod := obj.(*coreapiv1.Pod)
			return podMatchesOwnerReference(pod, kpps.OwnerReference) && (podImagePullError(pod) != nil || podImagePullSucceeded(pod)) && pod.DeletionTimestamp == nil
		},
		Handler: cache.ResourceEventHandlerFuncs{
			AddFunc: func(obj interface{}) {
				kpps.handlePodEvent(ctx, obj.(*coreapiv1.Pod))
			},
			UpdateFunc: func(_, obj interface{}) {
				kpps.handlePodEvent(ctx, obj.(*coreapiv1.Pod))
			},
		},
	})

	informer.Run(ctx.Done())
}

func (kpps *KubernetesPodPullStrategy) PullImage(ctx context.Context, image string) error {
	createdPod, err := kpps.Client.CoreV1().Pods(kpps.Namespace).Create(ctx, &coreapiv1.Pod{
		ObjectMeta: v1.ObjectMeta{
			GenerateName: kpps.PodName + "-",
			Namespace:    kpps.Namespace,
			OwnerReferences: []v1.OwnerReference{
				kpps.OwnerReference,
			},
			Labels: map[string]string{
				"part-of": "image-cache-daemon",
			},
		},
		Spec: coreapiv1.PodSpec{
			// TODO: Allow this to be passed
			// PriorityClassName: "",

			// TODO: How should users pass image pull secrets?
			// Maybe we copy them from the daemonset or this pod?  That ends up being non-portable
			// if we implement different strategies though.  It might also be useful to have access
			// to those secrets in order to read the image metadata to be able to determine whether or not
			// we need to re-pull an image if the digest changes (think tags like latest).
			// ImagePullSecrets: descriptor.ImagePullSecrets,
			InitContainers: []coreapiv1.Container{
				{
					Name:  "copy-warden",
					Image: kpps.WardenImage,
					VolumeMounts: []coreapiv1.VolumeMount{
						{
							Name:      "warden",
							MountPath: "/var/run/image-cache-daemon",
						},
					},
				},
			},
			Containers: []coreapiv1.Container{
				{
					Name:            "main",
					Image:           image,
					ImagePullPolicy: coreapiv1.PullAlways,
					// warden is a simple statically compiled binary that does absolutely nothing.
					// the idea behind it is by mounting it on an emptyDir and setting it as the entry
					// point for the image we're pulling, we can successfully exit without actually doing
					// anything except having the side effect of having pulled the image.
					Command: []string{"/var/run/image-cache-daemon/warden"},
					VolumeMounts: []coreapiv1.VolumeMount{
						{
							Name:      "warden",
							MountPath: "/var/run/image-cache-daemon",
							ReadOnly:  true,
						},
					},
				},
			},

			Volumes: []coreapiv1.Volume{
				{
					Name: "warden",
					VolumeSource: coreapiv1.VolumeSource{
						EmptyDir: &coreapiv1.EmptyDirVolumeSource{},
					},
				},
			},

			// Assign the Pod to the same node where our Daemonset pod is running in order to trigger
			// an image pull on that specific node.
			NodeName: kpps.NodeName,

			// TODO: Copy tolerations from the Daemonset pod?
			Tolerations:   []coreapiv1.Toleration{},
			RestartPolicy: coreapiv1.RestartPolicyNever,
		},
	}, v1.CreateOptions{})

	if err != nil {
		return err
	}

	logrus.WithFields(logrus.Fields{
		"image": image,
		"pod":   createdPod.ObjectMeta.Name,
		"node":  kpps.NodeName,
	}).Info("image pull started")

	return nil
}
