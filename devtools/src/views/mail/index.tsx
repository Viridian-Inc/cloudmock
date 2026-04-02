import { useState, useEffect, useCallback } from 'preact/hooks';
import { api } from '../../lib/api';
import './mail.css';

interface EmailSummary {
  message_id: string;
  source?: string;
  subject?: string;
  timestamp?: string;
}

interface EmailDetail {
  message_id: string;
  source?: string;
  subject?: string;
  timestamp?: string;
  to_addresses?: string[];
  cc_addresses?: string[];
  html_body?: string;
  text_body?: string;
  body?: string;
}

function formatTime(ts: string | undefined): string {
  if (!ts) return '';
  const d = new Date(ts);
  if (isNaN(d.getTime())) return '';
  return d.toLocaleString([], {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

/**
 * Renders trusted HTML email body from our own SES mock service.
 * This is an internal developer tool -- all email content originates
 * from the local cloudmock SES service, not from external users.
 */
function HtmlEmailBody({ html }: { html: string }) {
  // Content is from our own SES mock, not external/untrusted sources.
  // Using an iframe sandbox as defense-in-depth.
  const srcDoc = `
    <!DOCTYPE html>
    <html>
    <head><style>body { margin: 0; font-family: sans-serif; color: #e0e0e0; background: transparent; }</style></head>
    <body>${html}</body>
    </html>
  `;
  return (
    <iframe
      srcDoc={srcDoc}
      sandbox="allow-same-origin"
      style={{
        width: '100%',
        minHeight: '300px',
        border: 'none',
        background: 'transparent',
      }}
      title="Email content"
    />
  );
}

export function MailView() {
  const [emails, setEmails] = useState<EmailSummary[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [detail, setDetail] = useState<EmailDetail | null>(null);

  const loadEmails = useCallback(() => {
    api<EmailSummary[]>('/api/ses/emails')
      .then(setEmails)
      .catch(() => {});
  }, []);

  // Initial load + auto-refresh every 10s
  useEffect(() => {
    loadEmails();
    const iv = setInterval(loadEmails, 10000);
    return () => clearInterval(iv);
  }, [loadEmails]);

  function viewEmail(email: EmailSummary) {
    setSelectedId(email.message_id);
    api<EmailDetail>(`/api/ses/emails/${email.message_id}`)
      .then(setDetail)
      .catch(() => {});
  }

  function closeDetail() {
    setDetail(null);
    setSelectedId(null);
  }

  function renderEmailBody(d: EmailDetail) {
    if (d.html_body) {
      return <HtmlEmailBody html={d.html_body} />;
    }
    return (
      <pre>{d.text_body || d.body || '(empty)'}</pre>
    );
  }

  return (
    <div class="mail-view">
      <div class="mail-header">
        <h2 class="mail-title">SES Mailbox</h2>
        <p class="mail-desc">Captured email messages from SES mock</p>
      </div>

      <div class="mail-refresh-row">
        <button class="mail-refresh-btn" onClick={loadEmails}>
          Refresh
        </button>
      </div>

      <div class="mail-body">
        {/* Email list */}
        <div class="mail-list">
          {emails.length === 0 ? (
            <div class="mail-empty">No emails captured yet</div>
          ) : (
            <div class="mail-list-card">
              {emails.map((e) => (
                <div
                  key={e.message_id}
                  class={`mail-row ${selectedId === e.message_id ? 'mail-row-selected' : ''}`}
                  onClick={() => viewEmail(e)}
                >
                  <div class="mail-from">
                    {e.source || 'Unknown'}
                  </div>
                  <div class="mail-subject">
                    {e.subject || '(no subject)'}
                  </div>
                  <div class="mail-date">
                    {formatTime(e.timestamp)}
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Detail panel */}
        {detail && (
          <div class="mail-detail">
            <div class="mail-detail-header">
              <div>
                <div class="mail-detail-subject">
                  {detail.subject || '(no subject)'}
                </div>
                <div class="mail-detail-meta">
                  From: {detail.source || ''}
                  {(detail.to_addresses?.length ?? 0) > 0 && (
                    <span> | To: {detail.to_addresses!.join(', ')}</span>
                  )}
                  {(detail.cc_addresses?.length ?? 0) > 0 && (
                    <span> | CC: {detail.cc_addresses!.join(', ')}</span>
                  )}
                </div>
              </div>
              <button class="mail-close-btn" onClick={closeDetail}>
                x
              </button>
            </div>
            <div class="mail-detail-body">
              {renderEmailBody(detail)}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
