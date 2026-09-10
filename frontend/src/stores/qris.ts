import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import {
  GetPaymentSession,
  RequestShopeeOtp,
  VerifyShopeeOtp,
  CompleteShopeeLogin,
  LogoutPayment,
  SetPaymentStaticQris,
  CreatePayment,
} from '../../wailsjs/go/main/App'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useChatStore } from './chat'

export interface QrisProduct {
  id: string
  name: string
  price: number
  category?: string
}

export interface QrisTransaction {
  id: string
  paymentId?: string
  productId?: string
  productName?: string
  amount: number
  uniqueAmount?: number
  qrDataUrl?: string
  status: 'pending' | 'paid' | 'cancelled'
  createdAt: number
  paidAt?: number
  expiresAt?: number
  notes?: string
}

export interface MerchantSession {
  loggedIn: boolean
  merchantName?: string
  storeId?: string
  needsRelogin?: boolean
}

export interface MerchantSummary {
  id: string
  name: string
  status: number
}

const STORAGE_KEYS = {
  QRIS_STRING: 'wakupi_qris_string',
  PRODUCTS: 'wakupi_qris_products',
  TRANSACTIONS: 'wakupi_qris_transactions',
}

export const useQrisStore = defineStore('qris', () => {
  const qrisString = ref<string>('')
  const products = ref<QrisProduct[]>([])
  const transactions = ref<QrisTransaction[]>([])

  // Load from localStorage on init
  function loadFromStorage() {
    try {
      const storedQris = localStorage.getItem(STORAGE_KEYS.QRIS_STRING)
      const storedProducts = localStorage.getItem(STORAGE_KEYS.PRODUCTS)
      const storedTransactions = localStorage.getItem(STORAGE_KEYS.TRANSACTIONS)

      if (storedQris) qrisString.value = storedQris
      if (storedProducts) products.value = JSON.parse(storedProducts)
      if (storedTransactions) transactions.value = JSON.parse(storedTransactions)
    } catch (e) {
      console.error('Failed to load QRIS data from storage:', e)
    }
  }

  function saveToStorage() {
    try {
      localStorage.setItem(STORAGE_KEYS.QRIS_STRING, qrisString.value)
      localStorage.setItem(STORAGE_KEYS.PRODUCTS, JSON.stringify(products.value))
      localStorage.setItem(STORAGE_KEYS.TRANSACTIONS, JSON.stringify(transactions.value))
    } catch (e) {
      console.error('Failed to save QRIS data to storage:', e)
    }
  }

  function setQrisString(str: string) {
    qrisString.value = str
    saveToStorage()
  }

  function addProduct(product: Omit<QrisProduct, 'id'>) {
    const newProduct: QrisProduct = {
      id: `prod-${Date.now()}`,
      ...product,
    }
    products.value.push(newProduct)
    saveToStorage()
    return newProduct
  }

  function updateProduct(id: string, updates: Partial<QrisProduct>) {
    const index = products.value.findIndex((p) => p.id === id)
    if (index >= 0) {
      products.value[index] = { ...products.value[index], ...updates }
      saveToStorage()
    }
  }

  function deleteProduct(id: string) {
    products.value = products.value.filter((p) => p.id !== id)
    saveToStorage()
  }

  function addTransaction(transaction: Omit<QrisTransaction, 'id' | 'createdAt' | 'status'>) {
    const newTransaction: QrisTransaction = {
      id: `txn-${Date.now()}`,
      ...transaction,
      status: 'pending',
      createdAt: Date.now(),
    }
    transactions.value.unshift(newTransaction)
    saveToStorage()
    return newTransaction
  }

  function updateTransactionStatus(id: string, status: QrisTransaction['status']) {
    const transaction = transactions.value.find((t) => t.id === id)
    if (transaction) {
      transaction.status = status
      if (status === 'paid') transaction.paidAt = Date.now()
      saveToStorage()
    }
  }

  function deleteTransaction(id: string) {
    transactions.value = transactions.value.filter((t) => t.id !== id)
    saveToStorage()
  }

  function clearAllData() {
    qrisString.value = ''
    products.value = []
    transactions.value = []
    localStorage.removeItem(STORAGE_KEYS.QRIS_STRING)
    localStorage.removeItem(STORAGE_KEYS.PRODUCTS)
    localStorage.removeItem(STORAGE_KEYS.TRANSACTIONS)
  }

  // === Merchant Session ===

  const session = ref<MerchantSession>({ loggedIn: false })
  const loginError = ref('')
  const loginStep = ref<'idle' | 'number' | 'password' | 'otp' | 'merchant'>('idle')
  const loginPhone = ref('')
  const loginPassword = ref('')
  const loginOtp = ref('')
  const loginMerchants = ref<MerchantSummary[]>([])
  const loginLoading = ref(false)

  async function refreshSession() {
    try {
      const s = await GetPaymentSession()
      session.value = {
        loggedIn: s.loggedIn,
        merchantName: s.merchantName,
        storeId: s.storeId,
        needsRelogin: s.needsRelogin,
      }
      if (s.loggedIn && qrisString.value) {
        await syncQrisToBackend()
      }
    } catch {
      session.value = { loggedIn: false }
    }
  }

  async function syncQrisToBackend() {
    if (!qrisString.value || !session.value.loggedIn) return
    try {
      await SetPaymentStaticQris(qrisString.value)
    } catch (e: unknown) {
      console.error('Failed to sync QRIS to backend:', e)
    }
  }

  async function requestOtp(phone: string, password: string) {
    loginLoading.value = true
    loginError.value = ''
    try {
      await RequestShopeeOtp(phone, password)
      loginPhone.value = phone
      loginStep.value = 'otp'
    } catch (e: unknown) {
      loginError.value = e instanceof Error ? e.message : String(e)
    } finally {
      loginLoading.value = false
    }
  }

  async function verifyOtp(otp: string) {
    loginLoading.value = true
    loginError.value = ''
    try {
      const result = await VerifyShopeeOtp(otp)
      if (result.status === 'complete') {
        loginStep.value = 'idle'
        await refreshSession()
      } else if (result.status === 'merchant-selection-required') {
        loginMerchants.value = result.merchants || []
        loginStep.value = 'merchant'
      }
    } catch (e: unknown) {
      loginError.value = e instanceof Error ? e.message : String(e)
    } finally {
      loginLoading.value = false
    }
  }

  async function completeLogin(merchantID: string, storeID: string) {
    loginLoading.value = true
    loginError.value = ''
    try {
      await CompleteShopeeLogin(merchantID, storeID)
      loginStep.value = 'idle'
      await refreshSession()
    } catch (e: unknown) {
      loginError.value = e instanceof Error ? e.message : String(e)
    } finally {
      loginLoading.value = false
    }
  }

  async function logoutMerchant() {
    try {
      await LogoutPayment()
    } catch { /* ignore */ }
    session.value = { loggedIn: false }
  }

  function resetLogin() {
    loginStep.value = 'idle'
    loginPhone.value = ''
    loginPassword.value = ''
    loginOtp.value = ''
    loginMerchants.value = []
    loginError.value = ''
  }

  // === Payment via backend ===

  interface BackendPaymentResult {
    id: string
    uniqueAmount: number
    expiresAt: number
    qrString: string
  }

  async function createPayment(amount: number, reference: string): Promise<BackendPaymentResult | null> {
    if (!session.value.loggedIn) return null
    const result = await CreatePayment(amount, reference)
    return {
      id: result.id,
      uniqueAmount: result.uniqueAmount,
      expiresAt: result.expiresAt,
      qrString: result.qrString,
    }
  }

  async function setQris(qris: string) {
    setQrisString(qris)
    if (session.value.loggedIn) {
      try {
        await SetPaymentStaticQris(qris)
      } catch (e: unknown) {
        console.error('Failed to set backend QRIS:', e)
      }
    }
  }

  // === Settlement events ===

  interface PaymentPaidEvent {
    id: string
    uniqueAmount: number
    reference: string
    status: string
  }

  function bindSettlementEvents() {
    EventsOn('payment:paid', (data: PaymentPaidEvent) => {
      const txn = transactions.value.find((t) => t.paymentId === data.id)
      if (txn) {
        txn.status = 'paid'
        txn.paidAt = Date.now()
        saveToStorage()

        // Auto-send WhatsApp notification to the active chat
        const chat = useChatStore()
        if (chat.activeChatId) {
          const caption = `✅ Pembayaran Berhasil!\n\nRp ${new Intl.NumberFormat('id-ID').format(data.uniqueAmount)}\n${data.reference ? 'Ref: ' + data.reference : ''}\n\nTerima kasih.`.trim()
          chat.sendTextToChat(chat.activeChatId, caption)
        }
      }
    })
    EventsOn('payment:expired', (data: PaymentPaidEvent) => {
      const txn = transactions.value.find((t) => t.paymentId === data.id)
      if (txn) {
        txn.status = 'cancelled'
        saveToStorage()
      }
    })
    EventsOn('payment:error', (data: { error: string }) => {
      console.error('Payment error:', data.error)
    })
  }

  // Stats
  const todayTransactions = computed(() => {
    const today = new Date()
    today.setHours(0, 0, 0, 0)
    const todayStart = today.getTime()
    return transactions.value.filter((t) => t.createdAt >= todayStart)
  })

  const totalToday = computed(() =>
    todayTransactions.value.filter((t) => t.status === 'paid').reduce((sum, t) => sum + t.amount, 0)
  )

  const totalPending = computed(() =>
    transactions.value.filter((t) => t.status === 'pending').reduce((sum, t) => sum + t.amount, 0)
  )

  // Initialize
  loadFromStorage()
  bindSettlementEvents()

  return {
    qrisString,
    products,
    transactions,
    session,
    loginError,
    loginStep,
    loginPhone,
    loginPassword,
    loginOtp,
    loginMerchants,
    loginLoading,
    setQrisString,
    setQris,
    addProduct,
    updateProduct,
    deleteProduct,
    addTransaction,
    updateTransactionStatus,
    deleteTransaction,
    clearAllData,
    refreshSession,
    requestOtp,
    verifyOtp,
    completeLogin,
    logoutMerchant,
    resetLogin,
    createPayment,
    todayTransactions,
    totalToday,
    totalPending,
  }
})