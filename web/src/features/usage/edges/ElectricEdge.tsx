import { memo } from 'react';
import { BaseEdge, EdgeProps, getBezierPath } from '@xyflow/react';
import './ElectricEdge.scss';

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

  const isActive = data?.isActive;
  const color = data?.color || '#6366f1';

  if (isActive) {
    return (
      <g className="topology-edge-electric">
        <defs>
          <filter id={`topo-electric-${id}`} x="-40%" y="-40%" width="180%" height="180%">
            <feTurbulence type="fractalNoise" baseFrequency="0.9" numOctaves="2" seed="2" result="noise">
              <animate 
                attributeName="baseFrequency" 
                values="0.8;1.4;0.8" 
                dur="0.25s" 
                repeatCount="indefinite"
              />
            </feTurbulence>
            <feDisplacementMap in="SourceGraphic" in2="noise" scale="3.5" xChannelSelector="R" yChannelSelector="G" />
          </filter>
        </defs>

        {/* Halo */}
        <path
          d={edgePath}
          fill="none"
          stroke="#22d3ee"
          strokeWidth="10"
          strokeOpacity="0.35"
          strokeLinecap="round"
          filter={`url(#topo-electric-${id})`}
          className="topology-edge-halo"
        />

        {/* Plasma */}
        <path
          d={edgePath}
          fill="none"
          stroke="#4ade80"
          strokeWidth="5"
          strokeOpacity="0.85"
          strokeLinecap="round"
          filter={`url(#topo-electric-${id})`}
          className="topology-edge-plasma"
        />

        {/* Core line */}
        <path
          id={id}
          className="react-flow__edge-path topology-edge-kame"
          d={edgePath}
          fill="none"
          stroke="#f8fafc"
          strokeWidth="2.2"
          style={{ opacity: 1 }}
        />

        {/* Interaction path */}
        <path
          d={edgePath}
          fill="none"
          strokeOpacity="0"
          strokeWidth="20"
          className="react-flow__edge-interaction"
        />

        {/* Animated particles */}
        {[0, 0.09, 0.18, 0.27, 0.36, 0.45].map((delay, i) => (
          <circle key={i} r={i % 3 === 0 ? 4 : 2.5} fill={i % 2 === 0 ? '#fde047' : i % 3 === 0 ? '#fff' : '#67e8f9'} opacity="0.95" style={{ filter: 'drop-shadow(0 0 4px #22d3ee)' }}>
            <animateMotion dur={`${0.4 + i * 0.08}s`} repeatCount="indefinite" path={edgePath} begin={`${delay}s`} />
          </circle>
        ))}

        {/* Sparkles */}
        {[0, 0.07, 0.14, 0.21, 0.28].map((delay, i) => (
          <circle key={`spark-${i}`} r="1.8" fill="#e0f2fe" opacity="0">
            <animate attributeName="opacity" values="0;1;0;0;1;0" dur={`${0.35 + i * 0.1}s`} begin={`${delay}s`} repeatCount="indefinite" />
            <animateMotion dur={`${0.28 + i * 0.05}s`} repeatCount="indefinite" path={edgePath} begin={`${delay + 0.11 * i}s`} />
          </circle>
        ))}
      </g>
    );
  }

  // Inactive edge
  return (
    <BaseEdge 
      id={id} 
      path={edgePath}
      style={{ 
        stroke: 'var(--border-color)', 
        strokeWidth: 1, 
        opacity: 0.3 
      }}
    />
  );
});

ElectricEdge.displayName = 'ElectricEdge';
