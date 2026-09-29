import { useEffect, useRef, useState } from 'react';

const successDurationMs = 1000;

export function useSaveFeedback() {
  const [saved, setSaved] = useState(false);
  const resetTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (resetTimer.current) clearTimeout(resetTimer.current);
  }, []);

  const clearSaved = () => {
    if (resetTimer.current) clearTimeout(resetTimer.current);
    resetTimer.current = null;
    setSaved(false);
  };

  const showSaved = () => {
    clearSaved();
    setSaved(true);
    resetTimer.current = setTimeout(() => {
      setSaved(false);
      resetTimer.current = null;
    }, successDurationMs);
  };

  return { saved, clearSaved, showSaved };
}
