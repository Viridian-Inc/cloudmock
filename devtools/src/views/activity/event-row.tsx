import type { RequestEvent } from '../../lib/types';

interface EventRowProps {
  event: RequestEvent;
  selected: boolean;
  onClick: () => void;
}

function formatTime(ts: number | string): string {
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '--:--:--';
  return d.toTimeString().slice(0, 8); // HH:MM:SS
}

function statusClass(status: number): string {
  if (status >= 200 && status < 300) return 'status-2xx';
  if (status >= 300 && status < 400) return 'status-3xx';
  if (status >= 400 && status < 500) return 'status-4xx';
  if (status >= 500) return 'status-5xx';
  return '';
}

export function EventRow({ event, selected, onClick }: EventRowProps) {
  return (
    <div
      class={`event-row ${selected ? 'event-row-selected' : ''}`}
      onClick={onClick}
    >
      <span class="event-row-time">{formatTime(event.timestamp)}</span>
      <span class="event-row-source">{event.service}</span>
      <span class={`status-pill ${statusClass(event.status_code)}`}>
        {event.status_code}
      </span>
      <span class="event-row-action">{event.action}</span>
      <span class="event-row-latency">
        {event.latency_ms != null ? `${event.latency_ms < 1 ? event.latency_ms.toFixed(2) : Math.round(event.latency_ms)}ms` : ''}
      </span>
    </div>
  );
}
