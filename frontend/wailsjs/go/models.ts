export namespace ai {
	
	export class Config {
	    provider: string;
	    apiKey: string;
	    baseUrl: string;
	    model: string;
	    enabled: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.provider = source["provider"];
	        this.apiKey = source["apiKey"];
	        this.baseUrl = source["baseUrl"];
	        this.model = source["model"];
	        this.enabled = source["enabled"];
	    }
	}
	export class ImageOptions {
	    model: string;
	    size: string;
	    count: number;
	
	    static createFrom(source: any = {}) {
	        return new ImageOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.size = source["size"];
	        this.count = source["count"];
	    }
	}
	export class ImageResult {
	    url: string;
	    revisedPrompt?: string;
	    width?: number;
	    height?: number;
	    model?: string;
	    b64Json?: string;
	
	    static createFrom(source: any = {}) {
	        return new ImageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.url = source["url"];
	        this.revisedPrompt = source["revisedPrompt"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.model = source["model"];
	        this.b64Json = source["b64Json"];
	    }
	}

}

export namespace main {
	
	export class PlaygroundMessage {
	    role: string;
	    content: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaygroundMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.role = source["role"];
	        this.content = source["content"];
	    }
	}
	export class PlaygroundOptions {
	    model: string;
	    temperature: number;
	    system: string;
	
	    static createFrom(source: any = {}) {
	        return new PlaygroundOptions(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.temperature = source["temperature"];
	        this.system = source["system"];
	    }
	}
	export class QuotedArg {
	    id: string;
	    participant: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new QuotedArg(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.participant = source["participant"];
	        this.text = source["text"];
	    }
	}

}

export namespace payment {
	
	export class CompleteLoginResult {
	    status: string;
	    merchantId: string;
	    merchantName: string;
	
	    static createFrom(source: any = {}) {
	        return new CompleteLoginResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.merchantId = source["merchantId"];
	        this.merchantName = source["merchantName"];
	    }
	}
	export class MerchantSummary {
	    id: string;
	    name: string;
	    status: number;
	
	    static createFrom(source: any = {}) {
	        return new MerchantSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.status = source["status"];
	    }
	}
	export class OtpResponse {
	    phoneNumber: string;
	    channel: number;
	    hasPassword: boolean;
	
	    static createFrom(source: any = {}) {
	        return new OtpResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phoneNumber = source["phoneNumber"];
	        this.channel = source["channel"];
	        this.hasPassword = source["hasPassword"];
	    }
	}
	export class PaymentInfo {
	    id: string;
	    baseAmount: number;
	    uniqueAmount: number;
	    status: string;
	    reference: string;
	    qrString: string;
	    createdAt: number;
	    expiresAt: number;
	
