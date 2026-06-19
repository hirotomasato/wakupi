<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { X, Headset, Save, CheckCircle2, Loader2, MessageCircle } from '@lucide/vue'
import { useCSBotStore, defaults } from '../stores/csbot'
import { useUIStore } from '../stores/ui'

const cs = useCSBotStore()
const ui = useUIStore()
const local = ref<any>({ ...cs.config })
const saving = ref(false)
const message = ref('')

watch(() => ui.showCSBotSettings, async (open) => {
  if (open) {
    if (!cs.loaded) await cs.load()
    local.value = { ...cs.config, systemPrompt: cs.config.systemPrompt || '' }
    message.value = ''
  }
})

const presets = [
  { id: 'openai', label: 'OpenAI', provider: 'openai', baseUrl: '', model: 'gpt-4o-mini' },
  { id: 'anthropic', label: 'Anthropic Claude', provider: 'anthropic', baseUrl: '', model: 'claude-haiku-4-5-20251001' },
  { id: 'gemini', label: 'Google Gemini', provider: 'gemini', baseUrl: '', model: 'gemini-1.5-flash' },
  { id: 'ollama', label: 'Ollama (Lokal)', provider: 'ollama', baseUrl: 'http://localhost:11434/api/chat', model: 'llama3.2' },
  { id: 'openrouter', label: 'OpenRouter', provider: 'openai', baseUrl: 'https://openrouter.ai/api/v1/chat/completions', model: 'anthropic/claude-3.5-haiku' },
  { id: 'deepseek', label: 'DeepSeek', provider: 'openai', baseUrl: 'https://api.deepseek.com/v1/chat/completions', model: 'deepseek-chat' },
  { id: 'groq', label: 'Groq', provider: 'openai', baseUrl: 'https://api.groq.com/openai/v1/chat/completions', model: 'llama-3.1-70b-versatile' },
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
    await cs.save(local.value)
    message.value = 'Pengaturan CS Bot tersimpan'
    if (local.value.enabled) {
      const ok = await cs.testConnection(local.value)
      if (!ok) message.value = 'Tersimpan, tapi koneksi gagal: ' + cs.connMessage
    }
    setTimeout(() => (message.value = ''), 1500)
  } catch (e: any) {
    message.value = 'Gagal: ' + (e?.message || e)
  } finally {
    saving.value = false
  }
}

async function testNow() {
  await cs.testConnection(local.value)
}
</script>

