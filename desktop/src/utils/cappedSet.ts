/** Cap a Set by dropping the oldest insertion-order entries (FIFO prune). */
export function addToCappedSet(set: Set<string>, key: string, max = 500): void {
  set.add(key);
  if (set.size <= max) return;
  const excess = set.size - max;
  let removed = 0;
  for (const existing of set) {
    set.delete(existing);
    removed += 1;
    if (removed >= excess) break;
  }
}
