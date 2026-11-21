/**
 * Warehouse 3D — Three.js scene engine.
 * Gaya: cartoon semi-realistic, palet hangat/industri (tanpa neon/ungu).
 * Semua mesh dibuat dari data layout (bukan hardcoded layout).
 */
import * as THREE from 'three';
import { OrbitControls } from 'three/addons/controls/OrbitControls.js';

import type {
  Wh3dArea,
  Wh3dEmployee,
  Wh3dEquipment,
  Wh3dObject,
  Wh3dWarehouse,
} from '#/api/warehouse/warehouse3d';
import { num } from '#/api/warehouse/warehouse3d';

export type EditorMode = '3d' | '2d';
export type TransformMode = 'select' | 'move' | 'rotate' | 'resize';

export interface SceneCallbacks {
  onSelect?: (uid: string | null) => void;
  onMove?: (uid: string, x: number, y: number) => void;
  onRotate?: (uid: string, rotation: number) => void;
  onResize?: (uid: string, length: number, width: number) => void;
  onTransformEnd?: (uid: string, patch: Record<string, any>) => void;
  onHover?: (uid: string | null) => void;
}

interface Entry {
  uid: string;
  root: THREE.Group;
  baseColor: THREE.Color | null;
  kind: string;
  rotation: number;
  length: number;
  width: number;
}

const PALETTE: Record<string, string> = {
  floor: '#D9D3C7',
  floorLine: '#B9B1A2',
  wall: '#E8E4DA',
  wallTrim: '#94A3A8',
  door: '#3F7D8C',
  window: '#9CC9D4',
  pillar: '#8C9AA3',
  rack: '#E08A2E',
  rackBeam: '#2F6F8F',
  pallet: '#A9763F',
  carton: '#C98A4B',
  bin: '#5B8C5A',
  forklift: '#E8A317',
  handpallet: '#4E7E9B',
  conveyor: '#6C7A86',
  packing: '#3F8F7A',
  dock: '#7A8791',
  worker: '#2F7D8C',
  area: '#3B82F6',
};

function mat(color: string | undefined, extra: Record<string, any> = {}): THREE.MeshStandardMaterial {
  return new THREE.MeshStandardMaterial({
    color: color || '#6C8F94',
    roughness: 0.78,
    metalness: 0.06,
    flatShading: true,
    ...extra,
  });
}

function box(
  w: number,
  h: number,
  d: number,
  color: string | undefined,
  x = 0,
  y = 0,
  z = 0,
  extra: Record<string, any> = {},
): THREE.Mesh {
  const m = new THREE.Mesh(new THREE.BoxGeometry(Math.max(w, 0.01), Math.max(h, 0.01), Math.max(d, 0.01)), mat(color, extra));
  m.position.set(x, y, z);
  m.castShadow = true;
  m.receiveShadow = true;
  return m;
}

function cylinder(r: number, h: number, color: string | undefined, x = 0, y = 0, z = 0): THREE.Mesh {
  const m = new THREE.Mesh(new THREE.CylinderGeometry(r, r, h, 12), mat(color));
  m.position.set(x, y, z);
  m.castShadow = true;
  m.receiveShadow = true;
  return m;
}

function labelSprite(text: string | undefined, color = '#ffffff', bg = 'rgba(20,38,48,0.82)'): THREE.Sprite {
  const canvas = document.createElement('canvas');
  canvas.width = 512;
  canvas.height = 96;
  const ctx = canvas.getContext('2d')!;
  ctx.fillStyle = bg;
  ctx.fillRect(4, 8, canvas.width - 8, 72);
  ctx.fillStyle = color;
  ctx.font = 'bold 34px Inter, Arial, sans-serif';
  ctx.textAlign = 'center';
  ctx.textBaseline = 'middle';
  ctx.fillText(String(text ?? '').slice(0, 26), canvas.width / 2, 45);
  const texture = new THREE.CanvasTexture(canvas);
  texture.colorSpace = THREE.SRGBColorSpace;
  const sprite = new THREE.Sprite(new THREE.SpriteMaterial({ map: texture, transparent: true, depthTest: false }));
  sprite.scale.set(6, 1.1, 1);
  sprite.renderOrder = 999;
  return sprite;
}

