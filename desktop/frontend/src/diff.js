// Linear-memory, ordered line comparison. Review stays responsive at the 24 KiB limit.
export function lineDiff(current, proposed) {
  const oldLines = current.split("\n"),
    newLines = proposed.split("\n");
  const oldPositions = new Map();
  oldLines.forEach((line, index) => {
    const entry = oldPositions.get(line) || {positions: [], next: 0};
    entry.positions.push(index);
    oldPositions.set(line, entry);
  });
  let previous = -1;
  const unchangedNew = new Set(),
    unchangedOld = new Set();
  newLines.forEach((line, index) => {
    const entry = oldPositions.get(line);
    while (entry && entry.next < entry.positions.length && entry.positions[entry.next] <= previous) entry.next++;
    const position = entry?.positions[entry.next];
    if (position !== undefined) {
      entry.next++;
      unchangedNew.add(index);
      unchangedOld.add(position);
      previous = position;
    }
  });
  return {
    current: oldLines.map((line, index) => ({
      line,
      changed: !unchangedOld.has(index),
    })),
    proposed: newLines.map((line, index) => ({
      line,
      changed: !unchangedNew.has(index),
    })),
  };
}
