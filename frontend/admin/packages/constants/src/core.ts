/**
 * @zh_CN 登录页面 url 地址
 */
export const LOGIN_PATH = '/auth/login';

/**
 * Normalisasi homePath lama agar tidak mengarah ke route /fcl/* yang sudah diganti.
 */
export function normalizeHomePath(path?: string | null): string {
  const value = String(path ?? '').trim();
  if (!value.startsWith('/') || value.startsWith('/fcl/')) {
    return DEFAULT_HOME_PATH;
  }
  return value;
}

/**
 * @zh_CN 默认首页地址
 */
export const DEFAULT_HOME_PATH = '/warehouse/dashboard';

/**
 * Language option type
 */
export interface LanguageOption {
  label: string;
  value: 'en-US' | 'zh-CN' | 'id-ID';
}

/**
 * Supported languages
 */
export const SUPPORT_LANGUAGES: LanguageOption[] = [
  {
    label: '简体中文',
    value: 'zh-CN',
  },
  {
    label: 'English',
    value: 'en-US',
  },
  {
    label: 'Indonesia',
    value: 'id-ID',
  },
];
