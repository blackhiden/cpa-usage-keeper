import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import './ProviderNode.scss';

interface ProviderNodeData {
  name: string;
  icon?: string;
  color?: string;
  isActive?: boolean;
  requestCount?: number;
  tokens?: number;
}

export const ProviderNode = memo(({ data }: { data: ProviderNodeData }) => {
  return (
    <div 
      className={`topology-provider-node ${data.isActive ? 'active' : ''}`}
      style={{
        borderColor: data.isActive && data.color ? data.color : undefined,
        boxShadow: data.isActive && data.color ? `0 0 16px ${data.color}40` : undefined,
      }}
    >
      <Handle type="target" position={Position.Top} id="top" />
      <Handle type="target" position={Position.Bottom} id="bottom" />
      <Handle type="target" position={Position.Left} id="left" />
      <Handle type="target" position={Position.Right} id="right" />
      
      <div className="provider-content">
        {data.icon && (
          <div 
            className="provider-icon"
            style={{ backgroundColor: data.color ? `${data.color}15` : undefined }}
          >
            <img src={data.icon} alt={data.name} />
          </div>
        )}
        <span 
          className="provider-name"
          style={{ color: data.isActive && data.color ? data.color : undefined }}
        >
          {data.name}
        </span>
        {data.isActive && (
          <span className="provider-pulse" style={{ backgroundColor: data.color }} />
        )}
      </div>
    </div>
  );
});

ProviderNode.displayName = 'ProviderNode';