export class Warehouse3DScene {
  private container: HTMLElement;
  private cb: SceneCallbacks;
  private renderer: THREE.WebGLRenderer;
  private scene = new THREE.Scene();
  private camera: THREE.OrthographicCamera;
  private controls: OrbitControls;
  private raycaster = new THREE.Raycaster();
  private pointer = new THREE.Vector2();
  private ground = new THREE.Plane(new THREE.Vector3(0, 1, 0), 0);
  private entries = new Map<string, Entry>();
  private dynamic: Array<{ obj: THREE.Object3D; data: any; kind: string }> = [];
  private selected: string | null = null;
  private highlight: THREE.BoxHelper | null = null;
  private building = new THREE.Group();
  private objectsRoot = new THREE.Group();
  private clock = new THREE.Clock();
  private frame = 0;
  private dragging: string | null = null;
  private dragMode: TransformMode = 'select';
  private dragOffset = new THREE.Vector3();
  private startClient = { x: 0, y: 0 };
  private startRotation = 0;
  private startLength = 1;
  private startWidth = 1;
  private buildingLength = 100;
  private buildingWidth = 50;
  private resizeObserver: ResizeObserver | null = null;
  private mode: EditorMode = '3d';
  private transform: TransformMode = 'select';
  private frustum = 60;
  private disposed = false;

  constructor(container: HTMLElement, cb: SceneCallbacks = {}) {
    this.container = container;
    this.cb = cb;
    this.renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    this.renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    this.renderer.setSize(container.clientWidth || 800, container.clientHeight || 600);
    this.renderer.shadowMap.enabled = true;
    this.renderer.shadowMap.type = THREE.PCFSoftShadowMap;
    this.renderer.outputColorSpace = THREE.SRGBColorSpace;
    container.appendChild(this.renderer.domElement);

    this.scene.background = new THREE.Color('#eef1f3');
    this.scene.fog = new THREE.Fog('#eef1f3', 180, 420);

    const aspect = (container.clientWidth || 800) / (container.clientHeight || 600);
    this.camera = new THREE.OrthographicCamera(
      -this.frustum * aspect, this.frustum * aspect, this.frustum, -this.frustum, 0.1, 2000,
    );
    this.camera.position.set(90, 90, 90);
    this.camera.lookAt(0, 0, 0);

    this.controls = new OrbitControls(this.camera, this.renderer.domElement);
    this.controls.enableDamping = true;
    this.controls.dampingFactor = 0.08;
    this.controls.screenSpacePanning = true;
    this.controls.maxPolarAngle = Math.PI / 2.05;
    this.controls.minZoom = 0.35;
    this.controls.maxZoom = 5;
    this.controls.mouseButtons = {
      LEFT: THREE.MOUSE.ROTATE,
      MIDDLE: THREE.MOUSE.PAN,
      RIGHT: THREE.MOUSE.PAN,
    };

    const hemi = new THREE.HemisphereLight('#ffffff', '#b9b0a0', 1.25);
    this.scene.add(hemi);
    const sun = new THREE.DirectionalLight('#fff4dd', 2.1);
    sun.position.set(60, 110, 40);
    sun.castShadow = true;
    sun.shadow.mapSize.set(2048, 2048);
    sun.shadow.camera.left = -140;
    sun.shadow.camera.right = 140;
    sun.shadow.camera.top = 140;
    sun.shadow.camera.bottom = -140;
    sun.shadow.camera.far = 400;
    sun.shadow.bias = -0.0008;
    this.scene.add(sun);
    const fill = new THREE.DirectionalLight('#cfe3ee', 0.5);
    fill.position.set(-60, 50, -60);
    this.scene.add(fill);

    this.scene.add(this.building);
    this.scene.add(this.objectsRoot);

    const el = this.renderer.domElement;
    el.addEventListener('pointerdown', this.onPointerDown);
    el.addEventListener('pointermove', this.onPointerMove);
    el.addEventListener('pointerup', this.onPointerUp);
    el.addEventListener('dblclick', this.onDoubleClick);
    window.addEventListener('resize', this.onResize);
    if (typeof ResizeObserver !== 'undefined') {
      this.resizeObserver = new ResizeObserver(() => this.onResize());
      this.resizeObserver.observe(container);
    }
    this.animate();
  }

