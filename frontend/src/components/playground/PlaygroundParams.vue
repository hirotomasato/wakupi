<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { PanelRightClose, RefreshCw, Settings2 } from '@lucide/vue'
import { usePlaygroundStore } from '../../stores/playground'
import { useUIStore } from '../../stores/ui'
import { useAIStore } from '../../stores/ai'

const pg = usePlaygroundStore()
const ui = useUIStore()
const ai = useAIStore()

const models = ref<string[]>([])
const loadingModels = ref(false)

const session = computed(() => pg.activeSession)
const inheritedModel = computed(() => ai.config.model || '(default provider)')

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

// Pull model list when panel opens.
watch(
  () => ui.pgRightCollapsed,
  (collapsed) => {
    if (!collapsed && models.value.length === 0 && ai.config.enabled) loadModels()
  },
  { immediate: true }
)
</script>

<template>
  <div class="h-full flex flex-col bg-wa-panel dark:bg-[#111b21] border-l border-wa-border dark:border-wa-border-dark">
    <header class="flex items-center justify-between px-4 py-3 border-b border-wa-border dark:border-wa-border-dark">
      <span class="text-sm font-semibold flex items-center gap-2 text-wa-text dark:text-wa-text-dark">
        <Settings2 :size="16" class="text-violet-500" />
        Parameter
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
    <div v-if="session" class="flex-1 overflow-y-auto p-4 space-y-5">
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

    <!-- No session -->
    <div v-else class="flex-1 p-4 flex items-center justify-center text-sm text-wa-muted dark:text-wa-muted-dark">
      Belum ada sesi aktif.
    </div>
  </div>
</template>
