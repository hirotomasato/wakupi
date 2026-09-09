<script setup lang="ts">
import { ref, computed, nextTick, watch } from 'vue'
import {
  MoreVertical,
  Smile,
  Paperclip,
  Mic,
  Send,
  Users,
  X,
  Image as ImageIcon,
  Video,
  FileText,
  Music,
  Sparkles,
  Info,
  PanelLeftOpen,
  TrendingUp,
  Megaphone,
} from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'
import { useAIStore } from '../stores/ai'
import { usePlaygroundStore } from '../stores/playground'
import MessageBubble from './MessageBubble.vue'
import VoiceRecorder from './VoiceRecorder.vue'
import EmojiPicker from './EmojiPicker.vue'
import QuickInvoice from './QuickInvoice.vue'

const store = useChatStore()
const ui = useUIStore()
const ai = useAIStore()
const pg = usePlaygroundStore()
const draft = ref('')
const scroller = ref<HTMLElement | null>(null)
const showAttach = ref(false)
const recording = ref(false)
const showHeaderMenu = ref(false)
const composing = ref(false)
const showEmoji = ref(false)
const showQrisGenerator = ref(false)

async function handleQrisSendToChat(amount: number, qrDataUrl: string) {
  const caption = `💳 Invoice QRIS - Rp ${new Intl.NumberFormat('id-ID').format(amount)}`
  await store.sendImageBlob(qrDataUrl, caption)
  showQrisGenerator.value = false
}

function pickEmoji(e: string) {
  draft.value += e
}

const initials = (name: string) =>
  name
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()

const messages = computed(() => store.activeMessages)
const chat = computed(() => store.activeChat)

// Build a flat render list: date separators + messages with grouping flags,
// so consecutive messages from the same sender hug together (only the first
// in a run gets the bubble tail and the group sender name), just like WhatsApp.
type Row =
  | { kind: 'date'; id: string; label: string }
  | { kind: 'msg'; id: string; msg: any; showTail: boolean; showSender: boolean }

function dayKey(ts?: number): string {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  return `${d.getFullYear()}-${d.getMonth()}-${d.getDate()}`
}

function dateLabel(ts?: number): string {
  if (!ts) return ''
  const d = new Date(ts * 1000)
  const today = new Date()
  const yesterday = new Date()
  yesterday.setDate(yesterday.getDate() - 1)
  if (d.toDateString() === today.toDateString()) return 'Hari ini'
  if (d.toDateString() === yesterday.toDateString()) return 'Kemarin'
  return d.toLocaleDateString('id-ID', { weekday: 'long', day: 'numeric', month: 'long', year: 'numeric' })
}

const GROUP_GAP = 120 // seconds: break a run if messages are >2min apart

const rows = computed<Row[]>(() => {
  const out: Row[] = []
  const list = messages.value
  let prevDay = ''
  for (let i = 0; i < list.length; i++) {
    const m = list[i]
    const dk = dayKey(m._ts)
    if (dk !== prevDay) {
      out.push({ kind: 'date', id: 'date-' + dk, label: dateLabel(m._ts) })
      prevDay = dk
    }
    const prev = list[i - 1]
    const sameRunAsPrev =
      !!prev &&
      prev.fromMe === m.fromMe &&
      (prev._senderJID || '') === (m._senderJID || '') &&
      dayKey(prev._ts) === dk &&
      (m._ts || 0) - (prev._ts || 0) <= GROUP_GAP &&
      !prev.deleted
    out.push({
      kind: 'msg',
      id: m.id,
      msg: m,
      showTail: !sameRunAsPrev,
      showSender: !sameRunAsPrev,
    })
  }
  return out
})

