import { memo } from 'react';
import { BaseEdge, EdgeProps, getBezierPath } from '@xyflow/react';

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
  const isActive = data?.isActive === true;
  const color = typeof data?.color === 'string' ? data.color : 'var(--primary-active)';

  return (
    <BaseEdge
      id={id}
      path={edgePath}
      style={isActive
        ? { stroke: color, strokeWidth: 2.5, opacity: 0.92 }
        : { stroke: 'var(--border-color)', strokeWidth: 1, opacity: 0.35 }}
    />
  );
});

ElectricEdge.displayName = 'ElectricEdge';
