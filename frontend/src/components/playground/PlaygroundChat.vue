<script setup lang="ts">
import { ref, nextTick, watch, computed, onMounted, onBeforeUnmount } from 'vue'
import {
  Send, Square, PanelLeftOpen, PanelRightOpen, Sparkles, User,
  RefreshCw, Copy, Trash2, Bot, MessageCircle, ArrowDown,
} from '@lucide/vue'
import { usePlaygroundStore } from '../../stores/playground'
import { useUIStore } from '../../stores/ui'
import { useAIStore } from '../../stores/ai'
import { useChatStore } from '../../stores/chat'
import { useSettingsStore } from '../../stores/settings'
import MarkdownContent from '../MarkdownContent.vue'
import { CopyToClipboard } from '../../../wailsjs/go/main/App'

const pg = usePlaygroundStore()
const ui = useUIStore()
const ai = useAIStore()
const chat = useChatStore()
const settings = useSettingsStore()

const input = ref('')
const scroller = ref<HTMLElement | null>(null)
const textarea = ref<HTMLTextAreaElement | null>(null)
const atBottom = ref(true)

const session = computed(() => pg.activeSession)
const canSend = computed(() => input.value.trim().length > 0 && !pg.streaming && ai.config.enabled)

const quickPrompts = [
  { label: 'Jelaskan kode', text: 'Jelaskan potongan kode berikut langkah demi langkah:\n\n```\n\n```' },
  { label: 'Draf email', text: 'Bantu saya menulis email yang sopan dan profesional tentang: ' },
  { label: 'Terjemahkan', text: 'Terjemahkan teks berikut ke Bahasa Inggris:\n\n' },
  { label: 'Ringkas', text: 'Ringkas poin-poin penting dari teks berikut:\n\n' },
]

// Consider the view "at bottom" when within 80px of the end.
function updateAtBottom() {
  const el = scroller.value
  if (!el) return
  atBottom.value = el.scrollHeight - el.scrollTop - el.clientHeight < 80
}

function scrollToBottom(force = false) {
  nextTick(() => {
    const el = scroller.value
    if (!el) return
    if (force || atBottom.value) {
      el.scrollTop = el.scrollHeight
      atBottom.value = true
    }
  })
}

// Auto-follow streaming output only when the user is already near the bottom.
watch(
  () => session.value?.messages.map((m) => m.content.length).join(','),
  () => scrollToBottom()
)
// Always jump to bottom when switching sessions.
watch(() => pg.activeId, () => scrollToBottom(true))

onMounted(() => {
  scroller.value?.addEventListener('scroll', updateAtBottom, { passive: true })
  scrollToBottom(true)
  applyPending()
})
onBeforeUnmount(() => scroller.value?.removeEventListener('scroll', updateAtBottom))

// Pick up a prompt staged by "Tanya AI" from a WhatsApp chat.
function applyPending() {
  if (pg.pendingInput) {
    input.value = pg.consumePendingInput()
    nextTick(() => {
      autosize()
      textarea.value?.focus()
    })
  }
}
watch(() => pg.pendingInput, (v) => { if (v) applyPending() })

function autosize() {
  const el = textarea.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = Math.min(el.scrollHeight, 200) + 'px'
}

async function submit() {
  if (!canSend.value) return
  const text = input.value
  input.value = ''
  nextTick(autosize)
  scrollToBottom(true)
  await pg.send(text)
}

function useQuickPrompt(text: string) {
  input.value = text
  nextTick(() => {
    autosize()
    textarea.value?.focus()
  })
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey && settings.enterToSend) {
    e.preventDefault()
    submit()
  }
}

function copyMessage(content: string) {
  CopyToClipboard(content).catch(() => {
    // Fallback ke JS API kalau binding gagal
    navigator.clipboard?.writeText(content)
  })
}

const hasAccounts = computed(() => chat.accounts.length > 0)

