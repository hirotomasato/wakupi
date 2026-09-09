<script setup lang="ts">
import { X, Sun, Moon, Monitor, LogOut, ExternalLink, Heart } from '@lucide/vue'
import { useSettingsStore } from '../stores/settings'
import { useChatStore } from '../stores/chat'

const settings = useSettingsStore()
const chat = useChatStore()

defineProps<{ open: boolean }>()
const emit = defineEmits<{ (e: 'close'): void }>()

function close() {
  emit('close')
}

async function logoutAccount(id: string) {
  if (!confirm('Yakin keluar dari akun ini?')) return
  await chat.logout(id)
}
</script>

<template>
  <div
    v-if="open"
    class="fixed inset-0 z-40 flex items-center justify-center bg-black/50"
    @click.self="close"
  >
    <div class="w-[520px] max-w-[92vw] bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl overflow-hidden">
      <header class="flex items-center justify-between px-6 py-4 border-b border-wa-border dark:border-wa-border-dark">
        <h2 class="text-lg font-semibold text-wa-text dark:text-wa-text-dark">Pengaturan</h2>
        <button @click="close" class="w-8 h-8 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark">
          <X :size="18" />
        </button>
      </header>

      <div class="p-6 space-y-6 max-h-[70vh] overflow-y-auto">
        <section>
          <h3 class="text-sm font-semibold mb-2 text-wa-text dark:text-wa-text-dark">Tema</h3>
          <div class="grid grid-cols-3 gap-2">
            <button
              @click="settings.setTheme('light')"
              class="flex flex-col items-center gap-1 p-3 rounded-lg border-2 transition"
              :class="settings.theme === 'light' ? 'border-wa-green bg-wa-green/5' : 'border-wa-border dark:border-wa-border-dark'"
            >
              <Sun :size="22" class="text-amber-500" />
              <span class="text-xs">Terang</span>
            </button>
            <button
              @click="settings.setTheme('dark')"
              class="flex flex-col items-center gap-1 p-3 rounded-lg border-2 transition"
              :class="settings.theme === 'dark' ? 'border-wa-green bg-wa-green/5' : 'border-wa-border dark:border-wa-border-dark'"
            >
              <Moon :size="22" class="text-indigo-500" />
              <span class="text-xs">Gelap</span>
            </button>
            <button
              @click="settings.setTheme('auto')"
              class="flex flex-col items-center gap-1 p-3 rounded-lg border-2 transition"
              :class="settings.theme === 'auto' ? 'border-wa-green bg-wa-green/5' : 'border-wa-border dark:border-wa-border-dark'"
            >
              <Monitor :size="22" class="text-wa-green" />
              <span class="text-xs">Sistem</span>
            </button>
          </div>
        </section>

        <section>
          <h3 class="text-sm font-semibold mb-2 text-wa-text dark:text-wa-text-dark">Notifikasi</h3>
          <label class="flex items-center justify-between py-2 cursor-pointer">
            <span class="text-sm">Tampilkan notifikasi pesan</span>
            <input v-model="settings.notifications" type="checkbox" class="w-4 h-4 accent-wa-green" />
          </label>
          <label class="flex items-center justify-between py-2 cursor-pointer">
            <span class="text-sm">Suara notifikasi</span>
            <input v-model="settings.notificationSound" type="checkbox" class="w-4 h-4 accent-wa-green" />
          </label>
        </section>

        <section>
          <h3 class="text-sm font-semibold mb-2 text-wa-text dark:text-wa-text-dark">Pesan</h3>
          <label class="flex items-center justify-between py-2 cursor-pointer">
            <span class="text-sm">Tekan Enter untuk kirim</span>
            <input v-model="settings.enterToSend" type="checkbox" class="w-4 h-4 accent-wa-green" />
          </label>
        </section>

        <section>
          <h3 class="text-sm font-semibold mb-2 text-wa-text dark:text-wa-text-dark">Akun</h3>
          <ul class="space-y-2">
            <li
              v-for="acc in chat.accounts"
              :key="acc.id"
              class="flex items-center gap-3 p-3 rounded-lg bg-wa-panel dark:bg-wa-hover-dark"
            >
              <div class="w-10 h-10 rounded-full bg-wa-green text-white flex items-center justify-center font-semibold">
                {{ (acc.name || acc.phone)[0]?.toUpperCase() }}
              </div>
              <div class="flex-1 min-w-0">
                <div class="font-medium truncate">{{ acc.name }}</div>
                <div class="text-xs text-wa-muted dark:text-wa-muted-dark">{{ acc.phone }}</div>
              </div>
              <span
                class="text-xs px-2 py-0.5 rounded-full"
                :class="acc.connected ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'"
              >
                {{ acc.connected ? 'Online' : 'Offline' }}
              </span>
              <button
                @click="logoutAccount(acc.id)"
                class="w-8 h-8 rounded-full hover:bg-red-100 dark:hover:bg-red-900/30 flex items-center justify-center text-red-500"
                title="Keluar"
              >
                <LogOut :size="16" />
              </button>
            </li>
            <li v-if="chat.accounts.length === 0" class="text-center text-sm text-wa-muted dark:text-wa-muted-dark py-4">
              Belum ada akun
            </li>
          </ul>
          <button
            @click="chat.startLogin('')"
            class="mt-3 w-full bg-wa-green hover:bg-wa-green-dark text-white py-2 rounded-lg text-sm"
          >
            Tambah akun
          </button>
        </section>

        <!-- About -->
        <section>
          <h3 class="text-sm font-semibold mb-3 text-wa-text dark:text-wa-text-dark">Tentang</h3>
          <div class="text-center space-y-3 p-4 rounded-lg bg-wa-panel dark:bg-wa-hover-dark">
            <div class="text-2xl font-bold text-wa-green">Wakupi</div>
            <p class="text-sm text-wa-muted dark:text-wa-muted-dark">
              WhatsApp desktop client — AI & QRIS universal untuk bisnis.
            </p>
            <div class="text-xs text-wa-muted dark:text-wa-muted-dark space-y-1">
              <p>Made with <Heart :size="12" class="inline text-red-500 fill-red-500" /> by <strong>Masanto</strong></p>
              <p>v1.0.0 • Linux & Windows</p>
            </div>
            <a
              href="https://github.com/hirotomasato/wakupi"
              target="_blank"
              class="inline-flex items-center gap-2 px-4 py-2 bg-gray-800 dark:bg-gray-700 text-white rounded-lg hover:bg-gray-900 dark:hover:bg-gray-600 transition text-sm"
            >
              <ExternalLink :size="16" />
              github.com/hirotomasato/wakupi
            </a>
          </div>
        </section>
      </div>
    </div>
  </div>
</template>
