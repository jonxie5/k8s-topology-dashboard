// Package graph builds a topology graph from live Kubernetes objects.
package graph

import (
	"context"
	"fmt"
	"strings"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// BuildOptions controls what gets included in the graph.
type BuildOptions struct {
	Namespaces  []string // empty / ["*"] means all namespaces
	Q           string   // substring filter on node name (case-insensitive)
	IncludePods bool     // when true, emit real Pod nodes instead of PodSet
	MaxNodes    int
	MaxEdges    int
}

// Builder constructs topology graphs from a Kubernetes cluster.
type Builder struct {
	client kubernetes.Interface
}

// NewBuilder returns a Builder backed by the given clientset.
func NewBuilder(client kubernetes.Interface) *Builder {
	return &Builder{client: client}
}

// Build fetches cluster resources and returns a topology graph.
func (b *Builder) Build(ctx context.Context, opts BuildOptions) (*Graph, error) {
	if opts.MaxNodes <= 0 {
		opts.MaxNodes = 500
	}
	if opts.MaxEdges <= 0 {
		opts.MaxEdges = 2000
	}

	// Resolve namespace list.
	namespaces, err := b.resolveNamespaces(ctx, opts.Namespaces)
	if err != nil {
		return nil, err
	}

	g := &graph{
		nodes:    make(map[string]Node),
		edges:    make(map[string]Edge),
		maxNodes: opts.MaxNodes,
		maxEdges: opts.MaxEdges,
	}

	// ── Cluster-scoped resources ─────────────────────────────────────────────

	// Kubernetes Nodes
	nodeList, err := b.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}
	for i := range nodeList.Items {
		addNode(g, kubeNodeToNode(&nodeList.Items[i]))
	}

	// Namespaces
	nsList, err := b.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing namespaces: %w", err)
	}
	for i := range nsList.Items {
		addNode(g, namespaceToNode(&nsList.Items[i]))
	}

	// ── Namespaced resources ─────────────────────────────────────────────────
	for _, ns := range namespaces {
		// Deployments
		deploys, err := b.client.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing deployments in %s: %w", ns, err)
		}
		for i := range deploys.Items {
			addNode(g, deploymentToNode(&deploys.Items[i]))
		}

		// ReplicaSets
		rsets, err := b.client.AppsV1().ReplicaSets(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing replicasets in %s: %w", ns, err)
		}
		for i := range rsets.Items {
			addNode(g, replicaSetToNode(&rsets.Items[i]))
		}

		// Services
		svcs, err := b.client.CoreV1().Services(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing services in %s: %w", ns, err)
		}
		for i := range svcs.Items {
			addNode(g, serviceToNode(&svcs.Items[i]))
		}

		// Ingresses
		ings, err := b.client.NetworkingV1().Ingresses(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing ingresses in %s: %w", ns, err)
		}
		for i := range ings.Items {
			addNode(g, ingressToNode(&ings.Items[i]))
		}

		// Pods
		pods, err := b.client.CoreV1().Pods(ns).List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing pods in %s: %w", ns, err)
		}
		if opts.IncludePods {
			for i := range pods.Items {
				addNode(g, podToNode(&pods.Items[i]))
			}
		} else {
			addPodSets(g, pods.Items)
		}

		// ── Edges ────────────────────────────────────────────────────────────

		// Deployment → ReplicaSet (ownerReference)
		for i := range rsets.Items {
			rs := &rsets.Items[i]
			for _, ref := range rs.OwnerReferences {
				if ref.Kind == "Deployment" {
					fromID := nodeID("apps", "v1", "Deployment", rs.Namespace, ref.Name)
					toID := nodeID("apps", "v1", "ReplicaSet", rs.Namespace, rs.Name)
					addEdge(g, Edge{
						ID:     edgeID(fromID, toID, "owns"),
						From:   fromID,
						To:     toID,
						Type:   "owns",
						Reason: "ownerReference",
						UI:     EdgeUI{Style: "solid"},
					})
				}
			}
		}

		// ReplicaSet → Pod / PodSet (ownerReference)
		for i := range pods.Items {
			pod := &pods.Items[i]
			for _, ref := range pod.OwnerReferences {
				if ref.Kind == "ReplicaSet" {
					fromID := nodeID("apps", "v1", "ReplicaSet", pod.Namespace, ref.Name)
					var toID string
					if opts.IncludePods {
						toID = nodeID("", "v1", "Pod", pod.Namespace, pod.Name)
					} else {
						toID = podSetID(pod.Namespace, ref.Name)
					}
					addEdge(g, Edge{
						ID:     edgeID(fromID, toID, "owns"),
						From:   fromID,
						To:     toID,
						Type:   "owns",
						Reason: "ownerReference",
						UI:     EdgeUI{Style: "solid"},
					})
				}
			}
		}

		// Service → Pod / PodSet (labelSelector)
		for i := range svcs.Items {
			svc := &svcs.Items[i]
			if len(svc.Spec.Selector) == 0 {
				continue
			}
			sel := labels.Set(svc.Spec.Selector).AsSelector()
			svcID := nodeID("", "v1", "Service", svc.Namespace, svc.Name)

			if opts.IncludePods {
				for j := range pods.Items {
					pod := &pods.Items[j]
					if sel.Matches(labels.Set(pod.Labels)) {
						toID := nodeID("", "v1", "Pod", pod.Namespace, pod.Name)
						addEdge(g, Edge{
							ID:     edgeID(svcID, toID, "selects"),
							From:   svcID,
							To:     toID,
							Type:   "selects",
							Reason: "labelSelector",
							Meta:   map[string]any{"selector": svc.Spec.Selector},
							UI:     EdgeUI{Style: "dashed"},
						})
					}
				}
			} else {
				// Connect to PodSets whose labels match the selector.
				added := map[string]bool{}
				for j := range pods.Items {
					pod := &pods.Items[j]
					if !sel.Matches(labels.Set(pod.Labels)) {
						continue
					}
					// Find owning ReplicaSet name to derive PodSet ID.
					psID := podSetIDForPod(pod)
					if psID == "" || added[psID] {
						continue
					}
					added[psID] = true
					addEdge(g, Edge{
						ID:     edgeID(svcID, psID, "selects"),
						From:   svcID,
						To:     psID,
						Type:   "selects",
						Reason: "labelSelector",
						Meta:   map[string]any{"selector": svc.Spec.Selector},
						UI:     EdgeUI{Style: "dashed"},
					})
				}
			}
		}

		// Ingress → Service (backend reference)
		for i := range ings.Items {
			ing := &ings.Items[i]
			ingID := nodeID("networking.k8s.io", "v1", "Ingress", ing.Namespace, ing.Name)
			for _, rule := range ing.Spec.Rules {
				if rule.HTTP == nil {
					continue
				}
				for _, path := range rule.HTTP.Paths {
					if path.Backend.Service == nil {
						continue
					}
					toID := nodeID("", "v1", "Service", ing.Namespace, path.Backend.Service.Name)
					addEdge(g, Edge{
						ID:     edgeID(ingID, toID, "routes"),
						From:   ingID,
						To:     toID,
						Type:   "routes",
						Reason: "backendRef",
						UI:     EdgeUI{Style: "solid"},
					})
				}
			}
		}

		// Pod → Node (spec.nodeName) – only when includePods=true
		if opts.IncludePods {
			for i := range pods.Items {
				pod := &pods.Items[i]
				if pod.Spec.NodeName == "" {
					continue
				}
				fromID := nodeID("", "v1", "Pod", pod.Namespace, pod.Name)
				toID := nodeID("", "v1", "Node", "", pod.Spec.NodeName)
				addEdge(g, Edge{
					ID:     edgeID(fromID, toID, "schedules"),
					From:   fromID,
					To:     toID,
					Type:   "schedules",
					Reason: "nodeName",
					UI:     EdgeUI{Style: "dashed"},
				})
			}
		}
	}

	// ── Apply name filter ────────────────────────────────────────────────────
	q := strings.ToLower(opts.Q)
	if q != "" {
		for id, n := range g.nodes {
			if !strings.Contains(strings.ToLower(n.Name), q) &&
				!strings.Contains(strings.ToLower(n.Namespace), q) {
				delete(g.nodes, id)
			}
		}
		// Remove edges that reference deleted nodes.
		for id, e := range g.edges {
			if _, ok := g.nodes[e.From]; !ok {
				delete(g.edges, id)
				continue
			}
			if _, ok := g.nodes[e.To]; !ok {
				delete(g.edges, id)
			}
		}
	}

	// ── Build response ───────────────────────────────────────────────────────
	truncated := false
	nodes := make([]Node, 0, len(g.nodes))
	for _, n := range g.nodes {
		nodes = append(nodes, n)
	}
	if len(nodes) > opts.MaxNodes {
		nodes = nodes[:opts.MaxNodes]
		truncated = true
	}

	// Build a set of valid node IDs after truncation.
	validIDs := make(map[string]bool, len(nodes))
	for _, n := range nodes {
		validIDs[n.ID] = true
	}

	edges := make([]Edge, 0, len(g.edges))
	for _, e := range g.edges {
		if !validIDs[e.From] || !validIDs[e.To] {
			continue
		}
		edges = append(edges, e)
	}
	if len(edges) > opts.MaxEdges {
		edges = edges[:opts.MaxEdges]
		truncated = true
	}

	nsFilter := opts.Namespaces
	if len(nsFilter) == 0 {
		nsFilter = []string{"*"}
	}

	return &Graph{
		Meta: Meta{
			SnapshotTime: time.Now().UTC(),
			SnapshotID:   fmt.Sprintf("%d", time.Now().UnixNano()),
			Truncated:    truncated,
			Counts:       Counts{Nodes: len(nodes), Edges: len(edges)},
			Filters: Filters{
				Namespaces:  nsFilter,
				Q:           opts.Q,
				IncludePods: opts.IncludePods,
			},
		},
		Nodes: nodes,
		Edges: edges,
	}, nil
}

