import { memo } from 'react';
import { BaseEdge, EdgeProps, getBezierPath } from '@xyflow/react';

type FlowEdgeData = {
  isActive?: boolean;
  color?: string;
  pulse?: { id: string; phase: 'started' | 'completed'; status: number };
};

export const ElectricEdge = memo(({
  id,
  sourceX,
  sourceY,
  targetX,
  targetY,
  sourcePosition,
  targetPosition,
  data,
}: EdgeProps) => {
  const [edgePath] = getBezierPath({
    sourceX,
    sourceY,
    sourcePosition,
    targetX,
    targetY,
    targetPosition,
  });
  const edgeData = data as FlowEdgeData | undefined;
  const isActive = edgeData?.isActive === true;
  const pulse = edgeData?.pulse;
  const color = typeof edgeData?.color === 'string' ? edgeData.color : 'var(--primary-active)';
  const pulseColor = pulse?.phase === 'started'
    ? 'var(--warning-color)'
    : pulse?.status && pulse.status >= 400
      ? 'var(--error-color)'
      : 'var(--success-color)';

  return (
    <g>
      <BaseEdge
        id={id}
        path={edgePath}
        style={isActive
          ? { stroke: color, strokeWidth: 2.5, opacity: 0.92 }
          : { stroke: 'var(--border-color)', strokeWidth: 1, opacity: 0.35 }}
      />
      {pulse ? (
        <circle key={pulse.id} r="4" fill={pulseColor} style={{ filter: `drop-shadow(0 0 4px ${pulseColor})` }}>
          <animateMotion dur="0.72s" path={edgePath} fill="freeze" />
          <animate attributeName="opacity" values="0;1;1;0" dur="0.72s" fill="freeze" />
        </circle>
      ) : null}
    </g>
  );
});

ElectricEdge.displayName = 'ElectricEdge';
