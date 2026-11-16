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
import type { LiveFlowTrail } from './liveFlow';
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
  trails?: Record<string, LiveFlowTrail>;
}

// React Flow zoom controls use a 1.2x increment, so 0.7 is approximately two
// zoom-out steps below the default 1.0 viewport.
export const liveFlowFitViewOptions = { padding: 0.42, minZoom: 0.35, maxZoom: 0.7 };

// Provider blocks are roughly 190px wide once the name, count and in-flight
// badge are laid out. The ring radius grows with the model count so
// neighbouring blocks keep a visible gap instead of overlapping.
const providerNodeWidth = 190;
const providerRingBaseRadius = 250;
const providerRingMinGap = 28;

export function liveFlowRingRadius(nodeCount: number): number {
  if (nodeCount <= 1) return 0;
  // Chord length between neighbours must cover one block plus a gap.
  const chord = providerNodeWidth + providerRingMinGap;
  const required = chord / (2 * Math.sin(Math.PI / nodeCount));
  return Math.max(providerRingBaseRadius, Math.ceil(required));
}

// In-flight blocks are raised above idle ones so a busy block is never hidden
// behind a neighbour and its label stays readable.
const providerNodeBaseZIndex = 1;
const providerNodeActiveZIndex = 10;

export function liveFlowNodeZIndex(isActive: boolean): number {
  return isActive ? providerNodeActiveZIndex : providerNodeBaseZIndex;
}

export function FlowTopology({ providers, trails = {} }: FlowTopologyProps) {
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
      zIndex: activeRequests > 0 ? providerNodeActiveZIndex : providerNodeBaseZIndex,
    };

    const radius = liveFlowRingRadius(providers.length);
    // A block is active while it has in-flight requests or a live trail, so its
    // label is never covered by a neighbouring block.
    const activeTrailModels = new Set(
      Object.values(trails)
        .filter((trail) => trail.state === 'pending')
        .map((trail) => trail.modelID),
    );

    const providerNodes: Node[] = providers.map((provider, index) => {
      const total = providers.length;
      const angle = (index / total) * 2 * Math.PI - Math.PI / 2;
      // Center each block on its ring point; otherwise every block hangs to the
      // right of its anchor and neighbouring labels collide.
      const x = Math.cos(angle) * radius - providerNodeWidth / 2;
      const y = Math.sin(angle) * radius;
      const isActive = (provider.activeRequests ?? 0) > 0 || activeTrailModels.has(provider.id);

      return {
        id: `provider-${provider.id}`,
        type: 'provider',
        position: { x, y },
        zIndex: liveFlowNodeZIndex(isActive),
        className: isActive ? 'live-flow-node-active' : undefined,
        data: {
          name: provider.name,
          color: provider.color,
          isActive,
          requestCount: provider.requestCount,
          failures: provider.failures,
          latestStatus: provider.latestStatus,
          latestLatencyMs: provider.latestLatencyMs,
          activeRequests: provider.activeRequests,
        },
      };
    });

    return [routerNode, ...providerNodes];
  }, [providers, trails, activeRequests]);

  const edges: Edge[] = useMemo(() => {
    return providers.map((provider) => {
      const providerTrails = Object.values(trails).filter((trail) => trail.modelID === provider.id)
      return {
        id: `e-provider-${provider.id}`,
        source: 'router',
        target: `provider-${provider.id}`,
        type: 'topology',
        data: {
          isActive: (provider.activeRequests ?? 0) > 0,
          color: provider.color,
          trails: providerTrails,
        },
      };
    });
  }, [providers, trails]);

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
