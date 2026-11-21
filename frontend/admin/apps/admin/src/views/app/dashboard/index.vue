<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue';

import { useRouter } from 'vue-router';

import { Page } from '@vben/common-ui';
import {
  LucideCoins,
  LucideInbox,
  LucideTrendingDown,
  LucideTrendingUp,
  LucideWallet,
} from '@vben/icons';

import { Alert, Card, Table, notification } from 'ant-design-vue';

import { fclQuery, formatRp } from '#/api/warehouse';
import {
  DIVISI_COLORS,
  matchRosterDivisiKeyword,
  normalizeRosterRows,
  summarizeRosterDay,
} from '#/utils/warehouse-roster';

const router = useRouter();
const loading = ref(true);
const loadError = ref('');

const saldoGudang = ref(0);
const saldoTB = ref(0);
const kasMasuk = ref(0);
const kasKeluar = ref(0);
const rosterSummary = ref<ReturnType<typeof summarizeRosterDay> | null>(null);
const lemburTop = ref<Array<{ nama: string; divisi: string; totalJam: number; jumlah: number }>>([]);
const recentHistory = ref<Array<any>>([]);
const laporanDivisi = ref<Array<{ divisi: string; total: number }>>([]);

const lemburColumns = [
  { title: 'Nama', dataIndex: 'nama' },
  { title: 'Divisi', dataIndex: 'divisi' },
  { title: 'Jumlah Jam', dataIndex: 'totalJam', align: 'right' as const },
  { title: 'Jml Lembur', dataIndex: 'jumlah', align: 'right' as const },
];

const historyColumns = [
  { title: 'Tanggal', dataIndex: 'tanggal', width: 110 },
  { title: 'Tipe', dataIndex: 'tipe', width: 130 },
  { title: 'Keterangan', dataIndex: 'keterangan', ellipsis: true },
  { title: 'Nominal', dataIndex: 'nominal', width: 150, align: 'right' as const },
  { title: 'Kategori', dataIndex: 'kategori', width: 130 },
];

const maxLaporanTotal = computed(() =>
  Math.max(1, ...laporanDivisi.value.map((d) => d.total)),
);

