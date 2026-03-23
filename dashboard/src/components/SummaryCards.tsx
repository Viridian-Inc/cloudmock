interface Card {
  label: string
  value: number | string
  color?: string
}

export function SummaryCards({ cards }: { cards: Card[] }) {
  return (
    <div class="summary-cards">
      {cards.map(c => (
        <div class="summary-card">
          <div class="summary-card-value" style={c.color ? { color: c.color } : undefined}>
            {c.value}
          </div>
          <div class="summary-card-label">{c.label}</div>
        </div>
      ))}
    </div>
  )
}
