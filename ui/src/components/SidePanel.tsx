import { useEffect, useState } from 'react';
import { fetchObject, type GraphNode, type ObjectDetail } from '../api';

interface Props {
  node: GraphNode | null;
  onClose: () => void;
}

export default function SidePanel({ node, onClose }: Props) {
  const [detail, setDetail] = useState<ObjectDetail | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [yamlOpen, setYamlOpen] = useState(false);

  useEffect(() => {
    if (!node) {
      setDetail(null);
      setError(null);
      return;
    }
    setLoading(true);
    setError(null);
    setDetail(null);
    setYamlOpen(false);
    fetchObject(node.kind, node.namespace, node.name)
      .then(setDetail)
      .catch((e: Error) => setError(e.message))
      .finally(() => setLoading(false));
  }, [node]);

  if (!node) return null;

  return (
    <aside
      style={{
        position: 'fixed',
        top: 0,
        right: 0,
        width: '380px',
        height: '100vh',
        background: '#16213e',
        color: '#e0e0e0',
        boxShadow: '-4px 0 16px rgba(0,0,0,0.6)',
        display: 'flex',
        flexDirection: 'column',
        zIndex: 100,
        overflowY: 'auto',
      }}
    >
      {/* Header */}
      <div
        style={{
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          padding: '12px 16px',
          background: '#0f3460',
          borderBottom: '1px solid #1a4a80',
          flexShrink: 0,
        }}
      >
        <span style={{ fontWeight: 700, fontSize: '15px' }}>
          {node.kind}: {node.name}
        </span>
        <button
          onClick={onClose}
          style={{
            background: 'none',
            border: 'none',
            color: '#e0e0e0',
            fontSize: '20px',
            cursor: 'pointer',
            lineHeight: 1,
          }}
          aria-label="Close panel"
        >
          ×
        </button>
      </div>

      {/* Content */}
      <div style={{ padding: '16px', flex: 1 }}>
        {/* Basic info */}
        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
          <tbody>
            {[
              ['Kind', node.kind],
              ['API Version', node.apiVersion],
              ['Namespace', node.namespace || '(cluster-scoped)'],
              ['Name', node.name],
            ].map(([k, v]) => (
              <tr key={k} style={{ borderBottom: '1px solid #1a4a80' }}>
                <td style={{ padding: '6px 4px', color: '#9ab', fontWeight: 600, whiteSpace: 'nowrap' }}>
                  {k}
                </td>
                <td style={{ padding: '6px 4px' }}>{v}</td>
              </tr>
            ))}
          </tbody>
        </table>

        {/* Status */}
        {Object.keys(node.status).length > 0 && (
          <section style={{ marginTop: '16px' }}>
            <h3 style={{ fontSize: '13px', color: '#9ab', marginBottom: '8px' }}>Status</h3>
            <pre
              style={{
                background: '#0d1b2a',
                padding: '10px',
                borderRadius: '6px',
                fontSize: '12px',
                overflow: 'auto',
                margin: 0,
              }}
            >
              {JSON.stringify(node.status, null, 2)}
            </pre>
          </section>
        )}

        {/* Labels */}
        {Object.keys(node.labels ?? {}).length > 0 && (
          <section style={{ marginTop: '16px' }}>
            <h3 style={{ fontSize: '13px', color: '#9ab', marginBottom: '8px' }}>Labels</h3>
            <div style={{ display: 'flex', flexWrap: 'wrap', gap: '6px' }}>
              {Object.entries(node.labels).map(([k, v]) => (
                <span
                  key={k}
                  style={{
                    background: '#0f3460',
                    borderRadius: '4px',
                    padding: '2px 8px',
                    fontSize: '11px',
                  }}
                >
                  {k}={v}
                </span>
              ))}
            </div>
          </section>
        )}

        {/* YAML */}
        <section style={{ marginTop: '16px' }}>
          <button
            onClick={() => setYamlOpen((o) => !o)}
            style={{
              background: '#0f3460',
              border: '1px solid #1a4a80',
              color: '#e0e0e0',
              padding: '6px 14px',
              borderRadius: '4px',
              cursor: 'pointer',
              fontSize: '13px',
            }}
          >
            {yamlOpen ? '▼ Hide YAML' : '▶ Show YAML'}
          </button>
          {loading && <p style={{ color: '#9ab', fontSize: '13px' }}>Loading…</p>}
          {error && <p style={{ color: '#e74c3c', fontSize: '13px' }}>{error}</p>}
          {yamlOpen && detail && (
            <pre
              style={{
                background: '#0d1b2a',
                padding: '10px',
                borderRadius: '6px',
                fontSize: '11px',
                overflow: 'auto',
                marginTop: '8px',
                maxHeight: '400px',
              }}
            >
              {detail.yaml}
            </pre>
          )}
        </section>
      </div>
    </aside>
  );
}
