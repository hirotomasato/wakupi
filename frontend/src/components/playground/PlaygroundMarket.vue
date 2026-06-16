<script setup lang="ts">
import { ref, computed, watch, onMounted, onBeforeUnmount, nextTick } from 'vue'
import {
  TrendingUp, TrendingDown, Plus, Trash2, Loader2,
  BarChart3, RefreshCw, Clock, Minus, AlertCircle,
} from '@lucide/vue'
import { createChart, ColorType, CandlestickSeries, type IChartApi, type CandlestickData, type Time, type ISeriesApi } from 'lightweight-charts'
import { useMarketStore, type OHLC } from '../../stores/market'

const market = useMarketStore()

// --- Chart ---
const chartContainer = ref<HTMLDivElement | null>(null)
let chart: IChartApi | null = null
let candleSeries: ISeriesApi<'Candlestick'> | null = null
const chartReady = ref(false)

function destroyChart() {
  chart?.remove()
  chart = null
  candleSeries = null
  chartReady.value = false
}

function buildChart() {
  const el = chartContainer.value
  if (!el || chart) return
  // Container must have non-zero dimensions
  if (el.clientWidth === 0 || el.clientHeight === 0) return

  chart = createChart(el, {
    width: el.clientWidth,
    height: el.clientHeight,
    layout: {
      background: { type: ColorType.Solid, color: 'transparent' },
      textColor: '#8b9baa',
    },
    grid: {
      vertLines: { color: 'rgba(128,128,128,0.08)' },
      horzLines: { color: 'rgba(128,128,128,0.08)' },
    },
    crosshair: { mode: 0 },
    timeScale: {
      borderColor: 'rgba(128,128,128,0.12)',
      timeVisible: true,
      secondsVisible: false,
    },
    rightPriceScale: {
      borderColor: 'rgba(128,128,128,0.12)',
    },
  })

  candleSeries = chart.addSeries(CandlestickSeries, {
    upColor: '#00a884',
    downColor: '#ef4444',
    borderUpColor: '#00a884',
    borderDownColor: '#ef4444',
    wickUpColor: '#00a884',
    wickDownColor: '#ef4444',
  })

  chartReady.value = true

  // Keep chart sized to container
  const ro = new ResizeObserver(() => {
    if (el && chart) chart.applyOptions({ width: el.clientWidth, height: el.clientHeight })
  })
  ro.observe(el)
}

function setChartData(data: OHLC[]) {
  if (!candleSeries || !data.length) return
  const cd: CandlestickData[] = data.map((d) => ({
    time: d.time as Time,
    open: d.open,
    high: d.high,
    low: d.low,
    close: d.close,
  }))
  candleSeries.setData(cd)
}

// Build chart on mount + data
let initAttempts = 0
function tryInitChart() {
  const el = chartContainer.value
  if (!el || chart) return
  if (el.clientWidth === 0) {
    // Container not yet sized — retry soon
    if (initAttempts++ < 20) {
      requestAnimationFrame(tryInitChart)
    }
    return
  }
  buildChart()
  if (market.selectedChart.length > 0) {
    setChartData(market.selectedChart)
  }
}

// When data arrives, try to show it
watch(() => market.selectedChart, (data) => {
  if (data.length > 0) {
    nextTick(() => {
      tryInitChart()
      setChartData(data)
    })
  }
})

// When symbol changes, chart might need rebuild
watch(() => market.selectedSymbol, () => {
  if (chart) {
    // Chart exists — just update data
    if (market.selectedChart.length > 0) setChartData(market.selectedChart)
  } else {
    nextTick(() => tryInitChart())
  }
})

onMounted(async () => {
  await market.loadAll()
  market.startAutoRefresh(30) // 30s real-time
  // Give DOM a frame to paint, then init
  requestAnimationFrame(() => {
    requestAnimationFrame(() => {
      tryInitChart()
    })
  })
})

onBeforeUnmount(() => {
  market.stopAutoRefresh()
  destroyChart()
})

// --- Add symbol ---
const newSymbol = ref('')
function addSymbol() {
  const s = newSymbol.value.trim()
  if (!s) return
  market.addToWatchlist(s)
  newSymbol.value = ''
}

