<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import {
  Loader2, Image as ImageIcon, Download, MessageCircle, Sparkles,
  PanelRightClose, Settings2, RefreshCw, Zap, X, CheckCircle2, Save,
} from '@lucide/vue'
import { useImageGenStore } from '../stores/imagegen'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const ig = useImageGenStore()
const chat = useChatStore()
const ui = useUIStore()

const rightCollapsed = ref(false)
const showSettings = ref(false)

const hasAccounts = computed(() => chat.accounts.length > 0)

const dallESizes = [
  { id: '1024x1024', label: 'Square (1024×1024)' },
  { id: '1792x1024', label: 'Landscape (1792×1024)' },
  { id: '1024x1792', label: 'Portrait (1024×1792)' },
]

const presets = [
  { id: 'openai', label: 'OpenAI DALL-E', provider: 'openai', baseUrl: '', model: 'dall-e-3' },
  { id: 'gemini', label: 'Google Gemini Imagen', provider: 'gemini', baseUrl: '', model: 'imagen-3-flash' },
  { id: 'gamapi', label: 'GamAPI (Gratis)', provider: 'gamapi', baseUrl: '', model: 'imagen-3-flash' },
  { id: 'custom', label: 'Custom OpenAI-compat', provider: 'openai', baseUrl: '', model: '' },
]

const needsKey = computed(() => {
  return ig.config.provider !== 'ollama'
})

// Local copy for editing
const localCfg = ref<any>({})

onMounted(async () => {
  await ig.loadConfig()
  localCfg.value = { ...ig.config }
  if (ig.config.enabled) ig.testConnection(ig.config).catch(() => {})
  if (ig.isGamAPI) ig.loadGamAPIData()
})

function pickPreset(p: typeof presets[number]) {
  localCfg.value.provider = p.provider
  localCfg.value.baseUrl = p.baseUrl
  localCfg.value.model = p.model
}

async function saveSettingsCfg() {
  try {
    await ig.saveConfig(localCfg.value)
    if (localCfg.value.enabled) ig.testConnection(localCfg.value)
  } catch {}
}

async function testNow() {
  await ig.testConnection(localCfg.value)
}

async function loadModels() {
  // For image gen we just rely on presets — no model listing needed
}

function sendToWA(imageUrl: string) {
  ui.sendToWhatsApp = imageUrl
}

function downloadImage(url: string, idx: number) {
  const a = document.createElement('a')
  a.href = url
  a.download = `wakupi-image-${idx + 1}.png`
  a.click()
}
</script>

