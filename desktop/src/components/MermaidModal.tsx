import { useState, useEffect, useRef } from 'react';
import mermaid from 'mermaid';

interface MermaidModalProps {
  isOpen: boolean;
  onClose: () => void;
  content: string;
}

export function MermaidModal({ isOpen, onClose, content }: MermaidModalProps) {
  const [scale, setScale] = useState(1);
  const [position, setPosition] = useState({ x: 0, y: 0 });
  const [isDragging, setIsDragging] = useState(false);
  const [dragStart, setDragStart] = useState({ x: 0, y: 0 });
  const [lastPosition, setLastPosition] = useState({ x: 0, y: 0 });
  
  const containerRef = useRef<HTMLDivElement>(null);
  const diagramRef = useRef<HTMLDivElement>(null);
  const idRef = useRef(`mermaid-modal-${Math.random().toString(36).substr(2, 9)}`);
  const fullScreenScaleRef = useRef<number>(1);

  // Handle ESC key
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && isOpen) {
        onClose();
      }
    };

    if (isOpen) {
      document.addEventListener('keydown', handleKeyDown);
      // Prevent body scroll when modal is open
      document.body.style.overflow = 'hidden';
    }

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      document.body.style.overflow = 'unset';
    };
  }, [isOpen, onClose]);

  // Render diagram when modal opens
  useEffect(() => {
    if (isOpen && diagramRef.current && containerRef.current) {
      const renderDiagram = async () => {
        try {
          // Clear previous content
          diagramRef.current!.innerHTML = '';
          
          // Render the diagram
          const { svg } = await mermaid.render(idRef.current, content);
          diagramRef.current!.innerHTML = svg;
          
          // Wait for next frame to ensure SVG is rendered and measurable
          requestAnimationFrame(() => {
            if (!diagramRef.current || !containerRef.current) return;
            
            const svgElement = diagramRef.current.querySelector('svg');
            if (!svgElement) return;
            
            // Get container dimensions (95vw x 95vh, accounting for padding)
            const containerRect = containerRef.current.getBoundingClientRect();
            const padding = 32; // p-4 = 16px on each side = 32px total
            const availableWidth = containerRect.width - padding;
            const availableHeight = containerRect.height - padding;
            
            // Get SVG natural dimensions
            // Try multiple methods to get accurate dimensions
            let svgWidth = svgElement.getBoundingClientRect().width;
            let svgHeight = svgElement.getBoundingClientRect().height;
            
            // If dimensions are 0 or invalid, try viewBox
            if ((!svgWidth || svgWidth === 0) && svgElement.viewBox && svgElement.viewBox.baseVal) {
              svgWidth = svgElement.viewBox.baseVal.width;
              svgHeight = svgElement.viewBox.baseVal.height;
            }
            
            // If still no dimensions, use width/height attributes
            if ((!svgWidth || svgWidth === 0) && svgElement.hasAttribute('width')) {
              svgWidth = parseFloat(svgElement.getAttribute('width') || '800');
              svgHeight = parseFloat(svgElement.getAttribute('height') || '600');
            }
            
            // Fallback to defaults
            if (!svgWidth || svgWidth === 0) {
              svgWidth = 800;
              svgHeight = 600;
            }
            
            // Calculate scale to fit diagram to fill container
            const scaleX = availableWidth / svgWidth;
            const scaleY = availableHeight / svgHeight;
            const fitScale = Math.min(scaleX, scaleY);
            
            // Clamp to reasonable bounds
            const initialScale = Math.max(0.1, Math.min(5, fitScale));
            
            // Store full-screen scale for reset functionality
            fullScreenScaleRef.current = initialScale;
            
            // Set initial scale to full-screen fit
            setScale(initialScale);
          });
        } catch (error) {
          console.error('Mermaid rendering error:', error);
          diagramRef.current!.innerHTML = `
            <div class="p-4 bg-red-500/10 border border-red-500/20 rounded text-red-400 text-sm">
              <strong>Mermaid Diagram Error:</strong>
              <pre class="mt-2 text-xs">${error}</pre>
            </div>
          `;
          // On error, fall back to scale 1
          fullScreenScaleRef.current = 1;
          setScale(1);
        }
      };

      renderDiagram();
    }
  }, [isOpen, content]);

  // Reset position when modal opens (but not scale)
  useEffect(() => {
    if (isOpen) {
      setPosition({ x: 0, y: 0 });
      setLastPosition({ x: 0, y: 0 });
    }
  }, [isOpen]);

  const handleZoomIn = () => {
    setScale(prev => Math.min(prev * 1.2, 5));
  };

  const handleZoomOut = () => {
    setScale(prev => Math.max(prev / 1.2, 0.1));
  };

  const handleReset = () => {
    // Reset to full-screen fit scale, not scale 1
    setScale(fullScreenScaleRef.current);
    setPosition({ x: 0, y: 0 });
    setLastPosition({ x: 0, y: 0 });
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.target === diagramRef.current || diagramRef.current?.contains(e.target as Node)) {
      setIsDragging(true);
      setDragStart({ x: e.clientX, y: e.clientY });
      setLastPosition(position);
    }
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isDragging) {
      const deltaX = e.clientX - dragStart.x;
      const deltaY = e.clientY - dragStart.y;
      setPosition({
        x: lastPosition.x + deltaX,
        y: lastPosition.y + deltaY
      });
    }
  };

  const handleMouseUp = () => {
    setIsDragging(false);
  };

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    const delta = e.deltaY > 0 ? 0.9 : 1.1;
    setScale(prev => Math.max(0.1, Math.min(5, prev * delta)));
  };

  if (!isOpen) return null;

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/80 backdrop-blur-sm"
      onClick={onClose}
    >
      {/* Close button */}
      <button
        onClick={onClose}
        className="absolute top-4 right-4 z-10 p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded-full transition-colors"
        title="Close (ESC)"
      >
        <svg className="w-6 h-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
        </svg>
      </button>

      {/* Zoom controls */}
      <div className="absolute top-4 left-4 z-10 flex flex-col gap-2">
        <button
          onClick={handleZoomIn}
          className="p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
          title="Zoom In"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
        </button>
        <button
          onClick={handleZoomOut}
          className="p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
          title="Zoom Out"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M18 12H6" />
          </svg>
        </button>
        <button
          onClick={handleReset}
          className="p-2 bg-slack-bgHover hover:bg-slack-accent text-slack-text hover:text-white rounded transition-colors"
          title="Reset View"
        >
          <svg className="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
          </svg>
        </button>
      </div>

      {/* Scale indicator */}
      <div className="absolute bottom-4 left-4 z-10 px-3 py-1 bg-slack-bgHover text-slack-text rounded text-sm">
        {Math.round(scale * 100)}%
      </div>

      {/* Diagram container */}
      <div
        ref={containerRef}
        className="relative w-[95vw] h-[95vh] overflow-hidden cursor-grab active:cursor-grabbing flex items-center justify-center"
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        onWheel={handleWheel}
        onClick={(e) => e.stopPropagation()}
      >
        <div
          ref={diagramRef}
          className="p-4 bg-white rounded border border-slack-border flex items-center justify-center"
          style={{
            transform: `scale(${scale}) translate(${position.x / scale}px, ${position.y / scale}px)`,
            transformOrigin: 'center center',
            transition: isDragging ? 'none' : 'transform 0.1s ease-out'
          }}
        />
      </div>
    </div>
  );
}
