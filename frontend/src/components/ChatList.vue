<script setup lang="ts">
import { ref, computed } from 'vue'
import { MessageSquarePlus, MoreVertical, Filter, Users, Pin, BellOff, Archive, Image as ImageIcon, Video, Mic, FileText, Check, CheckCheck, PanelLeftClose, Megaphone } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useUIStore } from '../stores/ui'

const store = useChatStore()
const ui = useUIStore()
const query = ref('')
const showMenu = ref(false)
const showArchived = ref(false)

const filtered = computed(() => {
  const list = showArchived.value ? store.archivedChats : store.visibleChats
  const q = query.value.trim().toLowerCase()
  if (!q) return list
  return list.filter(
    (c) => c.name.toLowerCase().includes(q) || c.lastMessage.toLowerCase().includes(q)
  )
})

const initials = (name: string) =>
  name
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()

function isMuted(c: any) {
  return c.mutedUntil && c.mutedUntil > Math.floor(Date.now() / 1000)
}

function onContextMenu(e: MouseEvent, chatId: string) {
  e.preventDefault()
  ui.showChatMenu = { chatId, x: e.clientX, y: e.clientY }
}
</script>

<template>
  <section class="w-[380px] bg-white dark:bg-[#111b21] border-r border-wa-border dark:border-wa-border-dark flex flex-col">
    <header class="h-14 px-4 flex items-center justify-between bg-wa-panel dark:bg-wa-panel-dark">
      <h1 class="text-base font-semibold text-wa-text dark:text-wa-text-dark">
        {{ showArchived ? 'Arsip' : (store.activeAccount?.name || 'WhatsApp') }}
      </h1>
      <div class="flex items-center gap-1 text-wa-muted dark:text-wa-muted-dark">
        <button @click="ui.showNewChat = true" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center" title="Chat baru">
          <MessageSquarePlus :size="20" />
        </button>
        <div class="relative">
          <button @click="showMenu = !showMenu" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center" title="Menu">
            <MoreVertical :size="20" />
          </button>
          <div v-if="showMenu" class="absolute right-0 top-10 w-52 bg-white dark:bg-wa-panel-dark shadow-xl rounded-lg py-1 z-20 border border-wa-border dark:border-wa-border-dark">
            <button @click="showArchived = !showArchived; showMenu = false" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm flex items-center gap-2">
              <Archive :size="14" /> {{ showArchived ? 'Kembali ke chat' : 'Arsip' }}
            </button>
            <button @click="ui.showProfile = true; showMenu = false" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
              Profil saya
            </button>
            <button @click="ui.showAISettings = true; showMenu = false" class="w-full text-left px-4 py-2 hover:bg-wa-hover dark:hover:bg-wa-hover-dark text-sm">
              ✨ AI Assistant
            </button>
          </div>
        </div>
        <button @click="ui.waListCollapsed = true" class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center" title="Sembunyikan daftar chat">
          <PanelLeftClose :size="18" />
        </button>
      </div>
    </header>

    <div class="px-3 py-2 bg-white dark:bg-[#111b21]">
      <div class="flex items-center gap-2">
        <div class="flex-1 flex items-center gap-3 bg-wa-panel dark:bg-wa-panel-dark rounded-lg px-3 h-9">
          <Search :size="16" class="text-wa-muted dark:text-wa-muted-dark" />
          <input
            v-model="query"
            type="text"
            placeholder="Cari atau mulai chat baru"
            class="flex-1 bg-transparent outline-none text-sm placeholder:text-wa-muted dark:placeholder:text-wa-muted-dark dark:text-wa-text-dark"
          />
        </div>
        <button class="w-9 h-9 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark">
          <Filter :size="18" />
        </button>
      </div>
    </div>

    <ul class="flex-1 overflow-y-auto scrollbar-thin">
      <li
        v-for="chat in filtered"
        :key="chat.id"
        @click="store.selectChat(chat.id)"
        @contextmenu="onContextMenu($event, chat.id)"
        class="flex items-center gap-3 px-3 py-3 cursor-pointer hover:bg-wa-hover dark:hover:bg-wa-hover-dark border-b border-wa-border dark:border-wa-border-dark"
        :class="{ 'bg-wa-hover dark:bg-wa-hover-dark': store.activeChatId === chat.id }"
      >
        <div class="w-12 h-12 rounded-full bg-slate-400 text-white flex items-center justify-center font-semibold shrink-0 overflow-hidden">
          <img v-if="chat.avatarUrl" :src="chat.avatarUrl" class="w-full h-full object-cover" />
          <Users v-else-if="chat.isGroup" :size="22" />
          <Megaphone v-else-if="chat.isChannel" :size="22" />
          <span v-else>{{ initials(chat.name) }}</span>
        </div>
        <div class="flex-1 min-w-0">
          <div class="flex items-center justify-between">
            <span class="font-medium truncate text-wa-text dark:text-wa-text-dark">{{ chat.name }}</span>
            <span class="text-xs text-wa-muted dark:text-wa-muted-dark shrink-0 ml-2">{{ chat.lastTime }}</span>
          </div>
          <div class="flex items-center justify-between mt-0.5">
            <div class="flex items-center gap-1 text-sm truncate text-wa-muted dark:text-wa-muted-dark min-w-0">
              <CheckCheck v-if="chat.lastFromMe && chat.lastStatus === 'read'" :size="14" class="text-sky-500 shrink-0" />
              <CheckCheck v-else-if="chat.lastFromMe && chat.lastStatus === 'delivered'" :size="14" class="shrink-0" />
              <Check v-else-if="chat.lastFromMe" :size="14" class="shrink-0" />

              <ImageIcon v-if="chat.lastMediaType === 'image'" :size="14" class="shrink-0" />
              <Video v-else-if="chat.lastMediaType === 'video'" :size="14" class="shrink-0" />
              <Mic v-else-if="chat.lastMediaType === 'audio'" :size="14" class="shrink-0" />
              <FileText v-else-if="chat.lastMediaType === 'document'" :size="14" class="shrink-0" />
              <span class="truncate">{{ chat.lastMessage || (chat.lastMediaType === 'sticker' ? 'Stiker' : '') }}</span>
            </div>
            <div class="flex items-center gap-1 ml-2 shrink-0">
              <BellOff v-if="isMuted(chat)" :size="14" class="text-wa-muted dark:text-wa-muted-dark" />
              <Pin v-if="chat.pinned" :size="14" class="text-wa-muted dark:text-wa-muted-dark" />
              <span
                v-if="chat.unread > 0"
                class="bg-wa-green text-white text-xs font-semibold rounded-full min-w-[20px] h-5 px-1.5 flex items-center justify-center"
              >
                {{ chat.unread }}
              </span>
            </div>
          </div>
        </div>
      </li>
      <li v-if="filtered.length === 0" class="text-center text-sm text-wa-muted dark:text-wa-muted-dark py-8">
        Tidak ada chat ditemukan
      </li>
    </ul>
  </section>
</template>
