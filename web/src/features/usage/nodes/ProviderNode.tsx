import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import './ProviderNode.scss';

interface ProviderNodeData {
  name: string;
  color?: string;
  isActive?: boolean;
  requestCount?: number;
  activeRequests?: number;
  failures?: number;
  latestStatus?: number;
  latestLatencyMs?: number;
  pulse?: { id: string; phase: 'started' | 'completed'; status: number };
}

export const ProviderNode = memo(({ data }: { data: ProviderNodeData }) => {
  return (
    <div className={`topology-provider-node ${data.isActive ? 'active' : ''}`}>
      <Handle type="target" position={Position.Top} id="top" />
      <Handle type="target" position={Position.Bottom} id="bottom" />
      <Handle type="target" position={Position.Left} id="left" />
      <Handle type="target" position={Position.Right} id="right" />
      
      <div
        className="provider-content"
        style={{
          borderColor: data.isActive && data.color ? data.color : undefined,
          boxShadow: data.isActive && data.color ? `0 0 16px ${data.color}` : undefined,
        }}
      >
        <span
          className="provider-name"
          style={{ color: data.isActive && data.color ? data.color : undefined }}
        >
          {data.name}
        </span>
        <span className="provider-count">{data.requestCount ?? 0}</span>
        {data.activeRequests ? <span className="provider-inflight">{data.activeRequests}</span> : null}
        {data.isActive && <span className="provider-active" style={{ backgroundColor: data.color }} />}
        {data.pulse ? <span key={data.pulse.id} className={`provider-pulse provider-pulse-${data.pulse.phase}`} data-failed={data.pulse.status >= 400} /> : null}
      </div>
    </div>
  );
});

ProviderNode.displayName = 'ProviderNode';
