import { useState, useEffect } from 'preact/hooks';
import { api } from '../api';
import { XIcon } from '../components/Icons';
import { fmtTime } from '../utils';

// Internal developer dashboard -- email content comes from our own SES mock.

export function MailPage() {
  const [emails, setEmails] = useState<any[]>([]);
  const [selected, setSelected] = useState<string | null>(null);
  const [detail, setDetail] = useState<any>(null);

  useEffect(() => {
    api('/api/ses/emails').then(setEmails).catch(() => {});
  }, []);

  function viewEmail(email: any) {
    setSelected(email.message_id);
    api(`/api/ses/emails/${email.message_id}`).then(setDetail).catch(() => {});
  }

  function renderEmailBody(d: any) {
    if (d.html_body) {
      // Trusted content from our own SES mock service
      return <div dangerouslySetInnerHTML={{ __html: d.html_body }} />;
    }
    return <pre style="white-space:pre-wrap;font-size:14px">{d.text_body || d.body || '(empty)'}</pre>;
  }

  return (
    <div>
      <div class="mb-6">
        <h1 class="page-title">SES Mailbox</h1>
        <p class="page-desc">Captured email messages from /api/ses/emails</p>
      </div>

      <div class="flex gap-4">
        <div style="flex:1">
          {emails.length === 0 ? (
            <div class="card">
              <div class="empty-state" style="padding:48px">No emails captured yet</div>
            </div>
          ) : (
            <div class="mail-list">
              {emails.map((e: any) => (
                <div class={`mail-row ${selected === e.message_id ? 'expanded' : ''}`} onClick={() => viewEmail(e)}>
                  <div class="mail-from truncate">{e.source || 'Unknown'}</div>
                  <div class="mail-subject truncate">{e.subject || '(no subject)'}</div>
                  <div class="mail-date">{e.timestamp ? fmtTime(e.timestamp) : ''}</div>
                </div>
              ))}
            </div>
          )}
        </div>

        {detail && (
          <div style="width:45%;flex-shrink:0">
            <div class="card">
              <div class="card-header">
                <div style="flex:1">
                  <h3 style="font-weight:700;font-size:16px">{detail.subject || '(no subject)'}</h3>
                  <div class="text-sm text-muted mt-4">
                    From: {detail.source || ''} | To: {(detail.to_addresses || []).join(', ')}
                  </div>
                </div>
                <button class="btn-icon btn-sm btn-ghost" onClick={() => { setDetail(null); setSelected(null); }}>
                  <XIcon />
                </button>
              </div>
              <div class="card-body">
                {renderEmailBody(detail)}
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
