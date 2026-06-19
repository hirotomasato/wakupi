import { defineStore } from 'pinia'
import { ref, computed, watch } from 'vue'
import { AIChat, AIChatCancel } from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'

export type Role = 'user' | 'assistant'
export type PGTab = 'chat' | 'market'

export interface PlaygroundMessage {
  id: string
  role: Role
  content: string
  error?: boolean
  createdAt: number
}

export interface PlaygroundSession {
  id: string
  title: string
  messages: PlaygroundMessage[]
  // Per-session parameter overrides (empty = inherit from AI settings).
  model: string
  temperature: number
  system: string
  createdAt: number
  updatedAt: number
  // Which tab was last active for this session.
  tab?: PGTab
}

const STORAGE_KEY = 'wakupi.playground'

const DEFAULT_SYSTEM = 'You are a helpful assistant. Use Markdown for formatting when useful.'

function uid(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

function newSession(): PlaygroundSession {
  const now = Date.now()
  return {
    id: uid(),
    title: 'Percakapan baru',
    messages: [],
    model: '',
    temperature: 0.7,
    system: DEFAULT_SYSTEM,
    createdAt: now,
    updatedAt: now,
  }
}

interface PersistShape {
  sessions: PlaygroundSession[]
  activeId: string
}

function load(): PersistShape {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (raw) {
      const parsed = JSON.parse(raw) as PersistShape
      if (parsed.sessions?.length) return parsed
    }
  } catch {}
  const s = newSession()
  return { sessions: [s], activeId: s.id }
}

