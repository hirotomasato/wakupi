<script setup lang="ts">
import { ref, computed } from 'vue'
import { Check, CheckCheck, Reply, Trash2, Download, FileText, Play, Pause, Forward } from '@lucide/vue'
import type { Message } from '../types'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const props = withDefaults(
  defineProps<{ msg: Message; showTail?: boolean; showSender?: boolean }>(),
  { showTail: true, showSender: true }
)
const store = useChatStore()
const ui = useUIStore()
const showActions = ref(false)
const audioPlaying = ref(false)
const audioEl = ref<HTMLAudioElement | null>(null)

// WhatsApp-style palette for group sender names, picked deterministically per sender.
const senderPalette = [
  '#e542a3', '#0a7ade', '#1f9e6b', '#d9803a', '#9b59b6',
  '#e0696c', '#2aa39a', '#c77d2b', '#6a8caf', '#b0518f',
]
const senderColor = computed(() => {
  const key = props.msg._senderJID || props.msg.sender || ''
  let h = 0
  for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0
  return senderPalette[h % senderPalette.length]
})

const sizeLabel = computed(() => {
  if (!props.msg.fileSize) return ''
  const kb = props.msg.fileSize / 1024
  if (kb < 1024) return kb.toFixed(0) + ' KB'
  return (kb / 1024).toFixed(1) + ' MB'
})

const durationLabel = computed(() => {
  if (!props.msg.duration) return ''
  const s = props.msg.duration
  const m = Math.floor(s / 60)
  const r = s % 60
  return `${m}:${r.toString().padStart(2, '0')}`
})

function toggleAudio() {
  if (!audioEl.value) return
  if (audioEl.value.paused) {
    audioEl.value.play()
    audioPlaying.value = true
  } else {
    audioEl.value.pause()
    audioPlaying.value = false
  }
}

function onAudioEnd() {
  audioPlaying.value = false
}

function react(emoji: string) {
  store.reactToMessage(props.msg, emoji)
}

function reply() {
  store.setReply(props.msg)
}

function del() {
  store.deleteMessage(props.msg, props.msg.fromMe)
}

function forward() {
  ui.showForward = props.msg
}

function preview() {
  if (props.msg.mediaType === 'image' || props.msg.mediaType === 'video') {
    store.setPreview(props.msg)
  }
}

async function download() {
  if (!props.msg.mediaUrl) return

  const filename = props.msg.fileName || props.msg.mediaUrl.split('/').pop() || `download-${Date.now()}.jpg`

  try {
    const response = await fetch(props.msg.mediaUrl)
    if (!response.ok) throw new Error(`HTTP ${response.status}`)

    const blob = await response.blob()
    const url = window.URL.createObjectURL(blob)

    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.style.display = 'none'
    document.body.appendChild(a)
    a.click()

    // Cleanup
    setTimeout(() => {
      document.body.removeChild(a)
      window.URL.revokeObjectURL(url)
    }, 100)
  } catch (e) {
    console.error('Download failed:', e)
    alert('Gagal download. Coba buka di browser baru.')
  }
}
</script>

