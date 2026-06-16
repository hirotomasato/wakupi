<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { PanelRightClose, RefreshCw, Settings2, Image as ImageIcon } from '@lucide/vue'
import { usePlaygroundStore, type PGTab } from '../../stores/playground'
import { useUIStore } from '../../stores/ui'
import { useAIStore } from '../../stores/ai'

const pg = usePlaygroundStore()
const ui = useUIStore()
const ai = useAIStore()

const models = ref<string[]>([])
const loadingModels = ref(false)
const gamapiModels = ref<string[]>([])
const gamapiStyles = ref<Record<string, string>>({})
const gamapiRatios = ref<Record<string, string>>({})

const session = computed(() => pg.activeSession)
const inheritedModel = computed(() => ai.config.model || '(default provider)')
const isImageTab = computed(() => pg.pgTab === 'image')
const isGamAPI = computed(() => ai.config.provider === 'gamapi')

const dallESizes = [
  { id: '1024x1024', label: 'Square (1024×1024)' },
  { id: '1792x1024', label: 'Landscape (1792×1024)' },
  { id: '1024x1792', label: 'Portrait (1024×1792)' },
]

async function loadModels() {
  loadingModels.value = true
  try {
    models.value = await ai.listModels(ai.config)
  } catch {
    models.value = []
  } finally {
    loadingModels.value = false
  }
}

async function loadGamAPIData() {
  const [m, s, r] = await Promise.all([
    ai.getGamAPIModels().catch(() => [] as string[]),
    ai.getGamAPIStyles().catch(() => ({} as Record<string, string>)),
    ai.getGamAPIRatios().catch(() => ({} as Record<string, string>)),
  ])
  gamapiModels.value = m
  gamapiStyles.value = s
  gamapiRatios.value = r
}

// Pull model list when panel opens.
watch(
  () => ui.pgRightCollapsed,
  (collapsed) => {
    if (!collapsed && models.value.length === 0 && ai.config.enabled) loadModels()
    if (!collapsed && isGamAPI.value && gamapiModels.value.length === 0) loadGamAPIData()
  },
  { immediate: true }
)

// Reload when switching tab.
watch(
  () => pg.pgTab,
  (tab) => {
    if (!ui.pgRightCollapsed) {
      if (tab === 'chat' && models.value.length === 0 && ai.config.enabled) loadModels()
      if (tab === 'image' && isGamAPI.value && gamapiModels.value.length === 0) loadGamAPIData()
    }
  }
)
</script>

