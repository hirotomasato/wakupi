import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetCSBotConfig, SetCSBotConfig, CSBotTestConnection } from '../../wailsjs/go/main/App'

export interface CSBotConfig {
  provider: string
  apiKey: string
  baseUrl: string
  model: string
  enabled: boolean
  systemPrompt: string
  greetingMsg: string
  useGreeting: boolean
}

export type ConnStatus = 'off' | 'unknown' | 'ok' | 'error'

export const defaults: CSBotConfig = {
  provider: 'openai',
  apiKey: '',
  baseUrl: '',
  model: '',
  enabled: false,
  systemPrompt: '',
  greetingMsg: '',
  useGreeting: false,
}

export const useCSBotStore = defineStore('csbot', () => {
  const config = ref<CSBotConfig>({ ...defaults })
  const loaded = ref(false)
  const connStatus = ref<ConnStatus>('off')
  const connMessage = ref('')
  const testing = ref(false)

  async function load() {
    try {
      const cfg = (await GetCSBotConfig()) as any
      config.value = { ...defaults, ...cfg }
    } catch (e) {
      console.error('load CS Bot config', e)
    }
    loaded.value = true
    connStatus.value = config.value.enabled ? 'unknown' : 'off'
  }

  async function save(cfg: CSBotConfig) {
    await SetCSBotConfig(cfg as any)
    config.value = { ...cfg }
    if (!cfg.enabled) connStatus.value = 'off'
  }

  async function testConnection(cfg: CSBotConfig): Promise<boolean> {
    testing.value = true
    connMessage.value = ''
    try {
      await CSBotTestConnection(cfg as any)
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

  return {
    config,
    loaded,
    connStatus,
    connMessage,
    testing,
    load,
    save,
    testConnection,
  }
})
