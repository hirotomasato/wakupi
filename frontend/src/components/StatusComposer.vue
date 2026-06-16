<script setup lang="ts">
import { ref, computed } from 'vue'
import { X, Send, Image as ImageIcon, Trash2 } from '@lucide/vue'
import { useStatusStore } from '../stores/status'
import { useChatStore } from '../stores/chat'
import { PickFile } from '../../wailsjs/go/main/App'

const status = useStatusStore()
const chat = useChatStore()
const text = ref('')
const sending = ref(false)
const imagePath = ref<string>('')
const imageName = ref<string>('')

const colors = ['#00a884', '#1f2937', '#7c3aed', '#dc2626', '#f59e0b', '#0284c7', '#be185d']
const bgIdx = ref(0)

const isImageMode = computed(() => !!imagePath.value)
const canSend = computed(() => text.value.trim() || imagePath.value)

function close() {
  status.showComposer = false
  text.value = ''
  imagePath.value = ''
  imageName.value = ''
}

async function pickImage() {
  try {
    const path = await PickFile('image')
    if (!path) return
    imagePath.value = path
    imageName.value = path.split('/').pop() || path
    // Switch to a neutral background for images
    bgIdx.value = 1
  } catch {
    // user cancelled
  }
}

function removeImage() {
  imagePath.value = ''
  imageName.value = ''
}

async function send() {
  if (!canSend.value || !chat.activeAccountId) return
  sending.value = true
  try {
    if (imagePath.value) {
      await status.postImagePath(chat.activeAccountId, imagePath.value, text.value)
    } else {
      await status.postText(chat.activeAccountId, text.value)
    }
    close()
  } catch (e) {
    console.error('post status failed', e)
  } finally {
    sending.value = false
  }
}
</script>

<template>
  <div class="fixed inset-0 z-50 bg-black/95 flex flex-col">
    <header class="flex items-center justify-between px-6 py-4 text-white">
      <button @click="close" class="w-10 h-10 rounded-full hover:bg-white/10 flex items-center justify-center">
        <X :size="22" />
      </button>
      <div class="flex items-center gap-2">
        <button
          v-if="!isImageMode"
          v-for="(c, i) in colors"
          :key="c"
          @click="bgIdx = i"
          class="w-8 h-8 rounded-full border-2"
          :class="bgIdx === i ? 'border-white' : 'border-transparent'"
          :style="{ backgroundColor: c }"
        />
        <button
          @click="pickImage"
          :disabled="sending"
          class="w-10 h-10 rounded-full hover:bg-white/10 flex items-center justify-center disabled:opacity-50"
          title="Tambah gambar"
        >
          <ImageIcon :size="22" />
        </button>
      </div>
      <button
        @click="send"
        :disabled="!canSend || sending"
        class="w-12 h-12 rounded-full bg-wa-green text-white flex items-center justify-center disabled:opacity-50"
      >
        <Send :size="22" />
      </button>
    </header>

    <!-- Image preview -->
    <div v-if="isImageMode" class="flex-1 flex items-center justify-center p-8" :style="{ backgroundColor: colors[bgIdx] }">
      <div class="relative max-w-full max-h-full">
        <img
          :src="'file://' + imagePath"
          class="max-w-full max-h-[55vh] rounded-lg object-contain shadow-2xl"
          alt="Status preview"
        />
        <button
          @click="removeImage"
          class="absolute top-2 right-2 w-9 h-9 rounded-full bg-black/60 text-white flex items-center justify-center hover:bg-black/80"
          title="Hapus gambar"
        >
          <Trash2 :size="16" />
        </button>
      </div>
    </div>

    <!-- Text area -->
    <div
      class="flex-1 flex items-center justify-center p-12"
      :class="{ 'pt-4': isImageMode }"
      :style="{ backgroundColor: isImageMode ? colors[bgIdx] : colors[bgIdx] }"
    >
      <div class="w-full max-w-2xl flex flex-col items-center gap-4">
        <textarea
          v-model="text"
          :placeholder="isImageMode ? 'Tambah caption...' : 'Ketik status kamu...'"
          autofocus
          class="w-full bg-transparent text-white font-medium text-center outline-none resize-none placeholder:text-white/40"
          :class="isImageMode ? 'text-xl' : 'text-4xl'"
          rows="4"
        />
        <p v-if="isImageMode" class="text-white/50 text-xs">{{ imageName }}</p>
      </div>
    </div>
  </div>
</template>