// ── internal graph accumulator ───────────────────────────────────────────────

type graph struct {
	nodes    map[string]Node
	edges    map[string]Edge
	maxNodes int
	maxEdges int
}

func addNode(g *graph, n Node) {
	g.nodes[n.ID] = n
}

func addEdge(g *graph, e Edge) {
	g.edges[e.ID] = e
}

// ── helpers ──────────────────────────────────────────────────────────────────

// nodeID returns a stable, globally unique node ID.
// Format: group/version/Kind:namespace/name  (namespace empty for cluster-scoped)
func nodeID(group, version, kind, namespace, name string) string {
	gv := version
	if group != "" {
		gv = group + "/" + version
	}
	ns := namespace
	if ns != "" {
		ns = namespace + "/"
	}
	return fmt.Sprintf("%s/%s:%s%s", gv, kind, ns, name)
}

func podSetID(namespace, ownerName string) string {
	return fmt.Sprintf("internal/v1/PodSet:%s/%s", namespace, ownerName)
}

func podSetIDForPod(pod *corev1.Pod) string {
	for _, ref := range pod.OwnerReferences {
		if ref.Kind == "ReplicaSet" {
			return podSetID(pod.Namespace, ref.Name)
		}
	}
	return ""
}

func edgeID(from, to, t string) string {
	return fmt.Sprintf("%s--%s--%s", from, t, to)
}

