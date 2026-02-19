/**
 * Test component for debugging Mermaid diagram rendering issues
 * This component can be used to test wide diagram rendering
 */

import { useEffect, useRef, useState } from 'react';
import mermaid from 'mermaid';

// Initialize mermaid
mermaid.initialize({
  startOnLoad: false,
  theme: 'dark',
  securityLevel: 'loose',
  fontFamily: 'ui-monospace, monospace',
});

const WIDE_DIAGRAM = `graph TB
    subgraph "Client Layer"
        SunRun[SunRun Client<br/>🎯 Pilot Program]
        OtherClients[Other Clients<br/>Future Rollout]
    end
    
    subgraph "API Gateway"
        APIGW[ms-api-gateway<br/>Feature Flags & Routing]
        GraphQL[ms-graphql-gateway<br/>Federated GraphQL]
    end
    
    subgraph "New Route-Centric Services"
        Ingestion[ms-ingestion-engine<br/>Webhook Intake & Validation]
        Deliveries[ms-deliveries<br/>Delivery Processing & Shadow Mode]
        Dispatch[ms-dispatch<br/>Route Optimization & TSP]
        Automation[ms-automation-engine<br/>Custom Workflows]
    end
    
    subgraph "Enhanced Services"
        Monolith[ms-monolith<br/>Legacy Support]
        OrderQuote[ms-order-quote<br/>Route-Based Pricing]
        Locations[ms-locations<br/>Route Calculation]
    end
    
    subgraph "Driver Services"
        DriverEngagement[ms-driver-engagement<br/>Route Tracking]
        DriverTracker[ms-driver-order-tracker<br/>Multi-Stop Support]
        R2D2[ms-r2d2<br/>Route Notifications]
    end
    
    subgraph "New Data Model"
        OrderDB[("Order Database<br/>Order/Delivery/Route<br/>Shadow Tables")]
    end
    
    subgraph "Event-Driven Integration"
        Kafka[Kafka<br/>Event Streaming]
    end
    
    SunRun --> APIGW
    OtherClients --> APIGW
    APIGW --> Ingestion
    Ingestion --> Deliveries
    APIGW --> OrderQuote
    APIGW --> Locations
    
    Deliveries --> Dispatch
    Deliveries --> Automation
    Dispatch --> Automation
    
    Ingestion --> OrderDB
    Deliveries --> OrderDB
    Dispatch --> OrderDB
    Automation --> OrderDB
    
    Ingestion --> Kafka
    Deliveries --> Kafka
    Dispatch --> Kafka
    Automation --> Kafka
    
    Kafka --> DriverEngagement
    Kafka --> DriverTracker
    Kafka --> R2D2
    
    DriverEngagement --> OrderDB
    DriverTracker --> OrderDB`;

export function MermaidDiagramTest() {
  const containerRef = useRef<HTMLDivElement>(null);
  const [isRendering, setIsRendering] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [svgInfo, setSvgInfo] = useState<any>(null);

  useEffect(() => {
    const renderDiagram = async () => {
      if (!containerRef.current) return;

      setIsRendering(true);
      setError(null);

      try {
        containerRef.current.innerHTML = '';

        const id = `test-mermaid-${Date.now()}`;
        const { svg } = await mermaid.render(id, WIDE_DIAGRAM);
        containerRef.current.innerHTML = svg;

        const svgElement = containerRef.current.querySelector('svg');
        if (svgElement) {
          svgElement.style.maxWidth = 'none';
          svgElement.style.width = 'auto';
          svgElement.style.display = 'block';

          const rect = svgElement.getBoundingClientRect();
          const containerRect = containerRef.current.getBoundingClientRect();
          
          setSvgInfo({
            svgWidth: svgElement.getAttribute('width'),
            svgHeight: svgElement.getAttribute('height'),
            viewBox: svgElement.getAttribute('viewBox'),
            computedWidth: rect.width,
            computedHeight: rect.height,
            containerWidth: containerRect.width,
            containerHeight: containerRect.height,
            needsScrolling: rect.width > containerRect.width,
          });
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : String(err));
      } finally {
        setIsRendering(false);
      }
    };

    renderDiagram();
  }, []);

  return (
    <div className="p-6 bg-slack-bg text-slack-text">
      <h2 className="text-xl font-bold mb-4">Mermaid Diagram Test</h2>
      
      <div className="mb-4 p-4 bg-slack-bgHover rounded border border-slack-border">
        <h3 className="font-semibold mb-2">Container Info</h3>
        <div className="text-sm space-y-1">
          <div>Container: {containerRef.current?.getBoundingClientRect().width}px × {containerRef.current?.getBoundingClientRect().height}px</div>
        </div>
      </div>

      {svgInfo && (
        <div className="mb-4 p-4 bg-slack-bgHover rounded border border-slack-border">
          <h3 className="font-semibold mb-2">SVG Info</h3>
          <pre className="text-xs overflow-auto">{JSON.stringify(svgInfo, null, 2)}</pre>
          {svgInfo.needsScrolling && (
            <div className="mt-2 text-yellow-400">⚠️ Diagram is wider than container - horizontal scrolling should be visible</div>
          )}
        </div>
      )}

      {error && (
        <div className="mb-4 p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400">
          <strong>Error:</strong> {error}
        </div>
      )}

      {isRendering && (
        <div className="mb-4 text-slack-textMuted">Rendering diagram...</div>
      )}

      <div
        ref={containerRef}
        className="w-full min-h-[200px] p-6 bg-slack-bgHover rounded border border-slack-border overflow-x-auto overflow-y-visible relative"
        style={{ 
          display: 'block',
          position: 'relative',
        }}
      />
    </div>
  );
}

