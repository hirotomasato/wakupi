import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  GetImageGenConfig,
  SetImageGenConfig,
  ImageGenTestConnection,
  AIGenerateImage,
  AIGetGamAPIModels,
  AIGetGamAPIStyles,
  AIGetGamAPIRatios,
} from '../../wailsjs/go/main/App'

export type ConnStatus = 'off' | 'unknown' | 'ok' | 'error'

export interface ImageGenConfig {
  provider: string
  apiKey: string
  baseUrl: string
  model: string
  enabled: boolean
}

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

const defaults: ImageGenConfig = {
  provider: 'openai',
  apiKey: '',
  baseUrl: '',
  model: '',
  enabled: false,
}

export const useImageGenStore = defineStore('imagegen', () => {
  const config = ref<ImageGenConfig>({ ...defaults })
  const loaded = ref(false)
  const connStatus = ref<ConnStatus>('off')
  const connMessage = ref('')
  const testing = ref(false)

  const prompt = ref('')
  const images = ref<ImageResult[]>([])
  const generating = ref(false)
  const error = ref('')

  // Image parameters
  const imgModel = ref('imagen-3-flash')
  const imgStyle = ref('illustration')
  const imgRatio = ref('square')
  const imgSize = ref('1024x1024')

  // GamAPI cached data
  const gamapiModels = ref<string[]>([])
  const gamapiStyles = ref<Record<string, string>>({})
  const gamapiRatios = ref<Record<string, string>>({})

  const isGamAPI = computed(() => config.value.provider === 'gamapi')
  const canGenerate = computed(() => prompt.value.trim().length > 0 && !generating.value && config.value.enabled)

  // ===== Config management =====
  async function loadConfig() {
    try {
      const cfg = (await GetImageGenConfig()) as any
      config.value = { ...defaults, ...cfg }
    } catch (e) {
      console.error('load ImageGen config', e)
    }
    loaded.value = true
    connStatus.value = config.value.enabled ? 'unknown' : 'off'
  }

  async function saveConfig(cfg: ImageGenConfig) {
    await SetImageGenConfig(cfg as any)
    config.value = { ...cfg }
    if (!cfg.enabled) connStatus.value = 'off'
  }

  async function testConnection(cfg: ImageGenConfig): Promise<boolean> {
    testing.value = true
    connMessage.value = ''
    try {
      await ImageGenTestConnection(cfg as any)
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

  // ===== Image generation =====
  async function generate() {
    if (!canGenerate.value) return
    generating.value = true
    error.value = ''
    images.value = []
    try {
      const opts: ImageOptions = { model: imgModel.value }
      if (isGamAPI.value) {
        opts.style = imgStyle.value
        opts.aspectRatio = imgRatio.value
      }
      if (config.value.provider === 'openai') {
        opts.size = imgSize.value
      }
      const results = (await AIGenerateImage(prompt.value, opts as any)) || []
      images.value = results as ImageResult[]
    } catch (e: any) {
      error.value = e?.message || String(e)
    } finally {
      generating.value = false
    }
  }

  async function loadGamAPIData() {
    const [m, s, r] = await Promise.all([
      AIGetGamAPIModels().catch(() => [] as string[]),
      AIGetGamAPIStyles().catch(() => ({} as Record<string, string>)),
      AIGetGamAPIRatios().catch(() => ({} as Record<string, string>)),
    ])
    gamapiModels.value = m
    gamapiStyles.value = s
    gamapiRatios.value = r
  }

  return {
    config,
    loaded,
    connStatus,
    connMessage,
    testing,
    prompt,
    images,
    generating,
    error,
    imgModel,
    imgStyle,
    imgRatio,
    imgSize,
    isGamAPI,
    canGenerate,
    gamapiModels,
    gamapiStyles,
    gamapiRatios,
    loadConfig,
    saveConfig,
    testConnection,
    generate,
    loadGamAPIData,
  }
})
