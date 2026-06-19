<script setup lang="ts">
import { ref, computed } from 'vue'
import { Plus, MessageCircle, Circle, Settings as SettingsIcon, Sparkles, Headset, Image, User, Star, Search, Archive, Bot, TrendingUp } from '@lucide/vue'
import { useChatStore } from '../stores/chat'
import { useStatusStore } from '../stores/status'
import { useUIStore } from '../stores/ui'
import { useAIStore } from '../stores/ai'
import { useCSBotStore } from '../stores/csbot'
import QrisDashboard from './QrisDashboard.vue'

const store = useChatStore()
const status = useStatusStore()
const ui = useUIStore()
const ai = useAIStore()
const cs = useCSBotStore()

const emit = defineEmits<{ (e: 'open-settings'): void }>()

const showQrisDashboard = ref(false)
async function handleQrisSendToChat(amount: number, qrDataUrl: string) {
  const caption = `💳 Invoice QRIS - Rp ${new Intl.NumberFormat('id-ID').format(amount)}`
  await store.sendImageBlob(qrDataUrl, caption)
  showQrisDashboard.value = false
}

const initials = (name: string) =>
  name
    .split(' ')
    .map((s) => s[0])
    .slice(0, 2)
    .join('')
    .toUpperCase()

const accounts = computed(() => store.accounts)

const hasNewStatus = computed(() => status.grouped.length > 0)

const aiDotColor = computed(() => {
  if (!ai.config.enabled) return 'bg-gray-400'
  switch (ai.connStatus) {
    case 'ok':
      return 'bg-emerald-500'
    case 'error':
      return 'bg-red-500'
    default:
      return 'bg-amber-400'
  }
})

const csDotColor = computed(() => {
  if (!cs.config.enabled) return 'bg-gray-400'
  switch (cs.connStatus) {
    case 'ok':
      return 'bg-blue-500'
    case 'error':
      return 'bg-red-500'
    default:
      return 'bg-amber-400'
  }
})
</script>

<template>
  <aside class="w-[68px] bg-wa-panel dark:bg-[#111b21] border-r border-wa-border dark:border-wa-border-dark flex flex-col items-center py-3 gap-2">
    <button
      v-for="acc in accounts"
      :key="acc.id"
      @click="store.selectAccount(acc.id); status.showStatusPanel = false"
      class="relative w-11 h-11 rounded-full flex items-center justify-center text-white font-semibold text-sm transition-all"
      :class="[
        store.activeAccountId === acc.id
          ? 'bg-wa-green ring-2 ring-wa-green ring-offset-2 ring-offset-wa-panel dark:ring-offset-[#111b21]'
          : 'bg-slate-400 hover:bg-slate-500',
      ]"
      :title="acc.name + ' (' + acc.phone + ')'"
    >
      {{ initials(acc.name) }}
      <span
        class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-wa-panel dark:border-[#111b21]"
        :class="acc.connected ? 'bg-emerald-500' : 'bg-gray-400'"
      />
    </button>

    <button
      @click="store.startLogin('')"
      class="w-11 h-11 rounded-full flex items-center justify-center bg-wa-hover dark:bg-wa-hover-dark text-wa-muted dark:text-wa-muted-dark hover:bg-wa-green hover:text-white transition"
      title="Tambah akun"
    >
      <Plus :size="20" />
    </button>

    <div class="flex-1" />

    <button
      @click="status.showStatusPanel = false; ui.showPlayground = false; ui.showImageGen = false"
      class="w-11 h-11 rounded-full flex items-center justify-center transition"
      :class="!status.showStatusPanel && !ui.showPlayground ? 'bg-wa-green/10 text-wa-green' : 'text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark'"
      title="Chat"
    >
      <MessageCircle :size="20" />
    </button>

    <button
      @click="status.showStatusPanel = true; ui.showPlayground = false; ui.showImageGen = false"
      class="relative w-11 h-11 rounded-full flex items-center justify-center transition"
      :class="status.showStatusPanel && !ui.showPlayground ? 'bg-wa-green/10 text-wa-green' : 'text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark'"
      title="Status"
    >
      <Circle :size="20" />
      <span
        v-if="hasNewStatus"
        class="absolute top-1.5 right-1.5 w-2 h-2 rounded-full bg-wa-green"
      />
    </button>

    <button
      @click="ui.showPlayground = true; ui.showImageGen = false"
      class="relative w-11 h-11 rounded-full flex items-center justify-center transition"
      :class="ui.showPlayground ? 'bg-violet-500/15 text-violet-500' : 'text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark'"
      title="AI Playground"
    >
      <Bot :size="20" />
    </button>

    <button
      @click="ui.showImageGen = true; ui.showPlayground = false"
      class="relative w-11 h-11 rounded-full flex items-center justify-center transition"
      :class="ui.showImageGen ? 'bg-fuchsia-500/15 text-fuchsia-500' : 'text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark'"
      title="AI Image Generator"
    >
      <Image :size="20" />
    </button>

    <button
      @click="ui.showSearch = true"
      class="w-11 h-11 rounded-full flex items-center justify-center text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition"
      title="Cari"
    >
      <Search :size="20" />
    </button>

    <button
      @click="ui.showStarred = true"
      class="w-11 h-11 rounded-full flex items-center justify-center text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition"
      title="Pesan berbintang"
    >
      <Star :size="20" />
    </button>

    <button
      @click="ui.showAISettings = true"
      class="relative w-11 h-11 rounded-full flex items-center justify-center text-violet-500 hover:bg-violet-500/10 transition"
      title="AI Assistant"
    >
      <Sparkles :size="20" />
      <span
        class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-wa-panel dark:border-[#111b21]"
        :class="aiDotColor"
      />
    </button>

    <button
      @click="ui.showCSBotSettings = true"
      class="relative w-11 h-11 rounded-full flex items-center justify-center text-blue-500 hover:bg-blue-500/10 transition"
      title="CS Bot"
    >
      <Headset :size="20" />
      <span
        class="absolute -bottom-0.5 -right-0.5 w-3 h-3 rounded-full border-2 border-wa-panel dark:border-[#111b21]"
        :class="csDotColor"
      />
    </button>

    <button
      @click="showQrisDashboard = true"
      class="w-11 h-11 rounded-full flex items-center justify-center text-green-500 hover:bg-green-500/10 transition"
      title="Dashboard QRIS"
    >
      <TrendingUp :size="20" />
    </button>

    <button
      @click="ui.showProfile = true"
      class="w-11 h-11 rounded-full flex items-center justify-center text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition"
      title="Profil"
    >
      <User :size="20" />
    </button>

    <button
      @click="emit('open-settings')"
      class="w-11 h-11 rounded-full flex items-center justify-center text-wa-muted dark:text-wa-muted-dark hover:bg-wa-hover dark:hover:bg-wa-hover-dark transition"
      title="Pengaturan"
    >
      <SettingsIcon :size="20" />
    </button>

    <QrisDashboard v-if="showQrisDashboard" @close="showQrisDashboard = false" @send-to-chat="handleQrisSendToChat" />
  </aside>
</template>

