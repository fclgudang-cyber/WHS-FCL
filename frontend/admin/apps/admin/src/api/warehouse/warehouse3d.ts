/**
 * Warehouse 3D — klien API ke backend FCL di dalam admin-service (/api/warehouse3d).
 * Semua data berasal dari backend; tidak ada data dummy di frontend.
 * Memakai JWT utama (satu login dengan sistem ERP/WMS).
 */

import { useAccessStore } from '@vben/stores';

export interface Wh3dObject {
  id?: number;
  objectUid: string;
  objectType: string;
  category: 'BUILDING' | 'STORAGE' | 'EQUIPMENT' | 'AREA' | 'CUSTOM' | string;
  name: string;
  label?: string;
  positionX: number;
  positionY: number;
  positionZ: number;
  length: number;
  width: number;
  height: number;
  rotation: number;
  color?: string;
  parentUid?: string;
  groupUid?: string;
  locked?: boolean;
  visible?: boolean;
  zIndex?: number;
  areaUid?: string;
  rackUid?: string;
  binUid?: string;
  locationCode?: string;
  sku?: string;
  employeeId?: number | string;
  equipmentUid?: string;
  config?: Record<string, any>;
}

export interface Wh3dArea {
  id?: number;
  areaUid: string;
  code: string;
  name: string;
  areaType: string;
  shape: string;
  length: number;
  width: number;
  height: number;
  areaM2?: number;
  positionX: number;
  positionY: number;
  positionZ: number;
  rotation: number;
  color: string;
  capacity: number;
  description?: string;
  config?: Record<string, any>;
}

export interface Wh3dRack {
  id?: number;
  rackUid: string;
  code: string;
  name: string;
  rackType: string;
  pattern: string;
  length: number;
  width: number;
  height: number;
  levels: number;
  bays: number;
  bins: number;
  capacity: number;
  positionX: number;
  positionY: number;
  positionZ: number;
  rotation: number;
  color: string;
  config?: Record<string, any>;
}

export interface Wh3dBin {
  id?: number;
  rackUid: string;
  binUid: string;
  code: string;
  level: number;
  bay: number;
  positionX: number;
  positionY: number;
  positionZ: number;
  locationCode?: string;
  capacity: number;
  occupancy: number;
  status: string;
  config?: Record<string, any>;
}

export interface Wh3dEquipment {
  id?: number;
  equipmentUid: string;
  code: string;
  name: string;
  equipmentType: string;
  status: string;
  operatorId?: number | string;
  positionX: number;
  positionY: number;
  positionZ: number;
  rotation: number;
  speed: number;
  path: Array<{ x: number; y: number }>;
  color: string;
  config?: Record<string, any>;
}

export interface Wh3dEmployee {
  employeeId: number | string;
  fullName: string;
  jabatan?: string;
  divisi?: string;
  cabang?: string;
  employeeStatus?: string;
  fingerprintId?: string;
  photoUrl?: string;
  currentArea?: string;
  activity: string;
  activityLabel?: string;
  taskCode?: string;
  currentTask?: string;
  positionX: number;
  positionY: number;
  positionZ: number;
  rotation: number;
  animation: string;
}

export interface Wh3dWarehouse {
  id?: number;
  code: string;
  name: string;
  description?: string;
  address?: string;
  lengthM: number;
  widthM: number;
  heightM: number;
  areaM2?: number;
  totalFloor: number;
  capacity: number;
  status: string;
  buildingShape: string;
  wallThickness: number;
}

export interface Wh3dKpi {
  totalStock: string;
  totalSku: string;
  availableStock: string;
  lowStock: string;
  expiring: string;
  orderToday: string;
  pendingShipping: string;
  activeWorkers: string;
  occupiedLocations: string;
  emptyLocations: string;
  activePicking: string;
  activePacking: string;
  pendingReceiving: string;
  reservedStock: string;
  totalInventory: string;
  onHand: string;
  [key: string]: string;
}

export interface Wh3dBootstrap {
  warehouseId: number;
  layoutId: number;
  warehouse: Wh3dWarehouse;
  layout: { id?: number; code?: string; name?: string; status?: string; version?: number };
  objects: Wh3dObject[];
  areas: Wh3dArea[];
  racks: Wh3dRack[];
  bins: Wh3dBin[];
  equipment: Wh3dEquipment[];
  employees: Wh3dEmployee[];
  activities: any[];
  alerts: any[];
  library: any[];
  kpi: Wh3dKpi;
}

export interface Wh3dIssue {
  severity: 'ERROR' | 'WARNING' | 'INFO';
  code: string;
  message: string;
  objectUid: string;
  x: number;
  y: number;
}

export interface Wh3dSavePayload {
  layoutId: number;
  draft: boolean;
  note?: string;
  objects: Wh3dObject[];
  areas: Wh3dArea[];
  racks: Wh3dRack[];
  bins: Wh3dBin[];
  equipment: Wh3dEquipment[];
  employees: Wh3dEmployee[];
}

async function api<T>(path: string, init?: RequestInit): Promise<T> {
  const accessStore = useAccessStore();
  const token = accessStore.accessToken;
  const res = await fetch(`/api/warehouse3d${path}`, {
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...init?.headers,
    },
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
    const err: any = new Error(body?.message || body?.detail || `HTTP ${res.status}`);
    err.status = res.status;
    err.body = body;
    throw err;
  }
  return body as T;
}

export const warehouse3dApi = {
  bootstrap: (warehouseId?: number, layoutId?: number) =>
    api<Wh3dBootstrap>(
      `/bootstrap${warehouseId ? `?warehouseId=${warehouseId}` : ''}${layoutId ? `&layoutId=${layoutId}` : ''}`,
    ),
  kpi: (warehouseId?: number) =>
    api<Wh3dKpi>(`/kpi${warehouseId ? `?warehouseId=${warehouseId}` : ''}`),
  save: (payload: Wh3dSavePayload) =>
    api<{ ok: boolean; layoutId: number }>(`/save`, {
      method: 'POST',
      body: JSON.stringify(payload),
    }),
  publish: (layoutId: number, note = '') =>
    api<{ ok: boolean; version: number }>(`/publish`, {
      method: 'POST',
      body: JSON.stringify({ layoutId, note }),
    }),
  versions: (layoutId: number) => api<any[]>(`/versions?layoutId=${layoutId}`),
  restore: (layoutId: number, version: number) =>
    api<{ ok: boolean }>(`/restore`, {
      method: 'POST',
      body: JSON.stringify({ layoutId, version }),
    }),
  regenerateBuilding: (layoutId: number) =>
    api<{ ok: boolean; layoutId: number }>(`/building/regenerate?layoutId=${layoutId}`, {
      method: 'POST',
    }),
  validate: (layoutId: number, warehouseId: number) =>
    api<{ valid: boolean; issues: Wh3dIssue[]; count: number }>(
      `/validate?layoutId=${layoutId}&warehouseId=${warehouseId}`,
    ),
  search: (q: string) => api<any[]>(`/search?q=${encodeURIComponent(q)}`),
  streamUrl: (warehouseId?: number) => {
    const token = useAccessStore().accessToken ?? '';
    const params = new URLSearchParams();
    if (warehouseId) params.set('warehouseId', String(warehouseId));
    if (token) params.set('access_token', token);
    const qs = params.toString();
    return `/api/warehouse3d/activity/stream${qs ? `?${qs}` : ''}`;
  },
};

export function num(v: unknown): number {
  const n = Number(v);
  return Number.isFinite(n) ? n : 0;
}

export function fmt(n: unknown): string {
  return new Intl.NumberFormat('id-ID').format(Math.round(num(n)));
}
