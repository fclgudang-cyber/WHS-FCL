/**
 * @zh_CN 登录页面 url 地址
 */
export const LOGIN_PATH = '/auth/login';

/**
 * @zh_CN 默认首页地址
 */
export const DEFAULT_HOME_PATH = '/fcl/dashboard';

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
