export type Locale = 'en-US' | 'zh-CN' | 'id-ID';

export const messages: Record<Locale, Record<string, string>> = {
  'en-US': {
    cancel: 'Cancel',
    collapse: 'Collapse',
    confirm: 'Confirm',
    expand: 'Expand',
    reset: 'Reset',
    submit: 'Submit',
  },
  'zh-CN': {
    cancel: '取消',
    collapse: '收起',
    confirm: '确认',
    expand: '展开',
    reset: '重置',
    submit: '提交',
  },
  'id-ID': {
    cancel: 'Batal',
    collapse: 'Ciutkan',
    confirm: 'Konfirmasi',
    expand: 'Perluas',
    reset: 'Atur Ulang',
    submit: 'Kirim',
  },
};

export const getMessages = (locale: Locale) => messages[locale];
