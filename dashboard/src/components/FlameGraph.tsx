import { useMemo } from 'preact/hooks'

interface Props {
  data: string  // folded stacks text
}

interface Frame {
  name: string
  value: number
  children: Frame[]
}

function parseFolded(text: string): Frame {
  const root: Frame = { name: 'root', value: 0, children: [] }
  for (const line of text.trim().split('\n')) {
    if (!line) continue
    const lastSpace = line.lastIndexOf(' ')
    if (lastSpace === -1) continue
    const stack = line.substring(0, lastSpace).split(';')
    const count = parseInt(line.substring(lastSpace + 1))
    if (isNaN(count)) continue
    root.value += count
    let node = root
    for (const func of stack) {
      let child = node.children.find(c => c.name === func)
      if (!child) {
        child = { name: func, value: 0, children: [] }
        node.children.push(child)
      }
      child.value += count
      node = child
    }
  }
  return root
}

function hashColor(name: string): string {
  let hash = 0
  for (let i = 0; i < name.length; i++) hash = ((hash << 5) - hash) + name.charCodeAt(i)
  // Blue-teal-cyan range matching brand palette
  const hues = [200, 210, 185, 195, 220, 175, 230, 165]
  const h = hues[Math.abs(hash) % hues.length]
  const s = 55 + (Math.abs(hash >> 4) % 20)
  const l = 45 + (Math.abs(hash >> 8) % 15)
  return `hsl(${h}, ${s}%, ${l}%)`
}

function FlameRow({ frame, total, depth }: { frame: Frame; total: number; depth: number }) {
  const width = (frame.value / total) * 100
  if (width < 0.5) return null
  return (
    <div>
      <div
        class="flame-bar"
        style={{
          width: `${width}%`,
          backgroundColor: hashColor(frame.name),
          marginLeft: `${depth * 0}%`
        }}
        title={`${frame.name} (${frame.value} samples, ${width.toFixed(1)}%)`}
      >
        <span class="flame-label">{frame.name}</span>
      </div>
      {frame.children
        .sort((a, b) => b.value - a.value)
        .map(child => <FlameRow frame={child} total={total} depth={depth + 1} />)}
    </div>
  )
}

export function FlameGraph({ data }: Props) {
  const root = useMemo(() => parseFolded(data), [data])
  if (!root.children.length) return <div class="empty-state">No profile data</div>
  return (
    <div class="flame-graph">
      {root.children
        .sort((a, b) => b.value - a.value)
        .map(child => <FlameRow frame={child} total={root.value} depth={0} />)}
    </div>
  )
}