  private onResize = () => {
    if (this.disposed) return;
    const w = this.container.clientWidth || 800;
    const h = this.container.clientHeight || 600;
    const aspect = w / h;
    const span = Math.max(this.buildingLength, this.buildingWidth);
    this.frustum = Math.max(60, span / (2 * Math.max(aspect, 0.5)));
    this.camera.left = -this.frustum * aspect;
    this.camera.right = this.frustum * aspect;
    this.camera.top = this.frustum;
    this.camera.bottom = -this.frustum;
    this.camera.updateProjectionMatrix();
    this.renderer.setSize(w, h);
  };

  private updatePointer(e: PointerEvent | MouseEvent) {
    const rect = this.renderer.domElement.getBoundingClientRect();
    this.pointer.x = ((e.clientX - rect.left) / rect.width) * 2 - 1;
    this.pointer.y = -((e.clientY - rect.top) / rect.height) * 2 + 1;
    this.raycaster.setFromCamera(this.pointer, this.camera);
  }

  private pick(): THREE.Object3D | null {
    const roots = Array.from(this.entries.values()).map((e) => e.root);
    const hits = this.raycaster.intersectObjects(roots, true);
    if (!hits.length || !hits[0]) return null;
    let o: THREE.Object3D | null = hits[0].object;
    while (o && !o.userData.uid) o = o.parent;
    return o;
  }

  private onPointerDown = (e: PointerEvent) => {
    if (e.button !== 0 || this.mode === '2d') return;
    this.updatePointer(e);
    const hit = this.pick();
    const uid = hit?.userData.uid as string | undefined;
    if (!uid) {
      this.select(null);
      return;
    }
    const entry = this.entries.get(uid);
    this.select(uid);
    if (!entry || entry.root.userData.kind === 'EMPLOYEE') return;

    if (this.transform === 'move' || this.transform === 'rotate' || this.transform === 'resize') {
      this.dragging = uid;
      this.dragMode = this.transform;
      this.startClient = { x: e.clientX, y: e.clientY };
      this.startRotation = entry.rotation;
      this.startLength = entry.length;
      this.startWidth = entry.width;
      this.controls.enabled = false;
      if (this.transform === 'move') {
        const p = new THREE.Vector3();
        this.raycaster.ray.intersectPlane(this.ground, p);
        this.dragOffset.set(p.x - entry.root.position.x, 0, p.z - entry.root.position.z);
      }
      this.renderer.domElement.setPointerCapture?.(e.pointerId);
    }
  };

  private onPointerMove = (e: PointerEvent) => {
    if (this.mode === '2d') return;
    this.updatePointer(e);
    if (!this.dragging) {
      const hit = this.pick();
      const uid = (hit?.userData.uid as string) ?? null;
      this.cb.onHover?.(uid);
      this.renderer.domElement.style.cursor = uid ? 'grab' : 'default';
      return;
    }
    const entry = this.entries.get(this.dragging);
    if (!entry) return;

    if (this.dragMode === 'move') {
      const p = new THREE.Vector3();
      if (this.raycaster.ray.intersectPlane(this.ground, p)) {
        const x = Math.round((p.x - this.dragOffset.x) * 10) / 10;
        const y = Math.round((p.z - this.dragOffset.y) * 10) / 10;
        entry.root.position.set(x, entry.root.position.y, y);
        this.cb.onMove?.(this.dragging, x, y);
      }
    } else if (this.dragMode === 'rotate') {
      const deg = Math.round(
        (this.startRotation + (e.clientX - this.startClient.x) * 0.4 + 360) % 360,
      );
      entry.rotation = deg;
      entry.root.rotation.y = (deg * Math.PI) / 180;
      this.cb.onRotate?.(this.dragging, deg);
    } else if (this.dragMode === 'resize') {
      const length = Math.max(0.5, Math.round(
        (this.startLength * (1 + (e.clientX - this.startClient.x) * 0.03)) * 10,
      ) / 10);
      const width = Math.max(0.5, Math.round(
        (this.startWidth * (1 + (e.clientY - this.startClient.y) * 0.03)) * 10,
      ) / 10);
      entry.length = length;
      entry.width = width;
      entry.root.scale.set(length / this.startLength, 1, width / this.startWidth);
      this.cb.onResize?.(this.dragging, length, width);
    }
    this.refreshHighlight();
  };

