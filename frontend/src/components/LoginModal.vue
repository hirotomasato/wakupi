<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import QRCode from 'qrcode'
import { Smartphone, X, Loader2, CheckCircle2, RefreshCw, QrCode, Phone } from '@lucide/vue'
import { useChatStore } from '../stores/chat'

const store = useChatStore()
const canvas = ref<HTMLCanvasElement | null>(null)
const accountName = ref('')
const phoneInput = ref('')

async function render(code: string) {
  if (!canvas.value || !code) return
  await QRCode.toCanvas(canvas.value, code, {
    width: 264,
    margin: 1,
    color: { dark: '#111b21', light: '#ffffff' },
    errorCorrectionLevel: 'M',
  })
}

watch(
  () => store.qrCode,
  async (code) => {
    if (code) await render(code)
  }
)

onMounted(async () => {
  if (store.qrCode) await render(store.qrCode)
})

async function handleStart() {
  if (store.loginMethod === 'phone') {
    const cleaned = phoneInput.value.replace(/\D/g, '')
    if (!cleaned) return
    await store.startLoginWithPhone(accountName.value.trim(), cleaned)
  } else {
    await store.startLogin(accountName.value.trim())
  }
}

function handleClose() {
  store.showLogin = false
  store.loginMethod = 'qr'
  phoneInput.value = ''
  store.pairCode = ''
}
</script>

