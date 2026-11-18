<script lang="ts" setup>
/** Footer tabel standar FCL: Total records, ukuran halaman, navigasi halaman. */
import { computed } from 'vue';

const props = defineProps<{ total: number; page: number; pageSize: number }>();
const emit = defineEmits<{
  (e: 'update:page', value: number): void;
  (e: 'update:pageSize', value: number): void;
}>();

const totalPages = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)));
const from = computed(() => (props.total ? (props.page - 1) * props.pageSize + 1 : 0));
const to = computed(() => Math.min(props.total, props.page * props.pageSize));

const visiblePages = computed(() => {
  const total = totalPages.value;
  const cur = props.page;
  if (total <= 7) return Array.from({ length: total }, (_, i) => i + 1);
  const start = Math.max(1, Math.min(cur - 2, total - 6));
  return Array.from({ length: 7 }, (_, i) => start + i);
});

function go(p: number) {
  if (p >= 1 && p <= totalPages.value) emit('update:page', p);
}
function onSize(e: Event) {
  emit('update:pageSize', Number((e.target as HTMLSelectElement).value));
  emit('update:page', 1);
}
</script>

<template>
  <div class="fcl-pagination">
    <span class="fcl-pg-total">
      Total <b>{{ total }}</b> records
      <span v-if="total" class="fcl-pg-range">({{ from }}–{{ to }})</span>
    </span>
    <select class="fcl-pg-size" :value="pageSize" @change="onSize">
      <option v-for="n in [10, 20, 50, 100]" :key="n" :value="n">{{ n }}/page</option>
    </select>
    <div class="fcl-pg-pages">
      <button :disabled="page <= 1" title="Halaman pertama" @click="go(1)">|‹</button>
      <button :disabled="page <= 1" title="Sebelumnya" @click="go(page - 1)">‹</button>
      <button
        v-for="p in visiblePages"
        :key="p"
        :class="{ 'fcl-pg-active': p === page }"
        @click="go(p)"
      >{{ p }}</button>
      <button :disabled="page >= totalPages" title="Berikutnya" @click="go(page + 1)">›</button>
      <button :disabled="page >= totalPages" title="Halaman terakhir" @click="go(totalPages)">›|</button>
    </div>
  </div>
</template>

<style scoped>
.fcl-pagination {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  padding: 8px 10px;
  font-size: 11px;
  color: hsl(var(--muted-foreground));
}
.fcl-pg-total b { color: hsl(var(--foreground)); }
.fcl-pg-range { opacity: 0.7; margin-left: 4px; }
.fcl-pg-size {
  height: 26px; border-radius: 5px; padding: 2px 6px; font-size: 11px;
  border: 1px solid hsl(var(--border)); background: hsl(var(--background));
  color: hsl(var(--foreground)); outline: none;
}
.fcl-pg-pages { display: flex; gap: 3px; margin-left: auto; }
.fcl-pg-pages button {
  min-width: 26px; height: 26px; border-radius: 5px; font-size: 11px; font-weight: 700;
  border: 1px solid hsl(var(--border)); background: hsl(var(--muted));
  color: hsl(var(--foreground)); cursor: pointer;
}
.fcl-pg-pages button:disabled { opacity: 0.4; cursor: default; }
.fcl-pg-active { background: hsl(var(--primary)) !important; border-color: hsl(var(--primary)) !important; color: hsl(var(--primary-foreground)) !important; }
</style>
