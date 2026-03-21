interface ToastProps {
  message: string;
}

export function Toast({ message }: ToastProps) {
  if (!message) return null;
  return <div class="toast">{message}</div>;
}
