import React from 'react';
import ReactDOM from 'react-dom/client';
import { App } from './App';
import './styles/index.css';

// 动态加载 Wails 3 桌面端核心运行时（采用原生 script 标签注入，完全杜绝 Vite 静态构建期报文件不存在）
if (typeof window !== 'undefined') {
  const existing = document.querySelector('script[src*="/wails/runtime.js"]');
  if (!existing) {
    const script = document.createElement('script');
    script.type = 'module';
    script.src = '/wails/runtime.js';
    document.head.appendChild(script);
  }
}

ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>
);
