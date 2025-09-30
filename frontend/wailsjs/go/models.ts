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
	export class IncidentNote {
	    id: string;
	    content: string;
	    created_at: string;
	    user_name?: string;
	
	    static createFrom(source: any = {}) {
	        return new IncidentNote(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.created_at = source["created_at"];
	        this.user_name = source["user_name"];
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
	export class ServiceConfig {
	    id: any;
	    name: string;
	    disabled?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ServiceConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.disabled = source["disabled"];
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

