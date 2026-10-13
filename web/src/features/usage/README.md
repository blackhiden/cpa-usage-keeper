# Usage Features - Life Flow Topology

This directory contains the Life Flow visualization feature for the cpauk web UI.

## Overview

The Life Flow feature provides a dynamic, real-time visualization of provider connections and request flows in the CLIProxyAPI system. It displays a network topology diagram showing the central router and connected AI providers with animated edges indicating active connections.

## Components

### Core Components

- **FlowTopology.tsx** - Main component that orchestrates the flow diagram using React Flow
- **FlowTopology.scss** - Styles for the flow container and controls

### Nodes

- **RouterNode.tsx** - Central router node representing CLIProxyAPI
- **ProviderNode.tsx** - Individual AI provider nodes (Kiro, GitHub, OpenAI, etc.)

### Edges

- **ElectricEdge.tsx** - Animated connection edges with electric effects for active providers

## Features

### Dynamic Data
- Pulls real-time provider data from `OverviewRealtimeBlock.current_usage.ai_providers`
- Automatically updates when the realtime window changes (15m/30m/60m)
- Shows provider request counts and token usage

### Visual Effects
- **Circular Layout**: Providers arranged in a circle around the central router (250px radius)
- **Electric Edges**: Active connections show animated particles flowing along the edges
- **Pulse Animations**: Active providers display a pulsing indicator
- **Color Coding**: Active provider highlighted with custom color and glow effect

### Interactivity
- Zoom controls (zoom in, zoom out, fit view)
- Non-draggable nodes for consistent layout
- Responsive design (480px desktop, 320px mobile)
- Dark mode compatible

## Integration

The FlowTopology is integrated into the UsagePage realtime tab:

```typescript
import { FlowTopology, type Provider } from '@/features/usage';

// In the realtime tab section
<FlowTopology
  providers={currentRealtime.current_usage.ai_providers.map((provider): Provider => ({
    id: provider.key,
    name: provider.label,
    requestCount: provider.requests,
    tokens: provider.tokens,
    color: provider.key === currentRealtime.current_usage.ai_providers[0]?.key ? '#ff6b35' : undefined,
    isActive: provider.requests > 0,
  }))}
  activeProvider={currentRealtime.current_usage.ai_providers[0]?.key}
/>
```

## Data Structure

### Provider Interface
```typescript
interface Provider {
  id: string;          // Unique provider identifier
  name: string;        // Display name
  icon?: string;       // Provider icon URL
  color?: string;      // Theme color for active state
  isActive?: boolean;  // Whether provider has active requests
  requestCount?: number; // Number of requests
  tokens?: number;     // Token count
}
```

## Dependencies

- **@xyflow/react** (v12.11.2) - React Flow library for node-based UI
- No additional dependencies required

## Styling

The component uses CSS variables from the theme system:
- `--bg-primary` - Background color
- `--border-color` - Border colors
- `--floating-surface` - Card backgrounds
- `--text-primary` - Text color
- `--shadow-lg` - Box shadows

## Performance

- Memoized components prevent unnecessary re-renders
- Smooth animations using CSS and SVG
- Hardware-accelerated transformations
- Efficient React Flow rendering

## Browser Support

- Modern browsers with SVG and CSS animations support
- Responsive design for mobile and desktop
- Touch-friendly zoom controls

## Future Enhancements

Potential improvements:
- Click handlers for provider details
- Provider icon integration from backend
- Historical flow data visualization
- Network latency indicators
- Custom color schemes per provider type
