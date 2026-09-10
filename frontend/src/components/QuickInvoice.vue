<script setup lang="ts">
import { ref, watch } from 'vue'
import QRCode from 'qrcode'
import { X, Calculator, Send, Download, QrCode } from '@lucide/vue'
import { useQrisStore } from '../stores/qris'
import { useChatStore } from '../stores/chat'
import { makeDynamicQRIS } from '../lib/qris'

const emit = defineEmits<{
  close: []
  sendToChat: [payload: { amount: number; uniqueAmount: number; expiresAt: number; notes: string; qrDataUrl: string }]
}>()

const qrisStore = useQrisStore()
const chatStore = useChatStore()

const amount = ref<number>(0)
const notes = ref('')
const generatedQrDataUrl = ref('')
const isProcessing = ref(false)
const error = ref('')
const displayAmount = ref<number>(0)
const displayUniqueAmount = ref<number>(0)
const displayExpiresAt = ref<number>(0)
const isBackendPayment = ref(false)
const paymentId = ref('')
const transactionId = ref('')

// Auto-dismiss QR when payment is settled or expired.
watch(
  () => {
    const txn = qrisStore.transactions.find((t) => t.id === transactionId.value)
    return txn?.status
  },
  (status) => {
    if (status === 'paid') {
      error.value = ''
      // Keep QR visible for a moment so user sees the success, then auto-close
      setTimeout(() => emit('close'), 2000)
    } else if (status === 'cancelled') {
      error.value = 'QR sudah kadaluarsa / dibatalkan.'
      setTimeout(() => {
        error.value = ''
        emit('close')
      }, 2000)
    }
  }
)

async function generateQr() {
  if (!qrisStore.qrisString) {
    error.value = 'Upload QRIS dulu di Dashboard (icon hijau di sidebar)'
    return
  }
  if (amount.value <= 0) {
    error.value = 'Masukkan nominal valid'
    return
  }

  isProcessing.value = true
  error.value = ''

  try {
    let newQrisString: string
    let uniqueAmount = amount.value
    let expiresAt = 0
    let backendId = ''

    if (qrisStore.session.loggedIn) {
      const result = await qrisStore.createPayment(amount.value, notes.value)
      if (!result || !result.qrString) {
        throw new Error('Backend returned no QRIS')
      }
      newQrisString = result.qrString
      uniqueAmount = result.uniqueAmount
      expiresAt = result.expiresAt
      backendId = result.id
      isBackendPayment.value = true
      paymentId.value = result.id
    } else {
      newQrisString = makeDynamicQRIS(qrisStore.qrisString, amount.value)
      isBackendPayment.value = false
    }

    displayAmount.value = amount.value
    displayUniqueAmount.value = uniqueAmount
    displayExpiresAt.value = expiresAt

    generatedQrDataUrl.value = await QRCode.toDataURL(newQrisString, {
      width: 260,
      margin: 2,
      errorCorrectionLevel: 'M',
    })

    const txn = qrisStore.addTransaction({
      paymentId: backendId || undefined,
      amount: amount.value,
      uniqueAmount: uniqueAmount !== amount.value ? uniqueAmount : undefined,
      expiresAt: expiresAt || undefined,
      qrDataUrl: generatedQrDataUrl.value,
      notes: notes.value,
    })
    transactionId.value = txn.id
  } catch (e: unknown) {
    error.value = 'Gagal generate QR: ' + (e instanceof Error ? e.message : String(e))
  } finally {
    isProcessing.value = false
  }
}

function formatTime(ms: number): string {
  if (!ms) return ''
  return new Date(ms).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' })
}

function formatRupiah(value: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
  }).format(value)
}

function downloadQr() {
  if (!generatedQrDataUrl.value) return
  const link = document.createElement('a')
  link.href = generatedQrDataUrl.value
  link.download = `invoice-${amount.value}.png`
  link.click()
}