<template>
  <div
    class="relative group max-w-[75%]"
    @mouseenter="showActions = true"
    @mouseleave="showActions = false"
  >
    <!-- WhatsApp beak tail, only on the first message of a run -->
    <svg
      v-if="showTail"
      class="absolute top-0 z-0"
      :class="msg.fromMe
        ? 'right-[-8px] text-wa-bubble-out dark:text-wa-bubble-out-dark'
        : 'left-[-8px] text-wa-bubble dark:text-wa-bubble-dark'"
      width="8"
      height="13"
      viewBox="0 0 8 13"
      aria-hidden="true"
    >
      <path
        v-if="msg.fromMe"
        fill="currentColor"
        d="M5.188 0H0v11.193l6.467-8.625C7.526 1.156 6.958 0 5.188 0z"
      />
      <path
        v-else
        fill="currentColor"
        d="M2.812 0H8v11.193L1.533 2.568C.474 1.156 1.042 0 2.812 0z"
      />
    </svg>

    <div
      class="px-1.5 pt-1.5 pb-1 shadow-sm relative z-10"
      :class="[
        msg.fromMe
          ? 'bg-wa-bubble-out dark:bg-wa-bubble-out-dark text-wa-text dark:text-wa-text-dark'
          : 'bg-wa-bubble dark:bg-wa-bubble-dark text-wa-text dark:text-wa-text-dark',
        showTail
          ? (msg.fromMe ? 'rounded-lg rounded-tr-none' : 'rounded-lg rounded-tl-none')
          : 'rounded-lg',
      ]"
    >
      <div
        v-if="showSender && msg.sender"
        class="text-[12.5px] font-medium px-2 pt-0.5"
        :style="{ color: senderColor }"
      >{{ msg.sender }}</div>

      <div
        v-if="msg.quotedId"
        class="mx-1 my-1 px-2 py-1 rounded border-l-4 border-wa-green bg-black/5 dark:bg-white/5 text-xs"
      >
        <div class="font-medium text-wa-green truncate">{{ msg.quotedFrom?.split('@')[0] || 'Pesan' }}</div>
        <div class="truncate text-wa-muted dark:text-wa-muted-dark">{{ msg.quotedText }}</div>
      </div>

      <template v-if="msg.deleted">
        <div class="px-2 py-1 italic text-wa-muted dark:text-wa-muted-dark text-sm">{{ msg.text }}</div>
      </template>

      <template v-else>
        <div v-if="msg.mediaType === 'image' && msg.mediaUrl" class="cursor-pointer" @click="preview">
          <img :src="msg.mediaUrl" class="rounded-md max-w-[320px] max-h-[400px] object-cover" />
        </div>

        <div v-else-if="msg.mediaType === 'video' && msg.mediaUrl" class="cursor-pointer" @click="preview">
          <video :src="msg.mediaUrl" class="rounded-md max-w-[320px] max-h-[400px]" preload="metadata" />
        </div>

        <div
          v-else-if="msg.mediaType === 'audio' && msg.mediaUrl"
          class="flex items-center gap-3 px-2 py-2 min-w-[260px]"
        >
          <button
            @click="toggleAudio"
            class="w-9 h-9 rounded-full bg-wa-green text-white flex items-center justify-center"
          >
            <Pause v-if="audioPlaying" :size="18" />
            <Play v-else :size="18" />
          </button>
          <div class="flex-1">
            <audio ref="audioEl" :src="msg.mediaUrl" @ended="onAudioEnd" />
            <div class="h-1 bg-wa-muted/30 rounded-full" />
            <div class="text-[11px] text-wa-muted dark:text-wa-muted-dark mt-1">
              {{ msg.isPtt ? 'Voice note' : 'Audio' }} · {{ durationLabel || '' }}
            </div>
          </div>
        </div>

        <div
          v-else-if="msg.mediaType === 'document'"
          class="flex items-center gap-3 px-2 py-2 min-w-[260px] cursor-pointer hover:bg-black/5 dark:hover:bg-white/5 rounded"
          @click="download"
        >
          <div class="w-10 h-10 rounded bg-wa-green/20 flex items-center justify-center text-wa-green">
            <FileText :size="22" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="text-sm font-medium truncate">{{ msg.fileName || 'Dokumen' }}</div>
            <div class="text-[11px] text-wa-muted dark:text-wa-muted-dark">{{ sizeLabel }}</div>
          </div>
          <Download :size="18" class="text-wa-muted dark:text-wa-muted-dark" />
        </div>

        <div
          v-else-if="msg.mediaType === 'sticker' && msg.mediaUrl"
          class="p-1"
        >
          <img :src="msg.mediaUrl" class="w-32 h-32 object-contain" />
        </div>

        <div
          v-if="msg.text || msg.caption"
          class="text-[14.2px] whitespace-pre-wrap break-words px-2 pb-1 pr-12"
        >
          {{ msg.caption || msg.text }}
        </div>
      </template>

      <div
        v-if="msg.reactions && msg.reactions.length > 0"
        class="absolute -bottom-3 right-2 bg-white dark:bg-wa-panel-dark shadow rounded-full px-1.5 py-0.5 text-xs flex gap-0.5 border border-wa-border dark:border-wa-border-dark"
      >
        <span v-for="(r, i) in msg.reactions" :key="i">{{ r.emoji }}</span>
      </div>

      <div class="absolute bottom-0.5 right-1.5 flex items-center gap-1 text-[11px] text-wa-muted dark:text-wa-muted-dark px-1">
        <span>{{ msg.time }}</span>
        <CheckCheck v-if="msg.fromMe && msg.status === 'read'" :size="14" class="text-sky-500" />
        <CheckCheck v-else-if="msg.fromMe && msg.status === 'delivered'" :size="14" />
        <span v-else-if="msg.fromMe && msg.status === 'failed'" class="text-red-500 text-[10px]">!</span>
        <Check v-else-if="msg.fromMe" :size="14" />
      </div>
    </div>

    <div
      v-if="showActions && !msg.deleted"
      class="absolute -top-3 z-20 flex gap-0.5 bg-white dark:bg-wa-panel-dark shadow rounded-full px-1 py-0.5 border border-wa-border dark:border-wa-border-dark"
      :class="msg.fromMe ? 'right-0' : 'left-0'"
    >
      <button @click="react('❤️')" class="w-6 h-6 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">❤️</button>
      <button @click="react('👍')" class="w-6 h-6 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">👍</button>
      <button @click="react('😂')" class="w-6 h-6 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">😂</button>
      <button @click="reply" title="Balas" class="w-6 h-6 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"><Reply :size="14" /></button>
      <button @click="forward" title="Teruskan" class="w-6 h-6 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"><Forward :size="14" /></button>
      <button @click="del" title="Hapus" class="w-6 h-6 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"><Trash2 :size="14" /></button>
    </div>
  </div>
</template>
