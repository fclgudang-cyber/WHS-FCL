/**
 * 2D Layout Editor — top-view presisi, sinkron dengan scene 3D.
 * Grid / snap / zoom / pan / select / move / resize / multi-select / ukuran.
 */
import type { Wh3dArea, Wh3dEmployee, Wh3dEquipment, Wh3dObject, Wh3dWarehouse } from '#/api/warehouse/warehouse3d';
import { num } from '#/api/warehouse/warehouse3d';

export interface MapItem {
  uid: string;
  name: string;
  type: string;
  x: number;
  y: number;
  l: number;
  w: number;
  rotation: number;
  color: string;
  kind: 'object' | 'area' | 'equipment' | 'worker';
}

export interface Editor2DCallbacks {
  onSelect?: (uids: string[]) => void;
  onMove?: (uid: string, x: number, y: number) => void;
  onResize?: (uid: string, l: number, w: number) => void;
  onDraw?: (x: number, y: number, l: number, w: number) => void;
  onTransformEnd?: (uid: string, patch: Record<string, any>) => void;
}

type DragMode = 'none' | 'pan' | 'move' | 'resize' | 'draw';

export class Editor2D {
  private canvas: HTMLCanvasElement;
  private ctx: CanvasRenderingContext2D;
  private cb: Editor2DCallbacks;
  private items: MapItem[] = [];
  private warehouse: Wh3dWarehouse | null = null;
  private selected = new Set<string>();
  private scale = 6;
  private origin = { x: 0, y: 0 };
  private drag: DragMode = 'none';
  private start = { x: 0, y: 0 };
  private startItems = new Map<string, { x: number; y: number; l: number; w: number }>();
  private draft: { x: number; y: number; l: number; w: number } | null = null;
  private snap = true;
  private showGrid = true;
  private frame = 0;

  constructor(canvas: HTMLCanvasElement, cb: Editor2DCallbacks = {}) {
    this.canvas = canvas;
    this.ctx = canvas.getContext('2d')!;
    this.cb = cb;
    canvas.addEventListener('pointerdown', this.onDown);
    canvas.addEventListener('pointermove', this.onMove);
    canvas.addEventListener('pointerup', this.onUp);
    canvas.addEventListener('wheel', this.onWheel, { passive: false });
    this.loop();
  }

  setSnap(v: boolean) { this.snap = v; }
  setGrid(v: boolean) { this.showGrid = v; }

  setData(wh: Wh3dWarehouse | null, objects: Wh3dObject[], areas: Wh3dArea[], equipment: Wh3dEquipment[], workers: Wh3dEmployee[]) {
    this.warehouse = wh;
    const items: MapItem[] = [];
    for (const a of areas) {
      items.push({
        uid: a.areaUid, name: `${a.code} · ${a.name}`, type: a.areaType,
        x: num(a.positionX), y: num(a.positionY), l: num(a.length), w: num(a.width),
        rotation: num(a.rotation), color: a.color || '#3B82F6', kind: 'area',
      });
    }
    for (const o of objects) {
      items.push({
        uid: o.objectUid, name: o.name || o.objectType, type: o.objectType,
        x: num(o.positionX), y: num(o.positionY), l: num(o.length), w: num(o.width),
        rotation: num(o.rotation), color: o.color || this.colorFor(o.objectType), kind: 'object',
      });
    }
    for (const e of equipment) {
      items.push({
        uid: e.equipmentUid, name: e.name, type: e.equipmentType,
        x: num(e.positionX), y: num(e.positionY), l: 2.8, w: 1.5,
        rotation: num(e.rotation), color: e.color || '#E8A317', kind: 'equipment',
      });
    }
    for (const p of workers) {
      items.push({
        uid: `emp-${p.employeeId}`, name: p.fullName, type: 'EMPLOYEE',
        x: num(p.positionX), y: num(p.positionY), l: 1.2, w: 1.2,
        rotation: num(p.rotation), color: p.activity === 'IDLE' ? '#7A8C94' : '#2E8B9A', kind: 'worker',
      });
    }
    this.items = items;
    if (wh && this.scale === 6) this.fit();
  }

  private colorFor(t: string) {
    const x = t.toUpperCase();
    if (x.includes('RACK') || x.includes('SHELF')) return '#E08A2E';
    if (x.includes('PALLET')) return '#A9763F';
    if (x.includes('CARTON')) return '#C98A4B';
    if (x.includes('CONVEYOR')) return '#6C7A86';
    if (x.includes('PACKING')) return '#3F8F7A';
    if (x.includes('DOCK')) return '#7A8791';
    if (x.includes('DOOR')) return '#3F7D8C';
    if (x.includes('WINDOW')) return '#9CC9D4';
    if (x.includes('PILLAR')) return '#8C9AA3';
    return '#6C8F94';
  }

  setSelected(uids: string[]) {
    this.selected = new Set(uids);
  }

