<script setup lang="ts">
import { onMounted } from 'vue'
import { MessageSquareText, Image as ImageIcon, BarChart3 } from '@lucide/vue'
import { usePlaygroundStore } from '../../stores/playground'
import { useUIStore } from '../../stores/ui'
import PlaygroundSessions from './PlaygroundSessions.vue'
import PlaygroundChat from './PlaygroundChat.vue'
import PlaygroundImages from './PlaygroundImages.vue'
import PlaygroundMarket from './PlaygroundMarket.vue'
import PlaygroundParams from './PlaygroundParams.vue'
import SendToWhatsAppModal from './SendToWhatsAppModal.vue'

const pg = usePlaygroundStore()
const ui = useUIStore()

onMounted(() => pg.bindEvents())
</script>

<template>
  <div class="flex-1 flex h-full min-w-0">
    <!-- Left: sessions (collapsible) -->
    <transition name="pg-slide-left">
      <div v-show="!ui.pgLeftCollapsed" class="w-[260px] shrink-0 h-full">
        <PlaygroundSessions />
      </div>
    </transition>

    <!-- Center: chat / image / market, with tab toggle -->
    <div class="flex-1 min-w-0 h-full flex flex-col">
      <!-- Tab bar -->
      <div class="flex items-center gap-1 px-3 pt-2 pb-0 border-b border-wa-border dark:border-wa-border-dark">
        <button
          @click="pg.pgTab = 'chat'"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-t-lg text-sm font-medium transition"
          :class="pg.pgTab === 'chat'
            ? 'bg-white dark:bg-[#0b141a] text-wa-green border border-b-white dark:border-b-[#0b141a] border-wa-border dark:border-wa-border-dark'
            : 'text-wa-muted dark:text-wa-muted-dark hover:text-wa-text dark:hover:text-wa-text-dark'"
        >
          <MessageSquareText :size="16" /> Chat
        </button>
        <button
          @click="pg.pgTab = 'image'"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-t-lg text-sm font-medium transition"
          :class="pg.pgTab === 'image'
            ? 'bg-white dark:bg-[#0b141a] text-wa-green border border-b-white dark:border-b-[#0b141a] border-wa-border dark:border-wa-border-dark'
            : 'text-wa-muted dark:text-wa-muted-dark hover:text-wa-text dark:hover:text-wa-text-dark'"
        >
          <ImageIcon :size="16" /> Image
        </button>
        <button
          @click="pg.pgTab = 'market'"
          class="flex items-center gap-1.5 px-3 py-1.5 rounded-t-lg text-sm font-medium transition"
          :class="pg.pgTab === 'market'
            ? 'bg-white dark:bg-[#0b141a] text-wa-green border border-b-white dark:border-b-[#0b141a] border-wa-border dark:border-wa-border-dark'
            : 'text-wa-muted dark:text-wa-muted-dark hover:text-wa-text dark:hover:text-wa-text-dark'"
        >
          <BarChart3 :size="16" /> Market
        </button>
      </div>

      <div class="flex-1 min-h-0">
        <PlaygroundChat v-show="pg.pgTab === 'chat'" />
        <PlaygroundImages v-show="pg.pgTab === 'image'" />
        <PlaygroundMarket v-if="pg.pgTab === 'market'" />
      </div>
    </div>

    <!-- Right: parameters (collapsible — hidden for market) -->
    <transition name="pg-slide-right">
      <div v-show="!ui.pgRightCollapsed && pg.pgTab !== 'market'" class="w-[300px] shrink-0 h-full">
        <PlaygroundParams />
      </div>
    </transition>

    <SendToWhatsAppModal />
  </div>
</template>

<style scoped>
.pg-slide-left-enter-active,
.pg-slide-left-leave-active,
.pg-slide-right-enter-active,
.pg-slide-right-leave-active {
  transition: opacity 0.18s ease;
}
.pg-slide-left-enter-from,
.pg-slide-left-leave-to,
.pg-slide-right-enter-from,
.pg-slide-right-leave-to {
  opacity: 0;
}
</style>
