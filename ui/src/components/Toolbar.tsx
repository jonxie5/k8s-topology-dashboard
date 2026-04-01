import { type ChangeEvent } from 'react';

interface Props {
  value: string;
  onChange: (q: string) => void;
  includePods: boolean;
  onIncludePodsChange: (v: boolean) => void;
  namespaces: string;
  onNamespacesChange: (v: string) => void;
  onRefresh: () => void;
  truncated: boolean;
  nodeCount: number;
  edgeCount: number;
}

export default function Toolbar({
  value,
  onChange,
  includePods,
  onIncludePodsChange,
  namespaces,
  onNamespacesChange,
  onRefresh,
  truncated,
  nodeCount,
  edgeCount,
}: Props) {
  return (
    <div
      style={{
        position: 'fixed',
        top: 0,
        left: 0,
        right: 0,
        height: '52px',
        background: '#0f3460',
        display: 'flex',
        alignItems: 'center',
        gap: '12px',
        padding: '0 16px',
        zIndex: 50,
        boxShadow: '0 2px 8px rgba(0,0,0,0.4)',
      }}
    >
      {/* Title */}
      <span
        style={{
          color: '#e0e0e0',
          fontWeight: 700,
          fontSize: '16px',
          whiteSpace: 'nowrap',
          marginRight: '8px',
        }}
      >
        K8s Topology
      </span>

      {/* Search */}
      <input
        type="search"
        placeholder="Search nodes…"
        value={value}
        onChange={(e: ChangeEvent<HTMLInputElement>) => onChange(e.target.value)}
        style={{
          flex: '1 1 180px',
          maxWidth: '240px',
          padding: '5px 10px',
          borderRadius: '6px',
          border: '1px solid #1a4a80',
          background: '#16213e',
          color: '#e0e0e0',
          fontSize: '13px',
          outline: 'none',
        }}
      />

      {/* Namespaces */}
      <input
        type="text"
        placeholder="Namespaces (* = all)"
        value={namespaces}
        onChange={(e: ChangeEvent<HTMLInputElement>) => onNamespacesChange(e.target.value)}
        style={{
          flex: '1 1 160px',
          maxWidth: '200px',
          padding: '5px 10px',
          borderRadius: '6px',
          border: '1px solid #1a4a80',
          background: '#16213e',
          color: '#e0e0e0',
          fontSize: '13px',
          outline: 'none',
        }}
      />

      {/* Include Pods */}
      <label style={{ color: '#e0e0e0', fontSize: '13px', display: 'flex', alignItems: 'center', gap: '4px', whiteSpace: 'nowrap' }}>
        <input
          type="checkbox"
          checked={includePods}
          onChange={(e) => onIncludePodsChange(e.target.checked)}
        />
        Include Pods
      </label>

      {/* Refresh */}
      <button
        onClick={onRefresh}
        style={{
          background: '#4f83cc',
          border: 'none',
          color: '#fff',
          padding: '5px 14px',
          borderRadius: '6px',
          cursor: 'pointer',
          fontSize: '13px',
          whiteSpace: 'nowrap',
        }}
      >
        Refresh
      </button>

      {/* Stats */}
      <span style={{ color: '#9ab', fontSize: '12px', whiteSpace: 'nowrap', marginLeft: 'auto' }}>
        {nodeCount} nodes · {edgeCount} edges{truncated ? ' · ⚠ truncated' : ''}
      </span>
    </div>
  );
}