<template>
  <div class="flex-1 flex h-full min-w-0">
    <!-- ===== Main content ===== -->
    <div class="flex-1 min-w-0 h-full flex flex-col bg-wa-bg dark:bg-[#0b141a]">
      <!-- Header -->
      <header class="h-14 px-4 flex items-center justify-between bg-wa-panel dark:bg-wa-panel-dark border-b border-wa-border dark:border-wa-border-dark shrink-0">
        <div class="flex items-center gap-2">
          <ImageIcon :size="20" class="text-fuchsia-500" />
          <h1 class="font-semibold text-wa-text dark:text-wa-text-dark">AI Image Generator</h1>
        </div>
        <div class="flex items-center gap-2">
          <!-- Connection status dot -->
          <span
            class="w-2.5 h-2.5 rounded-full"
            :class="{
              'bg-emerald-500': ig.connStatus === 'ok',
              'bg-red-500': ig.connStatus === 'error',
              'bg-amber-400': ig.connStatus === 'unknown',
              'bg-gray-400': ig.connStatus === 'off',
            }"
            :title="ig.connMessage || (ig.config.enabled ? 'Checking...' : 'Disabled')"
          />
          <button
            @click="showSettings = !showSettings"
            class="p-2 rounded-lg hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-wa-muted dark:text-wa-muted-dark transition"
            :class="{ 'bg-wa-hover dark:bg-wa-hover-dark': showSettings }"
            title="Pengaturan Image Generator"
          >
            <Settings2 :size="18" />
          </button>
        </div>
      </header>

      <div class="flex-1 overflow-y-auto">
        <!-- ===== Settings panel (inline, collapsible) ===== -->
        <div v-if="showSettings" class="border-b border-wa-border dark:border-wa-border-dark bg-wa-panel dark:bg-[#111b21]">
          <div class="p-4 space-y-4 max-w-xl mx-auto">
            <div class="flex items-center justify-between">
              <h3 class="text-sm font-semibold text-wa-text dark:text-wa-text-dark">Pengaturan Provider</h3>
              <button @click="showSettings = false" class="text-wa-muted dark:text-wa-muted-dark hover:text-wa-text dark:hover:text-wa-text-dark">
                <X :size="16" />
              </button>
            </div>

            <label class="flex items-center justify-between p-3 bg-wa-panel dark:bg-wa-hover-dark rounded-lg cursor-pointer">
              <span class="text-sm font-medium flex items-center gap-2">
                <Zap :size="16" class="text-amber-500" /> Aktifkan Image Generator
              </span>
              <input v-model="localCfg.enabled" type="checkbox" class="w-4 h-4 accent-wa-green" />
            </label>

            <div>
              <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide mb-2 block">Preset</label>
              <div class="grid grid-cols-2 gap-2">
                <button
                  v-for="p in presets"
                  :key="p.id"
                  @click="pickPreset(p)"
                  class="text-left text-sm px-3 py-2 rounded-lg border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition"
                  :class="{ 'border-fuchsia-500 bg-fuchsia-500/5': localCfg.baseUrl === p.baseUrl && localCfg.provider === p.provider }"
                >
                  <div class="font-medium">{{ p.label }}</div>
                  <div class="text-xs text-wa-muted dark:text-wa-muted-dark truncate">{{ p.baseUrl || 'default' }}</div>
                </button>
              </div>
            </div>

            <div class="grid grid-cols-2 gap-3">
              <div class="col-span-2">
                <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Provider</label>
                <select v-model="localCfg.provider" class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none text-wa-text dark:text-wa-text-dark">
                  <option value="openai">OpenAI / DALL-E</option>
                  <option value="gemini">Google Gemini / Imagen</option>
                  <option value="gamapi">GamAPI (Image Generation)</option>
                  <option value="ollama">Ollama</option>
                </select>
              </div>
              <div class="col-span-2">
                <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Base URL</label>
                <input v-model="localCfg.baseUrl" placeholder="https://api.openai.com/v1" class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs" />
              </div>
              <div>
                <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Model</label>
                <input v-model="localCfg.model" placeholder="dall-e-3" class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs" />
              </div>
              <div v-if="needsKey">
                <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">API Key</label>
                <input v-model="localCfg.apiKey" type="password" placeholder="sk-..." class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none" />
              </div>
            </div>

            <!-- Status message -->
            <div v-if="ig.connStatus === 'ok' || ig.connStatus === 'error'" class="text-sm flex items-center gap-2 rounded-lg px-3 py-2"
              :class="ig.connStatus === 'ok' ? 'bg-wa-green/10 text-wa-green' : 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400'">
              <CheckCircle2 v-if="ig.connStatus === 'ok'" :size="16" class="shrink-0" />
              <X v-else :size="16" class="shrink-0" />
              <span class="break-words min-w-0">{{ ig.connMessage }}</span>
            </div>

            <div class="flex gap-2">
              <button @click="testNow" :disabled="ig.testing" class="flex-1 border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark py-2 rounded-lg font-medium text-sm flex items-center justify-center gap-2 disabled:opacity-50">
                <Loader2 v-if="ig.testing" :size="14" class="animate-spin" />
                <CheckCircle2 v-else :size="14" />
                Cek Koneksi
              </button>
              <button @click="saveSettingsCfg" class="flex-1 bg-fuchsia-500 hover:bg-fuchsia-600 text-white py-2 rounded-lg font-medium text-sm flex items-center justify-center gap-2">
                <Save :size="14" /> Simpan
              </button>
            </div>
          </div>
        </div>

        <div class="max-w-2xl mx-auto w-full px-4 py-6 gap-6 flex flex-col">
          <!-- Empty state -->
          <div v-if="ig.images.length === 0 && !ig.generating" class="flex-1 flex flex-col items-center justify-center text-center gap-4 py-20">
            <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center text-white">
              <ImageIcon :size="32" />
            </div>
            <div>
              <h2 class="text-lg font-semibold text-wa-text dark:text-wa-text-dark">AI Image Generator</h2>
              <p class="text-sm text-wa-muted dark:text-wa-muted-dark mt-1 max-w-sm">
                Tulis prompt di bawah lalu klik ✦ untuk generate gambar dengan AI.
              </p>
            </div>
            <div v-if="!ig.config.enabled" class="text-sm bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 rounded-lg p-3 max-w-xs">
              Image Generator belum aktif. Klik ⚙ untuk mengatur provider.
              <button @click="showSettings = true" class="block mt-2 text-fuchsia-600 dark:text-fuchsia-400 font-medium underline">Buka pengaturan</button>
            </div>
          </div>

          <!-- Result grid -->
          <div v-if="ig.images.length > 0" class="grid gap-3" :class="ig.images.length > 1 ? 'grid-cols-2' : 'grid-cols-1'">
            <div v-for="(img, i) in ig.images" :key="i" class="relative group rounded-xl overflow-hidden border border-wa-border dark:border-wa-border-dark bg-wa-panel dark:bg-wa-panel-dark">
              <img :src="img.url" :alt="img.revisedPrompt || ig.prompt" class="w-full h-auto object-cover" loading="lazy" />
              <div class="absolute inset-0 bg-black/0 group-hover:bg-black/40 transition flex items-end justify-end p-2 opacity-0 group-hover:opacity-100 gap-1.5">
                <button @click="downloadImage(img.url, i)" class="w-8 h-8 rounded-full bg-white/90 text-gray-700 hover:bg-white flex items-center justify-center" title="Download">
                  <Download :size="15" />
                </button>
                <button v-if="hasAccounts" @click="sendToWA(img.url)" class="w-8 h-8 rounded-full bg-wa-green text-white hover:bg-wa-green-dark flex items-center justify-center" title="Kirim ke WhatsApp">
                  <MessageCircle :size="15" />
                </button>
              </div>
              <div class="px-3 py-2 flex items-center justify-between text-xs text-wa-muted dark:text-wa-muted-dark">
                <span v-if="img.model" class="font-mono">{{ img.model }}</span>
                <span v-if="img.width">{{ img.width }}×{{ img.height }}</span>
                <span v-if="img.revisedPrompt && img.revisedPrompt !== ig.prompt" class="italic truncate ml-2">{{ img.revisedPrompt }}</span>
              </div>
            </div>
          </div>

          <!-- Error -->
          <div v-if="ig.error" class="text-sm text-red-500 bg-red-50 dark:bg-red-900/20 rounded-lg px-4 py-3">{{ ig.error }}</div>

          <!-- Loading -->
          <div v-if="ig.generating" class="flex items-center justify-center gap-2 py-8 text-wa-muted dark:text-wa-muted-dark">
            <Loader2 :size="20" class="animate-spin text-wa-green" />
            <span class="text-sm">Generating image...</span>
          </div>
        </div>
      </div>

      <!-- Composer -->
      <div class="border-t border-wa-border dark:border-wa-border-dark p-3 bg-wa-bg dark:bg-[#0b141a]">
        <div class="max-w-2xl mx-auto">
          <div class="flex items-end gap-2 bg-wa-panel dark:bg-wa-panel-dark rounded-2xl px-3 py-2 border border-wa-border dark:border-wa-border-dark focus-within:border-wa-green transition">
            <textarea
              v-model="ig.prompt"
              rows="1"
              placeholder="Deskripsikan gambar yang ingin dibuat..."
              :disabled="!ig.config.enabled || ig.generating"
              @keydown.enter.exact.prevent="ig.generate()"
              class="flex-1 bg-transparent outline-none resize-none text-sm leading-relaxed py-1 max-h-[120px] text-wa-text dark:text-wa-text-dark disabled:opacity-50"
            />
            <button v-if="ig.generating" disabled class="shrink-0 w-9 h-9 rounded-full bg-wa-muted dark:bg-wa-muted-dark text-white flex items-center justify-center">
              <Loader2 :size="16" class="animate-spin" />
            </button>
            <button
              v-else
              @click="ig.generate()"
              :disabled="!ig.canGenerate"
              class="shrink-0 w-9 h-9 rounded-full bg-gradient-to-br from-violet-500 to-fuchsia-500 hover:from-violet-600 hover:to-fuchsia-600 text-white flex items-center justify-center transition disabled:opacity-40 disabled:cursor-not-allowed"
              title="Generate"
            >
              <Sparkles :size="16" />
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- ===== Right params panel ===== -->
    <div v-show="!rightCollapsed" class="w-[280px] shrink-0 h-full flex flex-col bg-wa-panel dark:bg-[#111b21] border-l border-wa-border dark:border-wa-border-dark">
      <header class="flex items-center justify-between px-4 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <span class="text-sm font-semibold flex items-center gap-2 text-wa-text dark:text-wa-text-dark">
          <Settings2 :size="16" class="text-fuchsia-500" />
          Image Params
        </span>
        <button @click="rightCollapsed = true" class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark" title="Sembunyikan panel">
          <PanelRightClose :size="16" />
        </button>
      </header>

      <div class="flex-1 overflow-y-auto p-4 space-y-5">
        <div v-if="!ig.config.enabled" class="text-xs bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 rounded-lg p-3">
          Image Generator belum aktif.
        </div>

        <div class="p-3 rounded-lg bg-wa-panel dark:bg-wa-hover-dark border border-wa-border dark:border-wa-border-dark">
          <p class="text-xs font-medium text-wa-text dark:text-wa-text-dark">Provider</p>
          <p class="text-sm font-semibold text-wa-text dark:text-wa-text-dark mt-0.5">
            {{ ig.config.provider.toUpperCase() }}
            <span v-if="ig.isGamAPI" class="ml-1 text-wa-green text-xs"> • Unlimited</span>
          </p>
        </div>

        <div v-if="ig.isGamAPI && ig.gamapiModels.length">
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Model</label>
          <select v-model="ig.imgModel" class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark font-mono">
            <option v-for="m in ig.gamapiModels" :key="m" :value="m">{{ m }}</option>
          </select>
        </div>

        <div v-if="ig.isGamAPI && Object.keys(ig.gamapiStyles).length">
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Style</label>
          <select v-model="ig.imgStyle" class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark">
            <option v-for="[k, v] in Object.entries(ig.gamapiStyles)" :key="k" :value="k">{{ v }}</option>
          </select>
        </div>

        <div v-if="ig.isGamAPI && Object.keys(ig.gamapiRatios).length">
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Aspect Ratio</label>
          <select v-model="ig.imgRatio" class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark">
            <option v-for="[k, v] in Object.entries(ig.gamapiRatios)" :key="k" :value="k">{{ v }}</option>
          </select>
        </div>

        <div v-if="ig.config.provider === 'openai'">
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Size</label>
          <select v-model="ig.imgSize" class="mt-1.5 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none border border-wa-border dark:border-wa-border-dark text-wa-text dark:text-wa-text-dark">
            <option v-for="s in dallESizes" :key="s.id" :value="s.id">{{ s.label }}</option>
          </select>
        </div>

        <p class="text-xs text-wa-muted dark:text-wa-muted-dark pt-2">
          Ketik prompt lalu klik ✦ untuk generate.
        </p>
      </div>
    </div>

    <!-- Collapse toggle button -->
    <button v-if="rightCollapsed" @click="rightCollapsed = false" class="shrink-0 w-8 border-l border-wa-border dark:border-wa-border-dark bg-wa-panel dark:bg-[#111b21] flex items-center justify-center text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark">
      <Settings2 :size="16" />
    </button>
  </div>
</template>
