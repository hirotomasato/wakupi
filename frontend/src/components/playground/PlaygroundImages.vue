<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { Loader2, Image as ImageIcon, Download, MessageCircle, Sparkles } from '@lucide/vue'
import { usePlaygroundStore } from '../../stores/playground'
import { useAIStore, type ImageOptions } from '../../stores/ai'
import { useChatStore } from '../../stores/chat'
import { useUIStore } from '../../stores/ui'
import type { ImageResult } from '../../stores/ai'

const pg = usePlaygroundStore()
const ai = useAIStore()
const chat = useChatStore()
const ui = useUIStore()

const prompt = ref('')
const generating = ref(false)
const images = ref<ImageResult[]>([])
const error = ref('')

const isGamAPI = computed(() => ai.config.provider === 'gamapi')
const canGenerate = computed(() => prompt.value.trim().length > 0 && !generating.value && ai.config.enabled)

async function generate() {
  if (!canGenerate.value) return
  generating.value = true
  error.value = ''
  images.value = []
  try {
    const opts: ImageOptions = { model: pg.imgModel }
    if (isGamAPI.value) {
      opts.style = pg.imgStyle
      opts.aspectRatio = pg.imgRatio
    }
    if (ai.config.provider === 'openai') {
      opts.size = pg.imgSize
    }
    images.value = await ai.generateImage(prompt.value, opts)
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    generating.value = false
  }
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

const hasAccounts = computed(() => chat.accounts.length > 0)
</script>

<template>
  <div class="h-full flex flex-col bg-wa-bg dark:bg-[#0b141a] overflow-y-auto">
    <div class="flex-1 flex flex-col max-w-2xl mx-auto w-full px-4 py-6 gap-6">

      <!-- Empty state -->
      <div v-if="images.length === 0 && !generating" class="flex-1 flex flex-col items-center justify-center text-center gap-4 py-12">
        <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center text-white">
          <ImageIcon :size="32" />
        </div>
        <div>
          <h2 class="text-lg font-semibold text-wa-text dark:text-wa-text-dark">AI Image Generator</h2>
          <p class="text-sm text-wa-muted dark:text-wa-muted-dark mt-1 max-w-sm">
            Generate gambar AI dari teks — atur model & style di panel kanan →
          </p>
        </div>
        <div v-if="!ai.config.enabled" class="text-sm bg-amber-50 dark:bg-amber-900/20 text-amber-700 dark:text-amber-300 rounded-lg p-3 max-w-xs">
          AI belum aktif. Buka pengaturan AI (ikon ✨) untuk mengatur provider.
        </div>
      </div>

      <!-- Result grid -->
      <div v-if="images.length > 0" class="grid gap-3" :class="images.length > 1 ? 'grid-cols-2' : 'grid-cols-1'">
        <div
          v-for="(img, i) in images"
          :key="i"
          class="relative group rounded-xl overflow-hidden border border-wa-border dark:border-wa-border-dark bg-wa-panel dark:bg-wa-panel-dark"
        >
          <img
            :src="img.url"
            :alt="img.revisedPrompt || prompt"
            class="w-full h-auto object-cover"
            loading="lazy"
          />

          <!-- Hover actions -->
          <div class="absolute inset-0 bg-black/0 group-hover:bg-black/40 transition flex items-end justify-end p-2 opacity-0 group-hover:opacity-100 gap-1.5">
            <button
              @click="downloadImage(img.url, i)"
              class="w-8 h-8 rounded-full bg-white/90 text-gray-700 hover:bg-white flex items-center justify-center"
              title="Download"
            >
              <Download :size="15" />
            </button>
            <button
              v-if="hasAccounts"
              @click="sendToWA(img.url)"
              class="w-8 h-8 rounded-full bg-wa-green text-white hover:bg-wa-green-dark flex items-center justify-center"
              title="Kirim ke WhatsApp"
            >
              <MessageCircle :size="15" />
            </button>
          </div>

          <!-- Metadata -->
          <div class="px-3 py-2 flex items-center justify-between text-xs text-wa-muted dark:text-wa-muted-dark">
            <span v-if="img.model" class="font-mono">{{ img.model }}</span>
            <span v-if="img.width">{{ img.width }}×{{ img.height }}</span>
            <span v-if="img.revisedPrompt && img.revisedPrompt !== prompt" class="italic truncate ml-2">{{ img.revisedPrompt }}</span>
          </div>
        </div>
      </div>

      <!-- Error -->
      <div v-if="error" class="text-sm text-red-500 bg-red-50 dark:bg-red-900/20 rounded-lg px-4 py-3">
        {{ error }}
      </div>

      <!-- Loading -->
      <div v-if="generating" class="flex items-center justify-center gap-2 py-8 text-wa-muted dark:text-wa-muted-dark">
        <Loader2 :size="20" class="animate-spin text-wa-green" />
        <span class="text-sm">Generating image...</span>
      </div>
    </div>

    <!-- Composer — clean, just prompt + button -->
    <div class="border-t border-wa-border dark:border-wa-border-dark p-3 bg-wa-bg dark:bg-[#0b141a]">
      <div class="max-w-2xl mx-auto">
        <div class="flex items-end gap-2 bg-wa-panel dark:bg-wa-panel-dark rounded-2xl px-3 py-2 border border-wa-border dark:border-wa-border-dark focus-within:border-wa-green transition">
          <textarea
            v-model="prompt"
            rows="1"
            placeholder="Deskripsikan gambar yang ingin dibuat..."
            :disabled="!ai.config.enabled || generating"
            @keydown.enter.exact.prevent="generate()"
            class="flex-1 bg-transparent outline-none resize-none text-sm leading-relaxed py-1 max-h-[120px] text-wa-text dark:text-wa-text-dark disabled:opacity-50"
          />
          <button
            v-if="generating"
            disabled
            class="shrink-0 w-9 h-9 rounded-full bg-wa-muted dark:bg-wa-muted-dark text-white flex items-center justify-center"
          >
            <Loader2 :size="16" class="animate-spin" />
          </button>
          <button
            v-else
            @click="generate"
            :disabled="!canGenerate"
            class="shrink-0 w-9 h-9 rounded-full bg-gradient-to-br from-violet-500 to-fuchsia-500 hover:from-violet-600 hover:to-fuchsia-600 text-white flex items-center justify-center transition disabled:opacity-40 disabled:cursor-not-allowed"
            title="Generate"
          >
            <Sparkles :size="16" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