function sendToChat() {
  if (generatedQrDataUrl.value && chatStore.activeChat) {
    emit('sendToChat', {
      amount: displayAmount.value,
      uniqueAmount: displayUniqueAmount.value,
      expiresAt: displayExpiresAt.value,
      notes: notes.value,
      qrDataUrl: generatedQrDataUrl.value,
    })
  }
}

// Quick amounts
const quickAmounts = [10000, 15000, 20000, 25000, 50000, 100000]
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl w-full max-w-sm shadow-2xl">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700">
        <div class="flex items-center gap-2">
          <Calculator class="w-5 h-5 text-wa-green" />
          <h2 class="font-semibold">Buat Invoice</h2>
        </div>
        <button @click="emit('close')" class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Content -->
      <div class="p-4 space-y-4">
        <!-- Error -->
        <div v-if="error" class="p-3 bg-red-50 dark:bg-red-900/30 text-red-600 text-sm rounded-lg">
          {{ error }}
        </div>

        <!-- QRIS Status -->
        <div v-if="!qrisStore.qrisString" class="p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg text-sm text-amber-700">
          ⚠️ Upload QRIS dulu di Dashboard QRIS (menu sebelah kiri)
        </div>

        <!-- Amount Input -->
        <div>
          <label class="block text-sm text-gray-500 mb-1">Nominal (Rp)</label>
          <input
            v-model.number="amount"
            type="number"
            placeholder="Masukkan nominal"
            class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-lg font-semibold"
          />
        </div>

        <!-- Quick Amounts -->
        <div class="flex flex-wrap gap-2">
          <button
            v-for="val in quickAmounts"
            :key="val"
            @click="amount = val"
            class="px-3 py-1.5 text-xs bg-gray-100 dark:bg-gray-700 hover:bg-wa-green hover:text-white rounded-full transition-colors"
          >
            {{ formatRupiah(val) }}
          </button>
        </div>

        <!-- Notes -->
        <div>
          <label class="block text-sm text-gray-500 mb-1">Keterangan (opsional)</label>
          <input
            v-model="notes"
            type="text"
            placeholder="Contoh: Nasi Goreng"
            class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700 text-sm"
          />
        </div>

        <!-- Generated QR -->
        <div v-if="generatedQrDataUrl" class="text-center space-y-3">
          <img :src="generatedQrDataUrl" alt="QR Invoice" class="w-52 h-52 mx-auto rounded-lg" />
          <div v-if="isBackendPayment && displayUniqueAmount !== displayAmount" class="text-sm text-gray-500">
            Bayar tepat: <span class="font-bold text-wa-green">{{ formatRupiah(displayUniqueAmount) }}</span>
          </div>
          <div v-else class="text-2xl font-bold text-wa-green">{{ formatRupiah(displayAmount) }}</div>
          <div v-if="displayExpiresAt" class="text-xs text-amber-600">
            ⏳ Berlaku sampai {{ formatTime(displayExpiresAt) }}
          </div>
          <div v-if="isBackendPayment" class="text-xs text-gray-400">
            Status dipantau otomatis
          </div>
          <div class="flex gap-2">
            <button
              @click="downloadQr"
              class="flex-1 py-2 border dark:border-gray-600 rounded-lg flex items-center justify-center gap-1 text-sm"
            >
              <Download :size="16" /> Download
            </button>
            <button
              v-if="chatStore.activeChat"
              @click="sendToChat"
              class="flex-1 bg-wa-green hover:bg-wa-green-dark text-white py-2 rounded-lg flex items-center justify-center gap-1 text-sm"
            >
              <Send :size="16" /> Kirim
            </button>
          </div>
        </div>

        <!-- Generate Button -->
        <button
          v-else
          @click="generateQr"
          :disabled="isProcessing || amount <= 0 || !qrisStore.qrisString"
          class="w-full bg-wa-green hover:bg-wa-green-dark disabled:opacity-50 disabled:cursor-not-allowed text-white py-2.5 rounded-lg font-medium flex items-center justify-center gap-2"
        >
          <QrCode :size="18" />
          {{ isProcessing ? 'Memproses...' : 'Generate QR' }}
        </button>
      </div>
    </div>
  </div>
</template>
