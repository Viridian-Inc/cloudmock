import './profiler.css';

export function ProfilerView() {
  return (
    <div class="profiler-view">
      <div class="profiler-empty-state">
        <div class="profiler-hero-glyph">
          <svg width="48" height="48" viewBox="0 0 48 48" fill="none">
            <rect x="4" y="28" width="8" height="16" rx="2" fill="rgba(74,229,248,0.2)" stroke="var(--brand-teal)" stroke-width="1.5" />
            <rect x="15" y="20" width="8" height="24" rx="2" fill="rgba(9,127,245,0.2)" stroke="var(--brand-blue)" stroke-width="1.5" />
            <rect x="26" y="10" width="8" height="34" rx="2" fill="rgba(247,113,30,0.2)" stroke="var(--brand-orange)" stroke-width="1.5" />
            <rect x="37" y="4" width="8" height="40" rx="2" fill="rgba(255,78,94,0.15)" stroke="var(--error)" stroke-width="1.5" />
          </svg>
        </div>
        <h2>Profiler</h2>
        <p>CPU, heap, and goroutine profiling.</p>
        <p class="profiler-coming-soon">Coming soon</p>
      </div>
    </div>
  );
}
