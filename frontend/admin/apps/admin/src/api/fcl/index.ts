/**
 * Klien REST untuk backend FCL (fcl-service Go di /api).
 * Nonaktifkan axios ghost agar tidak kena interceptor auth/refresh ghost —
 * FCL punya otentikasi sendiri (localStorage `fcl_user`).
 *
 * Kontrak respons fcl-service (lihat backend/app/fcl-service):
 *  - GET /api/tables/{t}          → array baris (bare array)
 *  - GET ...?count=exact&head=1   → { count: n }
 *  - POST/PUT/PATCH/DELETE        → body JSON bebas
 *  Filter: `f_<col>` (eq) atau `f_<col>__<op>` (neq|gt|gte|lt|lte|in|ilike|isnull)
 *  Order:  `order=createdAt.desc` atau `order=col.asc,col2.desc`
 */

export interface FclFilter {
  col: string;
  op?: 'eq' | 'neq' | 'gt' | 'gte' | 'lt' | 'lte' | 'in' | 'ilike' | 'isnull';
  value: string | number | boolean;
}

export interface FclQuerySpec {
  select?: string;
  filters?: FclFilter[];
  order?: string;
  limit?: number;
  offset?: number;
}

function buildQuery(spec: FclQuerySpec): string {
  const params = new URLSearchParams();
  if (spec.select) params.set('select', spec.select);
  for (const f of spec.filters ?? []) {
    if (f.value === undefined || f.value === null) continue;
    params.set('f_' + f.col + (f.op && f.op !== 'eq' ? '__' + f.op : ''), String(f.value));
  }
  if (spec.order) params.set('order', spec.order);
  if (spec.limit != null) params.set('limit', String(spec.limit));
  if (spec.offset != null) params.set('offset', String(spec.offset));
  const qs = params.toString();
  return qs ? `?${qs}` : '';
}

async function fclFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', ...init?.headers },
    ...init,
  });
  const text = await res.text();
  let body: any = null;
  try {
    body = text ? JSON.parse(text) : null;
  } catch {
    body = { message: text };
  }
  if (!res.ok) {
    const err: any = new Error(body?.message || body?.error || `HTTP ${res.status}`);
    err.status = res.status;
    err.body = body;
    throw err;
  }
  return body as T;
}

export async function fclQuery<T = any>(table: string, spec: FclQuerySpec = {}): Promise<T[]> {
  const rows = await fclFetch<T[] | null>(`/api/tables/${encodeURIComponent(table)}${buildQuery(spec)}`);
  return Array.isArray(rows) ? rows : [];
}

export async function fclCount(table: string, filters: FclFilter[] = []): Promise<number> {
  const params = new URLSearchParams({ count: 'exact', head: '1' });
  for (const f of filters) {
    params.set('f_' + f.col + (f.op && f.op !== 'eq' ? '__' + f.op : ''), String(f.value));
  }
  const body = await fclFetch<{ count: number }>(`/api/tables/${encodeURIComponent(table)}?${params.toString()}`);
  return Number(body?.count ?? 0);
}

export async function fclInsert<T = any>(table: string, rows: Record<string, any> | Record<string, any>[]): Promise<T> {
  return fclFetch<T>(`/api/tables/${encodeURIComponent(table)}`, {
    method: 'POST',
    body: JSON.stringify(Array.isArray(rows) ? rows : [rows]),
  });
}

export async function fclUpsert<T = any>(table: string, rows: Record<string, any> | Record<string, any>[], onConflict = 'id'): Promise<T> {
  return fclFetch<T>(`/api/tables/${encodeURIComponent(table)}?onConflict=${encodeURIComponent(onConflict)}`, {
    method: 'PUT',
    body: JSON.stringify(rows),
  });
}

export async function fclPatch<T = any>(table: string, id: number | string, changes: Record<string, any>): Promise<T> {
  return fclFetch<T>(`/api/tables/${encodeURIComponent(table)}?f_id=${encodeURIComponent(String(id))}`, {
    method: 'PATCH',
    body: JSON.stringify(changes),
  });
}

export async function fclDelete(table: string, id: number | string): Promise<void> {
  await fclFetch(`/api/tables/${encodeURIComponent(table)}?f_id=${encodeURIComponent(String(id))}`, {
    method: 'DELETE',
  });
}

export function formatRp(n: number | string | undefined | null): string {
  const v = Number(n ?? 0);
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    maximumFractionDigits: 0,
  }).format(Number.isFinite(v) ? v : 0);
}

const FCL_USER_KEY = 'fcl_user';

export interface FclUser {
  id?: number | string;
  username?: string;
  nama?: string;
  role?: string;
  divisi?: string;
  [key: string]: any;
}

export function getFclUser(): FclUser | null {
  try {
    const raw = localStorage.getItem(FCL_USER_KEY);
    return raw ? (JSON.parse(raw) as FclUser) : null;
  } catch {
    return null;
  }
}

export async function fclLogin(username: string, password: string): Promise<FclUser> {
  const res = await fclFetch<{ success: boolean; user?: FclUser; message?: string }>('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username, password }),
  });
  if (!res?.success || !res.user) {
    throw new Error(res?.message || 'Login FCL gagal');
  }
  localStorage.setItem(FCL_USER_KEY, JSON.stringify(res.user));
  return res.user;
}

export function fclLogout() {
  localStorage.removeItem(FCL_USER_KEY);
}


export async function fclDeleteWhere(table: string, filters: FclFilter[]): Promise<void> {
  const params = new URLSearchParams();
  for (const f of filters) {
    params.set('f_' + f.col + (f.op && f.op !== 'eq' ? '__' + f.op : ''), String(f.value));
  }
  await fclFetch(`/api/tables/${encodeURIComponent(table)}?${params.toString()}`, {
    method: 'DELETE',
  });
}
