<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import AccountRail from './components/AccountRail.vue'
import ChatList from './components/ChatList.vue'
import ChatArea from './components/ChatArea.vue'
import LoginModal from './components/LoginModal.vue'
import MediaPreview from './components/MediaPreview.vue'
import StatusPanel from './components/StatusPanel.vue'
import StatusViewer from './components/StatusViewer.vue'
import StatusComposer from './components/StatusComposer.vue'
import SettingsModal from './components/SettingsModal.vue'
import SearchModal from './components/SearchModal.vue'
import NewChatModal from './components/NewChatModal.vue'
import ForwardModal from './components/ForwardModal.vue'
import GroupInfoPanel from './components/GroupInfoPanel.vue'
import ProfileEditor from './components/ProfileEditor.vue'
import StarredPanel from './components/StarredPanel.vue'
import AISettingsModal from './components/AISettingsModal.vue'
import CSBotSettingsModal from './components/CSBotSettingsModal.vue'
import ChatContextMenu from './components/ChatContextMenu.vue'
import PlaygroundView from './components/playground/PlaygroundView.vue'
import ImageGenView from './components/ImageGenView.vue'
import { useChatStore } from './stores/chat'
import { useStatusStore } from './stores/status'
import { useSettingsStore } from './stores/settings'
import { useAIStore } from './stores/ai'
import { useCSBotStore } from './stores/csbot'
import { useUIStore } from './stores/ui'
import { usePlaygroundStore } from './stores/playground'

const chat = useChatStore()
const status = useStatusStore()
const settings = useSettingsStore()
const ai = useAIStore()
const csbot = useCSBotStore()
const ui = useUIStore()
const pg = usePlaygroundStore()
const showSettings = ref(false)

let lastNotifTs = 0
function tryNotify(title: string, body: string) {
  if (!settings.notifications) return
  if (typeof Notification === 'undefined') return
  if (Notification.permission === 'granted') {
    try {
      new Notification(title, { body, silent: !settings.notificationSound })
    } catch {}
  } else if (Notification.permission !== 'denied') {
    Notification.requestPermission()
  }
}

onMounted(async () => {
  chat.bindEvents()
  status.bindEvents()
  pg.bindEvents()
  await chat.refreshSessions()
  await ai.load()
  if (ai.config.enabled) ai.testConnection(ai.config).catch(() => {})
  await csbot.load()
  if (csbot.config.enabled) csbot.testConnection(csbot.config).catch(() => {})

  if (typeof Notification !== 'undefined' && Notification.permission === 'default') {
    Notification.requestPermission().catch(() => {})
  }
})

watch(
  () => chat.chats.map((c) => c.id + ':' + c.unread).join(','),
  () => {
    const now = Date.now()
    if (now - lastNotifTs < 1500) return
    if (!chat.activeAccountId) return
    for (const c of chat.chats) {
      if (c.unread > 0 && c.id !== chat.activeChatId) {
        const muted = c.mutedUntil && c.mutedUntil > Math.floor(Date.now() / 1000)
        if (muted) continue
        tryNotify(c.name, c.lastMessage)
        lastNotifTs = now
        break
      }
    }
  }
)
</script>

<template>
  <div class="h-full w-full flex bg-wa-bg dark:bg-wa-bg-dark">
    <AccountRail @open-settings="showSettings = true" />

    <ImageGenView v-if="ui.showImageGen" />
    <PlaygroundView v-else-if="ui.showPlayground" />
    <template v-else-if="chat.activeAccountId">
      <StatusPanel v-if="status.showStatusPanel" />
      <ChatList v-else-if="!ui.waListCollapsed" />
      <ChatArea />
    </template>
    <div v-else class="flex-1 flex items-center justify-center text-wa-muted">
      <div class="text-center">
        <p class="text-lg">Belum ada akun terhubung</p>
        <button
          @click="chat.startLogin('')"
          class="mt-4 bg-wa-green hover:bg-wa-green-dark text-white px-5 py-2 rounded-lg"
        >
          Hubungkan WhatsApp
        </button>
      </div>
    </div>

    <LoginModal v-if="chat.showLogin" />
    <MediaPreview />
    <StatusViewer />
    <StatusComposer v-if="status.showComposer" />
    <SettingsModal :open="showSettings" @close="showSettings = false" />
    <SearchModal />
    <NewChatModal />
    <ForwardModal />
    <GroupInfoPanel />
    <ProfileEditor />
    <StarredPanel />
    <AISettingsModal />
    <CSBotSettingsModal />
    <ChatContextMenu />
  </div>
</template>