  private onPointerUp = (e: PointerEvent) => {
    if (!this.dragging) return;
    const uid = this.dragging;
    const entry = this.entries.get(uid);
    if (entry) {
      const patch: Record<string, any> = {};
      if (this.dragMode === 'move') {
        patch.positionX = Math.round(entry.root.position.x * 10) / 10;
        patch.positionY = Math.round(entry.root.position.z * 10) / 10;
      } else if (this.dragMode === 'rotate') {
        patch.rotation = entry.rotation;
      } else if (this.dragMode === 'resize') {
        patch.length = entry.length;
        patch.width = entry.width;
      }
      this.cb.onTransformEnd?.(uid, patch);
    }
    this.dragging = null;
    this.dragMode = 'select';
    this.controls.enabled = true;
    this.renderer.domElement.releasePointerCapture?.(e.pointerId);
    this.renderer.domElement.style.cursor = 'default';
  };

  private onDoubleClick = (e: MouseEvent) => {
    this.updatePointer(e);
    const hit = this.pick();
    if (hit?.userData.uid) this.focus(hit.userData.uid as string);
  };

  // -------------------------------------------------------------- public API

  setMode(mode: EditorMode) {
    this.mode = mode;
    this.renderer.domElement.style.display = mode === '3d' ? 'block' : 'none';
    if (mode === '3d') this.onResize();
  }

  setTransform(mode: TransformMode) {
    this.transform = mode;
  }

  setWarehouse(wh: Wh3dWarehouse) {
    this.buildingLength = num(wh.lengthM) || 100;
    this.buildingWidth = num(wh.widthM) || 50;
    this.buildBuilding(wh);
    this.onResize();
  }

  select(uid: string | null) {
    this.selected = uid;
    this.refreshHighlight();
    this.cb.onSelect?.(uid);
  }

  private refreshHighlight() {
    if (this.highlight) {
      this.scene.remove(this.highlight);
      this.highlight.geometry.dispose();
      (this.highlight.material as any).dispose?.();
      this.highlight = null;
    }
    if (!this.selected) return;
    const entry = this.entries.get(this.selected);
    if (!entry) return;
    this.highlight = new THREE.BoxHelper(entry.root, 0x16a3a3);
    (this.highlight.material as any).depthTest = false;
    this.highlight.renderOrder = 998;
    this.scene.add(this.highlight);
  }

  focus(uid: string) {
    const entry = this.entries.get(uid);
    if (!entry) return;
    const p = new THREE.Vector3();
    entry.root.getWorldPosition(p);
    this.controls.target.copy(p);
    this.camera.position.set(p.x + 45, p.y + 55, p.z + 45);
    this.camera.zoom = 1.5;
    this.camera.updateProjectionMatrix();
    this.controls.update();
    this.select(uid);
  }

  focusPoint(x: number, y: number) {
    this.controls.target.set(x, 0, y);
    this.camera.position.set(x + 45, 55, y + 45);
    this.camera.zoom = 1.5;
    this.camera.updateProjectionMatrix();
    this.controls.update();
  }

  resetCamera() {
    this.controls.target.set(0, 0, 0);
    this.camera.position.set(90, 90, 90);
    this.camera.zoom = 1;
    this.camera.updateProjectionMatrix();
    this.controls.update();
  }

  setView(view: 'iso' | 'top') {
    if (view === 'top') {
      this.camera.position.set(0, 160, 0.01);
      this.controls.target.set(0, 0, 0);
    } else {
      this.resetCamera();
    }
    this.controls.update();
  }

  zoom(factor: number) {
    this.camera.zoom = Math.min(5, Math.max(0.35, this.camera.zoom * factor));
    this.camera.updateProjectionMatrix();
  }

  // -------------------------------------------------------------- build scene

  private clearGroup(g: THREE.Group) {
    while (g.children.length) {
      const child = g.children.pop()!;
      child.traverse((o: any) => {
        o.geometry?.dispose?.();
        if (Array.isArray(o.material)) o.material.forEach((m: any) => m.dispose?.());
        else o.material?.dispose?.();
      });
    }
  }

  private buildBuilding(wh: Wh3dWarehouse) {
    // Komponen bangunan (lantai, dinding, pintu, jendela, tiang, dock) kini
    // adalah objek layout yang bisa dipilih & dipindah — lihat setLayout().
    // Di sini hanya grid referensi yang digambar sebagai latar.
    this.clearGroup(this.building);
    const L = num(wh.lengthM) || 100;
    const W = num(wh.widthM) || 50;
    const grid = new THREE.GridHelper(Math.max(L, W), Math.round(Math.max(L, W) / 5), PALETTE.floorLine, '#c9c2b5');
    grid.position.y = 0.01;
    (grid.material as any).opacity = 0.4;
    (grid.material as any).transparent = true;
    this.building.add(grid);
  }