<template>
  <div v-if="ui.showCSBotSettings" class="fixed inset-0 z-40 bg-black/40 flex items-center justify-center" @click.self="ui.showCSBotSettings = false">
    <div class="w-[600px] max-w-[92vw] max-h-[88vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden flex flex-col">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <div class="flex items-center gap-2">
          <Headset :size="18" class="text-blue-500" />
          <h2 class="font-semibold">CS Bot — Auto Reply</h2>
        </div>
        <button @click="ui.showCSBotSettings = false" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <div class="flex-1 overflow-y-auto p-5 space-y-5">
        <p class="text-sm text-wa-muted dark:text-wa-muted-dark">
          Aktifkan CS Bot untuk auto-reply otomatis ke setiap pesan masuk. CS Bot pakai AI provider dan API key terpisah dari AI personal (Playground).
        </p>

        <label class="flex items-center justify-between p-3 bg-wa-panel dark:bg-wa-hover-dark rounded-lg cursor-pointer">
          <span class="text-sm font-medium flex items-center gap-2">
            <MessageCircle :size="16" class="text-blue-500" /> Aktifkan CS Bot
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
              class="text-left text-sm px-3 py-2 rounded-lg border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition"
              :class="{ 'border-blue-500 bg-blue-500/5 dark:bg-blue-500/10': local.baseUrl === p.baseUrl && local.provider === p.provider }"
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
            </select>
          </div>

          <div>
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Base URL</label>
            <input
              v-model="local.baseUrl"
              placeholder="https://api.openai.com/v1/chat/completions"
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs"
            />
          </div>

          <div>
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Model</label>
            <input
              v-model="local.model"
              placeholder="gpt-4o-mini"
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none font-mono text-xs"
            />
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

        <!-- System Prompt -->
        <div class="border-t border-wa-border dark:border-wa-border-dark pt-4">
          <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide mb-2 block">System Prompt CS</label>
          <p class="text-xs text-wa-muted dark:text-wa-muted-dark mb-2">
            Petunjuk untuk AI bagaimana bersikap sebagai CS. Kosongkan untuk menggunakan default.
          </p>
          <textarea
            v-model="local.systemPrompt"
            rows="6"
            placeholder="Kamu adalah customer service yang ramah dan profesional..."
            class="w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none resize-y min-h-[120px] font-mono text-xs leading-relaxed"
          ></textarea>
        </div>

        <!-- Greeting -->
        <div class="border-t border-wa-border dark:border-wa-border-dark pt-4 space-y-3">
          <h3 class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark uppercase tracking-wide">Pesan Sambutan</h3>

          <label class="flex items-center justify-between p-3 bg-wa-panel dark:bg-wa-hover-dark rounded-lg cursor-pointer">
            <span class="text-sm font-medium">Kirim sambutan ke kontak baru</span>
            <input v-model="local.useGreeting" type="checkbox" class="w-4 h-4 accent-wa-green" />
          </label>

          <div v-if="local.useGreeting">
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Teks sambutan (dikirim sekali)</label>
            <textarea
              v-model="local.greetingMsg"
              rows="3"
              placeholder="Halo! Selamat datang. Ada yang bisa saya bantu?"
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none resize-none font-mono text-xs"
            ></textarea>
          </div>
        </div>

        <div v-if="message" class="text-sm text-wa-green text-center">{{ message }}</div>

        <div
          v-if="cs.connStatus === 'ok' || cs.connStatus === 'error'"
          class="text-sm flex items-center gap-2 rounded-lg px-3 py-2"
          :class="cs.connStatus === 'ok'
            ? 'bg-wa-green/10 text-wa-green'
            : 'bg-red-50 dark:bg-red-900/20 text-red-600 dark:text-red-400'"
        >
          <CheckCircle2 v-if="cs.connStatus === 'ok'" :size="16" class="shrink-0" />
          <X v-else :size="16" class="shrink-0 mt-0.5" />
          <span class="break-words min-w-0">{{ cs.connMessage }}</span>
        </div>

        <div class="bg-blue-50 dark:bg-blue-900/20 border border-blue-200 dark:border-blue-800 rounded-lg p-3 text-xs text-blue-700 dark:text-blue-300 space-y-1">
          <p class="font-semibold">💡 Catatan:</p>
          <p>• CS Bot pakai provider sendiri — tidak numpang sama AI personal (Playground)</p>
          <p>• Auto-reply aktif untuk SEMUA kontak (global)</p>
          <p>• Matikan CS Bot untuk menghentikan auto-reply</p>
          <p>• Greeting hanya dikirim SEKALI per kontak</p>
        </div>
      </div>

      <footer class="border-t border-wa-border dark:border-wa-border-dark px-5 py-3 flex gap-2">
        <button
          @click="testNow"
          :disabled="cs.testing"
          class="flex-1 border border-wa-border dark:border-wa-border-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark py-2.5 rounded-lg font-medium flex items-center justify-center gap-2 disabled:opacity-50"
        >
          <Loader2 v-if="cs.testing" :size="16" class="animate-spin" />
          <CheckCircle2 v-else :size="16" />
          Cek Koneksi
        </button>
        <button @click="save" :disabled="saving" class="flex-1 bg-blue-500 hover:bg-blue-600 text-white py-2.5 rounded-lg font-medium flex items-center justify-center gap-2 disabled:opacity-50">
          <Save :size="16" /> Simpan
        </button>
      </footer>
    </div>
  </div>
</template>
