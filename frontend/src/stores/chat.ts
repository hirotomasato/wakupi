import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Account, Chat, Message } from '../types'
import {
  ListSessions,
  StartLogin,
  Logout,
  SendText,
  SendImage,
  SendVideo,
  SendDocument,
  SendAudio,
  DeleteMessage,
  ReactMessage,
  MarkRead,
  SubscribePresence,
  SendChatPresence,
  PickFile,
  SaveTempBlob,
  LoadChats,
  LoadMessages,
  RefreshAvatar,
  PinChat,
  ArchiveChat,
  MuteChat,
  BlockChat,
  StarMessage,
  ListStarred,
  SearchMessages,
  ForwardMessage,
  IsOnWhatsApp,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

interface BackendSession {
  id: string
  name: string
  connected: boolean
  jid: string
  phone: string
}

interface BackendChat {
  id: string
  accountId: string
  jid: string
  name: string
  isGroup: boolean
  lastMessage: string
  lastTime: number
  avatarUrl?: string
  pinned?: boolean
  archived?: boolean
  mutedUntil?: number
  blocked?: boolean
}

interface BackendMessage {
  id: string
  chatId: string
  accountId: string
  jid: string
  sender: string
  text: string
  timestamp: number
  fromMe: boolean
  isGroup: boolean
  pushName: string
  mediaType?: string
  mediaUrl?: string
  mimeType?: string
  fileName?: string
  fileSize?: number
  width?: number
  height?: number
  duration?: number
  isPtt?: boolean
  caption?: string
  quotedId?: string
  quotedText?: string
  quotedFrom?: string
}

interface BackendReceipt {
  accountId: string
  jid: string
  sender: string
  messageIds: string[]
  type: 'read' | 'delivered' | 'played'
  timestamp: number
}

interface BackendPresence {
  accountId: string
  jid: string
  online: boolean
  lastSeen: number
}

interface BackendChatPresence {
  accountId: string
  jid: string
  state: 'composing' | 'paused'
  media: string
}

interface BackendReaction {
  accountId: string
  jid: string
  messageId: string
  sender: string
  fromMe: boolean
  emoji: string
  timestamp: number
}

interface BackendDeleted {
  accountId: string
  jid: string
  messageId: string
  sender: string
}

// --- localStorage persistence for last-viewed selection ---
// Backend persists chats + messages in SQLite, but the frontend Pinia store
// lives only in memory. On reload the chat list re-pulls from the DB, but
// the previously active chat is forgotten and its history is never loaded
// until the user clicks it. Persist the selection so reopen lands the user
// back on the same chat (and triggers loadHistoryForChat for it).
const LS_ACTIVE_ACCOUNT = 'wakupi:activeAccountId'
const lsActiveChat = (accountId: string) => `wakupi:activeChatId:${accountId}`

function loadPersistedAccount(): string {
  try { return localStorage.getItem(LS_ACTIVE_ACCOUNT) || '' } catch { return '' }
}
function loadPersistedChat(accountId: string): string {
  if (!accountId) return ''
  try { return localStorage.getItem(lsActiveChat(accountId)) || '' } catch { return '' }
}
function saveActiveAccount(id: string) {
  try {
    if (id) localStorage.setItem(LS_ACTIVE_ACCOUNT, id)
    else localStorage.removeItem(LS_ACTIVE_ACCOUNT)
  } catch {}
}
function saveActiveChat(accountId: string, chatId: string) {
  try {
    if (accountId && chatId) localStorage.setItem(lsActiveChat(accountId), chatId)
  } catch {}
}
function clearPersistedChat(accountId: string) {
  try { if (accountId) localStorage.removeItem(lsActiveChat(accountId)) } catch {}
}

function formatTime(unix: number): string {
  if (!unix) return ''
  const d = new Date(unix * 1000)
  const today = new Date()
  const isToday = d.toDateString() === today.toDateString()
  if (isToday) {
    return d.toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
  }
  const yesterday = new Date()
  yesterday.setDate(yesterday.getDate() - 1)
  if (d.toDateString() === yesterday.toDateString()) return 'Kemarin'
  return d.toLocaleDateString('id-ID', { day: '2-digit', month: '2-digit', year: '2-digit' })
}

// Prevent concurrent LoadChats calls for the same account.
const _loadingChats = new Set<string>()

export const useChatStore = defineStore('chat', () => {
  const accounts = ref<Account[]>([])
  const activeAccountId = ref<string>('')

  const chats = ref<Chat[]>([])
  const messages = ref<Record<string, Message[]>>({})
  const activeChatId = ref<string | null>(null)

  const showLogin = ref(false)
  const qrCode = ref<string>('')
  const qrSessionId = ref<string>('')
  const qrTimeoutSec = ref<number>(0)
  const loginStatus = ref<'idle' | 'waiting' | 'pairing' | 'success' | 'timeout' | 'error'>('idle')
  const loginError = ref<string>('')

  const presence = ref<Record<string, { online: boolean; lastSeen: number }>>({})
  const chatPresence = ref<Record<string, { state: 'composing' | 'paused'; media: string }>>({})

  const replyTo = ref<Message | null>(null)
  const previewMessage = ref<Message | null>(null)

  const visibleChats = computed(() => {
    const seen = new Set<string>()
    return chats.value
      .filter((c) => c.accountId === activeAccountId.value && !c.archived && !c.blocked)
      .filter((c) => {
        // Safety dedup — only first occurrence per id survives.
        if (seen.has(c.id)) return false
        seen.add(c.id)
        return true
      })
      .sort((a, b) => {
        const pinDiff = Number(!!b.pinned) - Number(!!a.pinned)
        if (pinDiff !== 0) return pinDiff
        return (b._sortKey || 0) - (a._sortKey || 0)
      })
  })

  const archivedChats = computed(() => {
    const seen = new Set<string>()
    return chats.value
      .filter((c) => c.accountId === activeAccountId.value && c.archived)
      .filter((c) => {
        if (seen.has(c.id)) return false
        seen.add(c.id)
        return true
      })
      .sort((a, b) => (b._sortKey || 0) - (a._sortKey || 0))
  })

  const activeChat = computed(() => chats.value.find((c) => c.id === activeChatId.value) || null)
  const activeMessages = computed(() => (activeChatId.value ? messages.value[activeChatId.value] || [] : []))
  const activeAccount = computed(() => accounts.value.find((a) => a.id === activeAccountId.value) || null)

  function applySessions(list: BackendSession[]) {
    accounts.value = list.map((s) => ({
      id: s.id,
      name: s.name || s.phone || s.id,
      phone: s.phone || '',
      connected: s.connected,
    }))
    if (!activeAccountId.value && accounts.value.length > 0) {
      activeAccountId.value = accounts.value[0].id
    }
    if (accounts.value.length === 0) {
      activeAccountId.value = ''
      showLogin.value = true
    }
  }

  function upsertSession(s: BackendSession) {
    const idx = accounts.value.findIndex((a) => a.id === s.id)
    const acc: Account = {
      id: s.id,
      name: s.name || s.phone || s.id,
      phone: s.phone || '',
      connected: s.connected,
    }
    if (idx >= 0) accounts.value[idx] = acc
    else accounts.value.push(acc)
    if (!activeAccountId.value) activeAccountId.value = acc.id
  }

  function upsertChat(c: BackendChat) {
    const id = `${c.accountId}::${c.jid}`
    // Atomic dedup: rebuild the entire array from a fresh Map so it is
    // mathematically impossible for two entries with the same id to coexist.
    const all = new Map<string, Chat>()
    for (const ch of chats.value) {
      all.set(ch.id, ch)
    }
    const prev = all.get(id)

    all.set(id, {
      id,
      accountId: c.accountId,
      jid: c.jid,
      name: c.name || prev?.name || c.jid.split('@')[0],
      avatarUrl: c.avatarUrl || prev?.avatarUrl || '',
      lastMessage: c.lastTime ? c.lastMessage : (prev?._sortKey ? prev.lastMessage : c.lastMessage || ''),
      lastTime: c.lastTime ? formatTime(c.lastTime) : (prev?._sortKey ? prev.lastTime : formatTime(c.lastTime || 0)),
      _sortKey: c.lastTime || prev?._sortKey || 0,
      unread: prev?.unread ?? 0,
      isGroup: c.isGroup,
      pinned: c.pinned ?? prev?.pinned ?? false,
      archived: c.archived ?? prev?.archived ?? false,
      mutedUntil: c.mutedUntil ?? prev?.mutedUntil ?? 0,
      blocked: c.blocked ?? prev?.blocked ?? false,
    })
    chats.value = [...all.values()]
  }

  function appendMessage(m: BackendMessage) {
    const chatId = `${m.accountId}::${m.jid}`
    if (!messages.value[chatId]) messages.value[chatId] = []
    if (messages.value[chatId].some((x) => x.id === m.id)) return

    const msg: Message = {
      id: m.id,
      chatId,
      text: m.text,
      time: formatTime(m.timestamp),
      fromMe: m.fromMe,
      status: m.fromMe ? 'sent' : undefined,
      sender: m.isGroup && !m.fromMe ? m.pushName || (m.sender || '').split('@')[0] : undefined,
      _ts: m.timestamp,
      _senderJID: m.sender,
      mediaType: (m.mediaType || '') as Message['mediaType'],
      mediaUrl: m.mediaUrl,
      mimeType: m.mimeType,
      fileName: m.fileName,
      fileSize: m.fileSize,
      width: m.width,
      height: m.height,
      duration: m.duration,
      isPtt: m.isPtt,
      caption: m.caption,
      quotedId: m.quotedId,
      quotedText: m.quotedText,
      quotedFrom: m.quotedFrom,
    }

    const arr = messages.value[chatId]
    const insertIdx = arr.findIndex((x) => (x._ts || 0) > m.timestamp)
    if (insertIdx === -1) arr.push(msg)
    else arr.splice(insertIdx, 0, msg)

    upsertChat({
      id: m.jid,
      accountId: m.accountId,
      jid: m.jid,
      name: '',
      isGroup: m.isGroup,
      lastMessage: m.text || lastMessagePreview(msg),
      lastTime: m.timestamp,
    })

    const chat = chats.value.find((c) => c.id === chatId)
    if (chat) {
      if (m.timestamp >= (chat._sortKey || 0)) {
        chat.lastMessage = m.text || lastMessagePreview(msg)
        chat.lastTime = formatTime(m.timestamp)
        chat._sortKey = m.timestamp
        chat.lastMediaType = (m.mediaType || '') as any
        chat.lastFromMe = m.fromMe
        chat.lastStatus = m.fromMe ? 'sent' : undefined
      }
      if (!m.fromMe && activeChatId.value !== chatId) {
        chat.unread = (chat.unread || 0) + 1
      }
    }
  }

  function lastMessagePreview(m: Message): string {
    if (m.mediaType === 'image') return (m.caption ? m.caption : 'Foto')
    if (m.mediaType === 'video') return (m.caption ? m.caption : 'Video')
    if (m.mediaType === 'audio') return m.isPtt ? 'Voice note' : 'Audio'
    if (m.mediaType === 'document') return m.fileName || 'Dokumen'
    if (m.mediaType === 'sticker') return 'Stiker'
    return m.text || ''
  }

  function applyReaction(r: BackendReaction) {
    const chatId = `${r.accountId}::${r.jid}`
    const arr = messages.value[chatId]
    if (!arr) return
    const msg = arr.find((x) => x.id === r.messageId)
    if (!msg) return
    if (!msg.reactions) msg.reactions = []
    const existingIdx = msg.reactions.findIndex((x) => x.sender === r.sender)
    if (!r.emoji) {
      if (existingIdx >= 0) msg.reactions.splice(existingIdx, 1)
      return
    }
    if (existingIdx >= 0) msg.reactions[existingIdx].emoji = r.emoji
    else msg.reactions.push({ emoji: r.emoji, fromMe: r.fromMe, sender: r.sender })
  }

  function applyDeleted(d: BackendDeleted) {
    const chatId = `${d.accountId}::${d.jid}`
    const arr = messages.value[chatId]
    if (!arr) return
    const msg = arr.find((x) => x.id === d.messageId)
    if (!msg) return
    msg.deleted = true
    msg.text = 'Pesan ini dihapus'
    msg.mediaUrl = ''
    msg.mediaType = '' as Message['mediaType']
  }

  async function togglePin(chat: Chat) {
    chat.pinned = !chat.pinned
    try { await PinChat(chat.accountId, chat.jid, chat.pinned) } catch (e) { console.error(e) }
  }

  async function toggleArchive(chat: Chat) {
    chat.archived = !chat.archived
    try { await ArchiveChat(chat.accountId, chat.jid, chat.archived) } catch (e) { console.error(e) }
  }

  async function toggleMute(chat: Chat, until: number) {
    chat.mutedUntil = until
    try { await MuteChat(chat.accountId, chat.jid, until) } catch (e) { console.error(e) }
  }

  async function toggleBlock(chat: Chat) {
    const next = !chat.blocked
    try {
      await BlockChat(chat.accountId, chat.jid, next)
      chat.blocked = next
    } catch (e) {
      console.error(e)
    }
  }

  async function searchAll(query: string) {
    if (!activeAccountId.value || !query.trim()) return [] as any[]
    try {
      return (await SearchMessages(activeAccountId.value, query, 100)) || []
    } catch (e) {
      console.error('search', e)
      return []
    }
  }

  async function getStarredList() {
    if (!activeAccountId.value) return [] as any[]
    try {
      return (await ListStarred(activeAccountId.value, 100)) || []
    } catch (e) {
      console.error(e)
      return []
    }
  }

  async function toggleStar(msg: Message) {
    const chat = activeChat.value
    if (!chat) return
    const next = !(msg as any).starred
    try {
      await StarMessage(chat.accountId, chat.jid, msg.id, next)
      ;(msg as any).starred = next
    } catch (e) { console.error(e) }
  }

  async function forwardTo(msg: Message, chatIds: string[]) {
    const chat = activeChat.value
    if (!chat) return
    const targets = chatIds
      .map((id) => chats.value.find((c) => c.id === id)?.jid)
      .filter((x): x is string => !!x)
    if (targets.length === 0) return
    try {
      await ForwardMessage(chat.accountId, chat.jid, msg.id, targets)
    } catch (e) { console.error(e) }
  }

  async function checkIsOnWA(phone: string): Promise<{ jid: string; onWA: boolean } | null> {
    if (!activeAccountId.value) return null
    const cleaned = phone.replace(/\D/g, '')
    if (!cleaned) return null
    try {
      const list = await IsOnWhatsApp(activeAccountId.value, [cleaned])
      if (list && list.length > 0) {
        return { jid: list[0].jid, onWA: list[0].onWhatsApp }
      }
    } catch (e) {
      console.error(e)
    }
    return null
  }

  function startChatWithJID(jid: string, name = '') {
    if (!activeAccountId.value) return
    const id = `${activeAccountId.value}::${jid}`
    if (!chats.value.find((c) => c.id === id)) {
      upsertChat({
        id,
        accountId: activeAccountId.value,
        jid,
        name: name || jid.split('@')[0],
        isGroup: jid.endsWith('@g.us'),
        lastMessage: '',
        lastTime: 0,
      })
    }
    selectChat(id)
  }

  // tryRestoreChatForAccount selects the persisted chat if it exists in the
  // current chat list, loading its message history. Only restores when no
  // chat is active yet — if the user has manually selected a chat we don't
  // yank them away.
  async function tryRestoreChatForAccount(accountId: string): Promise<boolean> {
    const saved = loadPersistedChat(accountId)
    if (!saved) return false
    if (!chats.value.some((c) => c.id === saved)) return false
    if (activeChatId.value === saved) return true // already active
    selectChat(saved)
    return true
  }

  async function refreshSessions() {
    try {
      // Restore last-viewed account first so applySessions doesn't override it
      // with the first account in the list.
      const savedAccount = loadPersistedAccount()
      if (savedAccount) activeAccountId.value = savedAccount

      const list = (await ListSessions()) as unknown as BackendSession[]
      applySessions(list || [])

      // If the saved account no longer exists, fall back to whatever
      // applySessions picked (first account) and persist that.
      if (savedAccount && !accounts.value.some((a) => a.id === savedAccount)) {
        if (activeAccountId.value) saveActiveAccount(activeAccountId.value)
      }

      for (const s of list || []) {
        await loadChatsForAccount(s.id)
      }
      // loadChatsForAccount already calls tryRestoreChatForAccount internally.
    } catch (e) {
      console.error('ListSessions error', e)
    }
  }

  async function loadChatsForAccount(accountId: string) {
    // Prevent re-entrant calls.
    if (_loadingChats.has(accountId)) return
    _loadingChats.add(accountId)
    try {
      const list = (await LoadChats(accountId)) as unknown as BackendChat[]

      // Build a fresh Map from DB results so we atomically replace the entire
      // chat list. This avoids race conditions where a wa:chat event arriving
      // mid-loop gets interleaved with individual upsertChat calls.
      const fresh = new Map<string, Chat>()
      for (const c of list || []) {
        const id = `${accountId}::${c.jid}`
        fresh.set(id, {
          id,
          accountId,
          jid: c.jid,
          name: c.name || c.jid.split('@')[0],
          avatarUrl: c.avatarUrl,
          lastMessage: c.lastMessage,
          lastTime: formatTime(c.lastTime),
          _sortKey: c.lastTime,
          unread: 0,
          isGroup: c.isGroup,
          pinned: c.pinned ?? false,
          archived: c.archived ?? false,
          mutedUntil: c.mutedUntil ?? 0,
          blocked: c.blocked ?? false,
        })
      }

      // Merge in any existing chats that are not in the DB result but have
      // local state worth preserving (unread, pinned, archived, etc.).
      for (const existing of chats.value) {
        if (existing.accountId !== accountId) continue
        if (!fresh.has(existing.id)) {
          fresh.set(existing.id, existing)
        } else {
          // Preserve local-only state from the old entry.
          const entry = fresh.get(existing.id)!
          entry.unread = existing.unread
          entry.pinned = existing.pinned
          entry.archived = existing.archived
          entry.mutedUntil = existing.mutedUntil
          entry.blocked = existing.blocked
        }
      }

      // Atomic replacement — single write to the reactive array.
      const accountChats: Chat[] = []
      for (const [, chat] of fresh) accountChats.push(chat)
      // Keep chats from other accounts, append new ones.
      const otherChats = chats.value.filter((c) => c.accountId !== accountId)
      chats.value = [...otherChats, ...accountChats]

      // Always try to restore the persisted chat for this account first —
      // selectChat will auto-load history via loadHistoryForChat if needed.
      const restored = await tryRestoreChatForAccount(accountId)

      // If we didn't restore (no persisted chat or it's not in the list yet),
      // still honour the caller's currently active chat.
      if (!restored && activeChat.value && activeChat.value.accountId === accountId) {
        await loadHistoryForChat(accountId, activeChat.value.jid)
      }
    } catch (e) {
      console.error('LoadChats error', e)
    } finally {
      _loadingChats.delete(accountId)
    }
  }

  async function loadHistoryForChat(accountId: string, jid: string) {
    try {
      const list = (await LoadMessages(accountId, jid, 100, 0)) as unknown as BackendMessage[]
      for (const m of list || []) {
        appendMessage({ ...m, accountId, jid })
      }
    } catch (e) {
      console.error('LoadMessages error', e)
    }
  }

  async function startLogin(name: string = '') {
    showLogin.value = true
    loginStatus.value = 'waiting'
    loginError.value = ''
    qrCode.value = ''
    try {
      qrSessionId.value = await StartLogin(name)
    } catch (e: any) {
      loginStatus.value = 'error'
      loginError.value = e?.message || String(e)
    }
  }

  async function logout(id: string) {
    try {
      await Logout(id)
      accounts.value = accounts.value.filter((a) => a.id !== id)
      chats.value = chats.value.filter((c) => c.accountId !== id)
      // Bersihkan messages milik akun yang logout
      for (const key of Object.keys(messages.value)) {
        if (key.startsWith(`${id}::`)) {
          delete messages.value[key]
        }
      }
      if (activeAccountId.value === id) {
        activeAccountId.value = accounts.value[0]?.id || ''
      }
      if (activeChatId.value && activeChatId.value.startsWith(`${id}::`)) {
        activeChatId.value = null
      }
    } catch (e) {
      console.error('logout error', e)
    }
  }

  function selectAccount(id: string) {
    activeAccountId.value = id
    activeChatId.value = null
    saveActiveAccount(id)
    // Load chats for the selected account
    loadChatsForAccount(id).catch(() => {})
  }

  function selectChat(id: string) {
    activeChatId.value = id
    const c = chats.value.find((x) => x.id === id)
    if (!c) return
    c.unread = 0
    replyTo.value = null
    saveActiveChat(c.accountId, id)

    if (!messages.value[id] || messages.value[id].length === 0) {
      loadHistoryForChat(c.accountId, c.jid).catch(() => {})
    }

    SubscribePresence(c.accountId, c.jid).catch(() => {})
    if (!c.avatarUrl) RefreshAvatar(c.accountId, c.jid).catch(() => {})

    const arr = messages.value[id] || []
    const unread = arr.filter((m) => !m.fromMe).map((m) => m.id)
    if (unread.length > 0) {
      const lastMsg = [...arr].reverse().find((m) => !m.fromMe)
      const senderJID = lastMsg?._senderJID || ''
      MarkRead(c.accountId, c.jid, senderJID, unread).catch(() => {})
    }
  }

  let typingTimer: any = null
  function setTyping(typing: boolean) {
    const chat = activeChat.value
    if (!chat) return
    SendChatPresence(chat.accountId, chat.jid, typing ? 'composing' : 'paused').catch(() => {})
    if (typingTimer) clearTimeout(typingTimer)
    if (typing) {
      typingTimer = setTimeout(() => {
        SendChatPresence(chat.accountId, chat.jid, 'paused').catch(() => {})
      }, 4000)
    }
  }

  function quotedArg(): { id: string; participant: string; text: string } | undefined {
    if (!replyTo.value) return undefined
    return {
      id: replyTo.value.id,
      participant: replyTo.value._senderJID || '',
      text: replyTo.value.text || '',
    }
  }

  function optimisticInsert(chatId: string, msg: Message) {
    if (!messages.value[chatId]) messages.value[chatId] = []
    messages.value[chatId].push(msg)
    const chat = chats.value.find((c) => c.id === chatId)
    if (chat) {
      chat.lastMessage = lastMessagePreview(msg)
      chat.lastTime = msg.time
      chat._sortKey = msg._ts
    }
  }

  async function sendMessage(text: string) {
    if (!activeChatId.value || !text.trim()) return
    const chat = chats.value.find((c) => c.id === activeChatId.value)
    if (!chat) return
    const tempId = 'tmp-' + Date.now()
    const ts = Math.floor(Date.now() / 1000)
    const msg: Message = {
      id: tempId,
      chatId: chat.id,
      text: text.trim(),
      time: formatTime(ts),
      fromMe: true,
      status: 'sent',
      _ts: ts,
      quotedId: replyTo.value?.id,
      quotedText: replyTo.value?.text,
      quotedFrom: replyTo.value?._senderJID,
    }
    optimisticInsert(chat.id, msg)
    const q = quotedArg()
    replyTo.value = null

    try {
      const realId = await SendText(chat.accountId, chat.jid, text.trim(), q as any)
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0 && realId) messages.value[chat.id][idx].id = realId
    } catch (e: any) {
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0) messages.value[chat.id][idx].status = 'failed'
      console.error('send failed', e)
    }
  }

  // sendTextToChat sends plain text to an arbitrary chat (not necessarily the
  // active one). Used by the AI playground to push a reply into WhatsApp.
  async function sendTextToChat(chatId: string, text: string) {
    const body = text.trim()
    if (!chatId || !body) return
    const chat = chats.value.find((c) => c.id === chatId)
    if (!chat) return

    const tempId = 'tmp-' + Date.now()
    const ts = Math.floor(Date.now() / 1000)
    const msg: Message = {
      id: tempId,
      chatId: chat.id,
      text: body,
      time: formatTime(ts),
      fromMe: true,
      status: 'sent',
      _ts: ts,
    }
    optimisticInsert(chat.id, msg)

    try {
      const realId = await SendText(chat.accountId, chat.jid, body, null as any)
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0 && realId) messages.value[chat.id][idx].id = realId
    } catch (e: any) {
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0) messages.value[chat.id][idx].status = 'failed'
      console.error('send to chat failed', e)
      throw e
    }
  }

  async function attachFile(kind: 'image' | 'video' | 'audio' | 'any', caption: string = '') {
    const chat = activeChat.value
    if (!chat) return
    let path: string
    try {
      path = await PickFile(kind === 'any' ? '' : kind)
    } catch (e) {
      return
    }
    if (!path) return

    const tempId = 'tmp-' + Date.now()
    const ts = Math.floor(Date.now() / 1000)
    const fileName = path.split('/').pop() || ''
    const msg: Message = {
      id: tempId,
      chatId: chat.id,
      text: caption,
      caption: caption,
      time: formatTime(ts),
      fromMe: true,
      status: 'sent',
      _ts: ts,
      mediaType: (kind === 'any' ? 'document' : kind) as Message['mediaType'],
      fileName,
      mediaUrl: '',
    }
    optimisticInsert(chat.id, msg)
    const q = quotedArg()
    replyTo.value = null

    try {
      let result: any
      if (kind === 'image') result = await SendImage(chat.accountId, chat.jid, path, caption, q as any)
      else if (kind === 'video') result = await SendVideo(chat.accountId, chat.jid, path, caption, q as any)
      else if (kind === 'audio') result = await SendAudio(chat.accountId, chat.jid, path, false, q as any)
      else result = await SendDocument(chat.accountId, chat.jid, path, q as any)
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0 && result) {
        messages.value[chat.id][idx].id = result.messageId
        if (result.localUrl) messages.value[chat.id][idx].mediaUrl = result.localUrl
        if (result.mimeType) messages.value[chat.id][idx].mimeType = result.mimeType
      }
    } catch (e: any) {
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0) messages.value[chat.id][idx].status = 'failed'
      console.error('attach failed', e)
    }
  }

  async function sendImageBlob(b64: string, caption: string = '') {
    const chat = activeChat.value
    if (!chat) return
    let path = ''
    try {
      path = await SaveTempBlob(b64, '.png')
    } catch (e) {
      console.error('save image blob', e)
      return
    }
    const tempId = 'tmp-' + Date.now()
    const ts = Math.floor(Date.now() / 1000)
    const msg: Message = {
      id: tempId,
      chatId: chat.id,
      text: caption,
      caption: caption,
      time: formatTime(ts),
      fromMe: true,
      status: 'sent',
      _ts: ts,
      mediaType: 'image',
      mediaUrl: '',
    }
    optimisticInsert(chat.id, msg)
    const q = quotedArg()
    replyTo.value = null

    try {
      const result = await SendImage(chat.accountId, chat.jid, path, caption, q as any)
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0 && result) {
        messages.value[chat.id][idx].id = result.messageId
        if (result.localUrl) messages.value[chat.id][idx].mediaUrl = result.localUrl
        if (result.mimeType) messages.value[chat.id][idx].mimeType = result.mimeType
      }
    } catch (e: any) {
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0) messages.value[chat.id][idx].status = 'failed'
      console.error('image blob send failed', e)
    }
  }

  async function sendVoiceBlob(b64: string) {
    const chat = activeChat.value
    if (!chat) return
    let path = ''
    try {
      path = await SaveTempBlob(b64, '.ogg')
    } catch (e) {
      console.error('save blob', e)
      return
    }
    const tempId = 'tmp-' + Date.now()
    const ts = Math.floor(Date.now() / 1000)
    const msg: Message = {
      id: tempId,
      chatId: chat.id,
      text: '',
      time: formatTime(ts),
      fromMe: true,
      status: 'sent',
      _ts: ts,
      mediaType: 'audio',
      isPtt: true,
    }
    optimisticInsert(chat.id, msg)
    const q = quotedArg()
    replyTo.value = null

    try {
      const result = await SendAudio(chat.accountId, chat.jid, path, true, q as any)
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0 && result) {
        messages.value[chat.id][idx].id = result.messageId
        if (result.localUrl) messages.value[chat.id][idx].mediaUrl = result.localUrl
        if (result.mimeType) messages.value[chat.id][idx].mimeType = result.mimeType
      }
    } catch (e: any) {
      const idx = messages.value[chat.id].findIndex((x) => x.id === tempId)
      if (idx >= 0) messages.value[chat.id][idx].status = 'failed'
      console.error('voice send failed', e)
    }
  }

  async function deleteMessage(msg: Message, forEveryone: boolean) {
    const chat = activeChat.value
    if (!chat) return
    try {
      await DeleteMessage(chat.accountId, chat.jid, msg.id, forEveryone)
      msg.deleted = true
      msg.text = forEveryone ? 'Anda menghapus pesan ini' : 'Pesan ini dihapus'
      msg.mediaUrl = ''
      msg.mediaType = '' as Message['mediaType']
    } catch (e) {
      console.error('delete failed', e)
    }
  }

  async function reactToMessage(msg: Message, emoji: string) {
    const chat = activeChat.value
    if (!chat) return
    try {
      await ReactMessage(chat.accountId, chat.jid, msg.id, msg._senderJID || '', emoji)
      if (!msg.reactions) msg.reactions = []
      const idx = msg.reactions.findIndex((r) => r.fromMe)
      if (!emoji) {
        if (idx >= 0) msg.reactions.splice(idx, 1)
      } else if (idx >= 0) msg.reactions[idx].emoji = emoji
      else msg.reactions.push({ emoji, fromMe: true, sender: 'me' })
    } catch (e) {
      console.error('react failed', e)
    }
  }

  function setReply(msg: Message | null) {
    replyTo.value = msg
  }

  function setPreview(msg: Message | null) {
    previewMessage.value = msg
  }

  function bindEvents() {
    EventsOn('wa:qr', (data: any) => {
      qrCode.value = data?.code || ''
      qrTimeoutSec.value = Math.round(data?.timeout || 0)
      qrSessionId.value = data?.sessionId || qrSessionId.value
      loginStatus.value = 'waiting'
    })
    EventsOn('wa:pair_success', () => {
      loginStatus.value = 'pairing'
    })
    EventsOn('wa:login_success', (s: BackendSession) => {
      loginStatus.value = 'success'
      qrCode.value = ''
      upsertSession(s)
      activeAccountId.value = s.id
      setTimeout(() => {
        showLogin.value = false
        loginStatus.value = 'idle'
      }, 1000)
    })
    EventsOn('wa:qr_timeout', () => {
      loginStatus.value = 'timeout'
      qrCode.value = ''
    })
    EventsOn('wa:connected', (s: BackendSession) => {
      upsertSession(s)
      // Don't reload all chats — refreshSessions already did it on startup.
      // For reconnection, seedChatsFromContacts sends wa:chat for each contact.
    })
    EventsOn('wa:disconnected', (s: BackendSession) => upsertSession(s))
    EventsOn('wa:logged_out', (s: BackendSession) => {
      accounts.value = accounts.value.filter((a) => a.id !== s.id)
      chats.value = chats.value.filter((c) => c.accountId !== s.id)
      for (const key of Object.keys(messages.value)) {
        if (key.startsWith(`${s.id}::`)) {
          delete messages.value[key]
        }
      }
      clearPersistedChat(s.id)
      if (activeAccountId.value === s.id) {
        activeAccountId.value = accounts.value[0]?.id || ''
        saveActiveAccount(activeAccountId.value)
      }
      if (activeChatId.value && activeChatId.value.startsWith(`${s.id}::`)) {
        activeChatId.value = null
      }
    })
    EventsOn('wa:chat', (c: BackendChat) => upsertChat(c))
    EventsOn('wa:message', (m: BackendMessage) => appendMessage(m))
    EventsOn('wa:receipt', (r: BackendReceipt) => {
      const chatId = `${r.accountId}::${r.jid}`
      const arr = messages.value[chatId]
      if (!arr) return
      const idSet = new Set(r.messageIds)
      let lastUpdated = false
      for (const msg of arr) {
        if (msg.fromMe && idSet.has(msg.id)) {
          if (r.type === 'read' && msg.status !== 'read') { msg.status = 'read'; lastUpdated = true }
          else if (r.type === 'delivered' && msg.status === 'sent') { msg.status = 'delivered'; lastUpdated = true }
        }
      }
      if (lastUpdated) {
        const chat = chats.value.find((c) => c.id === chatId)
        if (chat && chat.lastFromMe) {
          if (r.type === 'read') chat.lastStatus = 'read'
          else if (r.type === 'delivered' && chat.lastStatus === 'sent') chat.lastStatus = 'delivered'
        }
      }
    })
    EventsOn('wa:presence', (p: BackendPresence) => {
      const key = `${p.accountId}::${p.jid}`
      presence.value[key] = { online: p.online, lastSeen: p.lastSeen }
    })
    EventsOn('wa:chat_presence', (cp: BackendChatPresence) => {
      const key = `${cp.accountId}::${cp.jid}`
      chatPresence.value[key] = { state: cp.state, media: cp.media }
    })
    EventsOn('wa:reaction', (r: BackendReaction) => applyReaction(r))
    EventsOn('wa:deleted', (d: BackendDeleted) => applyDeleted(d))
    EventsOn('wa:avatar', (a: { accountId: string; jid: string; avatarUrl: string }) => {
      const id = `${a.accountId}::${a.jid}`
      const chat = chats.value.find((c) => c.id === id)
      if (chat) chat.avatarUrl = a.avatarUrl
    })
    EventsOn('wa:sync_complete', async (data: { sessionId: string }) => {
      console.log('history sync done', data?.sessionId)
      if (data?.sessionId) {
        // Always reload chats from DB after history sync completes.
        // seedChatsFromContacts populates stub chats but handleHistorySync
        // writes real lastMessage/lastTime to the DB. We need to replace
        // the stubs with the real data.
        await loadChatsForAccount(data.sessionId)

        // Also reload messages for the active chat so history sync
        // messages appear immediately instead of on first click.
        if (activeChat.value && activeChat.value.accountId === data.sessionId) {
          await loadHistoryForChat(data.sessionId, activeChat.value.jid)
        }
      }
    })
  }

  return {
    accounts,
    activeAccountId,
    activeAccount,
    chats,
    visibleChats,
    archivedChats,
    activeChat,
    activeChatId,
    activeMessages,
    showLogin,
    qrCode,
    qrSessionId,
    qrTimeoutSec,
    loginStatus,
    loginError,
    presence,
    chatPresence,
    replyTo,
    previewMessage,
    refreshSessions,
    startLogin,
    logout,
    selectAccount,
    selectChat,
    sendMessage,
    sendTextToChat,
    setTyping,
    attachFile,
    sendImageBlob,
    sendVoiceBlob,
    deleteMessage,
    reactToMessage,
    setReply,
    setPreview,
    bindEvents,
    togglePin,
    toggleArchive,
    toggleMute,
    toggleBlock,
    searchAll,
    getStarredList,
    toggleStar,
    forwardTo,
    checkIsOnWA,
    startChatWithJID,
  }
})
