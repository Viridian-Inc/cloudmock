import { useState, useEffect, useCallback } from 'preact/hooks';
import { nextTip, dismissTip, onTipsChange, type Tip } from '../../lib/tips';
import './tip-banner.css';

export function TipBanner() {
  const [tip, setTip] = useState<Tip | null>(() => nextTip());

  useEffect(() => onTipsChange(() => setTip(nextTip())), []);

  const dismiss = useCallback(() => {
    if (!tip) return;
    dismissTip(tip.id);
  }, [tip]);

  if (!tip) return null;

  return (
    <div class="tip-banner" role="status" aria-label="Tip">
      <span class="tip-banner-icon" aria-hidden="true">{'\u{1F4A1}'}</span>
      <span class="tip-banner-text">{tip.text}</span>
      <button
        class="tip-banner-close"
        onClick={dismiss}
        title="Dismiss this tip"
        aria-label="Dismiss tip"
      >
        {'\u2715'}
      </button>
    </div>
  );
}
