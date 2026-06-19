import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Message } from '../types'

export const useUIStore = defineStore('ui', () => {
  const showSearch = ref(false)
  const showNewChat = ref(false)
  const showGroupInfo = ref(false)
  const showProfile = ref(false)
  const showStarred = ref(false)
  const showAISettings = ref(false)
  const showCSBotSettings = ref(false)
  const showForward = ref<Message | null>(null)
  const showChatMenu = ref<{ chatId: string; x: number; y: number } | null>(null)

  // Playground (AI) mode + panel collapse state.
  const showPlayground = ref(false)
  const showImageGen = ref(false)
  const pgLeftCollapsed = ref(false)
  const pgRightCollapsed = ref(false)
  // Text pending to be sent into a WhatsApp chat from the playground (null = picker closed).
  const sendToWhatsApp = ref<string | null>(null)
  // Collapse state for the WhatsApp chat-list panel.
  const waListCollapsed = ref(false)

  return {
    showSearch,
    showNewChat,
    showGroupInfo,
    showProfile,
    showStarred,
    showAISettings,
    showCSBotSettings,
    showForward,
    showChatMenu,
    showPlayground,
    showImageGen,
    pgLeftCollapsed,
    pgRightCollapsed,
    sendToWhatsApp,
    waListCollapsed,
  }
})