export const usePlaygroundStore = defineStore('playground', () => {
  const initial = load()
  const sessions = ref<PlaygroundSession[]>(initial.sessions)
  const activeId = ref<string>(initial.activeId)
  const streaming = ref(false)
  const streamId = ref<string>('')
  const streamSessionId = ref<string>('')
  // Text to preload into the composer (e.g. from "Tanya AI" in a chat).
  const pendingInput = ref<string>('')
  // Active tab in the playground: chat or market.
  const pgTab = ref<PGTab>('chat')

  let bound = false

  const activeSession = computed(
    () => sessions.value.find((s) => s.id === activeId.value) || sessions.value[0]
  )

  const sortedSessions = computed(() =>
    [...sessions.value].sort((a, b) => b.updatedAt - a.updatedAt)
  )

  function persist() {
    try {
      localStorage.setItem(
        STORAGE_KEY,
        JSON.stringify({ sessions: sessions.value, activeId: activeId.value })
      )
    } catch {}
  }

  watch([sessions, activeId], persist, { deep: true })

  function createSession() {
    const s = newSession()
    sessions.value.push(s)
    activeId.value = s.id
    return s
  }

  // openWithContext starts a fresh session seeded with WhatsApp conversation
  // context and (optionally) auto-sends an instruction like "ringkas". When
  // autoSend is false, the prompt is staged into the composer for editing.
  function openWithContext(opts: {
    title: string
    context: string
    instruction: string
    autoSend?: boolean
  }) {
    if (streaming.value) cancel()
    const s = newSession()
    s.title = opts.title
    s.system =
      'You are a helpful WhatsApp assistant. The user will share a conversation excerpt and ask you to act on it ' +
      '(summarize, draft a reply, translate, etc). Reply in the same language as the conversation. Use Markdown when useful.'
    sessions.value.push(s)
    activeId.value = s.id

    const prompt =
      `${opts.instruction}\n\n--- Percakapan ---\n${opts.context}\n--- Akhir ---`

    if (opts.autoSend) {
      send(prompt)
    } else {
      pendingInput.value = prompt
    }
  }

  function consumePendingInput(): string {
    const v = pendingInput.value
    pendingInput.value = ''
    return v
  }

  function selectSession(id: string) {
    if (streaming.value) cancel()
    activeId.value = id
  }

  function deleteSession(id: string) {
    const idx = sessions.value.findIndex((s) => s.id === id)
    if (idx === -1) return
    sessions.value.splice(idx, 1)
    if (sessions.value.length === 0) {
      const s = newSession()
      sessions.value.push(s)
      activeId.value = s.id
    } else if (activeId.value === id) {
      activeId.value = sortedSessions.value[0].id
    }
  }

  function renameSession(id: string, title: string) {
    const s = sessions.value.find((x) => x.id === id)
    if (s) s.title = title.trim() || 'Percakapan baru'
  }

  function clearActive() {
    const s = activeSession.value
    if (!s) return
    s.messages = []
    s.title = 'Percakapan baru'
    s.updatedAt = Date.now()
  }

  function bindEvents() {
    if (bound) return
    bound = true

    EventsOn('ai:chat:delta', (payload: { id: string; delta: string }) => {
      if (payload.id !== streamId.value) return
      const s = sessions.value.find((x) => x.id === streamSessionId.value)
      if (!s) return
      const last = s.messages[s.messages.length - 1]
      if (last && last.role === 'assistant') {
        last.content += payload.delta
      }
    })

    EventsOn('ai:chat:done', (payload: { id: string; error?: string; cancelled?: boolean }) => {
      if (payload.id !== streamId.value) return
      const s = sessions.value.find((x) => x.id === streamSessionId.value)
      if (s) {
        const last = s.messages[s.messages.length - 1]
        if (last && last.role === 'assistant') {
          if (payload.error) {
            last.content = last.content || ''
            last.content += (last.content ? '\n\n' : '') + `⚠️ ${payload.error}`
            last.error = true
          } else if (payload.cancelled && !last.content) {
            last.content = '_(dihentikan)_'
          }
        }
        s.updatedAt = Date.now()
      }
      streaming.value = false
      streamId.value = ''
      streamSessionId.value = ''
    })
  }

  async function send(text: string) {
    const content = text.trim()
    if (!content || streaming.value) return
    const s = activeSession.value
    if (!s) return

    s.messages.push({ id: uid(), role: 'user', content, createdAt: Date.now() })

    // Derive a title from the first user message.
    if (s.messages.filter((m) => m.role === 'user').length === 1) {
      s.title = content.slice(0, 40) + (content.length > 40 ? '…' : '')
    }

    const assistant: PlaygroundMessage = {
      id: uid(),
      role: 'assistant',
      content: '',
      createdAt: Date.now(),
    }
    s.messages.push(assistant)
    s.updatedAt = Date.now()

    const id = uid()
    streamId.value = id
    streamSessionId.value = s.id
    streaming.value = true

    const history = s.messages
      .filter((m) => !(m.role === 'assistant' && m.content === ''))
      .map((m) => ({ role: m.role, content: m.content }))

    try {
      await AIChat(
        id,
        history as any,
        { model: s.model, temperature: s.temperature, system: s.system } as any
      )
    } catch (e: any) {
      assistant.content = `⚠️ ${e?.message || e}`
      assistant.error = true
      streaming.value = false
      streamId.value = ''
      streamSessionId.value = ''
    }
  }

  function cancel() {
    if (!streaming.value) return
    AIChatCancel()
    // Immediately clean up state so late-arriving events from the cancelled
    // stream are filtered out by their streamId check. The done handler may
    // also fire later but will be a no-op since the IDs no longer match.
    streamId.value = ''
    streamSessionId.value = ''
    streaming.value = false
  }

  function regenerate() {
    const s = activeSession.value
    if (!s || streaming.value) return
    // Drop trailing assistant message and resend the last user turn.
    let lastUser = -1
    for (let i = s.messages.length - 1; i >= 0; i--) {
      if (s.messages[i].role === 'user') {
        lastUser = i
        break
      }
    }
    if (lastUser === -1) return
    const userText = s.messages[lastUser].content
    s.messages.splice(lastUser)
    send(userText)
  }

  return {
    sessions,
    activeId,
    activeSession,
    sortedSessions,
    streaming,
    pendingInput,
    createSession,
    openWithContext,
    consumePendingInput,
    selectSession,
    deleteSession,
    renameSession,
    clearActive,
    bindEvents,
    send,
    cancel,
    regenerate,
    pgTab,
  }
})
