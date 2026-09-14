export namespace domain {
	
	export class ChannelStatus {
	    status: string;
	    errorMsg?: string;
	    uploadedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChannelStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.errorMsg = source["errorMsg"];
	        this.uploadedAt = source["uploadedAt"];
	    }
	}
	export class HistoryRecord {
	    id?: number;
	    filename: string;
	    title: string;
	    scheduledDate: string;
	    scheduledTime: string;
	    channels?: string[];
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new HistoryRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.title = source["title"];
	        this.scheduledDate = source["scheduledDate"];
	        this.scheduledTime = source["scheduledTime"];
	        this.channels = source["channels"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class PlatformInfo {
	    id: string;
	    displayName: string;
	    loginUrl: string;
	
	    static createFrom(source: any = {}) {
	        return new PlatformInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.displayName = source["displayName"];
	        this.loginUrl = source["loginUrl"];
	    }
	}
	export class Settings {
	    videoFolder: string;
	    chromeUserDataDir: string;
	    chromePath: string;
	    defaultTag: string;
	    goldenHours: string[];
	    scheduleGoldenHours: string[];
	    publishNowGoldenHours: string[];
	    maxDays: number;
	    headless: boolean;
	    cdpPort: number;
	    enabledChannels: string[];
	    autoStart: boolean;
	    closeToTray: boolean;
	    startHidden: boolean;
	    publishMode: string;
	    autoUploadEnabled: boolean;
	    missedSlotPolicy: string;
	    tiktokRestrictedPolicy?: string;
	    locale: string;
	
	    static createFrom(source: any = {}) {
	        return new Settings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.videoFolder = source["videoFolder"];
	        this.chromeUserDataDir = source["chromeUserDataDir"];
	        this.chromePath = source["chromePath"];
	        this.defaultTag = source["defaultTag"];
	        this.goldenHours = source["goldenHours"];
	        this.scheduleGoldenHours = source["scheduleGoldenHours"];
	        this.publishNowGoldenHours = source["publishNowGoldenHours"];
	        this.maxDays = source["maxDays"];
	        this.headless = source["headless"];
	        this.cdpPort = source["cdpPort"];
	        this.enabledChannels = source["enabledChannels"];
	        this.autoStart = source["autoStart"];
	        this.closeToTray = source["closeToTray"];
	        this.startHidden = source["startHidden"];
	        this.publishMode = source["publishMode"];
	        this.autoUploadEnabled = source["autoUploadEnabled"];
	        this.missedSlotPolicy = source["missedSlotPolicy"];
	        this.tiktokRestrictedPolicy = source["tiktokRestrictedPolicy"];
	        this.locale = source["locale"];
	    }
	}
	export class VideoItem {
	    id: string;
	    filename: string;
	    fullPath: string;
	    cleanTitle: string;
	    customTitle: string;
	    fileSize: number;
	    fileSizeHuman: string;
	    scheduledDate: string;
	    scheduledTime: string;
	    goldenHourSlot: string;
	    status: string;
	    publishMode?: string;
	    channels?: Record<string, ChannelStatus>;
	    errorMsg?: string;
	    uploadedAt?: string;
	
	    static createFrom(source: any = {}) {
	        return new VideoItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.filename = source["filename"];
	        this.fullPath = source["fullPath"];
	        this.cleanTitle = source["cleanTitle"];
	        this.customTitle = source["customTitle"];
	        this.fileSize = source["fileSize"];
	        this.fileSizeHuman = source["fileSizeHuman"];
	        this.scheduledDate = source["scheduledDate"];
	        this.scheduledTime = source["scheduledTime"];
	        this.goldenHourSlot = source["goldenHourSlot"];
	        this.status = source["status"];
	        this.publishMode = source["publishMode"];
	        this.channels = this.convertValues(source["channels"], ChannelStatus, true);
	        this.errorMsg = source["errorMsg"];
	        this.uploadedAt = source["uploadedAt"];
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

export namespace ports {
	
	export class JobItem {
	    id: string;
	    video: domain.VideoItem;
	    targetChannels: string[];
	    state: string;
	    retryCount: number;
	    maxRetries: number;
	    errorMsg?: string;
	    // Go type: time
	    createdAt: any;
	    // Go type: time
	    updatedAt: any;
	    // Go type: time
	    completedAt?: any;
	
	    static createFrom(source: any = {}) {
	        return new JobItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.video = this.convertValues(source["video"], domain.VideoItem);
	        this.targetChannels = source["targetChannels"];
	        this.state = source["state"];
	        this.retryCount = source["retryCount"];
	        this.maxRetries = source["maxRetries"];
	        this.errorMsg = source["errorMsg"];
	        this.createdAt = this.convertValues(source["createdAt"], null);
	        this.updatedAt = this.convertValues(source["updatedAt"], null);
	        this.completedAt = this.convertValues(source["completedAt"], null);
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

export namespace updater {
	
	export class UpdateInfo {
	    available: boolean;
	    currentVersion: string;
	    latestVersion: string;
	    releaseNotes: string;
	    releaseUrl: string;
	    publishedAt: string;
	    assetUrl: string;
	    assetName: string;
	    assetSize: number;
	    expectedChecksum: string;
	    isSupported: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.available = source["available"];
	        this.currentVersion = source["currentVersion"];
	        this.latestVersion = source["latestVersion"];
	        this.releaseNotes = source["releaseNotes"];
	        this.releaseUrl = source["releaseUrl"];
	        this.publishedAt = source["publishedAt"];
	        this.assetUrl = source["assetUrl"];
	        this.assetName = source["assetName"];
	        this.assetSize = source["assetSize"];
	        this.expectedChecksum = source["expectedChecksum"];
	        this.isSupported = source["isSupported"];
	    }
	}

}