  setLayout(
    objects: Wh3dObject[],
    areas: Wh3dArea[],
    employees: Wh3dEmployee[],
    equipment: Wh3dEquipment[],
  ) {
    this.clearGroup(this.objectsRoot);
    this.entries.clear();
    this.dynamic = [];

    for (const a of areas) this.objectsRoot.add(this.buildArea(a));

    for (const o of objects) {
      if (o.visible === false) continue;
      const root = this.buildObject(o);
      if (!root) continue;
      root.userData.uid = o.objectUid;
      root.userData.kind = o.objectType;
      root.position.set(num(o.positionX), num(o.positionZ), num(o.positionY));
      root.rotation.y = (num(o.rotation) * Math.PI) / 180;
      this.objectsRoot.add(root);
      this.entries.set(o.objectUid, {
        uid: o.objectUid, root, baseColor: null, kind: o.objectType,
        rotation: num(o.rotation), length: num(o.length), width: num(o.width),
      });
      this.dynamic.push({ obj: root, data: o, kind: o.objectType });
    }

    for (const e of equipment) {
      const root = this.buildEquipment(e);
      root.userData.uid = e.equipmentUid;
      root.userData.kind = e.equipmentType;
      root.position.set(num(e.positionX), 0, num(e.positionY));
      root.rotation.y = (num(e.rotation) * Math.PI) / 180;
      this.objectsRoot.add(root);
      this.entries.set(e.equipmentUid, {
        uid: e.equipmentUid, root, baseColor: null, kind: e.equipmentType,
        rotation: num(e.rotation), length: 2.8, width: 1.5,
      });
      this.dynamic.push({ obj: root, data: e, kind: 'EQUIPMENT' });
    }

    for (const p of employees) {
      const root = this.buildWorker(p);
      root.userData.uid = `emp-${p.employeeId}`;
      root.userData.kind = 'EMPLOYEE';
      root.position.set(num(p.positionX), 0, num(p.positionY));
      root.rotation.y = (num(p.rotation) * Math.PI) / 180;
      this.objectsRoot.add(root);
      this.entries.set(`emp-${p.employeeId}`, {
        uid: `emp-${p.employeeId}`, root, baseColor: null, kind: 'EMPLOYEE',
        rotation: num(p.rotation), length: 1, width: 1,
      });
      this.dynamic.push({ obj: root, data: p, kind: 'WORKER' });
    }
    this.refreshHighlight();
  }

  private buildArea(a: Wh3dArea): THREE.Group {
    const g = new THREE.Group();
    const l = num(a.length) || 1;
    const w = num(a.width) || 1;
    const color = a.color || PALETTE.area;
    const plane = new THREE.Mesh(
      new THREE.PlaneGeometry(l, w),
      new THREE.MeshStandardMaterial({ color, transparent: true, opacity: 0.16, side: THREE.DoubleSide, depthWrite: false }),
    );
    plane.rotation.x = -Math.PI / 2;
    plane.position.y = 0.05;
    plane.receiveShadow = true;
    g.add(plane);
    const edges = new THREE.LineSegments(
      new THREE.EdgesGeometry(new THREE.BoxGeometry(l, 0.12, w)),
      new THREE.LineBasicMaterial({ color, transparent: true, opacity: 0.85 }),
    );
    edges.position.y = 0.08;
    g.add(edges);
    g.position.set(num(a.positionX), 0, num(a.positionY));
    g.rotation.y = (num(a.rotation) * Math.PI) / 180;
    g.userData.uid = a.areaUid;
    const tag = labelSprite(`${a.code} · ${a.name}`, '#ffffff', 'rgba(30,80,120,0.85)');
    tag.position.set(0, 1.6, 0);
    tag.scale.set(7, 1.3, 1);
    g.add(tag);
    return g;
  }

