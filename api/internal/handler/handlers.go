package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/yaml"

	"github.com/jonxie5/k8s-topology-dashboard/api/internal/graph"
)

// API holds HTTP handlers for the topology dashboard.
type API struct {
	client   kubernetes.Interface
	builder  *graph.Builder
	cors     string // allowed origin for CORS (empty = same-origin only)
	maxNodes int
	maxEdges int
}

// New creates a new API handler.
func New(client kubernetes.Interface, corsOrigin string, maxNodes, maxEdges int) *API {
	return &API{
		client:   client,
		builder:  graph.NewBuilder(client),
		cors:     corsOrigin,
		maxNodes: maxNodes,
		maxEdges: maxEdges,
	}
}

// Register mounts all routes on mux.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("/healthz", a.wrap(a.handleHealthz))
	mux.HandleFunc("/api/graph", a.wrap(a.handleGraph))
	mux.HandleFunc("/api/object/", a.wrap(a.handleObject))
}

// wrap adds common middleware (CORS, timeout, JSON error recovery).
func (a *API) wrap(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// CORS
		if a.cors != "" {
			w.Header().Set("Access-Control-Allow-Origin", a.cors)
			w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Timeout
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		r = r.WithContext(ctx)

		h(w, r)
	}
}

// ── Handlers ─────────────────────────────────────────────────────────────────

func (a *API) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (a *API) handleGraph(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	q := r.URL.Query()

	nsParam := q.Get("namespaces")
	var namespaces []string
	if nsParam != "" && nsParam != "*" {
		namespaces = strings.Split(nsParam, ",")
	}

	includePods, _ := strconv.ParseBool(q.Get("includePods"))

	opts := graph.BuildOptions{
		Namespaces:  namespaces,
		Q:           q.Get("q"),
		IncludePods: includePods,
		MaxNodes:    a.maxNodes,
		MaxEdges:    a.maxEdges,
	}

	g, err := a.builder.Build(r.Context(), opts)
	if err != nil {
		writeError(w, http.StatusInternalServerError, fmt.Sprintf("building graph: %v", err))
		return
	}

	writeJSON(w, http.StatusOK, g)
}

func (a *API) handleObject(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Path: /api/object/{kind}/{namespace}/{name}
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/api/object/"), "/", 3)
	if len(parts) != 3 {
		writeError(w, http.StatusBadRequest, "path must be /api/object/{kind}/{namespace}/{name}")
		return
	}
	kind, namespace, name := parts[0], parts[1], parts[2]

	if strings.EqualFold(kind, "secret") {
		writeError(w, http.StatusBadRequest, "Secret retrieval is not supported")
		return
	}

	obj, err := fetchObject(r.Context(), a.client, kind, namespace, name)
	if err != nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("object not found: %v", err))
		return
	}

	yamlBytes, err := yaml.Marshal(obj)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "marshalling YAML")
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"kind":      kind,
		"namespace": namespace,
		"name":      name,
		"yaml":      string(yamlBytes),
		"object":    obj,
	})
}

// fetchObject retrieves a single Kubernetes object by kind/namespace/name.
func fetchObject(ctx context.Context, client kubernetes.Interface, kind, namespace, name string) (any, error) {
	opts := metav1.GetOptions{}
	switch strings.ToLower(kind) {
	case "namespace":
		return client.CoreV1().Namespaces().Get(ctx, name, opts)
	case "node":
		return client.CoreV1().Nodes().Get(ctx, name, opts)
	case "pod":
		return client.CoreV1().Pods(namespace).Get(ctx, name, opts)
	case "service":
		return client.CoreV1().Services(namespace).Get(ctx, name, opts)
	case "deployment":
		return client.AppsV1().Deployments(namespace).Get(ctx, name, opts)
	case "replicaset":
		return client.AppsV1().ReplicaSets(namespace).Get(ctx, name, opts)
	case "ingress":
		return client.NetworkingV1().Ingresses(namespace).Get(ctx, name, opts)
	default:
		return nil, fmt.Errorf("unsupported kind %q", kind)
	}
}

// ── helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