// --- Helpers ---
const fmtCur = (q: any) => {
  if (q.currency === 'IDR') return 'Rp'
  if (q.currency === 'USD') return '$'
  return q.currency || ''
}

const fmtPrice = (q: any) => {
  const p = q.price
  if (q.currency === 'IDR') {
    if (p >= 1000) return p.toLocaleString('id-ID', { maximumFractionDigits: 0 })
    return p.toLocaleString('id-ID', { maximumFractionDigits: 2 })
  }
  if (p >= 1) return p.toLocaleString('en-US', { maximumFractionDigits: 2 })
  return p.toLocaleString('en-US', { maximumFractionDigits: 6 })
}

const fmtVol = (n: number) => {
  if (n >= 1e9) return (n / 1e9).toFixed(1) + 'B'
  if (n >= 1e6) return (n / 1e6).toFixed(1) + 'M'
  if (n >= 1e3) return (n / 1e3).toFixed(1) + 'K'
  return n.toString()
}

const changeIcon = (q: any) => {
  if (q.change > 0) return TrendingUp
  if (q.change < 0) return TrendingDown
  return Minus
}

const changeColor = (q: any) => {
  if (q.change > 0) return 'text-wa-green'
  if (q.change < 0) return 'text-red-500'
  return 'text-wa-muted'
}

const ranges = [
  { id: '1d', label: '1H' },
  { id: '5d', label: '5H' },
  { id: '1mo', label: '1B' },
  { id: '3mo', label: '3B' },
  { id: '6mo', label: '6B' },
  { id: '1y', label: '1T' },
]

async function changeRange(rng: string) {
  market.selectedRange = rng
  chartReady.value = false
  await market.loadChart(market.selectedSymbol, rng)
  nextTick(() => {
    if (!chart) buildChart()
    setChartData(market.selectedChart)
    chartReady.value = true
  })
}
</script>