  fit() {
    const L = num(this.warehouse?.lengthM) || 100;
    const W = num(this.warehouse?.widthM) || 50;
    const rect = this.canvas.getBoundingClientRect();
    this.scale = Math.min((rect.width - 60) / L, (rect.height - 60) / W) || 6;
    this.origin = { x: rect.width / 2, y: rect.height / 2 };
  }

  zoom(f: number) { this.scale = Math.max(1, Math.min(60, this.scale * f)); }
  resetView() { this.fit(); }

  private world(px: number, py: number) {
    return { x: (px - this.origin.x) / this.scale, y: (py - this.origin.y) / this.scale };
  }
  private screen(x: number, y: number) {
    return { x: this.origin.x + x * this.scale, y: this.origin.y + y * this.scale };
  }
  private snapV(v: number) { return this.snap ? Math.round(v) : Math.round(v * 10) / 10; }
  private local(e: PointerEvent | WheelEvent) {
    const r = this.canvas.getBoundingClientRect();
    return { x: e.clientX - r.left, y: e.clientY - r.top };
  }

  private onDown = (e: PointerEvent) => {
    const p = this.local(e);
    if (e.button === 1 || e.button === 2 || e.altKey) {
      this.drag = 'pan';
      this.start = p;
      return;
    }
    const w = this.world(p.x, p.y);
    if (e.shiftKey) {
      this.drag = 'draw';
      this.draft = { x: this.snapV(w.x), y: this.snapV(w.y), l: 0, w: 0 };
      return;
    }
    const hit = [...this.items].reverse().find((it) =>
      Math.abs(w.x - it.x) <= Math.max(it.l / 2, 0.8) && Math.abs(w.y - it.y) <= Math.max(it.w / 2, 0.8));
    if (!hit) {
      this.selected.clear();
      this.cb.onSelect?.([]);
      return;
    }
    if (e.ctrlKey || e.metaKey) {
      if (this.selected.has(hit.uid)) this.selected.delete(hit.uid);
      else this.selected.add(hit.uid);
    } else if (!this.selected.has(hit.uid)) {
      this.selected.clear();
      this.selected.add(hit.uid);
    }
    this.cb.onSelect?.([...this.selected]);

    // resize handle pada item terpilih tunggal
    const s = this.screen(hit.x + hit.l / 2, hit.y + hit.w / 2);
    if (this.selected.size === 1 && Math.hypot(p.x - s.x, p.y - s.y) < 10) {
      this.drag = 'resize';
    } else {
      this.drag = 'move';
    }
    this.start = p;
    this.startItems.clear();
    for (const uid of this.selected) {
      const it = this.items.find((i) => i.uid === uid);
      if (it) this.startItems.set(uid, { x: it.x, y: it.y, l: it.l, w: it.w });
    }
  };

  private onMove = (e: PointerEvent) => {
    const p = this.local(e);
    if (this.drag === 'none') return;
    if (this.drag === 'pan') {
      this.origin.x += p.x - this.start.x;
      this.origin.y += p.y - this.start.y;
      this.start = p;
      return;
    }
    const w = this.world(p.x, p.y);
    if (this.drag === 'draw' && this.draft) {
      this.draft.l = this.snapV(w.x - this.draft.x);
      this.draft.w = this.snapV(w.y - this.draft.y);
      return;
    }
    const dx = w.x - this.world(this.start.x, this.start.y).x;
    const dy = w.y - this.world(this.start.x, this.start.y).y;
    for (const [uid, orig] of this.startItems) {
      const it = this.items.find((i) => i.uid === uid);
      if (!it) continue;
      if (this.drag === 'move') {
        it.x = this.snapV(orig.x + dx);
        it.y = this.snapV(orig.y + dy);
        this.cb.onMove?.(uid, it.x, it.y);
      } else if (this.drag === 'resize') {
        it.l = Math.max(0.5, this.snapV(orig.l + dx));
        it.w = Math.max(0.5, this.snapV(orig.w + dy));
        this.cb.onResize?.(uid, it.l, it.w);
      }
    }
  };

  private onUp = () => {
    if (this.drag === 'draw' && this.draft) {
      const d = this.draft;
      if (Math.abs(d.l) > 0.5 && Math.abs(d.w) > 0.5) {
        this.cb.onDraw?.(d.x + d.l / 2, d.y + d.w / 2, Math.abs(d.l), Math.abs(d.w));
      }
    } else if ((this.drag === 'move' || this.drag === 'resize') && this.startItems.size) {
      const uid = [...this.startItems.keys()][0];
      const item = this.items.find((i) => i.uid === uid);
      if (uid && item) {
        this.cb.onTransformEnd?.(uid, this.drag === 'move'
          ? { positionX: item.x, positionY: item.y }
          : { length: item.l, width: item.w });
      }
    }
    this.drag = 'none';
    this.draft = null;
    this.startItems.clear();
  };

  private onWheel = (e: WheelEvent) => {
    e.preventDefault();
    this.zoom(e.deltaY > 0 ? 0.9 : 1.1);
  };