<template>
  <div class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 backdrop-blur-sm">
    <div class="bg-white dark:bg-wa-panel-dark rounded-2xl shadow-2xl w-[480px] max-w-[92vw] overflow-hidden">
      <header class="flex items-center justify-between px-6 py-4 border-b border-wa-border dark:border-wa-border-dark">
        <h2 class="text-lg font-semibold text-wa-text dark:text-wa-text-dark">Tambah Akun WhatsApp</h2>
        <button
          v-if="store.accounts.length > 0"
          @click="handleClose"
          class="w-8 h-8 rounded-full hover:bg-wa-hover dark:hover:bg-wa-hover-dark flex items-center justify-center text-wa-muted dark:text-wa-muted-dark"
        >
          <X :size="18" />
        </button>
      </header>

      <div class="p-6">
        <div v-if="store.loginStatus === 'idle'" class="space-y-4">
          <!-- Method toggle -->
          <div class="flex bg-wa-panel dark:bg-wa-hover-dark rounded-lg p-1">
            <button
              @click="store.loginMethod = 'qr'"
              class="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-md text-sm font-medium transition"
              :class="store.loginMethod === 'qr' ? 'bg-white dark:bg-wa-panel-dark text-wa-green shadow-sm' : 'text-wa-muted dark:text-wa-muted-dark'"
            >
              <QrCode :size="16" /> QR Code
            </button>
            <button
              @click="store.loginMethod = 'phone'"
              class="flex-1 flex items-center justify-center gap-1.5 py-2 rounded-md text-sm font-medium transition"
              :class="store.loginMethod === 'phone' ? 'bg-white dark:bg-wa-panel-dark text-wa-green shadow-sm' : 'text-wa-muted dark:text-wa-muted-dark'"
            >
              <Phone :size="16" /> Nomor HP
            </button>
          </div>

          <div v-if="store.loginMethod === 'phone'" class="flex items-center gap-3 p-3 bg-wa-panel dark:bg-wa-hover-dark rounded-lg">
            <Phone :size="22" class="text-wa-green" />
            <p class="text-sm text-wa-text dark:text-wa-text-dark">
              Masukkan nomor WhatsApp-mu, lalu masukkan kode yang muncul di WhatsApp HP-mu.
            </p>
          </div>
          <div v-else class="flex items-center gap-3 p-3 bg-wa-panel dark:bg-wa-hover-dark rounded-lg">
            <Smartphone :size="22" class="text-wa-green" />
            <p class="text-sm text-wa-text dark:text-wa-text-dark">
              Buka WhatsApp di HP-mu, lalu pindai QR code untuk menghubungkan akun.
            </p>
          </div>

          <div v-if="store.loginMethod === 'phone'">
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Nomor WhatsApp</label>
            <input
              v-model="phoneInput"
              type="tel"
              placeholder="contoh: 08123456789"
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none text-wa-text dark:text-wa-text-dark"
            />
          </div>
          <div>
            <label class="text-xs font-medium text-wa-muted dark:text-wa-muted-dark">Label akun (opsional)</label>
            <input
              v-model="accountName"
              type="text"
              placeholder="contoh: Toko Saya"
              class="mt-1 w-full bg-wa-panel dark:bg-wa-hover-dark rounded-lg px-3 py-2 text-sm outline-none text-wa-text dark:text-wa-text-dark"
            />
          </div>
          <button
            @click="handleStart"
            class="w-full bg-wa-green hover:bg-wa-green-dark text-white font-medium py-2.5 rounded-lg transition"
          >
            {{ store.loginMethod === 'phone' ? 'Generate Kode' : 'Mulai Pairing' }}
          </button>
        </div>

        <div v-else-if="store.loginStatus === 'waiting'" class="flex flex-col items-center text-center gap-4">
          <div v-if="store.loginMethod === 'phone' && store.pairCode" class="space-y-3">
            <p class="text-sm text-wa-muted dark:text-wa-muted-dark">
              Masukkan kode ini di WhatsApp HP-mu
            </p>
            <div class="text-4xl font-mono font-bold tracking-[0.3em] text-wa-green select-all">{{ store.pairCode }}</div>
            <p class="text-xs text-wa-muted dark:text-wa-muted-dark">
              WhatsApp → Pengaturan → Perangkat Tertaut → Tautkan dengan nomor telepon
            </p>
          </div>
          <template v-else>
            <div class="relative bg-white p-4 rounded-xl border border-wa-border">
              <canvas v-show="store.qrCode" ref="canvas" class="block" />
              <div v-if="!store.qrCode" class="w-[264px] h-[264px] flex items-center justify-center">
                <Loader2 :size="32" class="text-wa-green animate-spin" />
              </div>
            </div>
            <ol class="text-sm text-wa-muted dark:text-wa-muted-dark text-left space-y-1 max-w-[320px]">
              <li>1. Buka WhatsApp di HP-mu</li>
              <li>2. Ketuk Menu / Pengaturan → Perangkat Tertaut</li>
              <li>3. Ketuk "Tautkan Perangkat"</li>
              <li>4. Arahkan kamera ke QR code di atas</li>
            </ol>
          </template>
        </div>

        <div v-else-if="store.loginStatus === 'pairing'" class="flex flex-col items-center text-center gap-3 py-8">
          <Loader2 :size="40" class="text-wa-green animate-spin" />
          <p class="text-sm text-wa-text dark:text-wa-text-dark">Menyambungkan ke WhatsApp...</p>
        </div>

        <div v-else-if="store.loginStatus === 'success'" class="flex flex-col items-center text-center gap-3 py-8">
          <CheckCircle2 :size="48" class="text-wa-green" />
          <p class="text-base font-medium text-wa-text dark:text-wa-text-dark">Akun berhasil terhubung</p>
        </div>

        <div v-else-if="store.loginStatus === 'timeout'" class="flex flex-col items-center text-center gap-4 py-6">
          <p class="text-sm text-wa-muted dark:text-wa-muted-dark">QR code kedaluwarsa. Coba lagi.</p>
          <button
            @click="handleStart"
            class="flex items-center gap-2 bg-wa-green hover:bg-wa-green-dark text-white px-4 py-2 rounded-lg"
          >
            <RefreshCw :size="16" />
            Generate QR Baru
          </button>
        </div>

        <div v-else-if="store.loginStatus === 'error'" class="text-center py-6">
          <p class="text-sm text-red-500">{{ store.loginError || 'Gagal memulai pairing' }}</p>
          <button
            @click="handleStart"
            class="mt-3 bg-wa-green hover:bg-wa-green-dark text-white px-4 py-2 rounded-lg text-sm"
          >
            Coba Lagi
          </button>
        </div>
      </div>
    </div>
  </div>
</template>
