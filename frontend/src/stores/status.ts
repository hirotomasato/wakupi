import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { PostStatusText, PostStatusImage, PickFile } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

export interface StatusItem {
  id: string
  accountId: string
  sender: string
  pushName: string
  text: string
  caption?: string
  mediaType?: string
  mediaUrl?: string
  timestamp: number
  fromMe: boolean
}

export interface StatusGroup {
  sender: string
  name: string
  items: StatusItem[]
  latestTime: number
}

export const useStatusStore = defineStore('status', () => {
  const items = ref<StatusItem[]>([])
  const showStatusPanel = ref(false)
  const showComposer = ref(false)
  const composerText = ref('')
  const viewer = ref<{ groupSender: string; index: number } | null>(null)

  const grouped = computed<StatusGroup[]>(() => {
    const map = new Map<string, StatusGroup>()
    const cutoff = Math.floor(Date.now() / 1000) - 24 * 3600
    for (const it of items.value) {
      if (it.timestamp < cutoff) continue
      const key = it.sender
      if (!map.has(key)) {
        map.set(key, { sender: key, name: it.pushName || key.split('@')[0], items: [], latestTime: 0 })
      }
      const g = map.get(key)!
      g.items.push(it)
      if (it.timestamp > g.latestTime) g.latestTime = it.timestamp
      if (it.pushName) g.name = it.pushName
    }
    for (const g of map.values()) {
      g.items.sort((a, b) => a.timestamp - b.timestamp)
    }
    return Array.from(map.values()).sort((a, b) => b.latestTime - a.latestTime)
  })

  const myStatus = computed(() => grouped.value.find((g) => g.items.some((i) => i.fromMe)))

  function bindEvents() {
    EventsOn('wa:status', (m: any) => {
      const exists = items.value.find((x) => x.id === m.id)
      if (exists) return
      items.value.push({
        id: m.id,
        accountId: m.accountId,
        sender: m.sender,
        pushName: m.pushName,
        text: m.text,
        caption: m.caption,
        mediaType: m.mediaType,
        mediaUrl: m.mediaUrl,
        timestamp: m.timestamp,
        fromMe: m.fromMe,
      })
    })
  }

  async function postText(accountId: string, text: string) {
    if (!text.trim()) return
    await PostStatusText(accountId, text.trim())
  }

  async function postImage(accountId: string, caption: string = '') {
    const path = await PickFile('image').catch(() => '')
    if (!path) return
    await postImagePath(accountId, path, caption)
  }

  async function postImagePath(accountId: string, path: string, caption: string) {
    await PostStatusImage(accountId, path, caption)
  }

  function openViewer(sender: string, index = 0) {
    viewer.value = { groupSender: sender, index }
  }

  function closeViewer() {
    viewer.value = null
  }

  return {
    items,
    grouped,
    myStatus,
    showStatusPanel,
    showComposer,
    composerText,
    viewer,
    bindEvents,
    postText,
    postImage,
    postImagePath,
    openViewer,
    closeViewer,
  }
})