<template>
  <div class="h-full flex flex-col bg-wa-panel dark:bg-[#111b21] border-l border-wa-border dark:border-wa-border-dark">
    <header class="flex items-center justify-between px-4 py-3 border-b border-wa-border dark:border-wa-border-dark">
      <span class="text-sm font-semibold flex items-center gap-2 text-wa-text dark:text-wa-text-dark">
        <ImageIcon v-if="isImageTab" :size="16" class="text-fuchsia-500" />
        <Settings2 v-else :size="16" class="text-violet-500" />
        {{ isImageTab ? 'Image Params' : 'Parameter' }}
      </span>
      <button
        @click="ui.pgRightCollapsed = true"
        class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        title="Sembunyikan panel"
      >
        <PanelRightClose :size="16" />
      </button>
    </header>

    <!-- CHAT PARAMETERS -->
    <div v-if="!isImageTab && session" class="flex-1 overflow-y-auto p-4 space-y-5">
      <div v-if="!ai.config.enabled" class="text-xs bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 rounded-lg p-3">
        AI belum aktif. Buka pengaturan AI (ikon ✨) untuk mengatur provider dan API key.
      </div>

      <div>
        <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Model</label>
        <div class="flex gap-2 mt-1.5">
          <input
            v-model="session.model"
            list="pg-model-list"
            :placeholder="inheritedModel"
            class="flex-1 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs border border-wa-border dark:border-wa-border-dark"
          />
          <button
            @click="loadModels"
            :disabled="loadingModels"
            class="shrink-0 px-2.5 rounded-lg border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark disabled:opacity-50"
            title="Muat daftar model"
          >
            <RefreshCw :size="14" :class="{ 'animate-spin': loadingModels }" />
          </button>
        </div>
        <datalist id="pg-model-list">
          <option v-for="m in models" :key="m" :value="m" />
        </datalist>
        <p class="text-xs text-wa-muted dark:text-wa-muted-dark mt-1">Kosongkan untuk pakai model dari pengaturan AI.</p>
      </div>

      <div>
        <div class="flex items-center justify-between">
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Temperature</label>
          <span class="text-xs font-mono text-wa-text dark:text-wa-text-dark">{{ session.temperature.toFixed(2) }}</span>
        </div>
        <input
          v-model.number="session.temperature"
          type="range"
          min="0"
          max="2"
          step="0.05"
          class="w-full mt-2 accent-wa-green"
        />
        <div class="flex justify-between text-[10px] text-wa-muted dark:text-wa-muted-dark mt-0.5">
          <span>presisi</span><span>kreatif</span>
        </div>
      </div>

      <div>
        <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">System prompt</label>
        <textarea
          v-model="session.system"
          rows="6"
          placeholder="Instruksi untuk asisten…"
          class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark resize-none leading-relaxed"
        />
      </div>
    </div>

    <!-- IMAGE PARAMETERS -->
    <div v-else-if="isImageTab" class="flex-1 overflow-y-auto p-4 space-y-5">
      <div v-if="!ai.config.enabled" class="text-xs bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 rounded-lg p-3">
        AI belum aktif. Buka pengaturan AI (ikon ✨) untuk mengatur provider dan API key.
      </div>

      <!-- Provider info -->
      <div class="p-3 rounded-lg bg-wa-panel dark:bg-wa-hover-dark border border-wa-border dark:border-wa-border-dark">
        <p class="text-xs font-medium text-wa-text dark:text-wa-text-dark">Provider</p>
        <p class="text-sm font-semibold text-wa-text dark:text-wa-text-dark mt-0.5">
          {{ ai.config.provider.toUpperCase() }}
          <span v-if="isGamAPI" class="ml-1 text-wa-green text-xs"> • Unlimited</span>
        </p>
      </div>

      <!-- GamAPI: Model selector -->
      <div v-if="isGamAPI && gamapiModels.length">
        <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Model</label>
        <select
          v-model="pg.imgModel"
          class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark font-mono"
        >
          <option v-for="m in gamapiModels" :key="m" :value="m">{{ m }}</option>
        </select>
      </div>

      <!-- GamAPI: Style selector -->
      <div v-if="isGamAPI && Object.keys(gamapiStyles).length">
        <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Style</label>
        <select
          v-model="pg.imgStyle"
          class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark"
        >
          <option v-for="[k, v] in Object.entries(gamapiStyles)" :key="k" :value="k">{{ v }}</option>
        </select>
      </div>

      <!-- GamAPI: Aspect Ratio selector -->
      <div v-if="isGamAPI && Object.keys(gamapiRatios).length">
        <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Aspect Ratio</label>
        <select
          v-model="pg.imgRatio"
          class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark"
        >
          <option v-for="[k, v] in Object.entries(gamapiRatios)" :key="k" :value="k">{{ v }}</option>
        </select>
      </div>

      <!-- OpenAI DALL-E: Size selector -->
      <div v-if="ai.config.provider === 'openai'">
        <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Size</label>
        <select
          v-model="pg.imgSize"
          class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark"
        >
          <option v-for="s in dallESizes" :key="s.id" :value="s.id">{{ s.label }}</option>
        </select>
      </div>

      <p class="text-xs text-wa-muted dark:text-wa-muted-dark pt-2">
        Ketik prompt di panel kiri lalu klik ✦ untuk generate.
      </p>
    </div>

    <!-- No session -->
    <div v-else class="flex-1 p-4 flex items-center justify-center text-sm text-wa-muted dark:text-wa-muted-dark">
      Belum ada sesi aktif.
    </div>
  </div>
</template>
