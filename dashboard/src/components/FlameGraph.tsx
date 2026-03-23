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
  const h = Math.abs(hash) % 60 + 10  // warm orange/red range
  return `hsl(${h}, 70%, 55%)`
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
