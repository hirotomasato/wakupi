import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetAIConfig, SetAIConfig, AISuggestReplies, AISummarize, AICompose, AITestConnection, AIListModels, AIGenerateImage, AIGetGamAPIModels, AIGetGamAPIStyles, AIGetGamAPIRatios } from '../../wailsjs/go/main/App'

export interface AIConfig {
  provider: 'openai' | 'anthropic' | 'gemini' | 'ollama' | 'gamapi' | ''
  apiKey: string
  baseUrl: string
  model: string
  enabled: boolean
}

export type ConnStatus = 'off' | 'unknown' | 'ok' | 'error'

// Image generation types
export interface ImageResult {
  url: string
  revisedPrompt?: string
  width?: number
  height?: number
  model?: string
  b64Json?: string
}

export interface ImageOptions {
  model?: string
  size?: string
  style?: string
  aspectRatio?: string
  count?: number
}

const defaults: AIConfig = {
  provider: 'openai',
  apiKey: '',
  baseUrl: '',
  model: '',
  enabled: false,
}

export const useAIStore = defineStore('ai', () => {
  const config = ref<AIConfig>({ ...defaults })
  const loaded = ref(false)
  const suggestions = ref<string[]>([])
  const suggesting = ref(false)
  const composing = ref(false)
  const connStatus = ref<ConnStatus>('off')
  const connMessage = ref('')
  const testing = ref(false)

  async function load() {
    try {
      const cfg = (await GetAIConfig()) as any
      config.value = { ...defaults, ...cfg }
    } catch (e) {
      console.error('load AI config', e)
    }
    loaded.value = true
    connStatus.value = config.value.enabled ? 'unknown' : 'off'
  }

  async function save(cfg: AIConfig) {
    await SetAIConfig(cfg as any)
    config.value = { ...cfg }
    if (!cfg.enabled) connStatus.value = 'off'
  }

  async function testConnection(cfg: AIConfig): Promise<boolean> {
    testing.value = true
    connMessage.value = ''
    try {
      await AITestConnection(cfg as any)
      connStatus.value = cfg.enabled ? 'ok' : 'off'
      connMessage.value = 'Terhubung'
      return true
    } catch (e: any) {
      connStatus.value = 'error'
      connMessage.value = e?.message || String(e)
      return false
    } finally {
      testing.value = false
    }
  }

  async function listModels(cfg: AIConfig): Promise<string[]> {
    return ((await AIListModels(cfg as any)) as string[]) || []
  }

  async function suggest(contactName: string, lastMessages: string) {
    if (!config.value.enabled) {
      suggestions.value = []
      return
    }
    suggesting.value = true
    try {
      const res = (await AISuggestReplies(contactName, lastMessages)) || []
      suggestions.value = res
    } catch (e) {
      console.error(e)
      suggestions.value = []
    } finally {
      suggesting.value = false
    }
  }

  function clearSuggestions() {
    suggestions.value = []
  }

  async function summarize(text: string) {
    return (await AISummarize(text)) || ''
  }

  async function compose(prompt: string, tone: string = 'friendly') {
    composing.value = true
    try {
      return (await AICompose(prompt, tone)) || ''
    } finally {
      composing.value = false
    }
  }

  // --- Image Generation ---

  let generating = ref(false)

  async function generateImage(prompt: string, opts: ImageOptions = {}): Promise<ImageResult[]> {
    if (!config.value.enabled) throw new Error('AI tidak aktif')
    generating.value = true
    try {
      return (await AIGenerateImage(prompt, opts as any)) || []
    } finally {
      generating.value = false
    }
  }

  async function getGamAPIModels(): Promise<string[]> {
    try { return (await AIGetGamAPIModels()) || [] } catch { return [] }
  }
  async function getGamAPIStyles(): Promise<Record<string, string>> {
    try { return (await AIGetGamAPIStyles()) || {} } catch { return {} }
  }
  async function getGamAPIRatios(): Promise<Record<string, string>> {
    try { return (await AIGetGamAPIRatios()) || {} } catch { return {} }
  }

  return {
    config,
    loaded,
    suggestions,
    suggesting,
    composing,
    connStatus,
    connMessage,
    testing,
    load,
    save,
    testConnection,
    listModels,
    suggest,
    clearSuggestions,
    summarize,
    compose,
    generating,
    generateImage,
    getGamAPIModels,
    getGamAPIStyles,
    getGamAPIRatios,
  }
})
