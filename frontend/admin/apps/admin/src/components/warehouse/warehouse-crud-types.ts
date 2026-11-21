/**
 * Tipe konfigurasi bersama untuk CRUD FCL (komponen generik).
 * Dipakai oleh warehouse-crud.vue & warehouse-header-detail.vue.
 */

export type FclFieldType =
  | 'date'
  | 'datetime'
  | 'input'
  | 'number'
  | 'select'
  | 'textarea';

export interface FclField {
  name: string;
  label: string;
  type?: FclFieldType;
  required?: boolean;
  placeholder?: string;
  options?: Array<{ label: string; value: string | number }> | string[];
  rows?: number;
  min?: number;
  max?: number;
  createOnly?: boolean;
}

export interface FclDisplayColumn {
  key: string;
  title: string;
  width?: number;
  align?: 'left' | 'center' | 'right';
  fmt?: 'bool' | 'date' | 'datetime' | 'rp' | 'text';
  tag?: Record<string, { color: string; text: string }>;
  ellipsis?: boolean;
  fixed?: 'left' | 'right';
}
