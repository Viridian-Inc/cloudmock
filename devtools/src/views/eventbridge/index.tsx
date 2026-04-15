import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './eventbridge.css';

interface Rule {
  name: string;
  state: string;
  targets: string[];
}

interface EventBus {
  name: string;
  arn: string;
  rules: Rule[];
}

export function EventBridgeView() {
  const [buses, setBuses] = useState<EventBus[]>([]);
  const [loading, setLoading] = useState(true);
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  useEffect(() => { loadBuses(); }, []);

  async function loadBuses() {
    setLoading(true);
    try {
      const data = await api<{ buses: EventBus[] }>('/api/eventbridge/buses');
      setBuses(data.buses || []);
    } catch {
      setBuses([]);
    }
    setLoading(false);
  }

  function toggle(name: string) {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(name)) next.delete(name);
      else next.add(name);
      return next;
    });
  }

  if (loading) {
    return <div class="eb-view"><div class="eb-empty">Loading event buses...</div></div>;
  }

  return (
    <div class="eb-view">
      <div class="eb-header">
        <h2>EventBridge</h2>
        <button class="btn btn-ghost btn-sm" onClick={loadBuses}>Refresh</button>
      </div>
      <div class="eb-list">
        {buses.length === 0 && <div class="eb-empty">No event buses found</div>}
        {buses.map((bus) => (
          <div class="eb-bus" key={bus.name}>
            <div class="eb-bus-header" onClick={() => toggle(bus.name)}>
              <span class="eb-bus-name">{bus.name}</span>
              <span class="eb-bus-badge">{bus.rules.length} rules</span>
            </div>
            {expanded.has(bus.name) && bus.rules.length > 0 && (
              <div class="eb-rules">
                {bus.rules.map((rule) => (
                  <div class="eb-rule" key={rule.name}>
                    <div class="eb-rule-name">{rule.name}</div>
                    <div class="eb-rule-targets">
                      Targets: {rule.targets.length > 0 ? rule.targets.join(', ') : 'none'}
                    </div>
                  </div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
