const API_BASE = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080';

export interface NodeUI {
  group: string;
  collapsed: boolean;
}

export interface GraphNode {
  id: string;
  kind: string;
  apiVersion: string;
  namespace: string;
  name: string;
  labels: Record<string, string>;
  status: Record<string, unknown>;
  tags: string[];
  ui: NodeUI;
}

export interface EdgeUI {
  style: string;
}

export interface GraphEdge {
  id: string;
  from: string;
  to: string;
  type: string;
  reason: string;
  meta?: Record<string, unknown>;
  ui: EdgeUI;
}

export interface GraphMeta {
  snapshotTime: string;
  snapshotId: string;
  truncated: boolean;
  counts: { nodes: number; edges: number };
  filters: { namespaces: string[]; q: string; includePods: boolean };
}

export interface Graph {
  meta: GraphMeta;
  nodes: GraphNode[];
  edges: GraphEdge[];
}

export interface ObjectDetail {
  kind: string;
  namespace: string;
  name: string;
  yaml: string;
  object: unknown;
}

export async function fetchGraph(params: {
  namespaces?: string;
  q?: string;
  includePods?: boolean;
}): Promise<Graph> {
  const search = new URLSearchParams();
  if (params.namespaces) search.set('namespaces', params.namespaces);
  if (params.q) search.set('q', params.q);
  if (params.includePods) search.set('includePods', 'true');

  const res = await fetch(`${API_BASE}/api/graph?${search.toString()}`);
  if (!res.ok) throw new Error(`Graph fetch failed: ${res.status}`);
  return res.json();
}

export async function fetchObject(
  kind: string,
  namespace: string,
  name: string,
): Promise<ObjectDetail> {
  const res = await fetch(
    `${API_BASE}/api/object/${encodeURIComponent(kind)}/${encodeURIComponent(namespace)}/${encodeURIComponent(name)}`,
  );
  if (!res.ok) throw new Error(`Object fetch failed: ${res.status}`);
  return res.json();
}
