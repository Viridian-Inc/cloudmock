import { useState, useEffect } from 'preact/hooks';
import { api } from '../../lib/api';
import './sns-browser.css';

interface SNSTopic {
  topicArn: string;
  name: string;
  subscriptionCount: number;
  recentMessages?: string[];
}

export function SNSBrowserView() {
  const [topics, setTopics] = useState<SNSTopic[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    loadTopics();
  }, []);

  async function loadTopics() {
    setLoading(true);
    try {
      const data = await api<{ topics: SNSTopic[] }>('/api/sns/topics');
      setTopics(data.topics || []);
    } catch {
      setTopics([]);
    }
    setLoading(false);
  }

  if (loading) {
    return (
      <div class="sns-view">
        <div class="sns-empty">Loading topics...</div>
      </div>
    );
  }

  return (
    <div class="sns-view">
      <div class="sns-header">
        <h2>SNS Topics</h2>
        <button class="btn btn-ghost btn-sm" onClick={loadTopics}>Refresh</button>
      </div>
      <div class="sns-list">
        {topics.length === 0 && (
          <div class="sns-empty">No SNS topics found</div>
        )}
        {topics.map((topic) => (
          <div class="sns-card" key={topic.topicArn}>
            <div class="sns-card-header">
              <span class="sns-card-name">{topic.name}</span>
              <span class="sns-badge">{topic.subscriptionCount} subscriptions</span>
            </div>
            <div style="font-size:11px;color:var(--text-tertiary);font-family:var(--font-mono)">
              {topic.topicArn}
            </div>
            {topic.recentMessages && topic.recentMessages.length > 0 && (
              <div class="sns-messages">
                <div style="font-size:11px;font-weight:600;margin-bottom:4px;color:var(--text-secondary)">
                  Recent Messages
                </div>
                {topic.recentMessages.slice(0, 5).map((msg, i) => (
                  <div class="sns-msg" key={i}>{msg}</div>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