<template>
  <div class="h-full flex bg-wa-bg dark:bg-[#0b141a]">
    <!-- LEFT: Watchlist -->
    <div class="w-[260px] shrink-0 border-r border-wa-border dark:border-wa-border-dark flex flex-col h-full">
      <header class="px-3 py-2.5 border-b border-wa-border dark:border-wa-border-dark flex items-center justify-between">
        <span class="text-sm font-semibold text-wa-text dark:text-wa-text-dark flex items-center gap-1.5">
          <BarChart3 :size="15" class="text-wa-green" /> Watchlist
        </span>
        <button
          @click="market.refreshQuotes()"
          class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
          title="Refresh"
        >
          <RefreshCw :size="14" :class="{ 'animate-spin': market.loading }" />
        </button>
      </header>

      <!-- Add symbol -->
      <div class="px-3 py-2 flex gap-1.5">
        <input
          v-model="newSymbol"
          placeholder="+ Tambah (BBCA.JK)"
          @keydown.enter="addSymbol"
          class="flex-1 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-2.5 py-1.5 text-xs outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark"
        />
        <button
          @click="addSymbol"
          :disabled="!newSymbol.trim()"
          class="shrink-0 w-7 h-7 rounded-lg bg-wa-green text-white flex items-center justify-center disabled:opacity-40"
        >
          <Plus :size="14" />
        </button>
      </div>

      <!-- Error -->
      <div v-if="market.error" class="mx-3 mt-2 px-2.5 py-2 text-xs text-red-600 dark:text-red-400 bg-red-50 dark:bg-red-900/20 rounded-lg flex items-center gap-1.5">
        <AlertCircle :size="12" /> {{ market.error }}
      </div>

      <!-- Quote list -->
      <div class="flex-1 overflow-y-auto scrollbar-thin">
        <div v-if="market.loading && market.sortedQuotes.length === 0" class="flex items-center justify-center py-12">
          <Loader2 :size="20" class="animate-spin text-wa-muted" />
        </div>

        <button
          v-for="q in market.sortedQuotes"
          :key="q.symbol"
          @click="market.selectSymbol(q.symbol)"
          class="w-full text-left px-3 py-2.5 border-b border-wa-border/50 dark:border-wa-border-dark/50 hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition flex items-center justify-between"
          :class="{ 'bg-wa-green/5 border-l-[3px] border-l-wa-green': market.selectedSymbol === q.symbol }"
        >
          <div class="min-w-0 flex-1">
            <div class="flex items-center gap-1.5">
              <span class="text-sm font-semibold text-wa-text dark:text-wa-text-dark">{{ q.symbol.replace('.JK', '') }}</span>
              <span class="text-[10px] text-wa-muted dark:text-wa-muted-dark truncate">{{ q.name }}</span>
            </div>
            <div class="text-[11px] text-wa-muted dark:text-wa-muted-dark flex items-center gap-1.5 mt-0.5">
              <Clock :size="10" />
              <span>Vol {{ fmtVol(q.volume) }}</span>
            </div>
          </div>
          <div class="text-right shrink-0 ml-2">
            <div class="text-sm font-semibold tabular-nums text-wa-text dark:text-wa-text-dark">
              {{ fmtCur(q) }} {{ fmtPrice(q) }}
            </div>
            <div class="text-xs font-medium tabular-nums flex items-center justify-end gap-0.5" :class="changeColor(q)">
              <component :is="changeIcon(q)" :size="12" />
              {{ q.changePercent >= 0 ? '+' : '' }}{{ q.changePercent.toFixed(2) }}%
            </div>
          </div>
        </button>
      </div>
    </div>

    <!-- RIGHT: Chart -->
    <div class="flex-1 flex flex-col min-w-0">
      <header class="px-4 py-2.5 border-b border-wa-border dark:border-wa-border-dark flex items-center justify-between flex-wrap gap-2">
        <div v-if="market.selectedQuote" class="flex items-center gap-4 flex-wrap">
          <div>
            <span class="text-base font-bold text-wa-text dark:text-wa-text-dark">{{ market.selectedSymbol.replace('.JK', '') }}</span>
            <span class="text-xs text-wa-muted dark:text-wa-muted-dark ml-1.5">{{ market.selectedQuote.name }}</span>
          </div>
          <div class="flex items-center gap-3 text-sm">
            <span class="font-semibold tabular-nums text-wa-text dark:text-wa-text-dark">
              {{ fmtCur(market.selectedQuote) }} {{ fmtPrice(market.selectedQuote) }}
            </span>
            <span class="font-medium tabular-nums flex items-center gap-0.5" :class="changeColor(market.selectedQuote)">
              <component :is="changeIcon(market.selectedQuote)" :size="14" />
              {{ market.selectedQuote.changePercent >= 0 ? '+' : '' }}{{ market.selectedQuote.changePercent.toFixed(2) }}%
            </span>
          </div>
          <div class="flex gap-3 text-xs text-wa-muted dark:text-wa-muted-dark">
            <span>O: {{ fmtCur(market.selectedQuote) }} {{ market.selectedQuote.open.toLocaleString('id-ID', { maximumFractionDigits: 0 }) }}</span>
            <span>H: {{ fmtCur(market.selectedQuote) }} {{ market.selectedQuote.high.toLocaleString('id-ID', { maximumFractionDigits: 0 }) }}</span>
            <span>L: {{ fmtCur(market.selectedQuote) }} {{ market.selectedQuote.low.toLocaleString('id-ID', { maximumFractionDigits: 0 }) }}</span>
          </div>
        </div>
        <div v-else-if="market.error" class="text-sm text-red-500">{{ market.error }}</div>

        <!-- Range selector -->
        <div class="flex gap-1">
          <button
            v-for="r in ranges" :key="r.id"
            @click="changeRange(r.id)"
            class="px-2.5 py-1 text-xs rounded-lg font-medium transition"
            :class="market.selectedRange === r.id
              ? 'bg-wa-green text-white'
              : 'text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark'"
          >
            {{ r.label }}
          </button>
        </div>
      </header>

      <!-- Chart area -->
      <div class="flex-1 relative min-h-0">
        <div
          ref="chartContainer"
          class="absolute inset-0"
          style="width: 100%; height: 100%;"
        />
        <!-- Loading overlay -->
        <div v-if="market.loading && !chartReady" class="absolute inset-0 flex items-center justify-center bg-white/50 dark:bg-[#0b141a]/50 z-10">
          <Loader2 :size="24" class="animate-spin text-wa-green" />
        </div>
      </div>
    </div>
  </div>
</template>