  private buildObject(o: Wh3dObject): THREE.Group | null {
    const t = o.objectType.toUpperCase();
    const L = num(o.length) || 1;
    const W = num(o.width) || 1;
    const H = num(o.height) || 1;
    const color = o.color || '';
    const g = new THREE.Group();

    if (t.includes('AREA')) return this.buildArea(o as any);

    if (t.includes('WALL')) {
      g.add(box(L, H, W, color || PALETTE.wall, 0, H / 2, 0, { transparent: true, opacity: 0.55 }));
      g.add(box(L, 0.25, W + 0.08, PALETTE.wallTrim, 0, 0.15, 0));
      return g;
    }
    if (t.includes('DOOR')) {
      g.add(box(L, H, 0.35, color || PALETTE.door, 0, H / 2, 0, { transparent: true, opacity: 0.8 }));
      g.add(box(0.25, H, 0.45, PALETTE.wallTrim, L / 2 - 0.2, H / 2, 0));
      return g;
    }
    if (t.includes('WINDOW')) {
      g.add(box(L, H, 0.2, color || PALETTE.window, 0, H / 2, 0, { transparent: true, opacity: 0.65 }));
      g.add(box(L + 0.2, 0.18, 0.3, PALETTE.wallTrim, 0, H / 2, 0));
      return g;
    }
    if (t.includes('PILLAR')) {
      g.add(box(L, H, W, color || PALETTE.pillar, 0, H / 2, 0));
      g.add(box(L + 0.6, 0.4, W + 0.6, PALETTE.wallTrim, 0, 0.2, 0));
      return g;
    }
    if (t.includes('DOCK')) {
      g.add(box(L, 0.8, W, color || PALETTE.dock, 0, 0.4, 0));
      g.add(box(L, 0.25, 0.5, '#E5B23C', 0, 0.85, W / 2 - 0.2));
      for (let i = -1; i <= 1; i++) g.add(box(0.3, 1.6, 0.3, PALETTE.wallTrim, i * L * 0.3, 1.5, -W / 2 + 0.3));
      return g;
    }
    if (t.includes('CONVEYOR')) {
      g.add(box(L, 0.5, W, color || PALETTE.conveyor, 0, 1.1, 0));
      const rollers = Math.max(3, Math.round(L / 0.8));
      for (let i = 0; i < rollers; i++) {
        const r = cylinder(W * 0.42, 0.12, '#B9C3C9', -L / 2 + (i + 0.5) * (L / rollers), 1.42, 0);
        r.rotation.x = Math.PI / 2;
        r.rotation.z = Math.PI / 2;
        g.add(r);
      }
      for (const x of [-L / 2 + 0.5, L / 2 - 0.5]) {
        g.add(box(0.25, 1.1, 0.25, PALETTE.wallTrim, x, 0.55, -W / 2 + 0.2));
        g.add(box(0.25, 1.1, 0.25, PALETTE.wallTrim, x, 0.55, W / 2 - 0.2));
      }
      return g;
    }
    if (t.includes('RACK') || t.includes('SHELF') || t.includes('BULK')) {
      return this.buildRack(g, L, W, H, color || PALETTE.rack || '#E08A2E', o.config);
    }
    if (t.includes('BIN')) {
      const levels = Number(o.config?.levels ?? 2);
      for (let i = 0; i < levels; i++) {
        const y = 0.4 + i * (H / levels);
        g.add(box(L, H / levels - 0.15, W, color || PALETTE.bin, 0, y, 0));
        g.add(box(L + 0.1, 0.1, W + 0.1, '#3C6E4C', 0, y + (H / levels - 0.15) / 2, 0));
      }
      return g;
    }
    if (t.includes('PALLET') || t.includes('CARTON')) {
      g.add(box(L, 0.22, W, PALETTE.pallet, 0, 0.12, 0));
      for (let i = -1; i <= 1; i++) g.add(box(L * 0.9, 0.18, 0.18, '#8B5E34', 0, 0.05, i * W * 0.32));
      const stack = Number(o.config?.stack ?? 2);
      for (let i = 0; i < stack; i++) {
        g.add(box(L * 0.86, H / stack * 0.82, W * 0.86, color || PALETTE.carton, 0, 0.22 + (i + 0.5) * (H / stack), 0));
        g.add(box(0.12, H / stack * 0.82, W * 0.87, '#A66A33', 0, 0.22 + (i + 0.5) * (H / stack), 0));
      }
      return g;
    }
    if (t.includes('PACKING') || t.includes('WORKSTATION') || t.includes('TABLE') || t.includes('TROLLEY')) {
      g.add(box(L, 0.22, W, color || PALETTE.packing, 0, H * 0.55, 0));
      for (const x of [-L / 2 + 0.2, L / 2 - 0.2]) {
        for (const z of [-W / 2 + 0.2, W / 2 - 0.2]) g.add(box(0.18, H * 0.55, 0.18, PALETTE.wallTrim, x, H * 0.275, z));
      }
      g.add(box(L * 0.35, H * 0.3, W * 0.5, PALETTE.carton, -L * 0.2, H * 0.55 + 0.2, 0));
      return g;
    }
    if (t.includes('FORKLIFT') || t.includes('HAND')) {
      return this.buildEquipment({ equipmentType: t, ...o } as any);
    }
    if (t.includes('EMPLOYEE') || t.includes('WORKER')) {
      return this.buildWorker({ ...o, animation: o.config?.animation ?? 'IDLE' } as any);
    }
    if (t.includes('FLOOR')) {
      const f = new THREE.Mesh(new THREE.PlaneGeometry(L, W), mat(o.color || PALETTE.floor));
      f.rotation.x = -Math.PI / 2;
      f.position.y = 0.04;
      g.add(f);
      return g;
    }
    // CUSTOM / default
    g.add(box(L, H, W, color || '#6C8F94', 0, H / 2, 0));
    const tag = labelSprite(o.name);
    tag.position.y = H + 1.2;
    g.add(tag);
    return g;
  }

