/*
Copyright © 2021 Daniel Herman <daniel.c.herman@gmail.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	argoclientset "github.com/argoproj/argo-workflows/v3/pkg/client/clientset/versioned"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/dcherman/image-cache-daemon/puller"
	"github.com/dcherman/image-cache-daemon/source"
	"github.com/dcherman/image-cache-daemon/strategy"
)

func NewImageCacheDaemonCommand() *cobra.Command {
	var (
		images            []string
		configmapSelector string
		nodeName          string
		podName           string
		podUUID           string
		podNamespace      string

		wardenImage                       string
		watchArgoWorkflowTemplates        bool
		watchArgoClusterWorkflowTemplates bool
		watchArgoCronWorkflows            bool
		watchConfigMaps                   bool
		resyncPeriod                      time.Duration
	)

	var rootCmd = &cobra.Command{
		Use: "image-cache-daemon",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(context.Background())

			loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
			loadingRules.DefaultClientConfig = &clientcmd.DefaultClientConfig
			overrides := clientcmd.ConfigOverrides{}

			clientConfig := clientcmd.NewInteractiveDeferredLoadingClientConfig(loadingRules, &overrides, os.Stdin)
			config, err := clientConfig.ClientConfig()

			if err != nil {
				panic(err)
			}

			kubeclient := kubernetes.NewForConfigOrDie(config)
			argoclient := argoclientset.NewForConfigOrDie(config)

			if err != nil {
				panic(err)
			}

			strat := strategy.NewKubernetesPodPullStrategy(&strategy.KubernetesPodPullStrategyOpts{
				Client:      kubeclient,
				NodeName:    nodeName,
				Namespace:   podNamespace,
				PodName:     podName,
				WardenImage: wardenImage,
				OwnerReference: v1.OwnerReference{
					APIVersion: "v1",
					Kind:       "Pod",
					Name:       podName,
					UID:        types.UID(podUUID),
				},
			})

			go strat.MonitorPods(ctx)

			ip := puller.NewImagePuller(strat, kubeclient, podNamespace, podName)

			if len(images) > 0 {
				staticSource := source.NewStaticImageSource(images, 0)
				ip.AddSource(ctx, staticSource)
				go staticSource.Run(ctx)
			}

			if watchArgoWorkflowTemplates {
				logrus.Info("watching workflow templates for images to pull")

				workflowTemplateSource := source.NewWorkflowTemplateSource(argoclient, resyncPeriod)
				ip.AddSource(ctx, workflowTemplateSource)
				go workflowTemplateSource.Run(ctx)
			}

			if watchArgoClusterWorkflowTemplates {
				logrus.Info("watching cluster workflow templates for images to pull")
				workflowTemplateSource := source.NewClusterWorkflowTemplateSource(argoclient, resyncPeriod)
				ip.AddSource(ctx, workflowTemplateSource)
				go workflowTemplateSource.Run(ctx)
			}

			if watchArgoCronWorkflows {
				logrus.Info("watching cron workflows for images to pull")
				workflowTemplateSource := source.NewCronWorkflowTemplateSource(argoclient, resyncPeriod)
				ip.AddSource(ctx, workflowTemplateSource)
				go workflowTemplateSource.Run(ctx)
			}

			if watchConfigMaps {
				logrus.Info("watching configmaps for images to pull")
				configmapSource := source.NewConfigMapSource(kubeclient, resyncPeriod, source.WithConfigMapSelector(configmapSelector))
				ip.AddSource(ctx, configmapSource)
				go configmapSource.Run(ctx)
			}

			go ip.Run(ctx)

			stopCh := make(chan os.Signal, 1)
			signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM)

			go func() {
				<-stopCh
				log.Print("SIGINT/SIGTERM Received, shutting down...")
				cancel()
			}()

			<-ctx.Done()
		},
	}

	rootCmd.Flags().StringArrayVar(&images, "image", []string{}, "Images that should be pre-fetched")
	rootCmd.Flags().StringVar(&nodeName, "node-name", os.Getenv("POD_NODE_NAME"), "The node name to pull to")
	rootCmd.Flags().StringVar(&podName, "pod-name", os.Getenv("POD_NAME"), "The pod name")
	rootCmd.Flags().StringVar(&podUUID, "pod-uid", os.Getenv("POD_UUD"), "The owning pod UID")
	rootCmd.Flags().StringVar(&podNamespace, "pod-namespace", os.Getenv("POD_NAMESPACE"), "The namespace this pod is running in")
	rootCmd.Flags().StringVar(&wardenImage, "warden-image", "exiges/image-cache-warden:latest", "The image that copies a binary to pulled containers to replace the entrypoint")
	rootCmd.Flags().StringVar(&configmapSelector, "configmap-selector", "app.kubernetes.io/part-of=image-cache-daemon", "The selector to use when monitoring for ConfigMap sources")
	rootCmd.Flags().BoolVar(&watchArgoWorkflowTemplates, "watch-argo-workflow-templates", true, "Whether or not to watch workflow templates")
	rootCmd.Flags().BoolVar(&watchArgoClusterWorkflowTemplates, "watch-argo-cluster-workflow-templates", true, "Whether or not to watch cluster workflow templates")
	rootCmd.Flags().BoolVar(&watchArgoCronWorkflows, "watch-argo-cron-workflows", true, "Whether or not to watch cron workflows")
	rootCmd.Flags().BoolVar(&watchConfigMaps, "watch-configmaps", true, "Whether or not to watch ConfigMaps for images to pull.  Must match the --config-map-selector")
	rootCmd.Flags().DurationVar(&resyncPeriod, "resync-period", time.Minute*15, "How often the daemon should re-pull images from all of the sources.  Set to 0 to disable.")

	return rootCmd
}
