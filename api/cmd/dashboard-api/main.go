package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/jonxie5/k8s-topology-dashboard/api/internal/handler"
	"github.com/jonxie5/k8s-topology-dashboard/api/internal/kube"
)

func main() {
	var (
		addr           = flag.String("addr", envOrDefault("ADDR", ":8080"), "listen address")
		kubeconfigPath = flag.String("kubeconfig", os.Getenv("KUBECONFIG"), "path to kubeconfig file")
		kubeContext    = flag.String("context", "", "kubeconfig context to use")
		corsOrigin     = flag.String("cors-origin", os.Getenv("CORS_ORIGIN"), "allowed CORS origin (e.g. http://localhost:5173)")
		maxNodes       = flag.Int("max-nodes", envOrDefaultInt("MAX_NODES", 500), "maximum nodes per response")
		maxEdges       = flag.Int("max-edges", envOrDefaultInt("MAX_EDGES", 2000), "maximum edges per response")
	)
	flag.Parse()

	client, err := kube.NewClient(kube.ClientOptions{
		KubeconfigPath: *kubeconfigPath,
		Context:        *kubeContext,
	})
	if err != nil {
		log.Fatalf("ERROR: %v\n\nHint: set KUBECONFIG env var or run inside a Kubernetes cluster.", err)
	}

	mux := http.NewServeMux()
	api := handler.New(client, *corsOrigin, *maxNodes, *maxEdges)
	api.Register(mux)

	log.Printf("Starting k8s-topology-dashboard API on %s (maxNodes=%d maxEdges=%d)", *addr, *maxNodes, *maxEdges)
	if err := http.ListenAndServe(*addr, mux); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envOrDefaultInt(key string, defaultVal int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return defaultVal
}
