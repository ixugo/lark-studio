import React, { createContext, useContext, useState, ReactNode } from 'react';
import { zh, TranslationKeys } from './locales/zh';
import { en } from './locales/en';

export type Locale = 'zh-CN' | 'en-US';

interface LanguageContextType {
  locale: Locale;
  setLocale: (locale: Locale) => void;
  toggleLocale: () => void;
  t: (path: string, defaultValueOrParams?: string | Record<string, string | number>, params?: Record<string, string | number>) => string;
  strings: TranslationKeys;
}

const STORAGE_KEY = 'vdub_locale';

const LanguageContext = createContext<LanguageContextType | undefined>(undefined);

export const LanguageProvider: React.FC<{ children: ReactNode }> = ({ children }) => {
  const [locale, setLocaleState] = useState<Locale>(() => {
    if (typeof window !== 'undefined') {
      const saved = localStorage.getItem(STORAGE_KEY) as Locale;
      if (saved === 'zh-CN' || saved === 'en-US') return saved;
    }
    return 'zh-CN';
  });

  const setLocale = (newLocale: Locale) => {
    setLocaleState(newLocale);
    if (typeof window !== 'undefined') {
      localStorage.setItem(STORAGE_KEY, newLocale);
    }
  };

  const toggleLocale = () => {
    setLocale(locale === 'zh-CN' ? 'en-US' : 'zh-CN');
  };

  const strings = locale === 'en-US' ? en : zh;

  // 根据点分隔路径查询，如 'sidebar.creationCenter'，并支持 {{key}} 参数插值
  const t = (
    path: string,
    defaultValueOrParams?: string | Record<string, string | number>,
    params?: Record<string, string | number>
  ): string => {
    const keys = path.split('.');
    let current: any = strings;
    let fallbackText = typeof defaultValueOrParams === 'string' ? defaultValueOrParams : path;
    const finalParams =
      typeof defaultValueOrParams === 'object' && defaultValueOrParams !== null
        ? defaultValueOrParams
        : params;

    for (const key of keys) {
      if (current && typeof current === 'object' && key in current) {
        current = current[key];
      } else {
        current = fallbackText;
        break;
      }
    }

    let result = typeof current === 'string' ? current : fallbackText;
    if (finalParams) {
      for (const [k, v] of Object.entries(finalParams)) {
        result = result.replace(new RegExp(`{{\\s*${k}\\s*}}`, 'g'), String(v));
      }
    }
    return result;
  };

  return (
    <LanguageContext.Provider value={{ locale, setLocale, toggleLocale, t, strings }}>
      {children}
    </LanguageContext.Provider>
  );
};

export const useTranslation = () => {
  const context = useContext(LanguageContext);
  if (!context) {
    throw new Error('useTranslation 必须在 LanguageProvider 内部使用');
  }
  return context;
};