  // ---------------------------------------------------------------- draw
  private loop = () => {
    this.frame = requestAnimationFrame(this.loop);
    this.draw();
  };

  private draw() {
    const r = this.canvas.getBoundingClientRect();
    const dpr = Math.min(window.devicePixelRatio, 2);
    if (this.canvas.width !== r.width * dpr || this.canvas.height !== r.height * dpr) {
      this.canvas.width = r.width * dpr;
      this.canvas.height = r.height * dpr;
    }
    const ctx = this.ctx;
    ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    ctx.clearRect(0, 0, r.width, r.height);
    ctx.fillStyle = '#f4f6f7';
    ctx.fillRect(0, 0, r.width, r.height);

    if (this.showGrid) {
      ctx.strokeStyle = '#dfe4e7';
      ctx.lineWidth = 1;
      const step = this.scale;
      const ox = ((this.origin.x % step) + step) % step;
      const oy = ((this.origin.y % step) + step) % step;
      for (let x = ox; x < r.width; x += step) {
        ctx.beginPath(); ctx.moveTo(x, 0); ctx.lineTo(x, r.height); ctx.stroke();
      }
      for (let y = oy; y < r.height; y += step) {
        ctx.beginPath(); ctx.moveTo(0, y); ctx.lineTo(r.width, y); ctx.stroke();
      }
    }
    // sumbu
    ctx.strokeStyle = '#9fb3bb';
    ctx.lineWidth = 1.5;
    ctx.beginPath(); ctx.moveTo(this.origin.x, 0); ctx.lineTo(this.origin.x, r.height); ctx.stroke();
    ctx.beginPath(); ctx.moveTo(0, this.origin.y); ctx.lineTo(r.width, this.origin.y); ctx.stroke();

    // bangunan
    if (this.warehouse) {
      const L = num(this.warehouse.lengthM);
      const W = num(this.warehouse.widthM);
      const a = this.screen(-L / 2, -W / 2);
      const b = this.screen(L / 2, W / 2);
      ctx.fillStyle = 'rgba(226,232,235,0.75)';
      ctx.fillRect(a.x, a.y, b.x - a.x, b.y - a.y);
      ctx.strokeStyle = '#5b6c74';
      ctx.lineWidth = 2;
      ctx.strokeRect(a.x, a.y, b.x - a.x, b.y - a.y);
      ctx.fillStyle = '#5b6c74';
      ctx.font = 'bold 12px Inter, Arial, sans-serif';
      ctx.fillText(`${this.warehouse.code || ''} ${L}m × ${W}m`, a.x + 6, a.y - 7);
    }

    for (const it of this.items) {
      const c = this.screen(it.x, it.y);
      const w = it.l * this.scale;
      const h = it.w * this.scale;
      ctx.save();
      ctx.translate(c.x, c.y);
      ctx.rotate((it.rotation * Math.PI) / 180);
      ctx.fillStyle = it.color + (it.kind === 'area' ? '2B' : 'CC');
      ctx.fillRect(-w / 2, -h / 2, w, h);
      ctx.strokeStyle = this.selected.has(it.uid) ? '#0F9B9B' : it.color;
      ctx.lineWidth = this.selected.has(it.uid) ? 3 : 1.5;
      ctx.strokeRect(-w / 2, -h / 2, w, h);
      ctx.restore();

      if (this.selected.has(it.uid)) {
        ctx.fillStyle = '#0F9B9B';
        ctx.fillRect(c.x + w / 2 - 4, c.y + h / 2 - 4, 8, 8);
        ctx.fillStyle = '#0F5F66';
        ctx.font = '11px Inter, Arial, sans-serif';
        ctx.fillText(`${it.l} × ${it.w} m`, c.x - w / 2, c.y - h / 2 - 6);
      }
      if (it.kind !== 'area' && w > 26 && h > 16) {
        ctx.fillStyle = '#37474F';
        ctx.font = '10px Inter, Arial, sans-serif';
        ctx.textAlign = 'center';
        ctx.fillText(it.name.slice(0, 14), c.x, c.y + 3);
        ctx.textAlign = 'left';
      }
    }

    if (this.draft) {
      const a = this.screen(this.draft.x, this.draft.y);
      ctx.fillStyle = 'rgba(15,155,155,0.18)';
      ctx.strokeStyle = '#0F9B9B';
      ctx.lineWidth = 2;
      ctx.fillRect(a.x, a.y, this.draft.l * this.scale, this.draft.w * this.scale);
      ctx.strokeRect(a.x, a.y, this.draft.l * this.scale, this.draft.w * this.scale);
    }
  }

  dispose() {
    cancelAnimationFrame(this.frame);
    this.canvas.removeEventListener('pointerdown', this.onDown);
    this.canvas.removeEventListener('pointermove', this.onMove);
    this.canvas.removeEventListener('pointerup', this.onUp);
    this.canvas.removeEventListener('wheel', this.onWheel);
  }
}
