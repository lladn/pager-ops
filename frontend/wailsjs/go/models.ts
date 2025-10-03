export namespace database {
	
	export class IncidentData {
	    incident_id: string;
	    incident_number: number;
	    title: string;
	    service_summary: string;
	    service_id: string;
	    status: string;
	    html_url: string;
	    // Go type: time
	    created_at: any;
	    // Go type: time
	    updated_at: any;
	    alert_count: number;
	    urgency: string;
	
	    static createFrom(source: any = {}) {
	        return new IncidentData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.incident_id = source["incident_id"];
	        this.incident_number = source["incident_number"];
	        this.title = source["title"];
	        this.service_summary = source["service_summary"];
	        this.service_id = source["service_id"];
	        this.status = source["status"];
	        this.html_url = source["html_url"];
	        this.created_at = this.convertValues(source["created_at"], null);
	        this.updated_at = this.convertValues(source["updated_at"], null);
	        this.alert_count = source["alert_count"];
	        this.urgency = source["urgency"];
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

export namespace main {
	
	export class NoteInput {
	    responses: store.NoteResponse[];
	    tags: store.NoteTag[];
	    freeform_content: string;
	
	    static createFrom(source: any = {}) {
	        return new NoteInput(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.responses = this.convertValues(source["responses"], store.NoteResponse);
	        this.tags = this.convertValues(source["tags"], store.NoteTag);
	        this.freeform_content = source["freeform_content"];
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
	export class NotificationConfig {
	    enabled: boolean;
	    sound: string;
	    snoozed: boolean;
	    // Go type: time
	    snoozeUntil: any;
	    browserRedirect: boolean;
	
	    static createFrom(source: any = {}) {
	        return new NotificationConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.sound = source["sound"];
	        this.snoozed = source["snoozed"];
	        this.snoozeUntil = this.convertValues(source["snoozeUntil"], null);
	        this.browserRedirect = source["browserRedirect"];
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

export namespace store {
	
	export class AlertLink {
	    href: string;
	    text: string;
	
	    static createFrom(source: any = {}) {
	        return new AlertLink(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.href = source["href"];
	        this.text = source["text"];
	    }
	}
	export class IncidentAlert {
	    id: string;
	    summary: string;
	    status: string;
	    created_at: string;
	    service_name?: string;
	    links?: AlertLink[];
	
	    static createFrom(source: any = {}) {
	        return new IncidentAlert(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.summary = source["summary"];
	        this.status = source["status"];
	        this.created_at = source["created_at"];
	        this.service_name = source["service_name"];
	        this.links = this.convertValues(source["links"], AlertLink);
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
	export class NoteTag {
	    tag_name: string;
	    selected_values: string[];
	
	    static createFrom(source: any = {}) {
	        return new NoteTag(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tag_name = source["tag_name"];
	        this.selected_values = source["selected_values"];
	    }
	}
	export class NoteResponse {
	    question: string;
	    answer: string;
	
	    static createFrom(source: any = {}) {
	        return new NoteResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.question = source["question"];
	        this.answer = source["answer"];
	    }
	}
	export class IncidentNote {
	    id: string;
	    content: string;
	    created_at: string;
	    user_name?: string;
	    service_id?: string;
	    responses?: NoteResponse[];
	    tags?: NoteTag[];
	    freeform_content?: string;
	
	    static createFrom(source: any = {}) {
	        return new IncidentNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.created_at = source["created_at"];
	        this.user_name = source["user_name"];
	        this.service_id = source["service_id"];
	        this.responses = this.convertValues(source["responses"], NoteResponse);
	        this.tags = this.convertValues(source["tags"], NoteTag);
	        this.freeform_content = source["freeform_content"];
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
	export class IncidentSidebarData {
	    incident_id: string;
	    alerts: IncidentAlert[];
	    notes: IncidentNote[];
	    loading: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new IncidentSidebarData(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.incident_id = source["incident_id"];
	        this.alerts = this.convertValues(source["alerts"], IncidentAlert);
	        this.notes = this.convertValues(source["notes"], IncidentNote);
	        this.loading = source["loading"];
	        this.error = source["error"];
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
	
	
	export class TagConfig {
	    name: string;
	    multiple?: string[];
	    single?: string[];
	
	    static createFrom(source: any = {}) {
	        return new TagConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.multiple = source["multiple"];
	        this.single = source["single"];
	    }
	}
	export class ServiceTypes {
	    questions?: string[];
	    tags?: TagConfig[];
	
	    static createFrom(source: any = {}) {
	        return new ServiceTypes(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.questions = source["questions"];
	        this.tags = this.convertValues(source["tags"], TagConfig);
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
	export class ServiceConfig {
	    id: any;
	    name: string;
	    disabled?: boolean;
	    types?: ServiceTypes;
	
	    static createFrom(source: any = {}) {
	        return new ServiceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.disabled = source["disabled"];
	        this.types = this.convertValues(source["types"], ServiceTypes);
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
	
	export class ServicesConfig {
	    services: ServiceConfig[];
	
	    static createFrom(source: any = {}) {
	        return new ServicesConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.services = this.convertValues(source["services"], ServiceConfig);
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

