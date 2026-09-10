<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import jsQR from 'jsqr'
import QRCode from 'qrcode'
import {
  X,
  Upload,
  QrCode,
  Plus,
  Trash2,
  Check,
  XCircle,
  Clock,
  TrendingUp,
  DollarSign,
  Package,
  Send,
  Download,
  RefreshCw,
} from '@lucide/vue'
import { useQrisStore } from '../stores/qris'
import { useChatStore } from '../stores/chat'
import { makeDynamicQRIS, parseEmvcoQr } from '../lib/qris'

const emit = defineEmits<{
  close: []
  sendToChat: [payload: { amount: number; uniqueAmount: number; expiresAt: number; notes: string; qrDataUrl: string }]
}>()

const qrisStore = useQrisStore()
const chatStore = useChatStore()

const activeTab = ref<'dashboard' | 'products' | 'history' | 'merchant'>('dashboard')
const showProductForm = ref(false)
const showGenerateForm = ref(false)
const editingProductId = ref<string | null>(null)

const newProduct = ref({ name: '', price: 0, category: '' })
const generateAmount = ref<number>(0)
const generateNotes = ref('')
const generatedQrDataUrl = ref('')
const generatedUniqueAmount = ref(0)
const generatedExpiresAt = ref(0)
const generatedTxnId = ref('')
const generatedPaymentStatus = ref('')
const isProcessing = ref(false)
const error = ref('')
const successMessage = ref('')

// Stats computed
const stats = computed(() => ({
  todaySales: qrisStore.totalToday,
  pendingAmount: qrisStore.totalPending,
  totalProducts: qrisStore.products.length,
  todayTransactions: qrisStore.todayTransactions.length,
}))

// Auto-dismiss generated QR when payment is settled or expired.
watch(
  () => {
    const txn = qrisStore.transactions.find((t) => t.id === generatedTxnId.value)
    return txn?.status
  },
  (status) => {
    if (status === 'paid') {
      generatedPaymentStatus.value = 'paid'
      setTimeout(() => {
        generatedQrDataUrl.value = ''
        generatedTxnId.value = ''
        generatedPaymentStatus.value = ''
      }, 2500)
    } else if (status === 'cancelled') {
      error.value = 'QR sudah kadaluarsa / dibatalkan.'
      setTimeout(() => {
        error.value = ''
        generatedQrDataUrl.value = ''
        generatedTxnId.value = ''
      }, 2000)
    }
  }
)

async function handleFileUpload(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  isProcessing.value = true
  error.value = ''

  try {
    const reader = new FileReader()
    const imageData = await new Promise<ImageData>((resolve, reject) => {
      reader.onload = () => {
        const img = new Image()
        img.onload = () => {
          const canvas = document.createElement('canvas')
          canvas.width = img.width
          canvas.height = img.height
          const ctx = canvas.getContext('2d')!
          ctx.drawImage(img, 0, 0)
          const data = ctx.getImageData(0, 0, img.width, img.height)
          resolve(data)
        }
        img.onerror = reject
        img.src = reader.result as string
      }
      reader.onerror = reject
      reader.readAsDataURL(file)
    })

    const qrCode = jsQR(imageData.data, imageData.width, imageData.height)
    if (!qrCode) {
      error.value = 'Tidak dapat membaca QR code dari gambar'
      return
    }

    const parts = parseEmvcoQr(qrCode.data)
    if (!parts['00']) {
      error.value = 'QR ini bukan QRIS valid'
      return
    }

    // Try to detect QRIS format (field 00 = format indicator, field 26 = merchant account)
    const isQris = parts['00'] === '01' || parts['26']
    if (!isQris) {
      error.value = 'QR ini bukan QRIS standar Indonesia'
      return
    }

    qrisStore.setQris(qrCode.data)
    successMessage.value = 'QRIS berhasil diupload!'
    setTimeout(() => (successMessage.value = ''), 3000)
  } catch (e: unknown) {
    error.value = 'Gagal memproses gambar: ' + (e instanceof Error ? e.message : String(e))
  } finally {
    isProcessing.value = false
  }
}

