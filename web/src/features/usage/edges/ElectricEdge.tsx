import { memo } from 'react';
import { BaseEdge, EdgeProps, getBezierPath } from '@xyflow/react';
import { LIVE_FLOW_PENDING_COLOR, type LiveFlowTrail } from '../liveFlow';

type FlowEdgeData = {
  isActive?: boolean;
  color?: string;
  trails?: LiveFlowTrail[];
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
  const trails = edgeData?.trails ?? [];
  const color = typeof edgeData?.color === 'string' ? edgeData.color : 'var(--primary-active)';

  return (
    <g>
      <BaseEdge
        id={id}
        path={edgePath}
        style={isActive
          ? { stroke: color, strokeWidth: 2.5, opacity: 0.92 }
          : { stroke: 'var(--border-color)', strokeWidth: 1, opacity: 0.35 }}
      />
      {trails.map((trail, index) => {
        const pending = trail.state === 'pending';
        const trailColor = pending
          ? LIVE_FLOW_PENDING_COLOR
          : trail.state === 'failure' ? 'var(--error-color)' : 'var(--success-color)';
        return (
          <g key={trail.id} className={`live-flow-trail live-flow-trail-${trail.state}`}>
            <path
              d={edgePath}
              fill="none"
              stroke={trailColor}
              strokeWidth={pending ? 5 : 3.5}
              strokeLinecap="round"
              strokeDasharray={pending ? '3 10' : undefined}
              strokeDashoffset={index * 1.75}
              className={pending ? 'live-flow-trail-pending' : undefined}
              style={pending ? undefined : { filter: `drop-shadow(0 0 3px ${trailColor})` }}
            />
            {pending ? (
              <circle r="3.25" fill={LIVE_FLOW_PENDING_COLOR} className="live-flow-trail-beacon">
                <animateMotion dur="0.9s" repeatCount="indefinite" path={edgePath} />
              </circle>
            ) : null}
          </g>
        );
      })}
    </g>
  );
});

ElectricEdge.displayName = 'ElectricEdge';
