package kube

import (
	"fmt"
	"os"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// ClientOptions controls how the Kubernetes client is configured.
type ClientOptions struct {
	KubeconfigPath string
	Context        string
}

// NewClient builds a Kubernetes clientset using the following strategy:
//  1. If KubeconfigPath is set OR the KUBECONFIG env var is present, use kubeconfig.
//  2. Else try in-cluster config.
//  3. Else return a clear error.
func NewClient(opts ClientOptions) (kubernetes.Interface, error) {
	cfg, err := buildConfig(opts)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(cfg)
}

func buildConfig(opts ClientOptions) (*rest.Config, error) {
	kubeconfigPath := opts.KubeconfigPath
	if kubeconfigPath == "" {
		kubeconfigPath = os.Getenv("KUBECONFIG")
	}

	if kubeconfigPath != "" {
		loadingRules := &clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath}
		configOverrides := &clientcmd.ConfigOverrides{}
		if opts.Context != "" {
			configOverrides.CurrentContext = opts.Context
		}
		cfg, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			loadingRules, configOverrides,
		).ClientConfig()
		if err != nil {
			return nil, fmt.Errorf("kubeconfig %q: %w", kubeconfigPath, err)
		}
		return cfg, nil
	}

	cfg, err := rest.InClusterConfig()
	if err == nil {
		return cfg, nil
	}

	return nil, fmt.Errorf(
		"no Kubernetes config found: provide --kubeconfig / KUBECONFIG env var, or run inside a cluster",
	)
}