// resolveNamespaces expands ["*"] or [] to the full list of namespace names.
func (b *Builder) resolveNamespaces(ctx context.Context, requested []string) ([]string, error) {
	if len(requested) == 0 || (len(requested) == 1 && requested[0] == "*") {
		list, err := b.client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
		if err != nil {
			return nil, fmt.Errorf("listing namespaces: %w", err)
		}
		out := make([]string, len(list.Items))
		for i, ns := range list.Items {
			out[i] = ns.Name
		}
		return out, nil
	}
	return requested, nil
}

// addPodSets groups pods by their owning ReplicaSet (or standalone) and creates
// a single PodSet virtual node per group.
func addPodSets(g *graph, pods []corev1.Pod) {
	type podSetKey struct{ namespace, owner string }
	type podSetStats struct {
		ready    int
		total    int
		restarts int
	}
	groups := map[podSetKey]*podSetStats{}
	ownerKinds := map[podSetKey]string{}

	for i := range pods {
		pod := &pods[i]
		var ownerName string
		var ownerKind string
		for _, ref := range pod.OwnerReferences {
			if ref.Kind == "ReplicaSet" || ref.Kind == "StatefulSet" || ref.Kind == "DaemonSet" {
				ownerName = ref.Name
				ownerKind = ref.Kind
				break
			}
		}
		if ownerName == "" {
			ownerName = pod.Name // standalone pod
			ownerKind = "Pod"
		}
		key := podSetKey{pod.Namespace, ownerName}
		if groups[key] == nil {
			groups[key] = &podSetStats{}
			ownerKinds[key] = ownerKind
		}
		groups[key].total++
		for _, cs := range pod.Status.ContainerStatuses {
			groups[key].restarts += int(cs.RestartCount)
		}
		if pod.Status.Phase == corev1.PodRunning {
			ready := true
			for _, cs := range pod.Status.ContainerStatuses {
				if !cs.Ready {
					ready = false
					break
				}
			}
			if ready {
				groups[key].ready++
			}
		}
	}

	for key, stats := range groups {
		id := podSetID(key.namespace, key.owner)
		_ = ownerKinds[key]
		addNode(g, Node{
			ID:         id,
			Kind:       "PodSet",
			APIVersion: "internal/v1",
			Namespace:  key.namespace,
			Name:       key.owner,
			Labels:     map[string]string{},
			Status: map[string]any{
				"pods":      stats.total,
				"readyPods": stats.ready,
				"restarts":  stats.restarts,
			},
			Tags: []string{"pods", "collapsed"},
			UI:   NodeUI{Group: "pods", Collapsed: true},
		})
	}
}

