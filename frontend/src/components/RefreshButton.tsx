import React, { useState } from 'react';
import { RotateCcw } from 'lucide-react';

const MIN_REFRESH_FEEDBACK_MS = 600;

export const RefreshButton: React.FC<{
  onRefresh: () => Promise<unknown>;
  className?: string;
  title: string;
  size?: number;
  children?: React.ReactNode;
}> = ({ onRefresh, className, title, size = 14, children }) => {
  const [refreshing, setRefreshing] = useState(false);

  const refresh = async () => {
    if (refreshing) return;
    setRefreshing(true);
    // 本地请求可能瞬间完成，至少保留一段可见动画，不延迟请求本身。
    const feedback = new Promise<void>(resolve => setTimeout(resolve, MIN_REFRESH_FEEDBACK_MS));
    try {
      await onRefresh();
    } finally {
      await feedback;
      setRefreshing(false);
    }
  };

  return (
    <button type="button" onClick={refresh} disabled={refreshing} aria-busy={refreshing}
      aria-label={title} title={title} className={`${className || ''} disabled:opacity-60 disabled:cursor-wait`}>
      <RotateCcw size={size} className={refreshing ? 'animate-spin' : undefined} />
      {children}
    </button>
  );
};
