<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { X, Megaphone, Search, Users, Loader2, ExternalLink } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'
import type { ChannelInfo } from '../types'

const chat = useChatStore()
const ui = useUIStore()
const inviteKey = ref('')
const preview = ref<ChannelInfo | null>(null)
const previewLoading = ref(false)
const previewError = ref('')
const followLoading = ref('')

onMounted(() => {
  chat.loadSubscribedChannels()
})

function initials(name: string) {
  return name
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()
}

function fmtCount(n: number) {
  if (n >= 1000000) return (n / 1000000).toFixed(1) + 'jt'
  if (n >= 1000) return (n / 1000).toFixed(1) + 'rb'
  return String(n)
}

async function resolve() {
  const key = inviteKey.value.trim()
  if (!key) return
  previewLoading.value = true
  previewError.value = ''
  preview.value = null
  try {
    preview.value = await chat.resolveChannelByInvite(key)
    if (!preview.value) previewError.value = 'Saluran tidak ditemukan'
  } catch (e: any) {
    previewError.value = e?.message || 'Gagal mencari saluran'
  } finally {
    previewLoading.value = false
  }
}

async function follow() {
  if (!preview.value) return
  followLoading.value = preview.value.jid
  try {
    await chat.followChannel(preview.value.jid)
    preview.value = null
    inviteKey.value = ''
    await chat.loadSubscribedChannels()
  } catch (e: any) {
    previewError.value = e?.message || 'Gagal mengikuti saluran'
  } finally {
    followLoading.value = ''
  }
}

async function unfollow(jid: string) {
  try {
    await chat.unfollowChannel(jid)
    await chat.loadSubscribedChannels()
  } catch (e: any) {
    // silent
  }
}

function openChannel(ch: ChannelInfo) {
  chat.startChatWithJID(ch.jid, ch.name)
  ui.showChannels = false
}

function close() {
  ui.showChannels = false
  inviteKey.value = ''
  preview.value = null
  previewError.value = ''
}
</script>

<template>
  <div v-if="ui.showChannels" class="fixed inset-0 z-40 bg-black/40 flex items-center justify-center" @click.self="close">
    <div class="w-[480px] max-w-[92vw] max-h-[80vh] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden flex flex-col">
      <header class="flex items-center justify-between px-5 py-3 border-b border-wa-border dark:border-wa-border-dark">
        <div class="flex items-center gap-2">
          <Megaphone :size="18" class="text-amber-500" />
          <h2 class="font-semibold">Saluran</h2>
        </div>
        <button @click="close" class="text-wa-muted dark:text-wa-muted-dark"><X :size="18" /></button>
      </header>

      <!-- Follow by invite -->
      <div class="px-5 py-3 border-b border-wa-border dark:border-wa-border-dark space-y-2">
        <div class="flex items-center gap-2">
          <div class="flex-1 flex items-center gap-2 bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 h-9">
            <Search :size="16" class="text-wa-muted dark:text-wa-muted-dark shrink-0" />
            <input
              v-model="inviteKey"
              @keydown.enter="resolve"
              placeholder="Link saluran atau kode invite"
              class="flex-1 bg-transparent outline-none text-sm placeholder:text-wa-muted dark:placeholder:text-wa-muted-dark dark:text-wa-text-dark"
            />
          </div>
          <button
            @click="resolve"
            :disabled="previewLoading || !inviteKey.trim()"
            class="text-sm px-3 py-1.5 rounded-lg bg-amber-500 text-white hover:bg-amber-600 disabled:opacity-50 shrink-0"
          >
            Cari
          </button>
        </div>

        <!-- Preview card -->
        <div v-if="preview" class="flex items-center gap-3 bg-wa-panel dark:bg-wa-hover-dark rounded-xl p-3">
          <div class="w-12 h-12 rounded-full bg-amber-500 text-white flex items-center justify-center font-semibold shrink-0 overflow-hidden">
            <img v-if="preview.avatarUrl" :src="preview.avatarUrl" class="w-full h-full object-cover" />
            <Megaphone v-else :size="20" />
          </div>
          <div class="flex-1 min-w-0">
            <div class="font-medium truncate dark:text-wa-text-dark">{{ preview.name }}</div>
            <div class="text-xs text-wa-muted dark:text-wa-muted-dark flex items-center gap-1">
              <Users :size="12" /> {{ fmtCount(preview.subscriberCount) }} pelanggan
            </div>
            <div v-if="preview.description" class="text-xs text-wa-muted dark:text-wa-muted-dark truncate mt-0.5">
              {{ preview.description }}
            </div>
          </div>
          <button
            @click="follow"
            :disabled="followLoading === preview.jid"
            class="text-sm px-3 py-1.5 rounded-lg bg-amber-500 text-white hover:bg-amber-600 disabled:opacity-50 shrink-0"
          >
            <Loader2 v-if="followLoading === preview.jid" :size="14" class="animate-spin" />
            <span v-else>Ikuti</span>
          </button>
        </div>

        <div v-if="previewError" class="text-sm text-red-500 px-1">{{ previewError }}</div>
      </div>

      <!-- Subscribed channels -->
      <div class="flex-1 overflow-y-auto scrollbar-thin">
        <div v-if="chat.loadingChannels" class="flex items-center justify-center py-12">
          <Loader2 :size="24" class="animate-spin text-wa-muted" />
        </div>

        <ul v-else>
          <li
            v-for="ch in chat.channelInfos"
            :key="ch.jid"
            class="flex items-center gap-3 px-5 py-3 cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark border-b border-wa-border dark:border-wa-border-dark"
          >
            <div
              class="w-12 h-12 rounded-full bg-amber-500 text-white flex items-center justify-center font-semibold shrink-0 overflow-hidden"
              @click="openChannel(ch)"
            >
              <img v-if="ch.avatarUrl" :src="ch.avatarUrl" class="w-full h-full object-cover" />
              <Megaphone v-else :size="20" />
            </div>
            <div class="flex-1 min-w-0" @click="openChannel(ch)">
              <div class="font-medium truncate dark:text-wa-text-dark">{{ ch.name }}</div>
              <div class="text-xs text-wa-muted dark:text-wa-muted-dark flex items-center gap-1">
                <Users :size="12" /> {{ fmtCount(ch.subscriberCount) }} pelanggan
              </div>
              <div v-if="ch.description" class="text-xs text-wa-muted dark:text-wa-muted-dark truncate mt-0.5">
                {{ ch.description }}
              </div>
            </div>
            <div class="flex items-center gap-2">
              <button
                @click="openChannel(ch)"
                class="text-sm px-3 py-1.5 rounded-lg bg-wa-green text-white hover:bg-wa-green-dark shrink-0"
              >
                <ExternalLink :size="14" />
              </button>
              <button
                @click="unfollow(ch.jid)"
                class="text-sm px-2 py-1.5 rounded-lg text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 shrink-0"
                title="Berhenti ikuti"
              >
                <X :size="14" />
              </button>
            </div>
          </li>
          <li v-if="chat.channelInfos.length === 0 && !chat.loadingChannels" class="px-5 py-8 text-center text-sm text-wa-muted dark:text-wa-muted-dark">
            Belum ada saluran diikuti. Cari saluran dengan link invite di atas.
          </li>
        </ul>
      </div>
    </div>
  </div>
</template>