	    static createFrom(source: any = {}) {
	        return new PaymentInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.baseAmount = source["baseAmount"];
	        this.uniqueAmount = source["uniqueAmount"];
	        this.status = source["status"];
	        this.reference = source["reference"];
	        this.qrString = source["qrString"];
	        this.createdAt = source["createdAt"];
	        this.expiresAt = source["expiresAt"];
	    }
	}
	export class SessionInfo {
	    loggedIn: boolean;
	    merchantName?: string;
	    storeId?: string;
	    needsRelogin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.loggedIn = source["loggedIn"];
	        this.merchantName = source["merchantName"];
	        this.storeId = source["storeId"];
	        this.needsRelogin = source["needsRelogin"];
	    }
	}
	export class VerifyLoginOutcome {
	    status: string;
	    merchantId?: string;
	    merchantName?: string;
	    merchants?: MerchantSummary[];
	
	    static createFrom(source: any = {}) {
	        return new VerifyLoginOutcome(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.merchantId = source["merchantId"];
	        this.merchantName = source["merchantName"];
	        this.merchants = this.convertValues(source["merchants"], MerchantSummary);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}

}

export namespace wa {
	
	export class ChannelInfo {
	    jid: string;
	    name: string;
	    description: string;
	    subscriberCount: number;
	    avatarUrl?: string;
	    isSubscribed: boolean;
	    role?: string;
	    inviteCode?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChannelInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.subscriberCount = source["subscriberCount"];
	        this.avatarUrl = source["avatarUrl"];
	        this.isSubscribed = source["isSubscribed"];
	        this.role = source["role"];
	        this.inviteCode = source["inviteCode"];
	    }
	}
	export class ChatInfo {
	    id: string;
	    accountId: string;
	    jid: string;
	    name: string;
	    isGroup: boolean;
	    isChannel: boolean;
	    channelRole?: string;
	    lastMessage: string;
	    lastTime: number;
	    avatarUrl?: string;
	    pinned: boolean;
	    archived: boolean;
	    mutedUntil: number;
	    blocked: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChatInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.accountId = source["accountId"];
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.isGroup = source["isGroup"];
	        this.isChannel = source["isChannel"];
	        this.channelRole = source["channelRole"];
	        this.lastMessage = source["lastMessage"];
	        this.lastTime = source["lastTime"];
	        this.avatarUrl = source["avatarUrl"];
	        this.pinned = source["pinned"];
	        this.archived = source["archived"];
	        this.mutedUntil = source["mutedUntil"];
	        this.blocked = source["blocked"];
	    }
	}
	export class ContactCheck {
	    phone: string;
	    jid: string;
	    onWhatsApp: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ContactCheck(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.phone = source["phone"];
	        this.jid = source["jid"];
	        this.onWhatsApp = source["onWhatsApp"];
	    }
	}
	export class GroupMember {
	    jid: string;
	    isAdmin: boolean;
	    isSuperAdmin: boolean;
	    pushName: string;
	
	    static createFrom(source: any = {}) {
	        return new GroupMember(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.isAdmin = source["isAdmin"];
	        this.isSuperAdmin = source["isSuperAdmin"];
	        this.pushName = source["pushName"];
	    }
	}
	export class GroupInfo {
	    jid: string;
	    name: string;
	    topic: string;
	    ownerJid: string;
	    created: number;
	    isAnnounce: boolean;
	    isLocked: boolean;
	    participants: GroupMember[];
	
	    static createFrom(source: any = {}) {
	        return new GroupInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.jid = source["jid"];
	        this.name = source["name"];
	        this.topic = source["topic"];
	        this.ownerJid = source["ownerJid"];
	        this.created = source["created"];
	        this.isAnnounce = source["isAnnounce"];
	        this.isLocked = source["isLocked"];
	        this.participants = this.convertValues(source["participants"], GroupMember);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	export class MessageInfo {
	    id: string;
	    chatId: string;
	    accountId: string;
	    jid: string;
	    sender: string;
	    text: string;
	    timestamp: number;
	    fromMe: boolean;
	    isGroup: boolean;
	    isChannel: boolean;
	    pushName: string;
	    mediaType?: string;
	    mediaUrl?: string;
	    mimeType?: string;
	    fileName?: string;
	    fileSize?: number;
	    width?: number;
	    height?: number;
	    duration?: number;
	    isPtt?: boolean;
	    caption?: string;
	    quotedId?: string;
	    quotedText?: string;
	    quotedFrom?: string;
	
	    static createFrom(source: any = {}) {
	        return new MessageInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.chatId = source["chatId"];
	        this.accountId = source["accountId"];
	        this.jid = source["jid"];
	        this.sender = source["sender"];
	        this.text = source["text"];
	        this.timestamp = source["timestamp"];
	        this.fromMe = source["fromMe"];
	        this.isGroup = source["isGroup"];
	        this.isChannel = source["isChannel"];
	        this.pushName = source["pushName"];
	        this.mediaType = source["mediaType"];
	        this.mediaUrl = source["mediaUrl"];
	        this.mimeType = source["mimeType"];
	        this.fileName = source["fileName"];
	        this.fileSize = source["fileSize"];
	        this.width = source["width"];
	        this.height = source["height"];
	        this.duration = source["duration"];
	        this.isPtt = source["isPtt"];
	        this.caption = source["caption"];
	        this.quotedId = source["quotedId"];
	        this.quotedText = source["quotedText"];
	        this.quotedFrom = source["quotedFrom"];
	    }
	}
	export class SendMediaResult {
	    messageId: string;
	    localUrl: string;
	    mimeType: string;
	
	    static createFrom(source: any = {}) {
	        return new SendMediaResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.messageId = source["messageId"];
	        this.localUrl = source["localUrl"];
	        this.mimeType = source["mimeType"];
	    }
	}
	export class SessionInfo {
	    id: string;
	    name: string;
	    connected: boolean;
	    jid: string;
	    phone: string;
	
	    static createFrom(source: any = {}) {
	        return new SessionInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.connected = source["connected"];
	        this.jid = source["jid"];
	        this.phone = source["phone"];
	    }
	}

}

