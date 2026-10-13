import { useMemo } from 'react';
import {
  ReactFlow,
  Background,
  Controls,
  Node,
  Edge,
  ConnectionLineType,
} from '@xyflow/react';
import '@xyflow/react/dist/style.css';
import { RouterNode } from './nodes/RouterNode';
import { ProviderNode } from './nodes/ProviderNode';
import { ElectricEdge } from './edges/ElectricEdge';
import './FlowTopology.scss';

const nodeTypes = {
  router: RouterNode,
  provider: ProviderNode,
};

const edgeTypes = {
  topology: ElectricEdge,
};

export interface Provider {
  id: string;
  name: string;
  color?: string;
  isActive?: boolean;
  requestCount?: number;
  failures?: number;
  latestStatus?: number;
  latestLatencyMs?: number;
}

interface FlowTopologyProps {
  providers: Provider[];
  activeProvider?: string;
}

export function FlowTopology({ providers, activeProvider }: FlowTopologyProps) {
  const nodes: Node[] = useMemo(() => {
    const routerNode: Node = {
      id: 'router',
      type: 'router',
      position: { x: 0, y: 0 },
      data: {
        label: 'CLIProxyAPI',
        count: providers.length,
      },
    };

    const providerNodes: Node[] = providers.map((provider, index) => {
      const total = providers.length;
      const angle = (index / total) * 2 * Math.PI - Math.PI / 2;
      const radius = 250;
      const x = Math.cos(angle) * radius;
      const y = Math.sin(angle) * radius;

      return {
        id: `provider-${provider.id}`,
        type: 'provider',
        position: { x, y },
        data: {
          name: provider.name,
          color: provider.color,
          isActive: provider.id === activeProvider,
          requestCount: provider.requestCount,
          failures: provider.failures,
          latestStatus: provider.latestStatus,
          latestLatencyMs: provider.latestLatencyMs,
        },
      };
    });

    return [routerNode, ...providerNodes];
  }, [providers, activeProvider]);

  const edges: Edge[] = useMemo(() => {
    return providers.map((provider) => ({
      id: `e-provider-${provider.id}`,
      source: 'router',
      target: `provider-${provider.id}`,
      type: 'topology',
      data: {
        isActive: provider.id === activeProvider,
        color: provider.color,
      },
    }));
  }, [providers, activeProvider]);

  return (
    <div className="flow-topology">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        connectionLineType={ConnectionLineType.SmoothStep}
        fitView
        nodesDraggable={false}
        nodesConnectable={false}
        elementsSelectable={false}
        zoomOnScroll={false}
        panOnScroll={false}
        preventScrolling={false}
        minZoom={0.5}
        maxZoom={1.5}
      >
        <Background />
        <Controls />
      </ReactFlow>
    </div>
  );
}
