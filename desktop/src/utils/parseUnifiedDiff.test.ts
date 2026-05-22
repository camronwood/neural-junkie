import { describe, expect, it } from 'vitest';
import { parseUnifiedDiff } from './parseUnifiedDiff';

describe('parseUnifiedDiff', () => {
  it('parses a single hunk', () => {
    const diff = `@@ -1,2 +1,3 @@
-old
+new
+line
`;
    const hunks = parseUnifiedDiff(diff);
    expect(hunks).toHaveLength(1);
    expect(hunks[0].removedLines).toEqual(['old']);
    expect(hunks[0].addedLines).toEqual(['new', 'line']);
  });
});
