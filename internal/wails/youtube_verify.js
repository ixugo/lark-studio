(() => {
  if (location.origin !== 'https://embed.dlsrv.online' || window.top !== window) return;
  const nonce = __NONCE__;
  const videoId = __VIDEO_ID__;
  const quality = __QUALITY__;
  const started = Date.now();
  let requesting = false;
  const send = payload => {
    const message = JSON.stringify({ ...payload, nonce });
    if (typeof window._wails?.invoke === 'function') window._wails.invoke(message);
    else if (window.webkit?.messageHandlers?.external) window.webkit.messageHandlers.external.postMessage(message);
    else if (window.chrome?.webview) window.chrome.webview.postMessage(message);
  };
  const timer = setInterval(async () => {
    if (Date.now() - started > 120000) { clearInterval(timer); return; }
    if (requesting) return;
    const token = sessionStorage.getItem('__session');
    if (!token) return;
    try {
      const claims = JSON.parse(atob(token.split('.')[1].replace(/-/g, '+').replace(/_/g, '/')));
      if (claims.exp * 1000 < Date.now() + 10000) return;
      requesting = true;
      // 在完成校验的同一浏览器环境请求地址，保留实际 Cookie、网络出口及浏览器请求特征。
      const touched = await fetch('/api/session/touch', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ videoId })
      });
      if (!touched.ok) throw new Error(`下载会话初始化失败（HTTP ${touched.status}）`);
      const response = await fetch('/api/download/mp4', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
        body: JSON.stringify({ videoId, format: 'mp4', quality: String(quality) })
      });
      const result = await response.json();
      if (!response.ok || !result.url) throw new Error(`下载服务未返回有效地址（HTTP ${response.status}）：${result.error || '请重新校验'}`);
      send({ kind: 'youtube-session', token, user_agent: navigator.userAgent, address: result.url });
      clearInterval(timer);
    } catch (error) {
      if (requesting) { clearInterval(timer); send({ kind: 'youtube-error', error: String(error) }); }
    }
  }, 500);
})();
