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
import type { LiveFlowPulse } from './liveFlow';
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
  activeRequests?: number;
}

interface FlowTopologyProps {
  providers: Provider[];
  pulses?: Record<string, LiveFlowPulse>;
}

// React Flow zoom controls use a 1.2x increment, so 0.7 is approximately two
// zoom-out steps below the default 1.0 viewport.
export const liveFlowFitViewOptions = { padding: 0.42, minZoom: 0.35, maxZoom: 0.7 };

export function FlowTopology({ providers, pulses = {} }: FlowTopologyProps) {
  const activeRequests = providers.reduce((total, provider) => total + (provider.activeRequests ?? 0), 0)
  const nodes: Node[] = useMemo(() => {
    const routerNode: Node = {
      id: 'router',
      type: 'router',
      position: { x: 0, y: 0 },
      data: {
        label: 'CLIProxyAPI',
        count: providers.length,
        activeRequests,
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
          isActive: (provider.activeRequests ?? 0) > 0,
          requestCount: provider.requestCount,
          failures: provider.failures,
          latestStatus: provider.latestStatus,
          latestLatencyMs: provider.latestLatencyMs,
          activeRequests: provider.activeRequests,
          pulse: pulses[provider.id],
        },
      };
    });

    return [routerNode, ...providerNodes];
  }, [providers, activeRequests, pulses]);

  const edges: Edge[] = useMemo(() => {
    return providers.map((provider) => ({
      id: `e-provider-${provider.id}`,
      source: 'router',
      target: `provider-${provider.id}`,
      type: 'topology',
      data: {
        isActive: (provider.activeRequests ?? 0) > 0,
        color: provider.color,
        pulse: pulses[provider.id],
      },
    }));
  }, [providers, pulses]);

  return (
    <div className="flow-topology">
      <ReactFlow
        nodes={nodes}
        edges={edges}
        nodeTypes={nodeTypes}
        edgeTypes={edgeTypes}
        connectionLineType={ConnectionLineType.SmoothStep}
        fitView
        fitViewOptions={liveFlowFitViewOptions}
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