async function load() {
  loading.value = true;
  loadError.value = '';
  try {
    const [kgData, tbData, lkData, lbData, rosterRaw] = await Promise.all([
      fclQuery<any>('kasgudang', { select: '*', order: 'createdAt.desc', limit: 200 }),
      fclQuery<any>('teambuilding', { select: '*', order: 'createdAt.desc', limit: 200 }),
      fclQuery<any>('laporankerja', { select: '*', order: 'tanggal.desc', limit: 500 }),
      fclQuery<any>('lembur', { select: '*', order: 'createdAt.desc', limit: 500 }),
      fclQuery<any>('jadwalroster', { select: '*', limit: 2000 }),
    ]);

    saldoGudang.value = kgData.reduce(
      (sum, d) => sum + (d.tipe === 'IN' ? Number(d.nominal || 0) : -Number(d.nominal || 0)),
      0,
    );
    kasMasuk.value = kgData
      .filter((d) => d.tipe === 'IN')
      .reduce((sum, d) => sum + Number(d.nominal || 0), 0);
    kasKeluar.value = kgData
      .filter((d) => d.tipe === 'OUT')
      .reduce((sum, d) => sum + Number(d.nominal || 0), 0);
    saldoTB.value = tbData.reduce(
      (sum, d) => sum + (d.tipe === 'Pemasukan' ? Number(d.nominal || 0) : -Number(d.nominal || 0)),
      0,
    );

    // Roster — hadir hari ini
    const now = new Date();
    const curMonth = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`;
    const rosterRows = normalizeRosterRows(rosterRaw, curMonth);
    const karMap: Record<string, string> = {};
    // Divisi diambil dari laporan kerja harian
    lkData.forEach((l) => {
      const nama = String(l.pic || '').trim().toLowerCase();
      const div = String(l.divisi || '').trim();
      if (nama && div && !karMap[nama]) karMap[nama] = div;
    });
    rosterSummary.value = summarizeRosterDay(rosterRows, now.getDate(), karMap);

    // Lembur top
    const lemburMap: Record<string, { nama: string; divisi: string; totalJam: number; jumlah: number }> = {};
    lbData.forEach((l) => {
      const nama = String(l.nama || '').trim();
      if (!nama) return;
      if (!lemburMap[nama]) lemburMap[nama] = { nama, divisi: l.divisi || '-', totalJam: 0, jumlah: 0 };
      lemburMap[nama].totalJam += Number(l.jumlahJam || 0);
      lemburMap[nama].jumlah += 1;
    });
    lemburTop.value = Object.values(lemburMap)
      .sort((a, b) => b.totalJam - a.totalJam || b.jumlah - a.jumlah)
      .slice(0, 5);

    // Laporan per divisi (dari tugas laporan bulan berjalan atau semuanya)
    const divCount: Record<string, number> = {};
    lkData.forEach((l) => {
      const key = matchRosterDivisiKeyword(String(l.divisi || '').toLowerCase());
      if (key) divCount[key] = (divCount[key] || 0) + 1;
    });
    laporanDivisi.value = Object.entries(divCount)
      .map(([divisi, total]) => ({ divisi, total }))
      .sort((a, b) => b.total - a.total);

    // Riwayat transaksi gabungan
    const history: any[] = [];
    kgData.forEach((k) => {
      history.push({
        tanggal: k.tanggal,
        tipe: k.tipe === 'IN' ? 'Kas Masuk' : 'Kas Keluar',
        keterangan: k.keterangan,
        nominal: Number(k.nominal || 0),
        kategori: 'Kas Gudang',
      });
    });
    tbData.forEach((t) => {
      history.push({
        tanggal: t.tanggal,
        tipe: t.tipe === 'Pemasukan' ? 'TB Masuk' : 'TB Keluar',
        keterangan: t.keterangan,
        nominal: Number(t.nominal || 0),
        kategori: 'Team Building',
      });
    });
    history.sort((a, b) => new Date(b.tanggal || 0).getTime() - new Date(a.tanggal || 0).getTime());
    recentHistory.value = history.slice(0, 10);
  } catch (err: any) {
    console.error('[warehouse/dashboard] load gagal', err);
    loadError.value = err?.message || 'Gagal memuat data dashboard';
    notification.error({ message: 'Dashboard FCL', description: loadError.value });
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <Page auto-content-height>
    <template #extra>
      <a-button size="small" @click="load" :loading="loading">Refresh</a-button>
    </template>

    <Alert
      v-if="loadError"
      class="mb-3"
      type="error"
      :message="loadError"
      show-icon
    />

    <!-- Stat cards -->
    <div class="wh-stat-grid">
      <Card :bordered="false" class="wh-stat-card wh-tone-teal">
        <div class="wh-stat-label"><LucideInbox :size="16" /> Saldo Gudang</div>
        <div class="wh-stat-value">{{ formatRp(saldoGudang) }}</div>
      </Card>
      <Card :bordered="false" class="wh-stat-card wh-tone-green">
        <div class="wh-stat-label"><LucideWallet :size="16" /> Saldo Team Building</div>
        <div class="wh-stat-value">{{ formatRp(saldoTB) }}</div>
      </Card>
      <Card :bordered="false" class="wh-stat-card wh-tone-amber">
        <div class="wh-stat-label"><LucideTrendingUp :size="16" /> Kas Masuk</div>
        <div class="wh-stat-value">{{ formatRp(kasMasuk) }}</div>
      </Card>
      <Card :bordered="false" class="wh-stat-card wh-tone-red">
        <div class="wh-stat-label"><LucideTrendingDown :size="16" /> Kas Keluar</div>
        <div class="wh-stat-value">{{ formatRp(kasKeluar) }}</div>
      </Card>
    </div>

    <div class="wh-two-col">
      <!-- Masuk hari ini -->
      <Card title="Masuk Hari Ini (Roster)" :bordered="false" :loading="loading">
        <template v-if="rosterSummary">
          <div class="wh-masuk-total">
            <span class="wh-masuk-num">{{ rosterSummary.total }}</span>
            <span class="wh-masuk-unit">orang</span>
            <span class="wh-shift-pill wh-pagi">🌅 Pagi {{ rosterSummary.pagi }}</span>
            <span class="wh-shift-pill wh-malam">🌙 Malam {{ rosterSummary.malam }}</span>
          </div>
          <div class="wh-chip-row">
            <span class="wh-chip wh-chip-off">OFF {{ rosterSummary.off }}</span>
            <span class="wh-chip wh-chip-cuti">CUTI {{ rosterSummary.cuti }}</span>
            <span class="wh-chip wh-chip-alfa">ALFA {{ rosterSummary.alfa }}</span>
            <span class="wh-chip wh-chip-sakit">SAKIT {{ rosterSummary.sakit }}</span>
            <span class="wh-chip wh-chip-ijin">IJIN {{ rosterSummary.ijin }}</span>
          </div>
          <div class="wh-chip-row">
            <span
              v-for="(s, div) in rosterSummary.perDivisi"
              :key="div"
              v-show="s.total > 0"
              class="wh-chip wh-chip-div"
            >
              <span class="wh-dot" :style="{ background: DIVISI_COLORS[div] || '#64748b' }"></span>
              <b>{{ div }}</b>: {{ s.total }}
              <span class="wh-chip-sub">(P:{{ s.pagi }} M:{{ s.malam }})</span>
            </span>
          </div>
        </template>
        <div v-else-if="!loading" class="wh-empty">Roster bulan ini belum ada.</div>
      </Card>

      <!-- Laporan per divisi -->
      <Card title="Laporan Kerja per Divisi" :bordered="false" :loading="loading">
        <div v-if="!laporanDivisi.length" class="wh-empty">Belum ada laporan.</div>
        <div v-for="d in laporanDivisi" :key="d.divisi" class="wh-bar-row">
          <span class="wh-bar-label">{{ d.divisi }}</span>
          <div class="wh-bar-track">
            <div
              class="wh-bar-fill"
              :style="{ width: `${(d.total / maxLaporanTotal) * 100}%`, background: DIVISI_COLORS[d.divisi] || '#0d74f2' }"
            ></div>
          </div>
          <span class="wh-bar-val">{{ d.total }}</span>
        </div>
      </Card>
    </div>

    <div class="wh-two-col">
      <!-- Top lembur -->
      <Card title="Top Lembur" :bordered="false" :loading="loading">
        <Table
          :columns="lemburColumns"
          :data-source="lemburTop"
          :pagination="false"
          :row-key="(r: any) => r.nama"
          size="small"
        />
      </Card>

      <!-- Riwayat transaksi -->
      <Card title="Transaksi Terakhir" :bordered="false" :loading="loading">
        <Table
          :columns="historyColumns"
          :data-source="recentHistory"
          :pagination="false"
          :row-key="(r: any) => `${r.tanggal}-${r.tipe}-${r.keterangan}`"
          size="small"
        >
          <template #bodyCell="{ column, record }">
            <span v-if="column.dataIndex === 'nominal'" :class="record.nominal >= 0 ? 'wh-plus' : 'wh-minus'">
              {{ formatRp(record.nominal) }}
            </span>
          </template>
        </Table>
      </Card>
    </div>

    <!-- Akses cepat ke fitur FCL lainnya (masih iframe) -->
    <Card title="Modul FCL Lainnya" :bordered="false" class="mt-3">
      <div class="wh-quick">
        <a-button type="link" @click="router.push('/fcl/warehouse/stock')">Stock Barang</a-button>
        <a-button type="link" @click="router.push('/fcl/warehouse/order')">Order</a-button>
        <a-button type="link" @click="router.push('/fcl/warehouse/inbound')">Inbound</a-button>
        <a-button type="link" @click="router.push('/fcl/keuangan/kasGudang')">Kas Gudang</a-button>
        <a-button type="link" @click="router.push('/fcl/keuangan/pettyCash')">Petty Cash</a-button>
        <a-button type="link" @click="router.push('/fcl/operasional/laporanKerja')">Laporan Kerja</a-button>
        <a-button type="link" @click="router.push('/fcl/sdm/jadwalShift')">Jadwal Shift</a-button>
      </div>
    </Card>
  </Page>
</template>

<style scoped>
.wh-stat-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
  margin-bottom: 12px;
}
.wh-stat-card .wh-stat-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: hsl(var(--muted-foreground));
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.wh-stat-value {
  font-size: 22px;
  font-weight: 800;
  margin-top: 6px;
  color: hsl(var(--foreground));
}
.wh-tone-teal .wh-stat-value { color: hsl(var(--success)); }
.wh-tone-green .wh-stat-value { color: hsl(var(--success)); }
.wh-tone-amber .wh-stat-value { color: hsl(var(--warning)); }
.wh-tone-red .wh-stat-value { color: hsl(var(--destructive)); }

.wh-two-col {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
  margin-bottom: 12px;
}
@media (max-width: 1200px) {
  .wh-two-col { grid-template-columns: 1fr; }
  .wh-stat-grid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
}

.wh-masuk-total {
  display: flex;
  align-items: baseline;
  gap: 10px;
  flex-wrap: wrap;
}
.wh-masuk-num { font-size: 40px; font-weight: 800; }
.wh-masuk-unit { color: hsl(var(--muted-foreground)); font-size: 14px; }
.wh-shift-pill {
  font-size: 12px;
  font-weight: 700;
  padding: 4px 10px;
  border-radius: 999px;
}
.wh-pagi { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); }
.wh-malam { background: hsl(var(--primary) / 0.16); color: hsl(var(--primary)); }

.wh-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}
.wh-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  padding: 3px 8px;
  border-radius: 8px;
  background: hsl(var(--muted));
  color: hsl(var(--foreground));
}
.wh-chip-off, .wh-chip-alfa { background: hsl(var(--destructive) / 0.14); color: hsl(var(--destructive)); }
.wh-chip-cuti, .wh-chip-sakit { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); }
.wh-chip-ijin { background: hsl(var(--warning) / 0.16); color: hsl(var(--warning)); }
.wh-chip-div { background: hsl(var(--muted)); }
.wh-chip-sub { color: hsl(var(--muted-foreground)); font-size: 10px; }
.wh-dot { width: 8px; height: 8px; border-radius: 999px; display: inline-block; }

.wh-bar-row {
  display: grid;
  grid-template-columns: 120px 1fr 40px;
  gap: 8px;
  align-items: center;
  margin-bottom: 6px;
  font-size: 12px;
}
.wh-bar-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.wh-bar-track { height: 10px; background: hsl(var(--muted)); border-radius: 999px; overflow: hidden; }
.wh-bar-fill { height: 100%; border-radius: 999px; transition: width 0.4s; }
.wh-bar-val { text-align: right; font-weight: 700; }

.wh-empty { color: hsl(var(--muted-foreground)); padding: 16px 0; text-align: center; }

.wh-quick { display: flex; flex-wrap: wrap; gap: 4px; }

.wh-plus { color: hsl(var(--success)); }
.wh-minus { color: hsl(var(--destructive)); }
</style>
