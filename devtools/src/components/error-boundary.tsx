import { Component } from 'preact';
import { t } from '../lib/i18n';

interface Props {
  children: preact.ComponentChildren;
  fallback?: (error: Error, reset: () => void) => preact.JSX.Element;
}

interface State {
  error: Error | null;
}

export class ErrorBoundary extends Component<Props, State> {
  state: State = { error: null };

  static getDerivedStateFromError(error: Error): State {
    return { error };
  }

  componentDidCatch(error: Error, info: any) {
    console.error('[ErrorBoundary]', error, info);
  }

  render() {
    if (this.state.error) {
      if (this.props.fallback) {
        return this.props.fallback(this.state.error, () => this.setState({ error: null }));
      }
      return (
        <div class="error-boundary">
          <div class="error-boundary-icon">&#x26A0;&#xFE0F;</div>
          <div class="error-boundary-title">{t('common.error')}</div>
          <div class="error-boundary-message">{this.state.error.message}</div>
          <button class="btn" onClick={() => this.setState({ error: null })}>
            {t('common.try_again')}
          </button>
        </div>
      );
    }
    return this.props.children;
  }
}
