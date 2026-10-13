import { memo } from 'react';
import { Handle, Position } from '@xyflow/react';
import './RouterNode.scss';

interface RouterNodeData {
  label: string;
  count: number;
}

export const RouterNode = memo(({ data }: { data: RouterNodeData }) => {
  return (
    <div className="topology-router-node">
      <Handle type="source" position={Position.Top} id="top" />
      <Handle type="source" position={Position.Bottom} id="bottom" />
      <Handle type="source" position={Position.Left} id="left" />
      <Handle type="source" position={Position.Right} id="right" />
      
      <div className="router-content">
        <div className="router-icon">
          <svg viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M12 2L2 7L12 12L22 7L12 2Z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M2 17L12 22L22 17" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
            <path d="M2 12L12 17L22 12" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round"/>
          </svg>
        </div>
        <span className="router-label">{data.label}</span>
        <span className="router-badge">{data.count}</span>
      </div>
    </div>
  );
});

RouterNode.displayName = 'RouterNode';
