<script lang="ts" setup>
/**
 * WAREHOUSE 3D — Digital Twin & Custom Layout Designer
 * - Semua objek berasal dari state/layout backend (bukan layout hardcoded).
 * - Karyawan 3D = master karyawan (tabel karyawan) via backend.
 * - Transaksi tetap dijalankan modul WMS/FCL yang sudah ada.
 */
import { computed, nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { useRouter } from 'vue-router';

import { Page } from '@vben/common-ui';
import { useAccessStore } from '@vben/stores';
import {
  LucideBuilding,
  LucideCheck,
  LucideCircleAlert,
  LucideDownload,
  LucideEye,
  LucidePencil,
  LucidePlus,
  LucideRotateCw,
  LucideTrash2,
} from '@vben/icons';
import { message, Modal, notification } from 'ant-design-vue';

import { fclCount, fclInsert, fclPatch, fclQuery, fclUpsert } from '#/api/warehouse';
import {
  fmt,
  num,
  warehouse3dApi,
  type Wh3dArea,
  type Wh3dBin,
  type Wh3dBootstrap,
  type Wh3dEmployee,
  type Wh3dEquipment,
  type Wh3dIssue,
  type Wh3dObject,
  type Wh3dRack,
} from '#/api/warehouse/warehouse3d';

import { Editor2D } from './editor2d';
import { Warehouse3DScene, type EditorMode, type TransformMode } from './scene';

// ---------------------------------------------------------------- state
const loading = ref(true);
const saving = ref(false);
const data = ref<Wh3dBootstrap | null>(null);
const objects = ref<Wh3dObject[]>([]);
const areas = ref<Wh3dArea[]>([]);
const racks = ref<Wh3dRack[]>([]);
const bins = ref<Wh3dBin[]>([]);
const equipment = ref<Wh3dEquipment[]>([]);
const employees = ref<Wh3dEmployee[]>([]);
const activities = ref<any[]>([]);
const alerts = ref<any[]>([]);
const kpi = ref<Record<string, string>>({});
const versions = ref<any[]>([]);

const selectedUid = ref<string | null>(null);
const selectedUids = ref<string[]>([]);
const mode = ref<EditorMode>('3d');
const transform = ref<TransformMode>('select');
const dirty = ref(false);
const showGrid = ref(true);
const snapToGrid = ref(true);
const rightTab = ref<'activity' | 'alerts' | 'properties'>('activity');
const searchText = ref('');
const searchResults = ref<any[]>([]);
const clipboard = ref<Wh3dObject[]>([]);
const issues = ref<Wh3dIssue[]>([]);

const history = ref<string[]>([]);
const historyIndex = ref(-1);
let restoreHistory = false;

const accessStore = useAccessStore();
watch(() => accessStore.accessToken, (token) => {
  if (token) startStream();
});

// Tampilan responsif: panel kiri/kanan jadi drawer di layar kecil.
const isMobile = ref(false);
const leftOpen = ref(false);
const rightOpen = ref(false);
function updateViewport() {
  isMobile.value = window.innerWidth <= 900;
  if (!isMobile.value) {
    leftOpen.value = false;
    rightOpen.value = false;
  }
}

const sceneEl = ref<HTMLDivElement | null>(null);
const canvas2d = ref<HTMLCanvasElement | null>(null);
const minimap = ref<HTMLCanvasElement | null>(null);
let scene: Warehouse3DScene | null = null;
let editor2d: Editor2D | null = null;
let stream: EventSource | null = null;

const warehouse = computed(() => data.value?.warehouse ?? ({} as any));
const layout = computed(() => data.value?.layout ?? ({} as any));
const selectedObject = computed(() => objects.value.find((o) => o.objectUid === selectedUid.value) ?? null);
const selectedArea = computed(() => areas.value.find((a) => a.areaUid === selectedUid.value) ?? null);
const selectedEquipment = computed(() => equipment.value.find((e) => e.equipmentUid === selectedUid.value) ?? null);
const selectedEmployee = computed(() => employees.value.find((e) => `emp-${e.employeeId}` === selectedUid.value) ?? null);

const kpiCards = computed(() => [
  { key: 'totalStock', label: 'TOTAL STOCK', value: fmt(kpi.value.totalStock), icon: '📦' },
  { key: 'orderToday', label: 'ORDER TODAY', value: fmt(kpi.value.orderToday), icon: '🧾' },
  { key: 'totalSku', label: 'TOTAL SKU', value: fmt(kpi.value.totalSku), icon: '🏷️' },
  { key: 'activePicking', label: 'PICKING', value: fmt(kpi.value.activePicking), icon: '🛒' },
  { key: 'activePacking', label: 'PACKING', value: fmt(kpi.value.activePacking), icon: '📋' },
  { key: 'activeWorkers', label: 'WORKERS', value: fmt(kpi.value.activeWorkers), icon: '👷' },
]);

// ---------------------------------------------------------------- katalog objek
interface CatalogItem { type: string; name: string; category: string; l: number; w: number; h: number; color: string; icon: string }
const catalog: CatalogItem[] = [
  { type: 'WALL', name: 'Dinding', category: 'BUILDING', l: 8, w: 0.25, h: 12, color: '#E8E4DA', icon: '▬' },
  { type: 'DOOR', name: 'Pintu', category: 'BUILDING', l: 4, w: 0.3, h: 5, color: '#3F7D8C', icon: '🚪' },
  { type: 'WINDOW', name: 'Jendela', category: 'BUILDING', l: 3, w: 0.2, h: 2, color: '#9CC9D4', icon: '▭' },
  { type: 'PILLAR', name: 'Tiang', category: 'BUILDING', l: 0.7, w: 0.7, h: 12, color: '#8C9AA3', icon: '▮' },
  { type: 'FLOOR', name: 'Lantai', category: 'BUILDING', l: 10, w: 10, h: 0.1, color: '#D9D3C7', icon: '▦' },
  { type: 'LOADING_DOCK', name: 'Loading Dock', category: 'BUILDING', l: 6, w: 3, h: 1, color: '#7A8791', icon: '🚚' },
  { type: 'RACK', name: 'Rak Pallet', category: 'STORAGE', l: 5.4, w: 1.1, h: 4.5, color: '#E08A2E', icon: '▤' },
  { type: 'SHELF', name: 'Rak Ringan', category: 'STORAGE', l: 3, w: 0.8, h: 2.4, color: '#E08A2E', icon: '▤' },
  { type: 'STORAGE_BIN', name: 'Storage Bin', category: 'STORAGE', l: 1.2, w: 1, h: 1.5, color: '#5B8C5A', icon: '◫' },
  { type: 'PALLET', name: 'Pallet', category: 'STORAGE', l: 1.2, w: 1, h: 1.5, color: '#A9763F', icon: '▣' },
  { type: 'CARTON', name: 'Kardus', category: 'STORAGE', l: 0.8, w: 0.6, h: 0.8, color: '#C98A4B', icon: '📦' },
  { type: 'BULK_STORAGE', name: 'Bulk Storage', category: 'STORAGE', l: 4, w: 3, h: 2, color: '#A9763F', icon: '▥' },
  { type: 'FORKLIFT', name: 'Forklift', category: 'EQUIPMENT', l: 2.8, w: 1.4, h: 2.6, color: '#E8A317', icon: '🚜' },
  { type: 'HAND_PALLET', name: 'Hand Pallet', category: 'EQUIPMENT', l: 1.6, w: 0.7, h: 1.2, color: '#4E7E9B', icon: '🛒' },
  { type: 'CONVEYOR', name: 'Conveyor', category: 'EQUIPMENT', l: 8, w: 1.1, h: 1.6, color: '#6C7A86', icon: '═' },
  { type: 'TROLLEY', name: 'Trolley', category: 'EQUIPMENT', l: 1.4, w: 0.8, h: 1.2, color: '#3F8F7A', icon: '🛞' },
  { type: 'PACKING_TABLE', name: 'Meja Packing', category: 'EQUIPMENT', l: 2.4, w: 1.2, h: 1.1, color: '#3F8F7A', icon: '🪑' },
  { type: 'WORKSTATION', name: 'Workstation', category: 'EQUIPMENT', l: 2, w: 1.2, h: 1.2, color: '#3F8F7A', icon: '💻' },
];
const catalogGroups = computed(() => {
  const g: Record<string, CatalogItem[]> = {};
  for (const c of catalog) (g[c.category] ||= []).push(c);
  return g;
});

const defaultAreas = [
  { code: 'RCV', name: 'Receiving Area', color: '#4C9F70' },
  { code: 'INSP', name: 'Inspection Area', color: '#5AA9A0' },
  { code: 'STR', name: 'Storage Area', color: '#3B82F6' },
  { code: 'PCK', name: 'Picking Area', color: '#E08A2E' },
  { code: 'PACK', name: 'Packing Area', color: '#3F8F7A' },
  { code: 'STG', name: 'Staging Area', color: '#8B7BC1' },
  { code: 'SHP', name: 'Shipping Area', color: '#C96A4B' },
];

// ---------------------------------------------------------------- load / save
async function load() {
  loading.value = true;
  try {
    const d = await warehouse3dApi.bootstrap();
    data.value = d;
    objects.value = d.objects ?? [];
    areas.value = d.areas ?? [];
    racks.value = d.racks ?? [];
    bins.value = d.bins ?? [];
    equipment.value = d.equipment ?? [];
    employees.value = d.employees ?? [];
    activities.value = d.activities ?? [];
    alerts.value = d.alerts ?? [];
    kpi.value = d.kpi ?? {};
    history.value = [snapshot()];
    historyIndex.value = 0;
    await nextTick();
    scene?.setWarehouse(warehouse.value);
    syncScene();
    dirty.value = false;
  } catch (e: any) {
    notification.error({ message: 'Warehouse 3D', description: e?.message || 'Gagal memuat data gudang' });
  } finally {
    loading.value = false;
  }
}

function snapshot() {
  return JSON.stringify({
    objects: objects.value, areas: areas.value, racks: racks.value,
    bins: bins.value, equipment: equipment.value,
  });
}

function pushHistory() {
  if (restoreHistory) return;
  const s = snapshot();
  if (history.value[historyIndex.value] === s) return;
  history.value = history.value.slice(0, historyIndex.value + 1);
  history.value.push(s);
  historyIndex.value = history.value.length - 1;
  dirty.value = true;
}

function applySnapshot(s: string) {
  restoreHistory = true;
  const p = JSON.parse(s);
  objects.value = p.objects ?? [];
  areas.value = p.areas ?? [];
  racks.value = p.racks ?? [];
  bins.value = p.bins ?? [];
  equipment.value = p.equipment ?? [];
  syncScene();
  restoreHistory = false;
}

function undo() {
  if (historyIndex.value <= 0) return;
  historyIndex.value--;
  applySnapshot(history.value[historyIndex.value] ?? '{}');
}
function redo() {
  if (historyIndex.value >= history.value.length - 1) return;
  historyIndex.value++;
  applySnapshot(history.value[historyIndex.value] ?? '{}');
}

function syncScene() {
  scene?.setLayout(objects.value, areas.value, employees.value, equipment.value);
  editor2d?.setData(warehouse.value, objects.value, areas.value, equipment.value, employees.value);
  drawMinimap();
}

function syncSelected() {
  scene?.select(selectedUid.value);
  editor2d?.setSelected(selectedUids.value);
  rightTab.value = 'properties';
}

function select(uid: string | null, multi = false) {
  if (!uid) {
    selectedUid.value = null;
    selectedUids.value = [];
  } else if (multi) {
    selectedUids.value = selectedUids.value.includes(uid)
      ? selectedUids.value.filter((u) => u !== uid)
      : [...selectedUids.value, uid];
    selectedUid.value = selectedUids.value.at(-1) ?? null;
  } else {
    selectedUid.value = uid;
    selectedUids.value = [uid];
  }
  syncSelected();
}

watch(mode, async (m) => {
  scene?.setMode(m);
  if (m === '2d') {
    await nextTick();
    editor2d ??= new Editor2D(canvas2d.value!, {
      onSelect: (uids) => {
        selectedUids.value = uids;
        selectedUid.value = uids.at(-1) ?? null;
        rightTab.value = 'properties';
        scene?.select(selectedUid.value);
      },
      onMove: (uid, x, y) => liveTransform(uid, { positionX: x, positionY: y }),
      onResize: (uid, l, w) => liveTransform(uid, { length: l, width: w }),
      onTransformEnd: commitTransform,
      onDraw: (x, y, l, w) => addArea(x, y, l, w),
    });
    editor2d.setSnap(snapToGrid.value);
    editor2d.setGrid(showGrid.value);
    editor2d.setData(warehouse.value, objects.value, areas.value, equipment.value, employees.value);
  }
});
watch(snapToGrid, (v) => editor2d?.setSnap(v));
watch(showGrid, (v) => editor2d?.setGrid(v));

// ---------------------------------------------------------------- objek
function addObject(c: CatalogItem, x = 0, y = 0) {
  const uid = `${c.type.toLowerCase()}-${Date.now().toString(36)}`;
  objects.value.push({
    objectUid: uid,
    objectType: c.type,
    category: c.category,
    name: c.name,
    label: c.name,
    positionX: Math.round(x * 10) / 10,
    positionY: Math.round(y * 10) / 10,
    positionZ: 0,
    length: c.l,
    width: c.w,
    height: c.h,
    rotation: 0,
    color: c.color,
    locked: false,
    visible: true,
    zIndex: objects.value.length,
    config: c.type === 'RACK' ? { levels: 3, bays: 2, showPallets: 1, cartonsPerBay: 2 } : {},
  });
  pushHistory();
  syncScene();
  select(uid);
  message.success(`${c.name} ditambahkan`);
}

function addArea(x = 0, y = 0, l = 10, w = 8, tpl?: (typeof defaultAreas)[number]) {
  const a = tpl ?? { code: 'CUSTOM', name: 'Custom Area', color: '#3B82F6' };
  const uid = `area-${Date.now().toString(36)}`;
  areas.value.push({
    areaUid: uid,
    code: `${a.code}-${areas.value.filter((v) => v.code.startsWith(a.code)).length + 1}`,
    name: a.name,
    areaType: tpl ? a.name.toUpperCase().replace(/\s+/g, '_') : 'CUSTOM',
    shape: 'RECTANGLE',
    length: l, width: w, height: 0,
    positionX: x, positionY: y, positionZ: 0, rotation: 0,
    color: a.color, capacity: 0,
    config: {},
  });
  pushHistory();
  syncScene();
  select(uid);
}

function findTransformable(uid: string): any {
  return objects.value.find((v) => v.objectUid === uid)
    ?? areas.value.find((v) => v.areaUid === uid)
    ?? equipment.value.find((v) => v.equipmentUid === uid)
    ?? null;
}

// Perubahan langsung saat drag (tanpa history / rebuild scene).
function liveTransform(uid: string, patch: Record<string, any>) {
  const target = findTransformable(uid);
  if (target) Object.assign(target, patch);
}

// Commit sekali saat drag selesai.
function commitTransform(uid: string, patch: Record<string, any>) {
  liveTransform(uid, patch);
  pushHistory();
  syncScene();
}


function patchSelected(patch: Record<string, any>) {
  const uid = selectedUid.value;
  if (!uid) return;
  const o: any = objects.value.find((v) => v.objectUid === uid)
    ?? areas.value.find((v) => v.areaUid === uid)
    ?? equipment.value.find((v) => v.equipmentUid === uid);
  if (!o) return;
  Object.assign(o, patch);
  pushHistory();
  syncScene();
}

function deleteSelected() {
  if (!selectedUids.value.length) return;
  const ids = new Set(selectedUids.value);
  objects.value = objects.value.filter((o) => !ids.has(o.objectUid));
  areas.value = areas.value.filter((a) => !ids.has(a.areaUid));
  equipment.value = equipment.value.filter((e) => !ids.has(e.equipmentUid));
  select(null);
  pushHistory();
  syncScene();
  message.success('Objek dihapus');
}

function duplicateSelected() {
  const o = selectedObject.value;
  if (!o) return;
  const copy: Wh3dObject = {
    ...o,
    objectUid: `${o.objectType.toLowerCase()}-${Date.now().toString(36)}`,
    name: `${o.name} (copy)`,
    positionX: o.positionX + 2,
    positionY: o.positionY + 2,
  };
  objects.value.push(copy);
  pushHistory();
  syncScene();
  select(copy.objectUid);
}
function copySelected() {
  if (selectedObject.value) {
    clipboard.value = [JSON.parse(JSON.stringify(selectedObject.value))];
    message.success('Objek disalin');
  }
}
function pasteSelected() {
  if (!clipboard.value.length) return;
  const c = clipboard.value[0];
  if (!c) return;
  const copy: Wh3dObject = { ...c, objectUid: `${c.objectType.toLowerCase()}-${Date.now().toString(36)}`, positionX: c.positionX + 2, positionY: c.positionY + 2 };
  objects.value.push(copy);
  pushHistory();
  syncScene();
  select(copy.objectUid);
}
function toggleLock() {
  const o: any = selectedObject.value ?? selectedArea.value;
  if (o) { o.locked = !o.locked; pushHistory(); syncScene(); }
}
function groupSelected() {
  if (selectedUids.value.length < 2) return;
  const gid = `group-${Date.now().toString(36)}`;
  for (const o of objects.value) if (selectedUids.value.includes(o.objectUid)) o.groupUid = gid;
  pushHistory();
  message.success(`${selectedUids.value.length} objek digrupkan`);
}
function alignSelected(axis: 'left' | 'right' | 'top' | 'bottom') {
  const sel = objects.value.filter((o) => selectedUids.value.includes(o.objectUid));
  if (sel.length < 2) return;
  const xs = sel.map((o) => o.positionX);
  const ys = sel.map((o) => o.positionY);
  const x = axis === 'left' ? Math.min(...xs) : axis === 'right' ? Math.max(...xs) : 0;
  const y = axis === 'top' ? Math.min(...ys) : axis === 'bottom' ? Math.max(...ys) : 0;
  for (const o of sel) {
    if (axis === 'left' || axis === 'right') o.positionX = x;
    else o.positionY = y;
  }
  pushHistory(); syncScene();
}
function distributeSelected() {
  const sel = objects.value.filter((o) => selectedUids.value.includes(o.objectUid)).sort((a, b) => a.positionX - b.positionX);
  if (sel.length < 3) return;
  const min = Math.min(...sel.map((o) => o.positionX));
  const max = Math.max(...sel.map((o) => o.positionX));
  const step = (max - min) / (sel.length - 1);
  sel.forEach((o, i) => { o.positionX = Math.round((min + i * step) * 10) / 10; });
  pushHistory(); syncScene();
}

function dragStart(e: DragEvent, c: CatalogItem) {
  e.dataTransfer?.setData('application/x-wh3d', JSON.stringify(c));
}
function onDrop(e: DragEvent) {
  const raw = e.dataTransfer?.getData('application/x-wh3d');
  if (!raw) return;
  const c = JSON.parse(raw) as CatalogItem;
  const rect = (sceneEl.value ?? canvas2d.value)?.getBoundingClientRect();
  if (!rect) return;
  addObject(c, (e.clientX - rect.left - rect.width / 2) / 8, (e.clientY - rect.top - rect.height / 2) / 8);
}

// ---------------------------------------------------------------- rack & bin
function addRack() {
  const code = `RACK-${String(racks.value.length + 1).padStart(3, '0')}`;
  const rack: Wh3dRack = {
    rackUid: `rack-${Date.now().toString(36)}`, code, name: code,
    rackType: 'PALLET_RACK', pattern: 'PARALLEL',
    length: 5.4, width: 1.1, height: 4.5,
    levels: 3, bays: 2, bins: 6, capacity: 6000,
    positionX: 0, positionY: 0, positionZ: 0, rotation: 0, color: '#E08A2E',
    config: { levels: 3, bays: 2, showPallets: 1, cartonsPerBay: 2 },
  };
  racks.value.push(rack);
  for (let lv = 1; lv <= rack.levels; lv++) {
    for (let b = 1; b <= Math.ceil(rack.bins / rack.levels); b++) {
      const idx = bins.value.length + 1;
      bins.value.push({
        rackUid: rack.rackUid,
        binUid: `${code}-B${String(idx).padStart(3, '0')}`,
        code: `${code}-L${String(lv).padStart(2, '0')}-${String(b).padStart(2, '0')}`,
        level: lv, bay: b,
        positionX: rack.positionX + (b - 1) * 2.4,
        positionY: rack.positionY,
        positionZ: lv * (rack.height / rack.levels),
        locationCode: '', capacity: 1000, occupancy: 0, status: 'EMPTY', config: {},
      });
    }
  }
  objects.value.push({
    objectUid: rack.rackUid, objectType: 'RACK', category: 'STORAGE', name: code,
    positionX: rack.positionX, positionY: rack.positionY, positionZ: 0,
    length: rack.length, width: rack.width, height: rack.height, rotation: 0,
    color: rack.color, rackUid: rack.rackUid, config: rack.config,
  });
  pushHistory(); syncScene(); select(rack.rackUid);
  message.success(`${code} dibuat dengan ${bins.value.filter((b) => b.rackUid === rack.rackUid).length} bin`);
}

// ---------------------------------------------------------------- aksi layout
async function save(draft = true) {
  if (!data.value) return;
  saving.value = true;
  try {
    await warehouse3dApi.save({
      layoutId: data.value.layoutId, draft,
      objects: objects.value, areas: areas.value, racks: racks.value,
      bins: bins.value, equipment: equipment.value, employees: employees.value,
    });
    dirty.value = false;
    notification.success({ message: draft ? 'Draft tersimpan' : 'Layout dipublish', description: `${objects.value.length} objek tersimpan di database` });
  } catch (e: any) {
    notification.error({ message: 'Gagal menyimpan', description: e?.message });
  } finally { saving.value = false; }
}

async function regenerateBuilding() {
  if (!data.value) return;
  await new Promise<void>((resolve) => {
    Modal.confirm({
      title: 'Buat ulang bangunan?',
      content: 'Dinding, pintu, jendela, tiang, lantai dan loading dock akan dibuat ulang sesuai dimensi gudang. Objek lain tidak dihapus.',
      okText: 'Buat Ulang',
      cancelText: 'Batal',
      onOk: () => resolve(),
      onCancel: () => resolve(),
    });
  });
  try {
    await warehouse3dApi.regenerateBuilding(data.value.layoutId);
    await load();
    message.success('Bangunan dibuat ulang dan bisa langsung dipindah/diedit');
  } catch (e: any) {
    notification.error({ message: 'Gagal membuat bangunan', description: e?.message });
  }
}

async function validate() {
  try {
    const r = await warehouse3dApi.validate(data.value!.layoutId, data.value!.warehouseId);
    issues.value = r.issues ?? [];
    Modal.info({
      title: r.valid ? 'Validasi Layout Lulus' : `${r.count} Masalah Ditemukan`,
      content: r.valid
        ? 'Tidak ditemukan konflik layout.'
        : r.issues.map((i) => `${i.code}: ${i.message}`).join('\n'),
      width: 520,
    });
    if (issues.value[0]) select(issues.value[0].objectUid || null);
  } catch (e: any) { notification.error({ message: 'Validasi gagal', description: e?.message }); }
}

async function publish() {
  const r = await warehouse3dApi.validate(data.value!.layoutId, data.value!.warehouseId);
  if (!r.valid) {
    const ok = await new Promise<boolean>((resolve) => {
      Modal.confirm({
        title: `${r.count} masalah ditemukan`,
        content: ' Tetap publish layout? Transaksi inventory tidak akan diubah.',
        onOk: () => resolve(true), onCancel: () => resolve(false),
      });
    });
    if (!ok) return;
  }
  await save(false);
  await warehouse3dApi.publish(data.value!.layoutId, 'Publish dari Warehouse 3D');
  await loadVersions();
  notification.success({ message: 'Layout dipublish', description: 'Version history diperbarui' });
}

async function loadVersions() {
  versions.value = await warehouse3dApi.versions(data.value!.layoutId);
}
async function restoreVersion(v: number) {
  await warehouse3dApi.restore(data.value!.layoutId, v);
  await load();
  message.success(`Versi ${v} dipulihkan`);
}

function exportLayout() {
  const payload = { objects: objects.value, areas: areas.value, racks: racks.value, bins: bins.value, equipment: equipment.value };
  const blob = new Blob([JSON.stringify(payload, null, 2)], { type: 'application/json' });
  const a = document.createElement('a');
  a.href = URL.createObjectURL(blob);
  a.download = `warehouse3d-layout-${Date.now()}.json`;
  a.click();
  URL.revokeObjectURL(a.href);
}
function importLayout(e: Event) {
  const input = e.target as HTMLInputElement;
  const file = input.files?.[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => {
    try {
      const p = JSON.parse(String(reader.result));
      objects.value = p.objects ?? [];
      areas.value = p.areas ?? [];
      racks.value = p.racks ?? [];
      bins.value = p.bins ?? [];
      equipment.value = p.equipment ?? [];
      pushHistory(); syncScene();
      message.success('Layout diimpor');
    } catch { message.error('File JSON tidak valid'); }
  };
  reader.readAsText(file);
  input.value = '';
}

// ---------------------------------------------------------------- search & kamera
async function doSearch() {
  if (!searchText.value.trim()) { searchResults.value = []; return; }
  try { searchResults.value = await warehouse3dApi.search(searchText.value); }
  catch { searchResults.value = []; }
}
function jumpTo(r: any) {
  const code = String(r.code ?? '');
  const found = objects.value.find((o) => o.sku === code || o.locationCode === code || o.objectUid === code)
    ?? bins.value.find((b) => b.locationCode === code || b.binUid === code)
    ?? equipment.value.find((e) => e.equipmentUid === code);
  if (found) {
    scene?.focusPoint(num((found as any).positionX), num((found as any).positionY));
    select((found as any).objectUid ?? (found as any).equipmentUid ?? null);
  } else if (r.type === 'EMPLOYEE') {
    const e2 = employees.value.find((p) => String(p.employeeId) === code);
    if (e2) { scene?.focusPoint(num(e2.positionX), num(e2.positionY)); select(`emp-${e2.employeeId}`); }
  }
  if (found) {
    notification.info({
      message: r.name || code,
      description: `Tipe: ${r.type} · Lokasi: ${r.location || '-'} · Ref: ${r.ref || '-'}`,
    });
  }
  searchText.value = '';
  searchResults.value = [];
}
function focusIssue(i: Wh3dIssue) {
  scene?.focusPoint(i.x, i.y);
  if (i.objectUid) select(i.objectUid);
}

// ---------------------------------------------------------------- live stream
function startStream() {
  stream?.close();
  try {
    stream = new EventSource(warehouse3dApi.streamUrl(data.value?.warehouseId));
    stream.addEventListener('tick', (ev: any) => {
      const p = JSON.parse(ev.data);
      if (Array.isArray(p.activities)) activities.value = p.activities;
      if (Array.isArray(p.alerts)) alerts.value = p.alerts;
      if (p.kpi) kpi.value = p.kpi;
    });
  } catch { /* SSE opsional */ }
}

// ---------------------------------------------------------------- minimap
function drawMinimap() {
  const c = minimap.value;
  if (!c) return;
  const ctx = c.getContext('2d');
  if (!ctx) return;
  const w = c.width = c.clientWidth * 2;
  const h = c.height = c.clientHeight * 2;
  ctx.clearRect(0, 0, w, h);
  ctx.fillStyle = 'rgba(245,247,248,0.94)';
  ctx.fillRect(0, 0, w, h);
  const L = num(warehouse.value.lengthM) || 100;
  const W = num(warehouse.value.widthM) || 50;
  const s = Math.min((w - 16) / L, (h - 16) / W);
  const cx = w / 2; const cy = h / 2;
  const X = (x: number) => cx + x * s;
  const Y = (y: number) => cy + y * s;
  ctx.strokeStyle = '#5b6c74'; ctx.lineWidth = 2;
  ctx.strokeRect(X(-L / 2), Y(-W / 2), L * s, W * s);
  for (const a of areas.value) {
    ctx.fillStyle = (a.color || '#3B82F6') + '44';
    ctx.fillRect(X(a.positionX - a.length / 2), Y(a.positionY - a.width / 2), a.length * s, a.width * s);
  }
  for (const o of objects.value) {
    if (o.objectType === 'FLOOR') continue;
    ctx.fillStyle = o.color || '#E08A2E';
    ctx.fillRect(X(o.positionX - o.length / 2), Y(o.positionY - o.width / 2), Math.max(2, o.length * s), Math.max(2, o.width * s));
  }
  for (const e of equipment.value) {
    ctx.fillStyle = '#E8A317';
    ctx.beginPath(); ctx.arc(X(e.positionX), Y(e.positionY), 4, 0, Math.PI * 2); ctx.fill();
  }
  for (const p of employees.value) {
    ctx.fillStyle = p.activity === 'IDLE' ? '#90A0A8' : '#157E8C';
    ctx.beginPath(); ctx.arc(X(p.positionX), Y(p.positionY), 3, 0, Math.PI * 2); ctx.fill();
  }
}

// ---------------------------------------------------------------- Warehouse info modal
const infoOpen = ref(false);
const infoForm = reactive({
  code: '', name: '', description: '', address: '',
  lengthM: 100, widthM: 50, heightM: 12, totalFloor: 1, capacity: 5000, status: 'ACTIVE',
});
const buildingArea = computed(() => (num(infoForm.lengthM) * num(infoForm.widthM)).toLocaleString('id-ID'));
function openInfo() {
  const w = warehouse.value;
  Object.assign(infoForm, {
    code: w.code ?? '', name: w.name ?? '', description: w.description ?? '', address: w.address ?? '',
    lengthM: num(w.lengthM) || 100, widthM: num(w.widthM) || 50, heightM: num(w.heightM) || 12,
    totalFloor: num(w.totalFloor) || 1, capacity: num(w.capacity) || 5000, status: w.status ?? 'ACTIVE',
  });
  infoOpen.value = true;
}
async function saveInfo() {
  if (!data.value) return;
  await fclUpsert('wh3d_warehouse', [{
    id: data.value.warehouseId, code: infoForm.code, name: infoForm.name,
    description: infoForm.description, address: infoForm.address,
    length_m: num(infoForm.lengthM), width_m: num(infoForm.widthM), height_m: num(infoForm.heightM),
    total_floor: num(infoForm.totalFloor), capacity: num(infoForm.capacity), status: infoForm.status,
    updated_at: new Date().toISOString(),
  }], 'id');
  infoOpen.value = false;
  await load();
  message.success('Informasi gudang tersimpan');
}

const properties = computed(() => {
  const o: any = selectedObject.value ?? selectedArea.value ?? selectedEquipment.value ?? selectedEmployee.value;
  return o ?? null;
});

// ---------------------------------------------------------------- action bar (backend nyata)
const router = useRouter();
const moduleRoutes: Record<string, string> = {
  Orders: '/warehouse/inventory/order',
  Inventory: '/warehouse/inventory/stock',
  Locations: '/warehouse/inventory/warehouse3d',
  Analytics: '/warehouse/inventory/analisis',
  Workers: '/opm/karyawan',
};

async function openModule(name: string) {
  if (name === 'Tasks') {
    const n = await fclCount('wh3d_task', [{ col: 'status', op: 'neq', value: 'DONE' }]);
    notification.info({ message: 'Tasks', description: `${n} task aktif di Warehouse 3D` });
    await warehouse3dApi.bootstrap().then((d) => { activities.value = d.activities ?? []; });
    rightTab.value = 'activity';
    return;
  }
  if (name === 'Equipment') {
    const rows = await fclQuery('wh3d_equipment', { select: 'equipment_uid,code,name,status', limit: 100 });
    notification.info({ message: 'Equipment', description: `${rows.length} unit terdaftar` });
    return;
  }
  if (name === 'Locations') {
    const n = await fclCount('wh3d_bin');
    notification.info({ message: 'Locations', description: `${n} bin terdaftar di layout` });
    return;
  }
  const path = moduleRoutes[name];
  if (path) await router.push(path);
}

async function pickForSale() {
  try {
    const orders = await fclQuery('order', { select: 'id,noresi,keterangan', order: 'createdAt.desc', limit: 1 });
    const order = orders[0];
    await fclInsert('wh3d_task', [{
      task_uid: `PICK-${Date.now()}`,
      task_type: 'PICKING',
      reference_no: String(order?.noresi ?? order?.id ?? '-'),
      location: selectedObject.value?.locationCode ?? null,
      status: 'PENDING',
      priority: 'NORMAL',
      created_at: new Date().toISOString(),
    }]);
    notification.success({ message: 'Pick for Sale', description: `Task picking dibuat untuk ${order?.noresi ?? 'order terbaru'}` });
  } catch (e: any) { notification.error({ message: 'Gagal membuat task', description: e?.message }); }
}

async function setLatestTaskStatus(status: string) {
  try {
    const tasks = await fclQuery('wh3d_task', { select: 'id,task_uid', order: 'created_at.desc', limit: 1 });
    const t = tasks[0];
    if (!t) { message.warning('Belum ada task'); return; }
    await fclPatch('wh3d_task', t.id, {
      status,
      ...(status === 'DONE' ? { finished_at: new Date().toISOString() } : {}),
    });
    notification.success({ message: 'Task diperbarui', description: `${t.task_uid} → ${status}` });
  } catch (e: any) { notification.error({ message: 'Gagal memperbarui task', description: e?.message }); }
}

onMounted(async () => {
  updateViewport();
  window.addEventListener('resize', updateViewport);
  await load();
  if (sceneEl.value) {
    scene = new Warehouse3DScene(sceneEl.value, {
      onSelect: (uid) => { selectedUid.value = uid; selectedUids.value = uid ? [uid] : []; if (uid) rightTab.value = 'properties'; },
      onMove: (uid, x, y) => liveTransform(uid, { positionX: x, positionY: y }),
      onRotate: (uid, rotation) => liveTransform(uid, { rotation }),
      onResize: (uid, length, width) => liveTransform(uid, { length, width }),
      onTransformEnd: commitTransform,
    });
    scene.setWarehouse(warehouse.value);
    scene.setLayout(objects.value, areas.value, employees.value, equipment.value);
  }
  startStream();
  await loadVersions();
});

onBeforeUnmount(() => {
  window.removeEventListener('resize', updateViewport);
  scene?.dispose();
  editor2d?.dispose();
  stream?.close();
});
</script>

<template>
  <Page auto-content-height class="wh3d-page">
    <!-- ================= TOP ================= -->
    <div class="wh3d-top">
      <div class="wh3d-title">
        <div class="wh3d-logo"><LucideBuilding :size="22" /></div>
        <div>
          <div class="wh3d-name">{{ warehouse.name || 'Warehouse 3D' }}</div>
          <div class="wh3d-sub">
            <span class="wh3d-code">{{ warehouse.code || '-' }}</span>
            <span class="dot">•</span>
            <span>{{ fmt(warehouse.areaM2 ?? num(warehouse.lengthM) * num(warehouse.widthM)) }} m²</span>
            <span class="dot">•</span>
            <span>{{ warehouse.status || '-' }}</span>
            <span class="dot">•</span>
            <span>Layout v{{ layout.version || 1 }} · {{ layout.status || 'DRAFT' }}</span>
          </div>
        </div>
        <a-button size="small" class="ml-2" @click="openInfo">
          <LucidePencil :size="14" /> Info Gudang
        </a-button>
      </div>
      <div class="wh3d-kpis">
        <div v-for="k in kpiCards" :key="k.key" class="kpi">
          <div class="kpi-icon">{{ k.icon }}</div>
          <div>
            <div class="kpi-value">{{ k.value }}</div>
            <div class="kpi-label">{{ k.label }}</div>
          </div>
        </div>
      </div>
    </div>

    <!-- ================= TOOLBAR ================= -->
    <div class="wh3d-toolbar">
      <div class="tool-group mobile-only">
        <a-button size="small" @click="leftOpen = !leftOpen">📦 Library</a-button>
        <a-button size="small" @click="rightOpen = !rightOpen">📊 Panel</a-button>
      </div>
      <span class="sep mobile-only" />
      <div class="tool-group">
        <a-tooltip title="Select"><a-button :type="transform === 'select' ? 'primary' : 'text'" size="small" @click="transform='select'; scene?.setTransform('select')"><LucideEye :size="15" /></a-button></a-tooltip>
        <a-tooltip title="Move"><a-button :type="transform === 'move' ? 'primary' : 'text'" size="small" @click="transform='move'; scene?.setTransform('move')">✥</a-button></a-tooltip>
        <a-tooltip title="Rotate"><a-button :type="transform === 'rotate' ? 'primary' : 'text'" size="small" @click="transform='rotate'; scene?.setTransform('rotate')"><LucideRotateCw :size="15" /></a-button></a-tooltip>
        <a-tooltip title="Resize"><a-button :type="transform === 'resize' ? 'primary' : 'text'" size="small" @click="transform='resize'; scene?.setTransform('resize')">⤢</a-button></a-tooltip>
      </div>
      <span class="sep" />
      <div class="tool-group">
        <a-button size="small" :type="mode === '2d' ? 'primary' : 'text'" @click="mode='2d'">▦ 2D</a-button>
        <a-button size="small" :type="mode === '3d' ? 'primary' : 'text'" @click="mode='3d'">▣ 3D</a-button>
      </div>
      <span class="sep" />
      <div class="tool-group">
        <a-tooltip title="Undo"><a-button size="small" :disabled="historyIndex <= 0" @click="undo">↶</a-button></a-tooltip>
        <a-tooltip title="Redo"><a-button size="small" :disabled="historyIndex >= history.length - 1" @click="redo">↷</a-button></a-tooltip>
        <a-tooltip title="Duplicate"><a-button size="small" @click="duplicateSelected">📋</a-button></a-tooltip>
        <a-popconfirm title="Hapus objek terpilih?" ok-text="Ya" cancel-text="Batal" @confirm="deleteSelected">
          <a-tooltip title="Delete"><a-button size="small" danger><LucideTrash2 :size="15" /></a-button></a-tooltip>
        </a-popconfirm>
        <a-tooltip title="Group"><a-button size="small" @click="groupSelected">⌘</a-button></a-tooltip>
        <a-tooltip title="Lock / Unlock"><a-button size="small" @click="toggleLock"><span>{{ selectedObject?.locked ? '🔓' : '🔒' }}</span></a-button></a-tooltip>
      </div>
      <span class="sep" />
      <div class="tool-group">
        <a-tooltip title="Align left"><a-button size="small" @click="alignSelected('left')">↤</a-button></a-tooltip>
        <a-tooltip title="Align right"><a-button size="small" @click="alignSelected('right')">↦</a-button></a-tooltip>
        <a-tooltip title="Distribute"><a-button size="small" @click="distributeSelected">⇹</a-button></a-tooltip>
        <a-tooltip title="Grid"><a-button size="small" :type="showGrid ? 'primary' : 'text'" @click="showGrid=!showGrid">▦</a-button></a-tooltip>
        <a-tooltip title="Snap to grid"><a-button size="small" :type="snapToGrid ? 'primary' : 'text'" @click="snapToGrid=!snapToGrid">SNAP</a-button></a-tooltip>
      </div>
      <span class="sep" />
      <div class="tool-group">
        <a-button size="small" @click="scene?.setView('iso')">Isometric</a-button>
        <a-button size="small" @click="scene?.setView('top')">Top View</a-button>
        <a-button size="small" @click="scene?.resetCamera()">Reset</a-button>
        <a-button size="small" @click="scene?.zoom(1.2)">+</a-button>
        <a-button size="small" @click="scene?.zoom(0.8)">−</a-button>
      </div>
      <div class="tool-spacer" />
      <div class="tool-group">
        <a-button size="small" @click="regenerateBuilding">🏗️ Build Ulang</a-button>
        <a-button size="small" @click="validate"><LucideCheck :size="14" /> Validasi</a-button>
        <a-button size="small" :loading="saving" @click="save(true)"><LucideCheck :size="14" /> Save Draft</a-button>
        <a-button size="small" type="primary" @click="publish">⬆ Publish</a-button>
      </div>
    </div>

    <div v-if="isMobile && (leftOpen || rightOpen)" class="drawer-backdrop" @click="leftOpen = false; rightOpen = false" />

    <!-- ================= BODY ================= -->
    <div class="wh3d-body">
      <!-- LEFT: Object library -->
      <aside class="wh3d-left" :class="{ open: leftOpen }">
        <div class="panel-head">
          <span>▦</span> OBJECT LIBRARY
          <a-button v-if="isMobile" class="mobile-only ml-auto" size="small" type="text" @click="leftOpen = false">✕</a-button>
        </div>
        <div class="search-wrap">
          <span>🔍</span>
          <a-input v-model:value="searchText" size="small" variant="borderless" placeholder="SKU / Bin / Karyawan / Order" @press-enter="doSearch" />
        </div>
        <div v-if="searchResults.length" class="search-results">
          <div v-for="(r, i) in searchResults" :key="i" class="search-item" @click="jumpTo(r)">
            <b>{{ r.code }}</b><span>{{ r.name }}</span><small>{{ r.type }} · {{ r.location }}</small>
          </div>
        </div>

        <div class="lib-section">
          <div class="lib-title"><LucideBuilding :size="14" /> AREA</div>
          <div class="area-grid">
            <a-button v-for="a in defaultAreas" :key="a.code" size="small" class="area-chip" :style="{ borderColor: a.color }" @click="addArea(0, 0, 12, 10, a)">
              <span class="swatch" :style="{ background: a.color }" />{{ a.name }}
            </a-button>
            <a-button size="small" class="area-chip custom" @click="addArea(0, 0, 10, 10)"><LucidePlus :size="12" /> Custom Area</a-button>
          </div>
        </div>

        <div v-for="(items, group) in catalogGroups" :key="group" class="lib-section">
          <div class="lib-title"><span>{{ group === 'BUILDING' ? '🏗️' : group === 'STORAGE' ? '📦' : '⚙️' }}</span> {{ group }}</div>
          <div class="lib-grid">
            <div
              v-for="c in items" :key="c.type" class="lib-item" draggable="true"
              @dragstart="dragStart($event, c)" @click="addObject(c)"
              :title="`Klik / tap untuk menambah · ${c.l} × ${c.w} × ${c.h} m`"
            >
              <div class="lib-icon">{{ c.icon }}</div>
              <span>{{ c.name }}</span>
            </div>
          </div>
        </div>

        <div class="lib-section">
          <div class="lib-title">RACK DESIGNER</div>
          <a-button block size="small" type="dashed" @click="addRack"><LucidePlus :size="14" /> Buat Rack + Bin</a-button>
        </div>
      </aside>

      <!-- CENTER: canvas -->
      <main class="wh3d-center" @dragover.prevent @drop.prevent="onDrop">
        <div ref="sceneEl" class="scene-host" :style="{ display: mode === '3d' ? 'block' : 'none' }" />
        <canvas ref="canvas2d" class="canvas2d" :style="{ display: mode === '2d' ? 'block' : 'none' }" />

        <div v-if="loading" class="scene-loading">Memuat Warehouse 3D…</div>
        <div v-else-if="!objects.length && !areas.length" class="scene-hint">
          Belum ada objek. Tap / klik item di <b>Object Library</b> untuk menambahkan.
        </div>

        <div class="scene-badge">
          <span class="pulse" /> LIVE · {{ employees.filter(e => e.activity !== 'IDLE').length }} pekerja aktif
          <span v-if="dirty" class="dirty">● belum disimpan</span>
        </div>

        <div v-if="issues.length" class="issue-strip">
          <LucideCircleAlert :size="14" /> {{ issues.length }} konflik layout
          <button v-for="(i, ix) in issues.slice(0, 3)" :key="ix" @click="focusIssue(i)">{{ i.code }}</button>
        </div>

        <!-- minimap -->
        <div class="minimap-wrap">
          <div class="minimap-title"><LucideEye :size="12" /> MINIMAP</div>
          <canvas ref="minimap" class="minimap-canvas" />
        </div>
      </main>

      <!-- RIGHT -->
      <aside class="wh3d-right" :class="{ open: rightOpen }">
        <div class="right-tabs">
          <a-button v-if="isMobile" class="mobile-only panel-close" size="small" type="text" @click="rightOpen = false">✕</a-button>
          <button :class="{ active: rightTab === 'activity' }" @click="rightTab='activity'">Aktivitas</button>
          <button :class="{ active: rightTab === 'alerts' }" @click="rightTab='alerts'">Cases & Alerts <b v-if="alerts.length">{{ alerts.length }}</b></button>
          <button :class="{ active: rightTab === 'properties' }" @click="rightTab='properties'">Properties</button>
        </div>

        <div v-if="rightTab === 'activity'" class="right-list">
          <div v-for="(a, i) in activities" :key="i" class="act-row">
            <div class="act-dot" :class="String(a.status || '').toLowerCase()" />
            <div class="act-body">
              <div class="act-title">{{ a.title || a.event_type }}</div>
              <div class="act-meta">{{ a.employee || '-' }} · {{ a.location || '-' }}</div>
              <div class="act-meta">{{ a.sku || '-' }} · {{ a.quantity ?? '' }} · <span class="time">{{ a.event_time }}</span></div>
            </div>
          </div>
          <div v-if="!activities.length" class="empty">Belum ada aktivitas live.</div>
        </div>

        <div v-else-if="rightTab === 'alerts'" class="right-list">
          <div v-for="a in alerts" :key="a.id" class="alert-row" @click="scene?.focusPoint(num(a.position_x), num(a.position_y)); a.object_uid && select(a.object_uid)">
            <LucideCircleAlert :size="16" :color="a.severity === 'CRITICAL' ? '#c0392b' : '#d68910'" />
            <div>
              <b>{{ a.title }}</b>
              <div class="act-meta">{{ a.message }}</div>
              <div class="act-meta">{{ a.location || '-' }}</div>
            </div>
          </div>
          <div v-if="!alerts.length" class="empty">Tidak ada alert terbuka.</div>
        </div>

        <div v-else class="props">
          <template v-if="properties">
            <div class="prop-head">
              <div class="prop-icon">▣</div>
              <div>
                <b>{{ properties.name || properties.fullName }}</b>
                <small>{{ properties.objectType || properties.areaType || properties.equipmentType || properties.jabatan }}</small>
              </div>
            </div>

            <template v-if="selectedEmployee">
              <div class="emp-card">
                <div class="avatar">{{ (selectedEmployee.fullName || '?').slice(0, 1) }}</div>
                <div>
                  <b>{{ selectedEmployee.fullName }}</b>
                  <div>{{ selectedEmployee.jabatan }} · {{ selectedEmployee.divisi }}</div>
                  <div>ID: {{ selectedEmployee.employeeId }} · {{ selectedEmployee.employeeStatus }}</div>
                  <div class="act-live">● {{ selectedEmployee.activityLabel || selectedEmployee.activity }}</div>
                  <div v-if="selectedEmployee.currentTask">Task: {{ selectedEmployee.currentTask }}</div>
                </div>
              </div>
            </template>

            <template v-else>
              <div class="prop-group-title">POSISI (meter)</div>
              <div class="prop-grid">
                <label>X<input type="number" step="0.1" :value="properties.positionX" @change="patchSelected({ positionX: num(($event.target as HTMLInputElement).value) })" /></label>
                <label>Y<input type="number" step="0.1" :value="properties.positionY" @change="patchSelected({ positionY: num(($event.target as HTMLInputElement).value) })" /></label>
                <label>Z<input type="number" step="0.1" :value="properties.positionZ" @change="patchSelected({ positionZ: num(($event.target as HTMLInputElement).value) })" /></label>
              </div>
              <div class="prop-group-title">DIMENSI (meter)</div>
              <div class="prop-grid">
                <label>Length<input type="number" step="0.1" :value="properties.length" @change="patchSelected({ length: num(($event.target as HTMLInputElement).value) })" /></label>
                <label>Width<input type="number" step="0.1" :value="properties.width" @change="patchSelected({ width: num(($event.target as HTMLInputElement).value) })" /></label>
                <label>Height<input type="number" step="0.1" :value="properties.height" @change="patchSelected({ height: num(($event.target as HTMLInputElement).value) })" /></label>
              </div>
              <div class="prop-group-title">ROTASI & WARNA</div>
              <div class="prop-grid">
                <label>Rotation (0-360)<input type="number" min="0" max="360" :value="properties.rotation" @change="patchSelected({ rotation: num(($event.target as HTMLInputElement).value) })" /></label>
                <label class="wide">Color<input type="color" :value="properties.color || '#3B82F6'" @change="patchSelected({ color: ($event.target as HTMLInputElement).value })" /></label>
              </div>
              <div class="prop-grid mt">
                <a-button size="small" @click="duplicateSelected">📋 Duplicate</a-button>
                <a-button size="small" @click="copySelected">Copy</a-button>
                <a-button size="small" @click="pasteSelected">Paste</a-button>
                <a-button size="small" danger @click="deleteSelected"><LucideTrash2 :size="13" /> Hapus</a-button>
              </div>
            </template>
          </template>
          <div v-else class="empty">Klik objek di scene / 2D untuk mengedit propertinya.</div>
        </div>

        <div class="right-foot">
          <div class="vers-title">VERSION HISTORY</div>
          <div class="vers-list">
            <div v-for="v in versions" :key="v.id" class="vers-row">
              <span>v{{ v.version }}</span><small>{{ String(v.created_at || '').slice(0, 16) }}</small>
              <a-button size="small" @click="restoreVersion(v.version)">Restore</a-button>
            </div>
            <div v-if="!versions.length" class="empty small">Belum ada versi publish.</div>
          </div>
          <div class="foot-actions">
            <a-button size="small" @click="exportLayout"><LucideDownload :size="13" /> Export</a-button>
            <label class="import-btn">⬆ Import<input type="file" accept="application/json" @change="importLayout" /></label>
          </div>
        </div>
      </aside>
    </div>

    <!-- ================= BOTTOM ================= -->
    <div class="wh3d-bottom">
      <div class="bottom-nav">
        <a-button v-for="n in ['Orders','Tasks','Inventory','Locations','Equipment','Workers','Analytics']" :key="n" size="small" @click="openModule(n)">{{ n }}</a-button>
      </div>
      <div class="bottom-actions">
        <a-button size="small" type="primary" @click="pickForSale">Pick for Sale</a-button>
        <a-button size="small" @click="setLatestTaskStatus('CANCELLED')">Cancel Task</a-button>
        <a-button size="small" @click="setLatestTaskStatus('DONE')">Mark Complete</a-button>
        <a-button size="small" @click="rightTab = 'properties'">More Actions</a-button>
        <a-button size="small" @click="rightTab = 'activity'">Watch</a-button>
      </div>
    </div>

    <!-- Warehouse info modal -->
    <a-modal v-model:open="infoOpen" title="Informasi Gudang" @ok="saveInfo" ok-text="Simpan" cancel-text="Batal">
      <a-form layout="vertical">
        <div class="form-grid">
          <a-form-item label="Warehouse Code"><a-input v-model:value="infoForm.code" /></a-form-item>
          <a-form-item label="Warehouse Name"><a-input v-model:value="infoForm.name" /></a-form-item>
        </div>
        <a-form-item label="Description"><a-input v-model:value="infoForm.description" /></a-form-item>
        <a-form-item label="Address"><a-input v-model:value="infoForm.address" /></a-form-item>
        <div class="form-grid">
          <a-form-item label="Building Length (m)"><a-input-number v-model:value="infoForm.lengthM" :min="1" class="w-full" /></a-form-item>
          <a-form-item label="Building Width (m)"><a-input-number v-model:value="infoForm.widthM" :min="1" class="w-full" /></a-form-item>
          <a-form-item label="Building Height (m)"><a-input-number v-model:value="infoForm.heightM" :min="1" class="w-full" /></a-form-item>
          <a-form-item label="Total Floor"><a-input-number v-model:value="infoForm.totalFloor" :min="1" class="w-full" /></a-form-item>
          <a-form-item label="Capacity"><a-input-number v-model:value="infoForm.capacity" :min="0" class="w-full" /></a-form-item>
          <a-form-item label="Status">
            <a-select v-model:value="infoForm.status">
              <a-select-option value="ACTIVE">ACTIVE</a-select-option>
              <a-select-option value="INACTIVE">INACTIVE</a-select-option>
              <a-select-option value="MAINTENANCE">MAINTENANCE</a-select-option>
            </a-select>
          </a-form-item>
        </div>
        <div class="area-auto">Building Area = Length × Width = <b>{{ buildingArea }} m²</b> (dihitung otomatis)</div>
      </a-form>
    </a-modal>
  </Page>
</template>

<style scoped>
.wh3d-page { padding: 0 !important; height: 100%; display: flex; flex-direction: column; overflow: hidden; }
.mobile-only { display: none; }
.wh3d-top { display:flex; align-items:center; justify-content:space-between; gap:16px; padding:10px 16px; background:hsl(var(--card)); border-bottom:1px solid hsl(var(--border)); flex-shrink:0; }
.wh3d-title { display:flex; align-items:center; gap:10px; }
.wh3d-logo { width:38px;height:38px;border-radius:10px;display:grid;place-items:center;background:#e6f2f4;color:#1f7a86; }
.wh3d-name { font-size:16px;font-weight:800; }
.wh3d-sub { font-size:11px;color:hsl(var(--muted-foreground)); display:flex; gap:7px; align-items:center; }
.wh3d-code { background:#eef3f5;padding:1px 7px;border-radius:4px;font-weight:700; }
.dot { opacity:.5; }
.ml-2 { margin-left:8px; }
.wh3d-kpis { display:flex; gap:8px; flex-wrap:wrap; }
.kpi { display:flex;gap:8px;align-items:center;background:hsl(var(--muted)/.6);border:1px solid hsl(var(--border));border-radius:10px;padding:6px 12px;min-width:112px; }
.kpi-icon { font-size:17px; }
.kpi-value { font-size:16px;font-weight:800;line-height:1.1; }
.kpi-label { font-size:9px;letter-spacing:.06em;color:hsl(var(--muted-foreground)); }
.wh3d-toolbar { display:flex;align-items:center;gap:8px;padding:6px 12px;background:hsl(var(--card));border-bottom:1px solid hsl(var(--border));flex-wrap:wrap; }
.tool-group { display:flex;gap:3px;align-items:center; }
.tool-spacer { flex:1; }
.sep { width:1px;height:22px;background:hsl(var(--border)); }
.wh3d-body { display:grid;grid-template-columns:238px 1fr 292px;gap:8px;padding:8px;flex:1;min-height:0; }
.wh3d-center { min-width:0; }
.scene-loading,.scene-hint { position:absolute;inset:0;display:grid;place-items:center;background:rgba(238,241,243,.82);z-index:5;font-size:12px;font-weight:700;color:#456;pointer-events:none; }
.scene-hint { background:rgba(238,241,243,.55); }
.drawer-backdrop { position:fixed;inset:0;background:rgba(15,30,38,.38);z-index:55; }
.wh3d-left,.wh3d-right { background:hsl(var(--card));border:1px solid hsl(var(--border));border-radius:12px;overflow:auto; }
.panel-head { display:flex;align-items:center;gap:7px;padding:10px 12px;font-size:11px;font-weight:800;letter-spacing:.08em;border-bottom:1px solid hsl(var(--border)); }
.search-wrap { display:flex;align-items:center;gap:6px;margin:8px;padding:6px 9px;border:1px solid hsl(var(--border));border-radius:8px; }
.search-wrap input { border:0;outline:0;background:transparent;width:100%;font-size:12px; }
.search-results { margin:0 8px 8px;border:1px solid hsl(var(--border));border-radius:8px;max-height:190px;overflow:auto; }
.search-item { padding:6px 9px;border-bottom:1px solid hsl(var(--border));cursor:pointer;font-size:11px;display:flex;flex-direction:column; }
.search-item:hover { background:hsl(var(--muted)/.7); }
.search-item small { color:hsl(var(--muted-foreground)); }
.lib-section { padding:8px 10px;border-top:1px solid hsl(var(--border)); }
.lib-title { display:flex;gap:6px;align-items:center;font-size:10px;font-weight:800;color:hsl(var(--muted-foreground));margin-bottom:7px;letter-spacing:.06em; }
.lib-grid { display:grid;grid-template-columns:repeat(3,1fr);gap:5px; }
.lib-item { border:1px solid hsl(var(--border));border-radius:8px;padding:6px 3px;text-align:center;cursor:grab;font-size:9px;background:hsl(var(--muted)/.35); }
.lib-item:hover { border-color:#1f8b96;background:#eaf5f6; }
.lib-icon { font-size:17px;line-height:1.2; }
.area-grid { display:flex;flex-wrap:wrap;gap:4px; }
.area-chip { display:inline-flex;gap:5px;align-items:center;border:1px solid;border-radius:20px;padding:3px 8px;height:auto;font-size:9px;background:transparent;cursor:pointer;box-shadow:none; }
.swatch { width:8px;height:8px;border-radius:50%; }
.area-chip.custom { border-style:dashed; }
.wh3d-center { position:relative;border-radius:12px;overflow:hidden;border:1px solid hsl(var(--border));background:#eef1f3; }
.scene-host,.canvas2d { width:100%;height:100%;display:block; }
.canvas2d { touch-action:none; }
.scene-badge { position:absolute;top:10px;left:10px;background:rgba(255,255,255,.92);border:1px solid #d8e0e4;border-radius:20px;padding:5px 12px;font-size:10px;font-weight:700;display:flex;gap:6px;align-items:center;color:#345; }
.pulse { width:7px;height:7px;border-radius:50%;background:#27a06b;box-shadow:0 0 0 3px rgba(39,160,107,.18); }
.dirty { color:#c26a10; }
.issue-strip { position:absolute;top:44px;left:10px;background:#fff5e8;border:1px solid #f0c48a;color:#8a5a12;border-radius:8px;padding:5px 10px;font-size:10px;display:flex;gap:6px;align-items:center;flex-wrap:wrap; }
.issue-strip button { border:0;background:#f5d9ad;border-radius:4px;font-size:9px;padding:1px 5px;cursor:pointer; }
.minimap-wrap { position:absolute;right:10px;bottom:10px;width:184px;background:rgba(255,255,255,.94);border:1px solid #d8e0e4;border-radius:10px;padding:5px;box-shadow:0 4px 14px rgba(0,0,0,.08); }
.minimap-title { font-size:9px;font-weight:800;color:#5b6c74;display:flex;gap:4px;align-items:center;margin-bottom:3px; }
.minimap-canvas { width:100%;height:104px;border-radius:6px; }
.right-tabs { display:flex;border-bottom:1px solid hsl(var(--border)); }
.right-tabs button { flex:1;border:0;background:transparent;padding:9px 4px;font-size:10px;font-weight:700;color:hsl(var(--muted-foreground));cursor:pointer; }
.right-tabs button.active { color:#127f8b;border-bottom:2px solid #127f8b; }
.right-tabs b { background:#e8a317;color:#fff;border-radius:8px;padding:0 5px;font-size:9px; }
.panel-close { flex: 0 0 auto; padding: 0 6px; }
.right-list { padding:8px; }
.act-row { display:flex;gap:7px;padding:7px 4px;border-bottom:1px solid hsl(var(--border)); }
.act-dot { width:8px;height:8px;border-radius:50%;background:#4a9ad4;margin-top:4px;flex-shrink:0; }
.act-dot.completed,.act-dot.success { background:#2fa36b; }
.act-dot.warning { background:#e8a317; }
.act-title { font-size:11px;font-weight:700; }
.act-meta { font-size:9px;color:hsl(var(--muted-foreground)); }
.time { opacity:.7; }
.alert-row { display:flex;gap:7px;padding:8px 4px;border-bottom:1px solid hsl(var(--border));cursor:pointer;font-size:11px; }
.empty { color:hsl(var(--muted-foreground));font-size:11px;text-align:center;padding:22px 8px; }
.empty.small { padding:8px; }
.props { padding:10px; }
.prop-head { display:flex;gap:9px;align-items:center;padding-bottom:10px;border-bottom:1px solid hsl(var(--border));margin-bottom:10px; }
.prop-icon { width:34px;height:34px;border-radius:8px;background:#e6f2f4;color:#1f7a86;display:grid;place-items:center; }
.prop-head b { display:block;font-size:13px; }
.prop-head small { color:hsl(var(--muted-foreground));font-size:10px; }
.prop-group-title { font-size:9px;font-weight:800;letter-spacing:.06em;color:hsl(var(--muted-foreground));margin:10px 0 5px; }
.prop-grid { display:grid;grid-template-columns:1fr 1fr;gap:6px; }
.prop-grid.mt { margin-top:12px; }
.prop-grid label { font-size:9px;color:hsl(var(--muted-foreground)); }
.prop-grid label.wide { grid-column:span 2; }
.prop-grid input { width:100%;border:1px solid hsl(var(--border));border-radius:6px;padding:5px 6px;font-size:11px;background:hsl(var(--background));color:hsl(var(--foreground)); }
.emp-card { display:flex;gap:9px;background:hsl(var(--muted)/.5);padding:9px;border-radius:9px;font-size:10px; }
.avatar { width:38px;height:38px;border-radius:50%;background:#1f7a86;color:#fff;display:grid;place-items:center;font-weight:800;font-size:16px;flex-shrink:0; }
.act-live { color:#1f9d68;font-weight:700;margin-top:3px; }
.right-foot { border-top:1px solid hsl(var(--border));padding:8px 10px; }
.vers-title { font-size:9px;font-weight:800;color:hsl(var(--muted-foreground));letter-spacing:.06em; }
.vers-row { display:flex;gap:7px;align-items:center;font-size:10px;padding:3px 0; }
.vers-row small { flex:1;color:hsl(var(--muted-foreground)); }
.foot-actions { display:flex;gap:5px;margin-top:8px; }
.import-btn { border:1px solid hsl(var(--border));border-radius:6px;padding:4px 8px;font-size:11px;cursor:pointer;display:flex;gap:4px;align-items:center; }
.import-btn input { display:none; }
.wh3d-bottom { display:flex;justify-content:space-between;align-items:center;gap:10px;padding:6px 12px;background:hsl(var(--card));border-top:1px solid hsl(var(--border)); }
.bottom-nav { display:flex;gap:4px;flex-wrap:wrap; }
.bottom-nav button { border:0;background:hsl(var(--muted)/.6);border-radius:16px;padding:5px 12px;height:auto;font-size:10px;font-weight:700;cursor:pointer;box-shadow:none; }
.bottom-actions { display:flex;gap:5px; }
.form-grid { display:grid;grid-template-columns:1fr 1fr;gap:8px; }
.w-full { width:100%; }
.area-auto { background:#eaf5f6;border-radius:8px;padding:8px 10px;font-size:11px;color:#1f6b75; }

/* ===================== Responsif / Mobile ===================== */
@media (max-width: 1180px) {
  .wh3d-top { flex-wrap: wrap; }
  .wh3d-title { width: 100%; }
  .wh3d-kpis { width: 100%; }
  .wh3d-body { grid-template-columns: 210px 1fr 250px; }
}

@media (max-width: 900px) {
  .mobile-only { display: inline-flex; }
  .wh3d-top { padding: 8px 10px; gap: 8px; }
  .wh3d-name { font-size: 14px; }
  .wh3d-sub { flex-wrap: wrap; font-size: 10px; }
  .wh3d-kpis { flex-wrap: nowrap; overflow-x: auto; padding-bottom: 2px; }
  .kpi { flex: 0 0 auto; min-width: 96px; padding: 5px 8px; }
  .kpi-value { font-size: 14px; }
  .wh3d-toolbar { flex-wrap: nowrap; overflow-x: auto; padding: 6px 8px; }
  .tool-group, .sep { flex-shrink: 0; }
  .tool-spacer { display: none; }
  .wh3d-body { display: block; position: relative; padding: 6px; }
  .wh3d-center { height: 100%; min-height: 52vh; }
  .wh3d-left, .wh3d-right {
    position: absolute; top: 6px; bottom: 6px; width: min(86vw, 320px);
    z-index: 60; border-radius: 12px;
    box-shadow: 0 10px 30px rgba(15, 35, 45, .25);
    transition: transform .22s ease;
  }
  .wh3d-left { left: 6px; transform: translateX(calc(-100% - 12px)); }
  .wh3d-left.open { transform: translateX(0); }
  .wh3d-right { right: 6px; transform: translateX(calc(100% + 12px)); }
  .wh3d-right.open { transform: translateX(0); }
  .minimap-wrap { width: 132px; }
  .minimap-canvas { height: 78px; }
  .scene-badge { font-size: 9px; padding: 4px 9px; }
  .issue-strip { top: 38px; right: 8px; left: 8px; }
  .wh3d-bottom { flex-wrap: wrap; padding: 6px 8px; }
  .bottom-nav, .bottom-actions { width: 100%; flex-wrap: nowrap; overflow-x: auto; }
  .bottom-nav button, .bottom-actions button { flex-shrink: 0; }
}

@media (max-width: 560px) {
  .kpi-icon { display: none; }
  .kpi { min-width: 84px; }
  .wh3d-logo { width: 32px; height: 32px; }
  .wh3d-sub .dot, .wh3d-sub span:nth-last-child(-n + 2) { display: none; }
  .minimap-wrap { width: 116px; }
  .minimap-canvas { height: 68px; }
}
</style>