function sendToWhatsApp(content: string) {
  ui.sendToWhatsApp = content
}
</script>

<template>
  <div class="h-full flex flex-col bg-white dark:bg-wa-bg-dark relative">
    <!-- Top bar -->
    <header class="flex items-center gap-2 px-3 py-2.5 border-b border-wa-border dark:border-wa-border-dark">
      <button
        v-if="ui.pgLeftCollapsed"
        @click="ui.pgLeftCollapsed = false"
        class="p-2 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        title="Tampilkan daftar"
      >
        <PanelLeftOpen :size="18" />
      </button>

      <div class="flex items-center gap-2 flex-1 min-w-0">
        <div class="w-8 h-8 rounded-lg bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center text-white shrink-0">
          <Sparkles :size="16" />
        </div>
        <div class="min-w-0">
          <p class="text-sm font-semibold truncate text-wa-text dark:text-wa-text-dark">{{ session?.title || 'Playground' }}</p>
          <p class="text-xs text-wa-muted dark:text-wa-muted-dark truncate">
            {{ session?.model || ai.config.model || 'model default' }}
          </p>
        </div>
      </div>

      <button
        v-if="session && session.messages.length"
        @click="pg.clearActive()"
        class="p-2 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        title="Bersihkan percakapan"
      >
        <Trash2 :size="18" />
      </button>
      <button
        v-if="ui.pgRightCollapsed"
        @click="ui.pgRightCollapsed = false"
        class="p-2 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
        title="Tampilkan parameter"
      >
        <PanelRightOpen :size="18" />
      </button>
    </header>

    <!-- Messages -->
    <div ref="scroller" class="flex-1 overflow-y-auto">
      <div
        v-if="!session || session.messages.length === 0"
        class="h-full flex flex-col items-center justify-center text-center px-6 gap-4"
      >
        <div class="w-16 h-16 rounded-2xl bg-gradient-to-br from-violet-500 to-fuchsia-500 flex items-center justify-center text-white">
          <Bot :size="32" />
        </div>
        <div>
          <h2 class="text-lg font-semibold text-wa-text dark:text-wa-text-dark">AI Playground</h2>
          <p class="text-sm text-wa-muted dark:text-wa-muted-dark mt-1 max-w-sm">
            Mulai percakapan dengan model AI. Mendukung Markdown, blok kode dengan syntax highlight, dan respons streaming.
          </p>
        </div>
        <div v-if="ai.config.enabled" class="grid grid-cols-2 gap-2 w-full max-w-md mt-2">
          <button
            v-for="qp in quickPrompts"
            :key="qp.label"
            @click="useQuickPrompt(qp.text)"
            class="text-left text-sm px-3 py-2.5 rounded-xl border border-wa-border dark:border-wa-border-dark hover:border-wa-green hover:bg-wa-green/5 transition text-wa-text dark:text-wa-text-dark"
          >
            {{ qp.label }}
          </button>
        </div>
      </div>

      <div v-else class="max-w-3xl mx-auto px-4 py-6 space-y-6">
        <div
          v-for="(m, i) in session.messages"
          :key="m.id"
          class="flex gap-3"
          :class="m.role === 'user' ? 'flex-row-reverse' : ''"
        >
          <div
            class="w-8 h-8 rounded-lg shrink-0 flex items-center justify-center text-white"
            :class="m.role === 'user'
              ? 'bg-wa-green'
              : 'bg-gradient-to-br from-violet-500 to-fuchsia-500'"
          >
            <User v-if="m.role === 'user'" :size="16" />
            <Sparkles v-else :size="16" />
          </div>

          <div class="min-w-0 flex-1" :class="m.role === 'user' ? 'flex flex-col items-end' : ''">
            <div
              class="rounded-2xl px-4 py-2.5 max-w-full"
              :class="m.role === 'user'
                ? 'bg-wa-green text-white'
                : 'bg-wa-panel dark:bg-wa-panel-dark text-wa-text dark:text-wa-text-dark'"
            >
              <p v-if="m.role === 'user'" class="whitespace-pre-wrap break-words text-sm leading-relaxed">{{ m.content }}</p>
              <template v-else>
                <span
                  v-if="m.content === '' && pg.streaming"
                  class="inline-flex items-center gap-1 text-wa-muted dark:text-wa-muted-dark"
                >
                  <span class="typing-dot" /><span class="typing-dot" /><span class="typing-dot" />
                </span>
                <MarkdownContent v-else :text="m.content" />
              </template>
            </div>

            <div
              v-if="m.role === 'assistant' && m.content && !(pg.streaming && i === session.messages.length - 1)"
              class="flex items-center gap-1 mt-1 px-1"
            >
              <button
                @click="copyMessage(m.content)"
                class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
                title="Salin"
              >
                <Copy :size="14" />
              </button>
              <button
                v-if="hasAccounts"
                @click="sendToWhatsApp(m.content)"
                class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark hover:text-wa-green"
                title="Kirim ke WhatsApp"
              >
                <MessageCircle :size="14" />
              </button>
              <button
                v-if="i === session.messages.length - 1"
                @click="pg.regenerate()"
                class="p-1.5 rounded-lg text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark"
                title="Buat ulang"
              >
                <RefreshCw :size="14" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Composer -->
    <div class="border-t border-wa-border dark:border-wa-border-dark p-3 relative">
      <!-- Scroll to bottom -->
      <transition name="fade">
        <button
          v-if="!atBottom && session && session.messages.length"
          @click="scrollToBottom(true)"
          class="absolute -top-12 left-1/2 -translate-x-1/2 w-9 h-9 rounded-full bg-white dark:bg-wa-panel-dark border border-wa-border dark:border-wa-border-dark shadow-lg flex items-center justify-center text-wa-muted dark:text-wa-muted-dark hover:text-wa-green transition"
          title="Ke pesan terbaru"
        >
          <ArrowDown :size="18" />
        </button>
      </transition>

      <div class="max-w-3xl mx-auto">
        <div class="flex items-end gap-2 bg-wa-panel dark:bg-wa-panel-dark rounded-2xl px-3 py-2 border border-wa-border dark:border-wa-border-dark focus-within:border-wa-green transition">
          <textarea
            ref="textarea"
            v-model="input"
            @input="autosize"
            @keydown="onKeydown"
            rows="1"
            :placeholder="ai.config.enabled ? 'Tulis pesan…' : 'Aktifkan AI dulu di pengaturan ✨'"
            :disabled="!ai.config.enabled"
            class="flex-1 bg-transparent outline-none resize-none text-sm leading-relaxed py-1 max-h-[200px] text-wa-text dark:text-wa-text-dark disabled:opacity-50"
          />
          <button
            v-if="pg.streaming"
            @click="pg.cancel()"
            class="shrink-0 w-9 h-9 rounded-full bg-red-500 hover:bg-red-600 text-white flex items-center justify-center transition"
            title="Hentikan"
          >
            <Square :size="16" fill="currentColor" />
          </button>
          <button
            v-else
            @click="submit"
            :disabled="!canSend"
            class="shrink-0 w-9 h-9 rounded-full bg-wa-green hover:bg-wa-green-dark text-white flex items-center justify-center transition disabled:opacity-40 disabled:cursor-not-allowed"
            title="Kirim"
          >
            <Send :size="16" />
          </button>
        </div>
        <p class="text-[11px] text-wa-muted dark:text-wa-muted-dark text-center mt-1.5">
          {{ settings.enterToSend ? 'Enter untuk kirim · Shift+Enter baris baru' : 'Shift+Enter atau tombol kirim' }}
        </p>
      </div>
    </div>
  </div>
</template>

<style scoped>
.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  transform: translate(-50%, 6px);
}
</style>
