const { marked } = require('marked');

// Configure marked
marked.setOptions({
  breaks: true,
  gfm: true,
});

const testContent = `# Test

\`\`\`mermaid
graph TD
    A[Start] --> B[End]
\`\`\`

More content here.`;

console.log('Input content:');
console.log(testContent);
console.log('\n---\n');

const html = marked.parse(testContent);
console.log('Marked output:');
console.log(html);
console.log('\n---\n');

// Test our regex on the original content
const mermaidRegex = /```mermaid\s*\r?\n([\s\S]*?)```/g;
let match;
let mermaidBlocks = [];

while ((match = mermaidRegex.exec(testContent)) !== null) {
  const mermaidContent = match[1].trim();
  console.log('Found Mermaid block in original:', mermaidContent);
  mermaidBlocks.push({
    type: 'mermaid',
    content: mermaidContent,
  });
}

console.log('Mermaid blocks found in original:', mermaidBlocks.length);

// Test our regex on the HTML output
const htmlMermaidRegex = /```mermaid\s*\r?\n([\s\S]*?)```/g;
let htmlMatch;
let htmlMermaidBlocks = [];

while ((htmlMatch = htmlMermaidRegex.exec(html)) !== null) {
  const mermaidContent = htmlMatch[1].trim();
  console.log('Found Mermaid block in HTML:', mermaidContent);
  htmlMermaidBlocks.push({
    type: 'mermaid',
    content: mermaidContent,
  });
}

console.log('Mermaid blocks found in HTML:', htmlMermaidBlocks.length);
