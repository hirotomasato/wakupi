<script setup lang="ts">
import { onMounted } from 'vue'
import { MessageSquareText } from '@lucide/vue'
import { usePlaygroundStore } from '../../stores/playground'
import { useUIStore } from '../../stores/ui'
import PlaygroundSessions from './PlaygroundSessions.vue'
import PlaygroundChat from './PlaygroundChat.vue'
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

    <!-- Center: chat -->
    <div class="flex-1 min-w-0 h-full flex flex-col">
      <div class="flex-1 min-h-0">
        <PlaygroundChat />
      </div>
    </div>

    <!-- Right: parameters (collapsible) -->
    <transition name="pg-slide-right">
      <div v-show="!ui.pgRightCollapsed" class="w-[300px] shrink-0 h-full">
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
