<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { X, Sparkles, Save, Zap, CheckCircle2, XCircle, Loader2, RefreshCw } from '@lucide/vue'
import { useAIStore } from '../stores/ai'
import { useUIStore } from '../stores/ui'

const ai = useAIStore()
const ui = useUIStore()
const local = ref<any>({ ...ai.config })
const saving = ref(false)
const message = ref('')
const models = ref<string[]>([])
const loadingModels = ref(false)
const modelsError = ref('')

watch(() => ui.showAISettings, async (open) => {
  if (open) {
    if (!ai.loaded) await ai.load()
    local.value = { ...ai.config }
    message.value = ''
    models.value = []
    modelsError.value = ''
  }
})

const presets = [
  { id: 'openai', label: 'OpenAI', provider: 'openai', baseUrl: '', model: 'gpt-4o-mini' },
  { id: 'anthropic', label: 'Anthropic Claude', provider: 'anthropic', baseUrl: '', model: 'claude-haiku-4-5-20251001' },
  { id: 'gemini', label: 'Google Gemini', provider: 'gemini', baseUrl: '', model: 'gemini-1.5-flash' },
  { id: 'ollama', label: 'Ollama (Lokal)', provider: 'ollama', baseUrl: 'http://localhost:11434/api/chat', model: 'llama3.2' },
  { id: 'gamapi', label: 'GamAPI (Image Gen)', provider: 'gamapi', baseUrl: '', model: 'imagen-3-flash' },
  { id: 'lmstudio', label: 'LM Studio (Lokal)', provider: 'openai', baseUrl: 'http://localhost:1234/v1/chat/completions', model: '' },
  { id: 'localai', label: 'LocalAI (Lokal)', provider: 'openai', baseUrl: 'http://localhost:8080/v1/chat/completions', model: '' },
  { id: 'openrouter', label: 'OpenRouter', provider: 'openai', baseUrl: 'https://openrouter.ai/api/v1/chat/completions', model: 'anthropic/claude-3.5-haiku' },
  { id: 'groq', label: 'Groq', provider: 'openai', baseUrl: 'https://api.groq.com/openai/v1/chat/completions', model: 'llama-3.1-70b-versatile' },
  { id: 'together', label: 'Together AI', provider: 'openai', baseUrl: 'https://api.together.xyz/v1/chat/completions', model: 'meta-llama/Llama-3.3-70B-Instruct-Turbo' },
  { id: 'deepseek', label: 'DeepSeek', provider: 'openai', baseUrl: 'https://api.deepseek.com/v1/chat/completions', model: 'deepseek-chat' },
  { id: 'mistral', label: 'Mistral', provider: 'openai', baseUrl: 'https://api.mistral.ai/v1/chat/completions', model: 'mistral-small-latest' },
  { id: 'custom', label: 'Custom (OpenAI-compatible)', provider: 'openai', baseUrl: '', model: '' },
]

function pickPreset(p: typeof presets[number]) {
  local.value.provider = p.provider
  local.value.baseUrl = p.baseUrl
  local.value.model = p.model
}

const needsKey = computed(() => {
  return local.value.provider !== 'ollama' && !(local.value.baseUrl || '').startsWith('http://localhost')
})

async function save() {
  saving.value = true
  try {
    await ai.save(local.value)
    message.value = 'Pengaturan AI tersimpan'
    if (local.value.enabled) ai.testConnection(local.value)
    setTimeout(() => (message.value = ''), 1500)
  } catch (e: any) {
    message.value = 'Gagal: ' + (e?.message || e)
  } finally {
    saving.value = false
  }
}

async function testNow() {
  await ai.testConnection(local.value)
}

async function loadModels() {
  loadingModels.value = true
  modelsError.value = ''
  try {
    models.value = await ai.listModels(local.value)
    if (models.value.length === 0) modelsError.value = 'Tidak ada model ditemukan'
  } catch (e: any) {
    modelsError.value = 'Provider tidak mendukung daftar model — ketik manual'
    console.error('list models', e)
  } finally {
    loadingModels.value = false
  }
}
</script>