const presenceText = computed(() => {
  if (!chat.value) return ''
  const key = `${chat.value.accountId}::${chat.value.jid}`
  const cp = store.chatPresence[key]
  if (cp?.state === 'composing') {
    return cp.media === 'audio' ? 'sedang merekam audio...' : 'sedang mengetik...'
  }
  const p = store.presence[key]
  if (p?.online) return 'online'
  if (p?.lastSeen) {
    const d = new Date(p.lastSeen * 1000)
    return 'terakhir dilihat ' + d.toLocaleString('id-ID', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' })
  }
  return ''
})

const isTyping = computed(() => presenceText.value.startsWith('sedang'))

watch(
  () => messages.value.length,
  async () => {
    await nextTick()
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
  }
)
watch(
  () => store.activeChatId,
  async () => {
    await nextTick()
    if (scroller.value) scroller.value.scrollTop = scroller.value.scrollHeight
    ai.clearSuggestions()
    triggerSuggest()
  }
)

watch(draft, (val, old) => {
  if (val && !old) store.setTyping(true)
  else if (!val && old) store.setTyping(false)
})

let suggestTimer: any = null
function triggerSuggest() {
  if (!ai.config.enabled) return
  if (suggestTimer) clearTimeout(suggestTimer)
  suggestTimer = setTimeout(async () => {
    if (!chat.value || messages.value.length === 0) return
    const lastMsg = messages.value[messages.value.length - 1]
    if (lastMsg.fromMe) return
    const tail = messages.value.slice(-6).map((m) => `${m.fromMe ? 'Me' : (m.sender || chat.value!.name)}: ${m.text || m.caption || '[media]'}`).join('\n')
    await ai.suggest(chat.value.name, tail)
  }, 600)
}

watch(() => messages.value.length, () => triggerSuggest())

function send() {
  if (!draft.value.trim()) return
  store.sendMessage(draft.value)
  draft.value = ''
  store.setTyping(false)
  ai.clearSuggestions()
}

function pick(kind: 'image' | 'video' | 'audio' | 'any') {
  showAttach.value = false
  store.attachFile(kind)
}

function clearReply() {
  store.setReply(null)
}

function useSuggestion(s: string) {
  draft.value = s
  ai.clearSuggestions()
}

async function aiCompose() {
  const prompt = window.prompt('Tulis topik pesan untuk AI:')
  if (!prompt) return
  composing.value = true
  try {
    const text = await ai.compose(prompt, 'friendly')
    if (text) draft.value = text
  } catch (e: any) {
    alert('Gagal: ' + (e?.message || e))
  } finally {
    composing.value = false
  }
}

// Build a plain-text excerpt of the recent conversation for AI context.
function buildContext(limit = 30): string {
  return messages.value
    .slice(-limit)
    .map((m) => `${m.fromMe ? 'Saya' : (m.sender || chat.value!.name)}: ${m.text || m.caption || '[media]'}`)
    .join('\n')
}

function askInPlayground(instruction: string, titlePrefix: string, autoSend: boolean) {
  if (!chat.value || messages.value.length === 0) return
  showHeaderMenu.value = false
  pg.openWithContext({
    title: `${titlePrefix}: ${chat.value.name}`,
    context: buildContext(),
    instruction,
    autoSend,
  })
  ui.showPlayground = true
}

function summarizeInPlayground() {
  askInPlayground('Ringkas percakapan berikut menjadi poin-poin penting.', 'Ringkasan', true)
}

function draftReplyInPlayground() {
  askInPlayground(
    'Berdasarkan percakapan berikut, buatkan draf balasan yang sesuai untuk saya kirim. Berikan 1 opsi balasan yang natural.',
    'Draf balasan',
    true
  )
}

function askAboutChat() {
  // Stage the context but let the user type their own question.
  askInPlayground('', 'Tanya AI', false)
}

function openInfo() {
  showHeaderMenu.value = false
  if (chat.value?.isGroup) ui.showGroupInfo = true
}
</script>

<template>
  <section class="flex-1 flex flex-col chat-bg-pattern relative">
    <template v-if="chat">
      <header
        class="h-14 px-4 flex items-center justify-between bg-wa-panel dark:bg-wa-panel-dark border-l border-wa-border dark:border-wa-border-dark cursor-pointer"
        @click="chat.isGroup && (ui.showGroupInfo = true)"
      >
        <div class="flex items-center gap-3">
          <button
            v-if="ui.waListCollapsed"
            @click.stop="ui.waListCollapsed = false"
            class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark shrink-0"
            title="Tampilkan daftar chat"
          >
            <PanelLeftOpen :size="18" />
          </button>
          <div class="w-10 h-10 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold overflow-hidden">
            <img v-if="chat.avatarUrl" :src="chat.avatarUrl" class="w-full h-full object-cover" />
            <Users v-else-if="chat.isGroup" :size="18" />
            <Megaphone v-else-if="chat.isChannel" :size="18" />
            <span v-else>{{ initials(chat.name) }}</span>
          </div>
          <div>
            <div class="font-medium text-wa-text dark:text-wa-text-dark">{{ chat.name }}</div>
            <div
              class="text-xs"
              :class="isTyping ? 'text-wa-green' : 'text-wa-muted dark:text-wa-muted-dark'"
            >
              {{ presenceText || ' ' }}
            </div>
          </div>
        </div>
        <div class="flex items-center gap-1 text-wa-muted dark:text-wa-muted-dark" @click.stop>
          <div class="relative">
            <button @click="showHeaderMenu = !showHeaderMenu" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center">
              <MoreVertical :size="18" />
            </button>
            <div v-if="showHeaderMenu" class="absolute right-0 top-10 w-52 bg-white dark:bg-wa-panel-dark shadow-xl rounded-lg py-1 z-20 border border-wa-border dark:border-wa-border-dark">
              <button v-if="chat.isGroup" @click="openInfo" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm flex items-center gap-2">
                <Info :size="14" /> Info grup
              </button>
              <template v-if="ai.config.enabled">
                <button @click="askAboutChat" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm flex items-center gap-2">
                  <Sparkles :size="14" class="text-violet-500" /> Tanya AI tentang chat
                </button>
                <button @click="summarizeInPlayground" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm flex items-center gap-2">
                  <Sparkles :size="14" class="text-violet-500" /> Ringkas chat (AI)
                </button>
                <button @click="draftReplyInPlayground" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm flex items-center gap-2">
                  <Sparkles :size="14" class="text-violet-500" /> Draf balasan (AI)
                </button>
              </template>
            </div>
          </div>
        </div>
      </header>

      <div ref="scroller" class="flex-1 overflow-y-auto scrollbar-thin px-[8%] py-4 space-y-0.5">
        <template v-for="row in rows" :key="row.id">
          <div v-if="row.kind === 'date'" class="flex justify-center my-3">
            <span class="date-pill">{{ row.label }}</span>
          </div>
          <div
            v-else
            class="flex"
            :class="[
              row.msg.fromMe ? 'justify-end' : 'justify-start',
              row.showTail ? 'mt-1.5' : 'mt-0.5',
            ]"
          >
            <MessageBubble :msg="row.msg" :show-tail="row.showTail" :show-sender="row.showSender" />
          </div>
        </template>
      </div>

      <div
        v-if="store.replyTo"
        class="bg-wa-panel dark:bg-wa-panel-dark px-3 pt-2"
      >
        <div class="flex items-start gap-2 bg-black/5 dark:bg-white/5 rounded-lg px-3 py-2 border-l-4 border-wa-green">
          <div class="flex-1 min-w-0">
            <div class="text-xs font-medium text-wa-green">Balas pesan</div>
            <div class="text-sm truncate text-wa-text dark:text-wa-text-dark">{{ store.replyTo.text || store.replyTo.caption || 'Media' }}</div>
          </div>
          <button @click="clearReply" class="text-wa-muted dark:text-wa-muted-dark hover:text-red-500">
            <X :size="18" />
          </button>
        </div>
      </div>

      <div v-if="ai.config.enabled && ai.suggestions.length > 0" class="px-3 pt-2 flex gap-2 overflow-x-auto scrollbar-thin">
        <button
          v-for="(s, i) in ai.suggestions"
          :key="i"
          @click="useSuggestion(s)"
          class="shrink-0 text-sm px-3 py-1.5 rounded-full bg-violet-50 dark:bg-violet-900/20 text-violet-700 dark:text-violet-300 hover:bg-violet-100 dark:hover:bg-violet-900/40 border border-violet-200 dark:border-violet-800 flex items-center gap-1"
        >
          <Sparkles :size="12" /> {{ s }}
        </button>
      </div>

      <footer v-if="chat.isChannel && chat.role !== 'admin' && chat.role !== 'owner'" class="bg-wa-panel dark:bg-wa-panel-dark px-4 py-3 flex items-center justify-center gap-2">
        <Megaphone :size="16" class="text-wa-muted" />
        <span class="text-sm text-wa-muted dark:text-wa-muted-dark">Saluran — hanya baca</span>
      </footer>

      <footer v-else class="bg-wa-panel dark:bg-wa-panel-dark px-3 py-2 flex items-center gap-2 relative">
        <VoiceRecorder v-if="recording" @done="recording = false" />

        <template v-else>
          <div class="relative">
            <button
              @click="showEmoji = !showEmoji; showAttach = false"
              class="w-10 h-10 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"
              :class="{ 'text-wa-green': showEmoji }"
            >
              <Smile :size="22" />
            </button>
            <div v-if="showEmoji" class="absolute bottom-12 left-0 z-20" @click.stop>
              <EmojiPicker @pick="pickEmoji" @close="showEmoji = false" />
            </div>
          </div>

          <button
            v-if="ai.config.enabled"
            @click="aiCompose"
            :disabled="composing"
            class="w-10 h-10 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-violet-500 disabled:opacity-50"
            title="AI Compose"
          >
            <Sparkles :size="20" />
          </button>

          <button
            @click="showQrisGenerator = true"
            class="w-10 h-10 rounded-full hover:bg-wa-green hover:text-white flex items-center justify-center transition-colors"
            title="Buat Invoice"
          >
            <TrendingUp :size="20" />
          </button>

          <div class="relative">
            <button
              @click="showAttach = !showAttach"
              class="w-10 h-10 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"
            >
              <Paperclip :size="22" />
            </button>
            <div
              v-if="showAttach"
              class="absolute bottom-12 left-0 bg-white dark:bg-wa-panel-dark shadow-xl rounded-xl py-2 w-44 z-10 border border-wa-border dark:border-wa-border-dark"
            >
              <button @click="pick('image')" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
                <ImageIcon :size="18" class="text-pink-500" /> Foto
              </button>
              <button @click="pick('video')" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
                <Video :size="18" class="text-purple-500" /> Video
              </button>
              <button @click="pick('audio')" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
                <Music :size="18" class="text-amber-500" /> Audio
              </button>
              <button @click="pick('any')" class="w-full flex items-center gap-3 px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
                <FileText :size="18" class="text-blue-500" /> Dokumen
              </button>
            </div>
          </div>

          <div class="flex-1 bg-white dark:bg-wa-hover-dark rounded-lg px-4 py-2">
            <input
              v-model="draft"
              @keydown.enter.prevent="send"
              type="text"
              placeholder="Ketik pesan"
              class="w-full bg-transparent outline-none text-sm text-wa-text dark:text-wa-text-dark placeholder:text-wa-muted dark:placeholder:text-wa-muted-dark"
            />
          </div>

          <button
            v-if="draft.trim()"
            @click="send"
            class="w-10 h-10 rounded-full bg-wa-green hover:bg-wa-green-dark flex items-center justify-center text-white"
          >
            <Send :size="20" />
          </button>
          <button
            v-else
            @click="recording = true"
            class="w-10 h-10 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"
          >
            <Mic :size="22" />
          </button>
        </template>
      </footer>
    </template>

    <div v-else class="flex-1 flex items-center justify-center flex-col gap-3 text-center px-8 relative">
      <button
        v-if="ui.waListCollapsed"
        @click="ui.waListCollapsed = false"
        class="absolute top-3 left-3 w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"
        title="Tampilkan daftar chat"
      >
        <PanelLeftOpen :size="18" />
      </button>
      <div class="w-40 h-40 rounded-full bg-wa-panel dark:bg-wa-panel-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark">
        <svg width="120" height="120" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 15a2 2 0 0 1-2 2H7l-4 4V5a2 2 0 0 1 2-2h14a2 2 0 0 1 2 2z" />
        </svg>
      </div>
      <h2 class="text-2xl font-light text-wa-text dark:text-wa-text-dark">Wakupi Desktop</h2>
      <p class="text-sm text-wa-muted dark:text-wa-muted-dark max-w-md">
        Pilih chat dari daftar di samping untuk mulai mengirim pesan. Pesanmu terenkripsi end-to-end.
      </p>
    </div>
  </section>

  <QuickInvoice v-if="showQrisGenerator" @close="showQrisGenerator = false" @send-to-chat="handleQrisSendToChat" />
</template>