// ── converters ───────────────────────────────────────────────────────────────

func kubeNodeToNode(n *corev1.Node) Node {
	ready := "Unknown"
	for _, c := range n.Status.Conditions {
		if c.Type == corev1.NodeReady {
			if c.Status == corev1.ConditionTrue {
				ready = "Ready"
			} else {
				ready = "NotReady"
			}
		}
	}
	return Node{
		ID:         nodeID("", "v1", "Node", "", n.Name),
		Kind:       "Node",
		APIVersion: "v1",
		Namespace:  "",
		Name:       n.Name,
		Labels:     n.Labels,
		Status:     map[string]any{"ready": ready},
		Tags:       []string{"infrastructure"},
		UI:         NodeUI{Group: "nodes"},
	}
}

func namespaceToNode(ns *corev1.Namespace) Node {
	return Node{
		ID:         nodeID("", "v1", "Namespace", "", ns.Name),
		Kind:       "Namespace",
		APIVersion: "v1",
		Namespace:  "",
		Name:       ns.Name,
		Labels:     ns.Labels,
		Status:     map[string]any{"phase": string(ns.Status.Phase)},
		Tags:       []string{"namespace"},
		UI:         NodeUI{Group: "namespaces"},
	}
}

func deploymentToNode(d *appsv1.Deployment) Node {
	return Node{
		ID:         nodeID("apps", "v1", "Deployment", d.Namespace, d.Name),
		Kind:       "Deployment",
		APIVersion: "apps/v1",
		Namespace:  d.Namespace,
		Name:       d.Name,
		Labels:     d.Labels,
		Status: map[string]any{
			"replicas":      d.Status.Replicas,
			"readyReplicas": d.Status.ReadyReplicas,
		},
		Tags: []string{"workload"},
		UI:   NodeUI{Group: "workloads"},
	}
}

func replicaSetToNode(rs *appsv1.ReplicaSet) Node {
	return Node{
		ID:         nodeID("apps", "v1", "ReplicaSet", rs.Namespace, rs.Name),
		Kind:       "ReplicaSet",
		APIVersion: "apps/v1",
		Namespace:  rs.Namespace,
		Name:       rs.Name,
		Labels:     rs.Labels,
		Status: map[string]any{
			"replicas":      rs.Status.Replicas,
			"readyReplicas": rs.Status.ReadyReplicas,
		},
		Tags: []string{"workload"},
		UI:   NodeUI{Group: "workloads"},
	}
}

func serviceToNode(svc *corev1.Service) Node {
	return Node{
		ID:         nodeID("", "v1", "Service", svc.Namespace, svc.Name),
		Kind:       "Service",
		APIVersion: "v1",
		Namespace:  svc.Namespace,
		Name:       svc.Name,
		Labels:     svc.Labels,
		Status:     map[string]any{"type": string(svc.Spec.Type)},
		Tags:       []string{"service"},
		UI:         NodeUI{Group: "services"},
	}
}

func ingressToNode(ing *networkingv1.Ingress) Node {
	return Node{
		ID:         nodeID("networking.k8s.io", "v1", "Ingress", ing.Namespace, ing.Name),
		Kind:       "Ingress",
		APIVersion: "networking.k8s.io/v1",
		Namespace:  ing.Namespace,
		Name:       ing.Name,
		Labels:     ing.Labels,
		Status:     map[string]any{},
		Tags:       []string{"ingress"},
		UI:         NodeUI{Group: "ingresses"},
	}
}

func podToNode(pod *corev1.Pod) Node {
	restarts := 0
	for _, cs := range pod.Status.ContainerStatuses {
		restarts += int(cs.RestartCount)
	}
	return Node{
		ID:         nodeID("", "v1", "Pod", pod.Namespace, pod.Name),
		Kind:       "Pod",
		APIVersion: "v1",
		Namespace:  pod.Namespace,
		Name:       pod.Name,
		Labels:     pod.Labels,
		Status: map[string]any{
			"phase":    string(pod.Status.Phase),
			"restarts": restarts,
		},
		Tags: []string{"pod"},
		UI:   NodeUI{Group: "pods"},
	}
}