  private buildRack(g: THREE.Group, L: number, W: number, H: number, color: string, config?: any): THREE.Group {
    const levels = Math.max(1, Number(config?.levels ?? 3));
    const bays = Math.max(1, Number(config?.bays ?? 1));
    const uprights = bays + 1;
    for (let i = 0; i < uprights; i++) {
      const x = -L / 2 + (i * L) / bays;
      for (const z of [-W / 2 + 0.1, W / 2 - 0.1]) {
        g.add(box(0.16, H, 0.16, PALETTE.rackBeam, x, H / 2, z));
      }
    }
    for (let lv = 0; lv < levels; lv++) {
      const y = 0.15 + (lv * (H - 0.4)) / levels;
      for (const z of [-W / 2 + 0.1, W / 2 - 0.1]) {
        g.add(box(L, 0.14, 0.14, color, 0, y, z));
      }
      g.add(box(L * 0.96, 0.08, W * 0.85, '#9AA6AC', 0, y + 0.06, 0));
      if (Number(config?.showPallets ?? 1) !== 0 && lv < levels - 1) {
        for (let b = 0; b < bays; b++) {
          const x = -L / 2 + (b + 0.5) * (L / bays);
          g.add(box((L / bays) * 0.8, 0.16, W * 0.72, PALETTE.pallet, x, y + 0.18, 0));
          const boxes = Number(config?.cartonsPerBay ?? 2);
          for (let c = 0; c < boxes; c++) {
            g.add(box((L / bays) * 0.32, H / levels * 0.4, W * 0.6, PALETTE.carton,
              x + (c - (boxes - 1) / 2) * (L / bays) * 0.34, y + 0.18 + H / levels * 0.22, 0));
          }
        }
      }
    }
    return g;
  }

  private buildEquipment(e: Wh3dEquipment): THREE.Group {
    const g = new THREE.Group();
    const type = String(e.equipmentType || '').toUpperCase();
    if (type.includes('HAND') || type.includes('PALLET')) {
      g.add(box(1.6, 0.16, 0.7, PALETTE.handpallet, 0, 0.2, 0));
      for (const z of [-0.28, 0.28]) g.add(cylinder(0.12, 0.14, '#37474F', -0.6, 0.12, z));
      g.add(box(0.12, 0.9, 0.12, PALETTE.wallTrim, 0.7, 0.65, 0));
      g.add(box(0.5, 0.12, 0.12, '#37474F', 0.7, 1.05, 0));
      return g;
    }
    // Forklift
    g.add(box(2.8, 0.55, 1.4, e.color || PALETTE.forklift, 0, 0.62, 0));
    g.add(box(1.2, 1.5, 1.3, e.color || PALETTE.forklift, -0.6, 1.6, 0));
    g.add(box(1.0, 0.75, 1.15, '#BFE0E8', -0.62, 1.75, 0, { transparent: true, opacity: 0.7 }));
    for (const x of [-0.95, 0.95]) {
      for (const z of [-0.72, 0.72]) g.add(cylinder(0.38, 0.26, '#2E3B41', x, 0.38, z).rotateZ(Math.PI / 2));
    }
    g.add(box(0.2, 1.9, 0.2, PALETTE.wallTrim, 1.25, 1.4, -0.5));
    g.add(box(0.2, 1.9, 0.2, PALETTE.wallTrim, 1.25, 1.4, 0.5));
    g.add(box(0.2, 0.2, 1.2, PALETTE.wallTrim, 1.25, 2.35, 0));
    for (const z of [-0.45, 0.45]) g.add(box(1.0, 0.12, 0.16, '#4A5A62', 1.75, 0.18, z));
    return g;
  }

