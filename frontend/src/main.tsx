import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import './styles/index.css';

const renderApp = () => {
  ReactDOM.createRoot(document.getElementById('root')!).render(
    <React.StrictMode><App /></React.StrictMode>,
  );
};

// 桌面端先等待连接就绪，避免首屏读取到浏览器演示配置。
const existing = document.querySelector<HTMLScriptElement>('script[src*="/wails/runtime.js"]');
const script = existing || document.createElement('script');
if (!existing) {
  script.type = 'module';
  script.src = '/wails/runtime.js';
}
if (window.location.protocol === 'wails:' && !window.wails?.Call?.ByName) {
  script.addEventListener('load', renderApp, { once: true });
  script.addEventListener('error', () => {
    document.getElementById('root')!.textContent = '桌面服务加载失败，请重新打开应用';
  }, { once: true });
} else {
  renderApp();
}
if (!existing) document.head.appendChild(script);
