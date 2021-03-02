package puller

import (
	"context"
	"log"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/dcherman/image-cache-daemon/source"
	"github.com/dcherman/image-cache-daemon/strategy"
)

type ImagePuller struct {
	strategy      strategy.PullStrategy
	kubeClient    kubernetes.Interface
	imageSourceCh chan string

	podNamespace string
	podName      string

	pendingImages map[string]bool
}

func NewImagePuller(strategy strategy.PullStrategy, kubeClient kubernetes.Interface, podNamespace, podName string) *ImagePuller {
	ip := ImagePuller{
		kubeClient:    kubeClient,
		strategy:      strategy,
		imageSourceCh: make(chan string),
		pendingImages: map[string]bool{},
		podNamespace:  podNamespace,
		podName:       podName,
	}

	return &ip
}

func (ip *ImagePuller) AddSource(ctx context.Context, src source.ImageSource) {
	imageCh := src.ImageCh()

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case image := <-imageCh:
				logrus.WithFields(logrus.Fields{
					"image":  image,
					"source": src.Name(),
				}).Info("image received")

				ip.imageSourceCh <- image
			}
		}
	}()
}

func (ip *ImagePuller) Run(ctx context.Context) {
	doneCh := ctx.Done()
	successCh := ip.strategy.ImagePullSuccessCh()
	errorCh := ip.strategy.ImagePullErrorCh()

	for {
		select {
		case <-doneCh:
			return
		case image := <-ip.imageSourceCh:
			if _, ok := ip.pendingImages[image]; !ok {
				ip.pendingImages[image] = true

				if err := ip.strategy.PullImage(ctx, image); err != nil {
					delete(ip.pendingImages, image)
					log.Print(err)
				}
			} else {
				logrus.WithField("image", image).Info("image pull is already pending, skipping")

				// TODO: Should we inspect what images already exist on a given node in order to avoid re-pulling
				// images?  We would need to inspect the image metadata in order to determine whether or not
				// the digest for a given tag has changed (if we were given a tag and not a digest).  If the tag
				// did not change, then running a pod anyway will have little effect and already does that check for us.
				// The biggest disadvantage of always running a pod is that we're temporarily consuming a spot on the node
				// for running a pod (nodes have a max number of pods that can run concurrently), and in the case of the aws-vpc
				// CNI plugin and maybe others, we're consuming an IP address for a short period of time and subsequently making
				// it go into a cooldown period, if that applies to the CNI.

				// For now, we'll always pull for simplicity, but this is potentially an area of improvement.
			}
		case successfulImage := <-successCh:
			logrus.WithField("image", successfulImage).Info("image successfully pulled")
			delete(ip.pendingImages, successfulImage)

			// The image locality scheduling plugin (enabled by default) already prefers nodes
			// that already have the image being referenced.  We might not need to use this hack
			// where we label ourselves and then use Pod Affinity to prefer scheduling on a node
			// with the image already pulled.
			// https://kubernetes.io/docs/reference/scheduling/config/#scheduling-plugins-1

			//if err := ip.LabelPodPostSuccess(ctx, successfulImage); err != nil {
			//logrus.Error(err)
			//}
		case erroredImage := <-errorCh:
			logrus.WithField("image", erroredImage).Info("failed to pull image")
			delete(ip.pendingImages, erroredImage)
		}
	}
}
