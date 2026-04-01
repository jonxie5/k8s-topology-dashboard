import { useCallback, useEffect, useState } from 'react';
import { fetchGraph, type Graph, type GraphNode } from './api';
import GraphCanvas from './components/GraphCanvas';
import SidePanel from './components/SidePanel';
import Toolbar from './components/Toolbar';

export default function App() {
  const [graph, setGraph] = useState<Graph | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(true);

  // Toolbar state
  const [q, setQ] = useState('');
  const [namespaces, setNamespaces] = useState('*');
  const [includePods, setIncludePods] = useState(false);

  // Selected node for side panel
  const [selectedNode, setSelectedNode] = useState<GraphNode | null>(null);

  const load = useCallback(() => {
    setLoading(true);
    setError(null);
    fetchGraph({ namespaces, q, includePods })
      .then(setGraph)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [namespaces, q, includePods]);

  // Initial load
  useEffect(() => { load(); }, [load]);

  const handleNodeClick = useCallback((node: GraphNode) => {
    setSelectedNode(node);
  }, []);

  return (
    <div style={{ width: '100vw', height: '100vh', background: '#1a1a2e', fontFamily: 'Inter, system-ui, sans-serif' }}>
      <Toolbar
        value={q}
        onChange={setQ}
        includePods={includePods}
        onIncludePodsChange={setIncludePods}
        namespaces={namespaces}
        onNamespacesChange={setNamespaces}
        onRefresh={load}
        truncated={graph?.meta.truncated ?? false}
        nodeCount={graph?.meta.counts.nodes ?? 0}
        edgeCount={graph?.meta.counts.edges ?? 0}
      />

      {/* Graph area (below toolbar) */}
      <div style={{ position: 'absolute', top: '52px', left: 0, right: 0, bottom: 0 }}>
        {loading && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#9ab', fontSize: '16px' }}>
            Loading cluster topology…
          </div>
        )}
        {error && (
          <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'center', height: '100%', color: '#e74c3c', fontSize: '16px', padding: '20px', textAlign: 'center' }}>
            ⚠ {error}
          </div>
        )}
        {!loading && !error && graph && (
          <GraphCanvas graph={graph} onNodeClick={handleNodeClick} />
        )}
      </div>

      <SidePanel node={selectedNode} onClose={() => setSelectedNode(null)} />
    </div>
  );
}