async function generateQr() {
  if (!qrisStore.qrisString) {
    error.value = 'Upload QRIS terlebih dahulu'
    return
  }
  if (generateAmount.value <= 0) {
    error.value = 'Masukkan nominal yang valid'
    return
  }

  isProcessing.value = true
  error.value = ''

  try {
    let newQrisString: string
    let uniqueAmount = generateAmount.value
    let expiresAt = 0
    let backendId = ''

    if (qrisStore.session.loggedIn) {
      const result = await qrisStore.createPayment(generateAmount.value, generateNotes.value)
      if (!result || !result.qrString) {
        throw new Error('Backend returned no QRIS')
      }
      newQrisString = result.qrString
      uniqueAmount = result.uniqueAmount
      expiresAt = result.expiresAt
      backendId = result.id
    } else {
      newQrisString = makeDynamicQRIS(qrisStore.qrisString, generateAmount.value)
    }

    generatedQrDataUrl.value = await QRCode.toDataURL(newQrisString, {
      width: 280,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
    generatedUniqueAmount.value = uniqueAmount
    generatedExpiresAt.value = expiresAt

    const txn = qrisStore.addTransaction({
      paymentId: backendId || undefined,
      amount: generateAmount.value,
      uniqueAmount: uniqueAmount !== generateAmount.value ? uniqueAmount : undefined,
      expiresAt: expiresAt || undefined,
      qrDataUrl: generatedQrDataUrl.value,
      notes: generateNotes.value,
    })
    generatedTxnId.value = txn.id

    successMessage.value = 'QR berhasil dibuat!'
    generateAmount.value = 0
    generateNotes.value = ''
  } catch (e: unknown) {
    error.value = 'Gagal generate QR: ' + (e instanceof Error ? e.message : String(e))
  } finally {
    isProcessing.value = false
  }
}

function addProduct() {
  if (!newProduct.value.name || newProduct.value.price <= 0) {
    error.value = 'Nama dan harga produk harus diisi'
    return
  }
  qrisStore.addProduct({
    name: newProduct.value.name,
    price: newProduct.value.price,
    category: newProduct.value.category,
  })
  newProduct.value = { name: '', price: 0, category: '' }
  showProductForm.value = false
  successMessage.value = 'Produk berhasil ditambahkan!'
  setTimeout(() => (successMessage.value = ''), 3000)
}

function selectProduct(productId: string) {
  const product = qrisStore.products.find((p) => p.id === productId)
  if (product) {
    generateAmount.value = product.price
    generateNotes.value = product.name
    showGenerateForm.value = true
  }
}

function markAsPaid(transactionId: string) {
  qrisStore.updateTransactionStatus(transactionId, 'paid')
  successMessage.value = 'Transaksi ditandai sudah dibayar!'
  setTimeout(() => (successMessage.value = ''), 3000)
}

function markAsCancelled(transactionId: string) {
  qrisStore.updateTransactionStatus(transactionId, 'cancelled')
}

function sendToChat() {
  if (generatedQrDataUrl.value && chatStore.activeChatId) {
    emit('sendToChat', {
      amount: generatedUniqueAmount.value || generateAmount.value,
      uniqueAmount: generatedUniqueAmount.value,
      expiresAt: generatedExpiresAt.value,
      notes: generateNotes.value,
      qrDataUrl: generatedQrDataUrl.value,
    })
    generatedQrDataUrl.value = ''
  }
}

function formatRupiah(value: number): string {
  return new Intl.NumberFormat('id-ID', {
    style: 'currency',
    currency: 'IDR',
    minimumFractionDigits: 0,
  }).format(value)
}

function formatDate(ts: number): string {
  return new Date(ts).toLocaleString('id-ID', {
    day: '2-digit',
    month: 'short',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function downloadQr(dataUrl: string, filename: string) {
  const link = document.createElement('a')
  link.href = dataUrl
  link.download = filename
  link.click()
}

const pendingTransactions = computed(() =>
  qrisStore.transactions.filter((t) => t.status === 'pending')
)

const paidTransactions = computed(() =>
  qrisStore.transactions.filter((t) => t.status === 'paid')
)

onMounted(() => {
  qrisStore.refreshSession()
})
</script>

<template>
  <div class="fixed inset-0 bg-black/50 flex items-center justify-center z-50 p-4">
    <div class="bg-white dark:bg-gray-800 rounded-xl w-full max-w-4xl h-[90vh] shadow-2xl flex flex-col">
      <!-- Header -->
      <div class="flex items-center justify-between p-4 border-b dark:border-gray-700 shrink-0">
        <div class="flex items-center gap-2">
          <TrendingUp class="w-5 h-5 text-wa-green" />
          <h2 class="font-semibold text-lg">Dashboard QRIS</h2>
        </div>
        <button @click="emit('close')" class="p-1 hover:bg-gray-100 dark:hover:bg-gray-700 rounded">
          <X class="w-5 h-5" />
        </button>
      </div>

      <!-- Tabs -->
      <div class="flex border-b dark:border-gray-700 shrink-0">
        <button
          @click="activeTab = 'dashboard'"
          class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === 'dashboard' ? 'border-wa-green text-wa-green' : 'border-transparent text-gray-500'"
        >
          Dashboard
        </button>
        <button
          @click="activeTab = 'products'"
          class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === 'products' ? 'border-wa-green text-wa-green' : 'border-transparent text-gray-500'"
        >
          Produk
        </button>
        <button
          @click="activeTab = 'history'"
          class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === 'history' ? 'border-wa-green text-wa-green' : 'border-transparent text-gray-500'"
        >
          Riwayat
        </button>
        <button
          @click="activeTab = 'merchant'"
          class="flex-1 px-4 py-2 text-sm font-medium border-b-2 transition-colors"
          :class="activeTab === 'merchant' ? 'border-wa-green text-wa-green' : 'border-transparent text-gray-500'"
        >
          Merchant
        </button>
      </div>

      <!-- Notifications -->
      <div v-if="error" class="mx-4 mt-4 p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded-lg">
        {{ error }}
      </div>
      <div v-if="successMessage" class="mx-4 mt-4 p-3 bg-green-50 dark:bg-green-900/30 text-green-600 dark:text-green-400 text-sm rounded-lg">
        {{ successMessage }}
      </div>

      <!-- Content -->
      <div class="flex-1 overflow-y-auto p-4">
        <!-- Dashboard Tab -->
        <div v-if="activeTab === 'dashboard'" class="space-y-4">
          <!-- Stats Cards -->
          <div class="grid grid-cols-2 md:grid-cols-4 gap-3">
            <div class="bg-wa-green/10 rounded-lg p-3">
              <div class="flex items-center gap-2 text-wa-green mb-1">
                <DollarSign :size="16" />
                <span class="text-xs">Penjualan Hari Ini</span>
              </div>
              <div class="text-xl font-bold text-wa-green">{{ formatRupiah(stats.todaySales) }}</div>
            </div>
            <div class="bg-amber-50 dark:bg-amber-900/20 rounded-lg p-3">
              <div class="flex items-center gap-2 text-amber-600 mb-1">
                <Clock :size="16" />
                <span class="text-xs">Pending</span>
              </div>
              <div class="text-xl font-bold text-amber-600">{{ formatRupiah(stats.pendingAmount) }}</div>
            </div>
            <div class="bg-blue-50 dark:bg-blue-900/20 rounded-lg p-3">
              <div class="flex items-center gap-2 text-blue-600 mb-1">
                <Package :size="16" />
                <span class="text-xs">Produk</span>
              </div>
              <div class="text-xl font-bold text-blue-600">{{ stats.totalProducts }}</div>
            </div>
            <div class="bg-purple-50 dark:bg-purple-900/20 rounded-lg p-3">
              <div class="flex items-center gap-2 text-purple-600 mb-1">
                <TrendingUp :size="16" />
                <span class="text-xs">Transaksi Hari Ini</span>
              </div>
              <div class="text-xl font-bold text-purple-600">{{ stats.todayTransactions }}</div>
            </div>
          </div>

          <!-- QR Upload Section -->
          <div v-if="!qrisStore.qrisString" class="border-2 border-dashed border-gray-300 dark:border-gray-600 rounded-lg p-6 text-center">
            <Upload class="w-10 h-10 mx-auto text-gray-400 mb-2" />
            <p class="text-sm text-gray-600 dark:text-gray-400 mb-2">Upload QRIS statis Anda (cukup sekali)</p>
            <label class="inline-block cursor-pointer bg-wa-green hover:bg-wa-green-dark text-white px-4 py-2 rounded-lg text-sm">
              Pilih Gambar QRIS
              <input type="file" accept="image/*" @change="handleFileUpload" class="hidden" />
            </label>
          </div>

          <div v-else class="bg-gray-50 dark:bg-gray-700 rounded-lg p-4">
            <div class="flex items-center justify-between mb-3">
              <span class="text-sm font-medium">QRIS Aktif</span>
              <label class="text-xs text-wa-green cursor-pointer hover:underline">
                Ganti QRIS
                <input type="file" accept="image/*" @change="handleFileUpload" class="hidden" />
              </label>
            </div>
            <button
              @click="showGenerateForm = true"
              class="w-full bg-wa-green hover:bg-wa-green-dark text-white py-2 rounded-lg font-medium"
            >
              + Buat QR Baru
            </button>
          </div>

          <!-- Quick Products -->
          <div v-if="qrisStore.products.length > 0">
            <h3 class="text-sm font-medium mb-2">Produk Cepat</h3>
            <div class="grid grid-cols-2 md:grid-cols-3 gap-2">
              <button
                v-for="product in qrisStore.products.slice(0, 6)"
                :key="product.id"
                @click="selectProduct(product.id)"
                class="p-3 border dark:border-gray-600 rounded-lg text-left hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors"
              >
                <div class="font-medium text-sm truncate">{{ product.name }}</div>
                <div class="text-wa-green font-semibold text-sm">{{ formatRupiah(product.price) }}</div>
              </button>
            </div>
          </div>

          <!-- Pending Transactions -->
          <div v-if="pendingTransactions.length > 0">
            <h3 class="text-sm font-medium mb-2">Menunggu Pembayaran</h3>
            <div class="space-y-2">
              <div
                v-for="txn in pendingTransactions.slice(0, 5)"
                :key="txn.id"
                class="flex items-center justify-between p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg"
              >
                <div>
                  <div class="font-medium">{{ formatRupiah(txn.amount) }}</div>
                  <div class="text-xs text-gray-500">{{ txn.notes || 'Tanpa keterangan' }}</div>
                  <div class="text-xs text-gray-400">{{ formatDate(txn.createdAt) }}</div>
                </div>
                <div class="flex gap-2">
                  <button
                    @click="markAsPaid(txn.id)"
                    class="p-2 bg-green-500 hover:bg-green-600 text-white rounded-lg"
                    title="Tandai sudah dibayar"
                  >
                    <Check :size="16" />
                  </button>
                  <button
                    @click="markAsCancelled(txn.id)"
                    class="p-2 bg-red-500 hover:bg-red-600 text-white rounded-lg"
                    title="Batalkan"
                  >
                    <XCircle :size="16" />
                  </button>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Products Tab -->
        <div v-else-if="activeTab === 'products'" class="space-y-4">
          <div class="flex justify-between items-center">
            <h3 class="font-medium">Daftar Produk</h3>
            <button
              @click="showProductForm = true"
              class="bg-wa-green hover:bg-wa-green-dark text-white px-3 py-1.5 rounded-lg text-sm flex items-center gap-1"
            >
              <Plus :size="16" /> Tambah Produk
            </button>
          </div>

          <!-- Product Form Modal -->
          <div v-if="showProductForm" class="fixed inset-0 bg-black/50 flex items-center justify-center z-10 p-4" @click.self="showProductForm = false">
            <div class="bg-white dark:bg-gray-800 rounded-lg p-4 w-full max-w-sm">
              <h4 class="font-medium mb-3">Tambah Produk</h4>
              <div class="space-y-3">
                <input
                  v-model="newProduct.name"
                  type="text"
                  placeholder="Nama produk"
                  class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
                <input
                  v-model.number="newProduct.price"
                  type="number"
                  placeholder="Harga (Rp)"
                  class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
                <input
                  v-model="newProduct.category"
                  type="text"
                  placeholder="Kategori (opsional)"
                  class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
                <div class="flex gap-2">
                  <button @click="showProductForm = false" class="flex-1 py-2 border dark:border-gray-600 rounded-lg">
                    Batal
                  </button>
                  <button @click="addProduct" class="flex-1 bg-wa-green hover:bg-wa-green-dark text-white py-2 rounded-lg">
                    Simpan
                  </button>
                </div>
              </div>
            </div>
          </div>

          <!-- Products List -->
          <div v-if="qrisStore.products.length === 0" class="text-center py-8 text-gray-500">
            Belum ada produk. Tambahkan produk untuk akses cepat.
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="product in qrisStore.products"
              :key="product.id"
              class="flex items-center justify-between p-3 border dark:border-gray-600 rounded-lg"
            >
              <div>
                <div class="font-medium">{{ product.name }}</div>
                <div class="text-sm text-gray-500">{{ product.category || 'Tanpa kategori' }}</div>
                <div class="text-wa-green font-semibold">{{ formatRupiah(product.price) }}</div>
              </div>
              <div class="flex gap-2">
                <button
                  @click="selectProduct(product.id); activeTab = 'dashboard'"
                  class="p-2 text-wa-green hover:bg-wa-green/10 rounded-lg"
                  title="Generate QR"
                >
                  <QrCode :size="16" />
                </button>
                <button
                  @click="qrisStore.deleteProduct(product.id)"
                  class="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg"
                  title="Hapus"
                >
                  <Trash2 :size="16" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- History Tab -->
        <div v-else-if="activeTab === 'history'" class="space-y-4">
          <h3 class="font-medium">Riwayat Transaksi</h3>
          <div v-if="qrisStore.transactions.length === 0" class="text-center py-8 text-gray-500">
            Belum ada transaksi.
          </div>
          <div v-else class="space-y-2">
            <div
              v-for="txn in qrisStore.transactions"
              :key="txn.id"
              class="flex items-center justify-between p-3 border dark:border-gray-600 rounded-lg"
              :class="{
                'bg-green-50 dark:bg-green-900/20': txn.status === 'paid',
                'bg-red-50 dark:bg-red-900/20': txn.status === 'cancelled',
                'bg-amber-50 dark:bg-amber-900/20': txn.status === 'pending',
              }"
            >
              <div>
                <div class="flex items-center gap-2">
                  <span class="font-medium">{{ formatRupiah(txn.amount) }}</span>
                  <span
                    class="text-xs px-2 py-0.5 rounded-full"
                    :class="{
                      'bg-green-500 text-white': txn.status === 'paid',
                      'bg-red-500 text-white': txn.status === 'cancelled',
                      'bg-amber-500 text-white': txn.status === 'pending',
                    }"
                  >
                    {{ txn.status === 'paid' ? 'Dibayar' : txn.status === 'cancelled' ? 'Dibatalkan' : 'Pending' }}
                  </span>
                </div>
                <div class="text-sm text-gray-500">{{ txn.notes || 'Tanpa keterangan' }}</div>
                <div class="text-xs text-gray-400">{{ formatDate(txn.createdAt) }}</div>
              </div>
              <div class="flex gap-2">
                <button
                  v-if="txn.qrDataUrl"
                  @click="downloadQr(txn.qrDataUrl, `qris-${txn.amount}-${txn.id}.png`)"
                  class="p-2 text-blue-500 hover:bg-blue-50 dark:hover:bg-blue-900/20 rounded-lg"
                  title="Download QR"
                >
                  <Download :size="16" />
                </button>
                <button
                  v-if="txn.status === 'pending'"
                  @click="markAsPaid(txn.id)"
                  class="p-2 text-green-500 hover:bg-green-50 dark:hover:bg-green-900/20 rounded-lg"
                  title="Tandai dibayar"
                >
                  <Check :size="16" />
                </button>
                <button
                  @click="qrisStore.deleteTransaction(txn.id)"
                  class="p-2 text-red-500 hover:bg-red-50 dark:hover:bg-red-900/20 rounded-lg"
                  title="Hapus"
                >
                  <Trash2 :size="16" />
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- Merchant Tab -->
        <div v-else-if="activeTab === 'merchant'" class="space-y-4">
          <!-- Logged in -->
          <div v-if="qrisStore.session.loggedIn" class="space-y-4">
            <div class="bg-green-50 dark:bg-green-900/20 rounded-lg p-4">
              <div class="flex items-center gap-2 mb-2">
                <div class="w-3 h-3 rounded-full bg-green-500"></div>
                <span class="text-sm font-medium text-green-700 dark:text-green-400">Tersambung</span>
              </div>
              <div class="text-sm text-gray-600 dark:text-gray-400">
                <div>Merchant: <strong>{{ qrisStore.session.merchantName }}</strong></div>
                <div v-if="qrisStore.session.storeId">Store: {{ qrisStore.session.storeId }}</div>
              </div>
            </div>
            <div class="p-4 border dark:border-gray-600 rounded-lg">
              <div class="flex items-center gap-3 mb-3">
                <QrCode class="w-5 h-5 text-wa-green" />
                <span class="text-sm font-medium">QRIS Merchant</span>
              </div>
              <div v-if="!qrisStore.qrisString" class="text-sm text-gray-500 mb-2">
                Upload QRIS statis untuk mulai generate QR dinamis.
              </div>
              <div v-else class="text-sm text-green-600 mb-2">
                QRIS statis sudah tersimpan. Generate QR dinamis via backend Shopee.
              </div>
              <label class="inline-block cursor-pointer bg-gray-100 dark:bg-gray-700 hover:bg-gray-200 dark:hover:bg-gray-600 px-4 py-2 rounded-lg text-sm">
                Upload QRIS Statis
                <input type="file" accept="image/*" @change="handleFileUpload" class="hidden" />
              </label>
            </div>
            <button
              @click="qrisStore.logoutMerchant()"
              class="w-full py-2 border border-red-300 dark:border-red-700 text-red-600 rounded-lg text-sm hover:bg-red-50 dark:hover:bg-red-900/20"
            >
              Logout Merchant
            </button>
          </div>

          <!-- Not logged in: Login flow -->
          <div v-else class="space-y-4">
            <!-- Needs re-login (passport session dead) -->
            <div v-if="qrisStore.session.needsRelogin" class="p-3 bg-amber-50 dark:bg-amber-900/20 rounded-lg text-sm text-amber-700">
              ⚠️ Sesi Shopee sudah kedaluwarsa. Silakan login ulang.
            </div>

            <h3 class="font-medium">Login Shopee Merchant</h3>

            <!-- Step: idle / number -->
            <div v-if="qrisStore.loginStep === 'idle' || qrisStore.loginStep === 'number'" class="space-y-3">
              <p class="text-sm text-gray-500">Masuk dengan nomor HP Shopee untuk akses merchant.</p>
              <div>
                <label class="text-sm text-gray-500 block mb-1">Nomor HP</label>
                <input
                  v-model="qrisStore.loginPhone"
                  type="tel"
                  placeholder="0812xxxx"
                  class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
              </div>
              <div>
                <label class="text-sm text-gray-500 block mb-1">Password</label>
                <input
                  v-model="qrisStore.loginPassword"
                  type="password"
                  placeholder="Password akun Shopee"
                  class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
              </div>
              <button
                @click="qrisStore.requestOtp(qrisStore.loginPhone, qrisStore.loginPassword)"
                :disabled="qrisStore.loginLoading || !qrisStore.loginPhone"
                class="w-full bg-wa-green hover:bg-wa-green-dark disabled:opacity-50 text-white py-2 rounded-lg"
              >
                {{ qrisStore.loginLoading ? 'Mengirim...' : 'Kirim OTP' }}
              </button>
            </div>

            <!-- Step: OTP -->
            <div v-else-if="qrisStore.loginStep === 'otp'" class="space-y-3">
              <p class="text-sm text-gray-500">OTP telah dikirim ke {{ qrisStore.loginPhone }}.</p>
              <div>
                <label class="text-sm text-gray-500 block mb-1">Kode OTP</label>
                <input
                  v-model="qrisStore.loginOtp"
                  type="text"
                  inputmode="numeric"
                  placeholder="123456"
                  maxlength="10"
                  class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
                />
              </div>
              <div class="flex gap-2">
                <button
                  @click="qrisStore.resetLogin()"
                  class="flex-1 py-2 border dark:border-gray-600 rounded-lg text-sm"
                >
                  Batal
                </button>
                <button
                  @click="qrisStore.verifyOtp(qrisStore.loginOtp)"
                  :disabled="qrisStore.loginLoading || !qrisStore.loginOtp"
                  class="flex-1 bg-wa-green hover:bg-wa-green-dark disabled:opacity-50 text-white py-2 rounded-lg"
                >
                  {{ qrisStore.loginLoading ? 'Verifikasi...' : 'Verifikasi' }}
                </button>
              </div>
            </div>

            <!-- Step: merchant selection -->
            <div v-else-if="qrisStore.loginStep === 'merchant'" class="space-y-3">
              <p class="text-sm text-gray-500">Pilih merchant yang aktif:</p>
              <div
                v-for="m in qrisStore.loginMerchants"
                :key="m.id"
                @click="qrisStore.completeLogin(m.id, '')"
                class="p-3 border dark:border-gray-600 rounded-lg cursor-pointer hover:bg-gray-50 dark:hover:bg-gray-700"
              >
                <div class="font-medium text-sm">{{ m.name }}</div>
                <div class="text-xs text-gray-500">ID: {{ m.id }}</div>
              </div>
              <button
                @click="qrisStore.resetLogin()"
                class="w-full py-2 border dark:border-gray-600 rounded-lg text-sm"
              >
                Batal
              </button>
            </div>

            <!-- Login error -->
            <div v-if="qrisStore.loginError" class="p-3 bg-red-50 dark:bg-red-900/30 text-red-600 dark:text-red-400 text-sm rounded-lg">
              {{ qrisStore.loginError }}
            </div>

            <!-- Info -->
            <div class="p-4 bg-blue-50 dark:bg-blue-900/20 rounded-lg text-sm text-blue-700 dark:text-blue-400">
              <p class="font-medium mb-1">Kenapa login?</p>
              <p>Dengan login Shopee Merchant, setiap QRIS yang dibuat dipantau settlement-nya otomatis lewat feed transaksi Shopee.</p>
            </div>
          </div>
        </div>
      </div>

      <!-- Generate QR Modal -->
      <div
        v-if="showGenerateForm && qrisStore.qrisString"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-10 p-4"
        @click.self="showGenerateForm = false"
      >
        <div class="bg-white dark:bg-gray-800 rounded-lg p-4 w-full max-w-sm">
          <h4 class="font-medium mb-3">Generate QR Dinamis</h4>
          <div class="space-y-3">
            <div>
              <label class="text-sm text-gray-500 block mb-1">Nominal (Rp)</label>
              <input
                v-model.number="generateAmount"
                type="number"
                placeholder="Masukkan nominal"
                class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
              />
            </div>
            <div>
              <label class="text-sm text-gray-500 block mb-1">Keterangan (opsional)</label>
              <input
                v-model="generateNotes"
                type="text"
                placeholder="Contoh: Nasi Goreng"
                class="w-full px-3 py-2 border dark:border-gray-600 rounded-lg bg-white dark:bg-gray-700"
              />
            </div>
            <button
              @click="generateQr"
              :disabled="isProcessing || generateAmount <= 0"
              class="w-full bg-wa-green hover:bg-wa-green-dark disabled:opacity-50 text-white py-2 rounded-lg"
            >
              {{ isProcessing ? 'Memproses...' : 'Generate QR' }}
            </button>
          </div>
        </div>
      </div>

      <!-- Generated QR Modal -->
      <div
        v-if="generatedQrDataUrl"
        class="fixed inset-0 bg-black/50 flex items-center justify-center z-20 p-4"
        @click.self="generatedQrDataUrl = ''"
      >
        <div class="bg-white dark:bg-gray-800 rounded-lg p-4 w-full max-w-sm text-center">
          <h4 class="font-medium mb-3">QR Berhasil Dibuat!</h4>
          <img :src="generatedQrDataUrl" alt="QRIS" class="w-64 h-64 mx-auto mb-3" />
          <div v-if="generatedPaymentStatus === 'paid'" class="mb-3 p-2 bg-green-50 dark:bg-green-900/30 rounded-lg text-green-600 font-semibold">
            ✅ Pembayaran Berhasil
          </div>
          <div v-if="generatedUniqueAmount" class="text-2xl font-bold text-wa-green mb-1">{{ formatRupiah(generatedUniqueAmount) }}</div>
          <div v-if="generatedExpiresAt" class="text-xs text-amber-600 mb-1">⏳ Berlaku sampai {{ new Date(generatedExpiresAt).toLocaleTimeString('id-ID', { hour: '2-digit', minute: '2-digit' }) }}</div>
          <div class="text-sm text-gray-500 mb-4">Scan untuk membayar</div>
          <div class="flex gap-2">
            <button
              @click="downloadQr(generatedQrDataUrl, `qris-${generateAmount}.png`)"
              class="flex-1 py-2 border dark:border-gray-600 rounded-lg flex items-center justify-center gap-1"
            >
              <Download :size="16" /> Download
            </button>
            <button
              v-if="chatStore.activeChatId"
              @click="sendToChat"
              class="flex-1 bg-wa-green hover:bg-wa-green-dark text-white py-2 rounded-lg flex items-center justify-center gap-1"
            >
              <Send :size="16" /> Kirim
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
