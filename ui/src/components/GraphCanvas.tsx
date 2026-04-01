import { useEffect, useRef } from 'react';
import cytoscape, { type Core, type ElementDefinition } from 'cytoscape';
import { type Graph, type GraphNode } from '../api';

// Node colours keyed by ui.group
const GROUP_COLORS: Record<string, { bg: string; border: string }> = {
  workloads:  { bg: '#4f83cc', border: '#2c5f9e' },
  services:   { bg: '#3eaf7c', border: '#2a7d58' },
  ingresses:  { bg: '#9c6ade', border: '#6b3fa0' },
  nodes:      { bg: '#e67e22', border: '#b35b00' },
  namespaces: { bg: '#7f8c8d', border: '#566061' },
  pods:       { bg: '#e74c3c', border: '#c0392b' },
};

function nodeColor(group: string) {
  return GROUP_COLORS[group] ?? { bg: '#95a5a6', border: '#7f8c8d' };
}

function nodeLabel(node: GraphNode): string {
  const ns = node.namespace ? `${node.namespace}/` : '';
  return `${node.kind}\n${ns}${node.name}`;
}

interface Props {
  graph: Graph;
  onNodeClick: (node: GraphNode) => void;
}

export default function GraphCanvas({ graph, onNodeClick }: Props) {
  const containerRef = useRef<HTMLDivElement>(null);
  const cyRef = useRef<Core | null>(null);

  useEffect(() => {
    if (!containerRef.current) return;

    const elements: ElementDefinition[] = [
      ...graph.nodes.map((n) => ({
        data: {
          id: n.id,
          label: nodeLabel(n),
          group: n.ui.group,
          nodeData: n,
        },
      })),
      ...graph.edges.map((e) => ({
        data: {
          id: e.id,
          source: e.from,
          target: e.to,
          type: e.type,
          style: e.ui.style,
        },
      })),
    ];

    const cy = cytoscape({
      container: containerRef.current,
      elements,
      style: [
        {
          selector: 'node',
          style: {
            label: 'data(label)',
            'text-wrap': 'wrap',
            'text-max-width': '120px',
            'font-size': '11px',
            'text-valign': 'center',
            'text-halign': 'center',
            width: '80px',
            height: '80px',
            'background-color': (ele) => nodeColor(ele.data('group')).bg,
            'border-color': (ele) => nodeColor(ele.data('group')).border,
            'border-width': '2px',
            color: '#fff',
            shape: 'roundrectangle',
          },
        },
        {
          selector: 'node[group="nodes"]',
          style: { shape: 'ellipse' },
        },
        {
          selector: 'node[group="ingresses"]',
          style: { shape: 'hexagon' },
        },
        {
          selector: 'node[group="namespaces"]',
          style: { shape: 'rectangle', opacity: 0.7 },
        },
        {
          selector: 'node:selected',
          style: {
            'border-width': '4px',
            'border-color': '#f1c40f',
          },
        },
        {
          selector: 'edge',
          style: {
            width: 2,
            'line-color': '#aaa',
            'target-arrow-color': '#aaa',
            'target-arrow-shape': 'triangle',
            'curve-style': 'bezier',
            label: 'data(type)',
            'font-size': '10px',
            color: '#555',
          },
        },
        {
          selector: 'edge[style="dashed"]',
          style: {
            'line-style': 'dashed',
          },
        },
      ],
      layout: {
        name: 'cose',
        animate: false,
        nodeRepulsion: () => 80000,
        idealEdgeLength: () => 120,
        padding: 40,
      },
    });

    cy.on('tap', 'node', (evt) => {
      const nodeData: GraphNode = evt.target.data('nodeData');
      onNodeClick(nodeData);
    });

    cyRef.current = cy;

    return () => {
      cy.destroy();
      cyRef.current = null;
    };
  }, [graph, onNodeClick]);

  return (
    <div
      ref={containerRef}
      style={{ width: '100%', height: '100%', background: '#1a1a2e' }}
    />
  );
}
