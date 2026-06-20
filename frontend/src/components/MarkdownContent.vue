<script setup lang="ts">
import { computed, ref, onMounted, onBeforeUnmount } from 'vue'
import { renderMarkdown, b64Decode } from '../lib/markdown'
import { CopyToClipboard } from '../../wailsjs/go/main/App'

const props = defineProps<{ text: string }>()
const root = ref<HTMLElement | null>(null)

const html = computed(() => renderMarkdown(props.text))

// Delegate clicks for "Copy" buttons injected into rendered code blocks.
function onClick(e: MouseEvent) {
  const target = (e.target as HTMLElement).closest('.code-block__copy') as HTMLElement | null
  if (!target) return
  const code = b64Decode(target.getAttribute('data-code') || '')
  CopyToClipboard(code).then(() => {
    const original = target.textContent
    target.textContent = 'Copied!'
    target.classList.add('is-copied')
    setTimeout(() => {
      target.textContent = original
      target.classList.remove('is-copied')
    }, 1200)
  })
}

onMounted(() => root.value?.addEventListener('click', onClick))
onBeforeUnmount(() => root.value?.removeEventListener('click', onClick))
</script>

<template>
  <div ref="root" class="markdown-body" v-html="html" />
</template>
