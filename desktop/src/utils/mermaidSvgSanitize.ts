import DOMPurify from 'dompurify';

/** Sanitize Mermaid SVG before innerHTML injection. */
export function sanitizeMermaidSvg(svg: string): string {
  return DOMPurify.sanitize(svg, {
    USE_PROFILES: { svg: true, svgFilters: true },
    ADD_TAGS: ['foreignObject'],
    ADD_ATTR: ['xmlns', 'viewBox', 'preserveAspectRatio'],
  });
}
