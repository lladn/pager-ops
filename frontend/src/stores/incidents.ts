import { writable, derived } from 'svelte/store';
import type { IncidentData, ServicesConfig, ServiceConfig } from '../../wailsjs/go/models';
import { GetOpenIncidents, GetResolvedIncidents, GetServicesConfig, GetSelectedServices } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

// Store for all incidents
export const openIncidents = writable<IncidentData[]>([]);
export const resolvedIncidents = writable<IncidentData[]>([]);

// Store for services configuration
export const servicesConfig = writable<ServicesConfig | null>(null);
export const selectedServices = writable<string[]>([]);

// Store for UI state
export const activeTab = writable<'open' | 'resolved'>('open');
export const settingsOpen = writable(false);
export const settingsTab = writable<'api' | 'services'>('api');
export const loading = writable(false);
export const error = writable<string | null>(null);

// Derived store for incident counts
export const openCount = derived(openIncidents, $incidents => $incidents.length);
export const resolvedCount = derived(resolvedIncidents, $incidents => $incidents.length);

// Load incidents based on selected services
export async function loadOpenIncidents() {
    loading.set(true);
    error.set(null);
    try {
        const services = await GetSelectedServices();
        const incidents = await GetOpenIncidents(services);
        openIncidents.set(incidents || []);
    } catch (err) {
        error.set(err?.toString() || 'Failed to load open incidents');
        openIncidents.set([]);
    } finally {
        loading.set(false);
    }
}

export async function loadResolvedIncidents() {
    loading.set(true);
    error.set(null);
    try {
        const services = await GetSelectedServices();
        const incidents = await GetResolvedIncidents(services);
        resolvedIncidents.set(incidents || []);
    } catch (err) {
        error.set(err?.toString() || 'Failed to load resolved incidents');
        resolvedIncidents.set([]);
    } finally {
        loading.set(false);
    }
}

export async function loadServicesConfig() {
    try {
        const config = await GetServicesConfig();
        servicesConfig.set(config);
        const selected = await GetSelectedServices();
        selectedServices.set(selected || []);
    } catch (err) {
        servicesConfig.set(null);
        selectedServices.set([]);
    }
}

// Initialize event listeners
export function initializeEventListeners() {
    // Listen for incident updates from backend polling
    EventsOn('incidents-updated', (type: string) => {
        if (type === 'open') {
            loadOpenIncidents();
        } else if (type === 'resolved') {
            loadResolvedIncidents();
        }
    });

    // Listen for services config updates
    EventsOn('services-config-updated', () => {
        loadServicesConfig();
    });
}

// Format time helper
export function formatTime(date: string | Date): string {
    const d = typeof date === 'string' ? new Date(date) : date;
    const now = new Date();
    const diff = Math.floor((now.getTime() - d.getTime()) / 1000);
    
    if (diff < 60) return 'just now';
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    
    return d.toLocaleDateString('en-US', { 
        month: 'numeric', 
        day: 'numeric', 
        year: 'numeric',
        hour: 'numeric',
        minute: '2-digit',
        hour12: true
    });
}

// Get urgency level from incident
export function getUrgency(incident: IncidentData): 'high' | 'low' | 'medium' {
    // Determine urgency based on title or other criteria
    const title = incident.title.toLowerCase();
    if (title.includes('critical') || title.includes('down') || title.includes('failure')) {
        return 'high';
    }
    if (title.includes('warning') || title.includes('degradation')) {
        return 'medium';
    }
    return 'low';
}