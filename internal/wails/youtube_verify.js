(() => {
  if (location.origin !== 'https://embed.dlsrv.online' || window.top !== window) return;
  const nonce = __NONCE__;
  const started = Date.now();
  // 页面自行完成其正常校验；这里只读取校验成功后的会话，不代答验证题。
  const timer = setInterval(() => {
    if (Date.now() - started > 120000) { clearInterval(timer); return; }
    const token = sessionStorage.getItem('__session');
    if (!token) return;
    try {
      const claims = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
      if (claims.exp * 1000 < Date.now() + 10000) return;
      const message = JSON.stringify({ kind: 'youtube-session', nonce, token, user_agent: navigator.userAgent });
      if (typeof window._wails?.invoke === 'function') window._wails.invoke(message);
      else if (window.webkit?.messageHandlers?.external) window.webkit.messageHandlers.external.postMessage(message);
      else if (window.chrome?.webview) window.chrome.webview.postMessage(message);
      else return;
      clearInterval(timer);
    } catch (error) { console.warn('Download session is not ready'); }
  }, 500);
})();
