export namespace app {
	
	export class ConfigPersistenceResult {
	    created: boolean;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ConfigPersistenceResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.created = source["created"];
	        this.path = source["path"];
	    }
	}

}

export namespace clockify {
	
	export class ProjectInfo {
	    id: string;
	    name: string;
	    clientName: string;
	
	    static createFrom(source: any = {}) {
	        return new ProjectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.clientName = source["clientName"];
	    }
	}
	export class WorkspaceInfo {
	    id: string;
	    name: string;
	
	    static createFrom(source: any = {}) {
	        return new WorkspaceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	    }
	}

}

export namespace config {
	
	export class Config {
	    ClockifyAPIKey: string;
	    ClockifyWorkspace: string;
	    JiraBaseURL: string;
	    JiraEmail: string;
	    JiraAPIToken: string;
	    MockMode: boolean;
	    AutoUpdate: boolean;
	    BetaChannel: boolean;
	    TrayTimerFormat: string;
	    TrayShowTimer: boolean;
	    LaunchOnStartup: boolean;
	    SummaryWordLimit: number;
	    LogRoundingMin: number;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ClockifyAPIKey = source["ClockifyAPIKey"];
	        this.ClockifyWorkspace = source["ClockifyWorkspace"];
	        this.JiraBaseURL = source["JiraBaseURL"];
	        this.JiraEmail = source["JiraEmail"];
	        this.JiraAPIToken = source["JiraAPIToken"];
	        this.MockMode = source["MockMode"];
	        this.AutoUpdate = source["AutoUpdate"];
	        this.BetaChannel = source["BetaChannel"];
	        this.TrayTimerFormat = source["TrayTimerFormat"];
	        this.TrayShowTimer = source["TrayShowTimer"];
	        this.LaunchOnStartup = source["LaunchOnStartup"];
	        this.SummaryWordLimit = source["SummaryWordLimit"];
	        this.LogRoundingMin = source["LogRoundingMin"];
	    }
	}

}

export namespace models {
	
	export class BranchDetection {
	    ticketKey: string;
	    branchName: string;
	    repoPath: string;
	    ide: string;
	
	    static createFrom(source: any = {}) {
	        return new BranchDetection(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticketKey = source["ticketKey"];
	        this.branchName = source["branchName"];
	        this.repoPath = source["repoPath"];
	        this.ide = source["ide"];
	    }
	}
	export class IntegrationStatus {
	    clockifyConnected: boolean;
	    clockifyError?: string;
	    jiraConnected: boolean;
	    jiraError?: string;
	
	    static createFrom(source: any = {}) {
	        return new IntegrationStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.clockifyConnected = source["clockifyConnected"];
	        this.clockifyError = source["clockifyError"];
	        this.jiraConnected = source["jiraConnected"];
	        this.jiraError = source["jiraError"];
	    }
	}
	export class JiraTicket {
	    key: string;
	    summary: string;
	    status: string;
	    assignee: string;
	    issueType: string;
	
	    static createFrom(source: any = {}) {
	        return new JiraTicket(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.key = source["key"];
	        this.summary = source["summary"];
	        this.status = source["status"];
	        this.assignee = source["assignee"];
	        this.issueType = source["issueType"];
	    }
	}
	export class ManualEntryRequest {
	    ticketKey: string;
	    description: string;
	    date: string;
	    startTime: string;
	    endTime: string;
	    projectId: string;
	
	    static createFrom(source: any = {}) {
	        return new ManualEntryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ticketKey = source["ticketKey"];
	        this.description = source["description"];
	        this.date = source["date"];
	        this.startTime = source["startTime"];
	        this.endTime = source["endTime"];
	        this.projectId = source["projectId"];
	    }
	}
	export class TimeEntry {
	    id: string;
	    ticketKey: string;
	    ticketSummary: string;
	    description: string;
	    // Go type: time
	    start: any;
	    // Go type: time
	    end: any;
	    duration: number;
	    clockifyId: string;
	    jiraWorklogId: string;
	
	    static createFrom(source: any = {}) {
	        return new TimeEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ticketKey = source["ticketKey"];
	        this.ticketSummary = source["ticketSummary"];
	        this.description = source["description"];
	        this.start = this.convertValues(source["start"], null);
	        this.end = this.convertValues(source["end"], null);
	        this.duration = source["duration"];
	        this.clockifyId = source["clockifyId"];
	        this.jiraWorklogId = source["jiraWorklogId"];
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
	export class TimerState {
	    running: boolean;
	    ticketKey: string;
	    ticketSummary: string;
	    // Go type: time
	    startedAt: any;
	    clockifyId: string;
	
	    static createFrom(source: any = {}) {
	        return new TimerState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.ticketKey = source["ticketKey"];
	        this.ticketSummary = source["ticketSummary"];
	        this.startedAt = this.convertValues(source["startedAt"], null);
	        this.clockifyId = source["clockifyId"];
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
	export class UpdateEntryRequest {
	    id: string;
	    ticketKey: string;
	    description: string;
	    start: string;
	    end: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateEntryRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ticketKey = source["ticketKey"];
	        this.description = source["description"];
	        this.start = source["start"];
	        this.end = source["end"];
	    }
	}
	export class UpdateInfo {
	    version: string;
	    isPreRelease: boolean;
	    downloadUrl: string;
	    releaseNotes: string;
	    size: number;
	    publishedAt: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.version = source["version"];
	        this.isPreRelease = source["isPreRelease"];
	        this.downloadUrl = source["downloadUrl"];
	        this.releaseNotes = source["releaseNotes"];
	        this.size = source["size"];
	        this.publishedAt = source["publishedAt"];
	    }
	}
	export class UpdatePreferences {
	    autoCheck: boolean;
	    betaChannel: boolean;
	
	    static createFrom(source: any = {}) {
	        return new UpdatePreferences(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.autoCheck = source["autoCheck"];
	        this.betaChannel = source["betaChannel"];
	    }
	}

}

