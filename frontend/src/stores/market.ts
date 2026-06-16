import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { MarketGetQuote, MarketGetQuotes, MarketGetChart } from '../../wailsjs/go/main/App'

export interface Quote {
  symbol: string
  name: string
  price: number
  change: number
  changePercent: number
  high: number
  low: number
  open: number
  volume: number
  currency: string
  exchange: string
}

export interface OHLC {
  time: number  // unix seconds
  open: number
  high: number
  low: number
  close: number
  volume: number
}

const STORAGE_KEY = 'wakupi.market.watchlist'

// Default watchlist: blue-chip IDX stocks + Bitcoin
const DEFAULT_WATCHLIST = [
  'BBCA.JK', 'TLKM.JK', 'BMRI.JK', 'BBNI.JK',
  'ASII.JK', 'UNVR.JK', 'ICBP.JK', 'ADRO.JK',
  'GOTO.JK', 'BTC-USD',
]

function loadWatchlist(): string[] {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) return JSON.parse(raw)
  } catch {}
  return [...DEFAULT_WATCHLIST]
}

function saveWatchlist(list: string[]) {
  try { localStorage.setItem(STORAGE_KEY, JSON.stringify(list)) } catch {}
}

export const useMarketStore = defineStore('market', () => {
  const watchlist = ref<string[]>(loadWatchlist())
  const quotes = ref<Record<string, Quote>>({})
  const charts = ref<Record<string, OHLC[]>>({})
  const selectedSymbol = ref<string>('BBCA.JK')
  const selectedRange = ref<string>('1d')  // default intraday
  const loading = ref(false)
  const error = ref('')
  let timer: ReturnType<typeof setInterval> | null = null

  const selectedQuote = computed(() => quotes.value[selectedSymbol.value] || null)
  const selectedChart = computed(() => charts.value[selectedSymbol.value] || [])

  const sortedQuotes = computed(() => {
    return watchlist.value
      .map((s) => quotes.value[s])
      .filter((q): q is Quote => !!q)
  })

  // --- Actions ---

  async function refreshQuotes() {
    if (watchlist.value.length === 0) return
    try {
      const list = (await MarketGetQuotes(watchlist.value)) as Quote[]
      for (const q of list) {
        quotes.value[q.symbol] = q
      }
      // Also refresh chart for the selected symbol
      if (selectedSymbol.value) {
        await loadChart(selectedSymbol.value, selectedRange.value)
      }
      error.value = ''
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function loadChart(symbol: string, rng: string = '1mo') {
    try {
      const data = await MarketGetChart(symbol, rng)
      charts.value[symbol] = data
    } catch (e: any) {
      error.value = e?.message || String(e)
    }
  }

  async function selectSymbol(symbol: string) {
    selectedSymbol.value = symbol
    if (!charts.value[symbol]) {
      loading.value = true
      await Promise.all([refreshQuotes(), loadChart(symbol, selectedRange.value)])
      loading.value = false
    }
  }

  async function loadAll() {
    loading.value = true
    error.value = ''
    try {
      await Promise.all([
        refreshQuotes(),
        loadChart(selectedSymbol.value, selectedRange.value),
      ])
    } finally {
      loading.value = false
    }
  }

  function addToWatchlist(symbol: string) {
    const s = symbol.trim().toUpperCase()
    if (!s || watchlist.value.includes(s)) return
    watchlist.value.push(s)
    saveWatchlist(watchlist.value)
    // Fetch the new symbol immediately
    MarketGetQuotes([s]).then((list) => {
      if (list.length > 0) quotes.value[s] = list[0]
    }).catch(() => {})
  }

  function removeFromWatchlist(symbol: string) {
    watchlist.value = watchlist.value.filter((s) => s !== symbol)
    saveWatchlist(watchlist.value)
    delete quotes.value[symbol]
    delete charts.value[symbol]
    if (selectedSymbol.value === symbol && watchlist.value.length > 0) {
      selectSymbol(watchlist.value[0])
    }
  }

  function startAutoRefresh(intervalSec = 60) {
    stopAutoRefresh()
    timer = setInterval(refreshQuotes, intervalSec * 1000)
  }

  function stopAutoRefresh() {
    if (timer) { clearInterval(timer); timer = null }
  }

  return {
    watchlist,
    quotes,
    charts,
    selectedSymbol,
    selectedRange,
    selectedQuote,
    selectedChart,
    sortedQuotes,
    loading,
    error,
    refreshQuotes,
    loadChart,
    selectSymbol,
    loadAll,
    addToWatchlist,
    removeFromWatchlist,
    startAutoRefresh,
    stopAutoRefresh,
  }
})