  private buildWorker(p: Wh3dEmployee): THREE.Group {
    const g = new THREE.Group();
    const active = p.activity !== 'IDLE';
    const shirt = active ? '#2E8B9A' : '#7A8C94';
    const g1 = new THREE.Group();
    g1.add(cylinder(0.38, 0.95, shirt, 0, 1.05, 0));
    const head = new THREE.Mesh(new THREE.SphereGeometry(0.34, 14, 10), mat('#D9A679'));
    head.position.y = 1.78;
    head.castShadow = true;
    g1.add(head);
    g1.add(box(0.72, 0.12, 0.72, '#F0C24B', 0, 2.02, 0));
    for (const x of [-0.18, 0.18]) g1.add(box(0.16, 0.65, 0.16, '#37474F', x, 0.33, 0));
    const name = labelSprite(String(p.fullName || '').trim() || `#${p.employeeId}`, '#ffffff', 'rgba(24,60,70,0.86)');
    name.position.y = 2.75;
    name.scale.set(4.6, 0.85, 1);
    g1.add(name);
    g.add(g1);
    g.userData.anim = g1;
    g.userData.worker = p;
    return g;
  }

  // -------------------------------------------------------------- loop

  private animate = () => {
    if (this.disposed) return;
    this.frame = requestAnimationFrame(this.animate);
    const t = this.clock.getElapsedTime();
    this.controls.update();

    for (const d of this.dynamic) {
      if (d.kind === 'WORKER') {
        const g = d.obj.userData.anim as THREE.Group | undefined;
        if (!g) continue;
        const anim = String(d.data.animation || 'IDLE');
        if (anim === 'WALKING' || anim === 'OPERATING_FORKLIFT' || anim === 'MOVING') {
          g.position.x = Math.sin(t * 0.9 + Number(d.data.employeeId)) * 1.1;
          g.rotation.z = Math.sin(t * 5) * 0.05;
        } else if (anim === 'PICKING' || anim === 'CARRYING_BOX' || anim === 'LOADING' || anim === 'UNLOADING') {
          g.rotation.z = Math.sin(t * 3) * 0.12;
        } else {
          g.position.y = Math.sin(t * 1.6 + Number(d.data.employeeId)) * 0.03;
        }
      } else if (d.kind === 'EQUIPMENT') {
        const path = Array.isArray(d.data.path) ? d.data.path : [];
        if (path.length > 1 && String(d.data.status) === 'MOVING') {
          const total = path.length - 1;
          const seg = (t * 0.12 * num(d.data.speed || 1)) % total;
          const i = Math.floor(seg);
          const f = seg - i;
          const a = path[i];
          const b = path[i + 1];
          d.obj.position.x = num(a.x) + (num(b.x) - num(a.x)) * f;
          d.obj.position.z = num(a.y) + (num(b.y) - num(a.y)) * f;
          d.obj.rotation.y = Math.atan2(num(b.x) - num(a.x), num(b.y) - num(a.y));
        }
      }
    }
    if (this.highlight) this.highlight.update();
    this.renderer.render(this.scene, this.camera);
  };

  dispose() {
    this.disposed = true;
    cancelAnimationFrame(this.frame);
    const el = this.renderer.domElement;
    el.removeEventListener('pointerdown', this.onPointerDown);
    el.removeEventListener('pointermove', this.onPointerMove);
    el.removeEventListener('pointerup', this.onPointerUp);
    el.removeEventListener('dblclick', this.onDoubleClick);
    window.removeEventListener('resize', this.onResize);
    this.resizeObserver?.disconnect();
    this.controls.dispose();
    this.renderer.dispose();
    el.remove();
  }
}