<template>
  <div v-if="ui.showAISettings" class="fixed inset-0 z-40 bg-black/40 flex items-center justify-center" @click.self="ui.showAISettings = false">
    <div class="w-[600px] max-w-[92vw] max-h-[88vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden flex flex-col">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <div class="flex items-center gap-2">
          <Sparkles :size="18" class="text-violet-500" />
          <h2 class="font-semibold">AI Assistant</h2>
        </div>
        <button @click="ui.showAISettings = false" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <div class="flex-1 overflow-y-auto p-5 space-y-5">
        <p class="text-sm text-wa-muted dark:text-wa-muted-dark">
          Aktifkan AI untuk Smart Reply, AI Compose, dan ringkas chat. Wakupi mendukung berbagai provider termasuk yang OpenAI-compatible.
        </p>

        <label class="flex items-center justify-between p-3 bg-wa-panel dark:bg-wa-hover-dark rounded-lg cursor-pointer">
          <span class="text-sm font-medium flex items-center gap-2">
            <Zap :size="16" class="text-amber-500" /> Aktifkan AI
          </span>
          <input v-model="local.enabled" type="checkbox" class="w-4 h-4 accent-wa-green" />
        </label>

        <div>
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide mb-2 block">Preset cepat</label>
          <div class="grid grid-cols-2 gap-2">
            <button
              v-for="p in presets"
              :key="p.id"
              @click="pickPreset(p)"
              class="text-left text-sm px-3 py-2 rounded-lg border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
              :class="{ 'border-wa-green bg-wa-green/5': local.baseUrl === p.baseUrl && local.provider === p.provider }"
            >
              <div class="font-medium">{{ p.label }}</div>
              <div class="text-xs text-wa-muted dark:text-wa-muted-dark truncate">{{ p.baseUrl || 'default' }}</div>
            </button>
          </div>
        </div>

        <div class="border-t border-wa-border dark:border-wa-border-dark pt-4 space-y-3">
          <div>
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Provider</label>
            <select v-model="local.provider" class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none text-wa-text dark:text-wa-text-dark">
              <option value="openai">OpenAI / OpenAI-compatible</option>
              <option value="anthropic">Anthropic Claude</option>
              <option value="gemini">Google Gemini</option>
              <option value="ollama">Ollama</option>
              <option value="gamapi">GamAPI (Image Generation)</option>
            </select>
            <p class="text-xs text-wa-muted dark:text-wa-muted-dark mt-1">
              Pilih "OpenAI" untuk OpenRouter, Groq, LM Studio, Together, dll. GamAPI khusus image generation.
            </p>
          </div>

          <div>
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Base URL</label>
            <input
              v-model="local.baseUrl"
              :placeholder="local.provider === 'gamapi' ? 'https://gamapi.proaccess.app/api' : 'https://api.openai.com/v1/chat/completions'"
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs"
            />
          </div>

          <div>
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Model</label>
            <div class="flex gap-2 mt-1">
              <input
                v-model="local.model"
                list="ai-model-list"
                placeholder="gpt-4o-mini"
                class="flex-1 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs"
              />
              <button
                @click="loadModels"
                :disabled="loadingModels"
                class="shrink-0 px-3 py-2 rounded-lg border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm flex items-center gap-1.5 disabled:opacity-50"
                title="Muat daftar model dari provider"
              >
                <Loader2 v-if="loadingModels" :size="14" class="animate-spin" />
                <RefreshCw v-else :size="14" />
                Muat model
              </button>
            </div>
            <datalist id="ai-model-list">
              <option v-for="m in models" :key="m" :value="m" />
            </datalist>
            <p v-if="models.length > 0" class="text-xs text-wa-green mt-1">{{ models.length }} model tersedia — ketik untuk memfilter</p>
            <p v-else-if="modelsError" class="text-xs text-amber-600 dark:text-amber-400 mt-1">{{ modelsError }}</p>
          </div>

          <div v-if="needsKey">
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">API Key</label>
            <input
              v-model="local.apiKey"
              type="password"
              placeholder="sk-..."
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none"
            />
          </div>
        </div>

        <div v-if="message" class="text-sm text-wa-green text-center">{{ message }}</div>

        <div
          v-if="ai.connStatus === 'ok' || ai.connStatus === 'error'"
          class="text-sm flex items-center gap-2 rounded-lg px-3 py-2"
          :class="ai.connStatus === 'ok'
            ? 'bg-wa-green/10 text-wa-green'
            : 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400'"
        >
          <CheckCircle2 v-if="ai.connStatus === 'ok'" :size="16" class="shrink-0" />
          <XCircle v-else :size="16" class="shrink-0 mt-0.5" />
          <span class="break-words min-w-0">{{ ai.connMessage }}</span>
        </div>

        <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3 text-xs text-blue-700 dark:text-blue-300 space-y-1">
          <p class="font-semibold">💡 Tips gratis:</p>
          <p>• Ollama: <code>ollama pull llama3.2</code> lalu pilih preset Ollama</p>
          <p>• LM Studio: download dari lmstudio.ai, load model, start server</p>
          <p>• OpenRouter: punya banyak model gratis (DeepSeek, Llama, Gemini)</p>
          <p>• GamAPI: image generation gratis — punya 14 model, unlimited quota</p>
        </div>
      </div>

      <footer class="border-t border-wa-border dark:border-wa-border-dark px-5 py-3 flex gap-2">
        <button
          @click="testNow"
          :disabled="ai.testing"
          class="flex-1 border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark py-2.5 rounded-lg font-medium flex items-center justify-center gap-2 disabled:opacity-50"
        >
          <Loader2 v-if="ai.testing" :size="16" class="animate-spin" />
          <CheckCircle2 v-else :size="16" />
          Cek Koneksi
        </button>
        <button @click="save" :disabled="saving" class="flex-1 bg-wa-green hover:bg-wa-green-dark text-white py-2.5 rounded-lg font-medium flex items-center justify-center gap-2 disabled:opacity-50">
          <Save :size="16" /> Simpan
        </button>
      </footer>
    </div>
  </div>
</template>

