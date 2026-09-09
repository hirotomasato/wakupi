export interface Account {
  id: string
  name: string
  phone: string
  avatar?: string
  connected: boolean
}

export interface Chat {
  id: string
  accountId: string
  jid: string
  name: string
  avatar?: string
  avatarUrl?: string
  lastMessage: string
  lastTime: string
  lastMediaType?: MediaType
  lastFromMe?: boolean
  lastStatus?: 'sent' | 'delivered' | 'read'
  unread: number
  pinned?: boolean
  archived?: boolean
  mutedUntil?: number
  blocked?: boolean
  isGroup?: boolean
  isChannel?: boolean
  role?: string
  _sortKey?: number
}

export type MediaType = '' | 'image' | 'video' | 'audio' | 'document' | 'sticker'

export interface ChannelInfo {
  jid: string
  name: string
  description: string
  subscriberCount: number
  avatarUrl?: string
  isSubscribed: boolean
  role?: string
  inviteCode?: string
}

export interface Message {
  id: string
  chatId: string
  text: string
  time: string
  fromMe: boolean
  sender?: string
  status?: 'sent' | 'delivered' | 'read' | 'failed'
  _ts?: number
  _senderJID?: string

  mediaType?: MediaType
  mediaUrl?: string
  mimeType?: string
  fileName?: string
  fileSize?: number
  width?: number
  height?: number
  duration?: number
  isPtt?: boolean
  caption?: string

  quotedId?: string
  quotedText?: string
  quotedFrom?: string

  deleted?: boolean
  reactions?: { emoji: string; fromMe: boolean; sender: string }[]
}
