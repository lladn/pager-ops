import { writable, derived, get } from 'svelte/store';
import { GetOpenIncidents, GetResolvedIncidents, GetServicesConfig, GetSelectedServices, GetIncidentSidebarData } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';
import type { database, store } from '../../wailsjs/go/models';

// Type aliases for cleaner code
type IncidentData = database.IncidentData;
type ServicesConfig = store.ServicesConfig;
type ServiceConfig = store.ServiceConfig;

// Store for all incidents
export const openIncidents = writable<IncidentData[]>([]);
export const resolvedIncidents = writable<IncidentData[]>([]);

// Store for services configuration
export const servicesConfig = writable<ServicesConfig | null>(null);
export const selectedServices = writable<string[]>([]);

// Store for UI state
export const activeTab = writable<'open' | 'resolved'>('open');
export const settingsOpen = writable(false);
export const settingsTab = writable<'general' | 'services'>('general');
export const panelOpen = writable(false);
export const panelWidth = writable(320);
export const loading = writable(false);
export const error = writable<string | null>(null);

// Store for selected incident (for panel display)
export const sidebarOpen = writable(false);
export const selectedIncident = writable<IncidentData | null>(null);
export const selectedIncidentID = derived(
    selectedIncident,
    $incident => $incident?.incident_id || null
);
export const sidebarLoading = writable(false);
export const sidebarError = writable<string | null>(null);
export const sidebarData = writable<{
    alerts: any[];
    notes: any[];
} | null>(null);

// Track loading state to prevent duplicate fetches
const loadingIncidents = new Set<string>();

// Track last fetched incidents to enable smart caching
const lastFetchedIncidentMetadata = new Map<string, { alertCount: number; updatedAt: string }>();

// Store for polling state to prevent loading flicker
let isPolling = false;
let lastOpenIncidentIds = new Set<string>();
let lastResolvedIncidentIds = new Set<string>();

// Derived store for incident counts
export const openCount = derived(openIncidents, $incidents => $incidents.length);
export const resolvedCount = derived(resolvedIncidents, $incidents => $incidents.length);

// Function to load sidebar data with deduplication - EXPORT THIS
export async function loadIncidentSidebarData(incidentId: string) {
    // Check if already loading this incident
    if (loadingIncidents.has(incidentId)) {
        return;
    }

    // Clear previous error but keep data until new data arrives
    sidebarError.set(null);
    
    // Mark as loading
    loadingIncidents.add(incidentId);
    sidebarLoading.set(true);
    
    try {
        const data = await GetIncidentSidebarData(incidentId);
        
        // Only update if this is still the selected incident
        if (get(selectedIncidentID) === incidentId) {
            if (data.error) {
                sidebarError.set(data.error);
            }
            sidebarData.set({
                alerts: data.alerts || [],
                notes: data.notes || []
            });
        }
    } catch (err) {
        if (get(selectedIncidentID) === incidentId) {
            sidebarError.set(err?.toString() || 'Failed to load incident details');
            // Don't clear data on error - keep showing previous data
        }
    } finally {
        loadingIncidents.delete(incidentId);
        if (get(selectedIncidentID) === incidentId) {
            sidebarLoading.set(false);
        }
    }
}

// Function to check if incident needs refresh based on metadata changes
function shouldRefetchIncident(incidentId: string, incident: IncidentData | null): boolean {
    if (!incident) return false;
    
    const lastMeta = lastFetchedIncidentMetadata.get(incidentId);
    if (!lastMeta) return true; // Never fetched before
    
    // Check if alert count changed or incident was updated
    const alertCountChanged = incident.alert_count !== lastMeta.alertCount;
    const updatedAtChanged = incident.updated_at !== lastMeta.updatedAt;
    
    return alertCountChanged || updatedAtChanged;
}

// Function to update incident metadata after fetch
function updateIncidentMetadata(incidentId: string, incident: IncidentData) {
    lastFetchedIncidentMetadata.set(incidentId, {
        alertCount: incident.alert_count || 0,
        updatedAt: incident.updated_at
    });
}

// Subscribe to selectedIncidentID changes to trigger fetch when panel is open
let previousIncidentID: string | null = null;
selectedIncidentID.subscribe(async (currentID) => {
    // Only fetch if panel is open and incident ID actually changed
    if (currentID && currentID !== previousIncidentID && get(panelOpen)) {
        previousIncidentID = currentID;
        
        // Clear error state when switching incidents
        sidebarError.set(null);
        
        // Load the incident data
        const incident = get(selectedIncident);
        await loadIncidentSidebarData(currentID);
        
        // Update metadata after successful fetch
        if (incident) {
            updateIncidentMetadata(currentID, incident);
        }
    } else if (!currentID) {
        // Clear when no incident selected
        previousIncidentID = null;
        sidebarData.set(null);
        sidebarError.set(null);
    } else {
        // Update previous ID tracker
        previousIncidentID = currentID;
    }
});

// Subscribe to panel open/close to trigger fetch when opening panel with selected incident
panelOpen.subscribe(async (isOpen) => {
    if (isOpen) {
        const incidentId = get(selectedIncidentID);
        if (incidentId) {
            const incident = get(selectedIncident);
            
            // Check if we should refetch based on metadata changes
            if (shouldRefetchIncident(incidentId, incident)) {
                await loadIncidentSidebarData(incidentId);
                if (incident) {
                    updateIncidentMetadata(incidentId, incident);
                }
            }
        }
    }
});

// Load incidents based on selected services
export async function loadOpenIncidents() {
    // Don't show loading state if polling
    if (!isPolling) {
        loading.set(true);
    }
    error.set(null);
    try {
        const services = await GetSelectedServices();
        const incidents = await GetOpenIncidents(services);
        
        // Track status changes
        const currentOpenIds = new Set(incidents.map((i: IncidentData) => i.incident_id));
        
        // Check if any incidents moved from open to resolved
        const movedToResolved = Array.from(lastOpenIncidentIds).filter(id => !currentOpenIds.has(id));
        
        if (movedToResolved.length > 0 && isPolling) {
            // Force reload resolved incidents to show newly resolved ones
            await loadResolvedIncidents();
        }
        
        lastOpenIncidentIds = currentOpenIds;
        openIncidents.set(incidents || []);
        
        // Check if selected incident's metadata changed and panel is open
        const currentIncidentID = get(selectedIncidentID);
        if (currentIncidentID && get(panelOpen)) {
            const currentIncident = incidents.find((i: IncidentData) => i.incident_id === currentIncidentID);
            if (currentIncident && shouldRefetchIncident(currentIncidentID, currentIncident)) {
                await loadIncidentSidebarData(currentIncidentID);
                updateIncidentMetadata(currentIncidentID, currentIncident);
            }
        }
    } catch (err) {
        error.set(err?.toString() || 'Failed to load open incidents');
        openIncidents.set([]);
    } finally {
        if (!isPolling) {
            loading.set(false);
        }
    }
}

export async function loadResolvedIncidents() {
    // Don't show loading state if polling
    if (!isPolling) {
        loading.set(true);
    }
    error.set(null);
    try {
        const services = await GetSelectedServices();
        const incidents = await GetResolvedIncidents(services);
        
        // Track resolved incidents
        const currentResolvedIds = new Set(incidents.map((i: IncidentData) => i.incident_id));
        lastResolvedIncidentIds = currentResolvedIds;
        
        resolvedIncidents.set(incidents || []);
        
        // Check if selected incident's metadata changed and panel is open
        const currentIncidentID = get(selectedIncidentID);
        if (currentIncidentID && get(panelOpen)) {
            const currentIncident = incidents.find((i: IncidentData) => i.incident_id === currentIncidentID);
            if (currentIncident && shouldRefetchIncident(currentIncidentID, currentIncident)) {
                await loadIncidentSidebarData(currentIncidentID);
                updateIncidentMetadata(currentIncidentID, currentIncident);
            }
        }
    } catch (err) {
        error.set(err?.toString() || 'Failed to load resolved incidents');
        resolvedIncidents.set([]);
    } finally {
        if (!isPolling) {
            loading.set(false);
        }
    }
}

export async function loadServicesConfig() {
    try {
        const config = await GetServicesConfig();
        servicesConfig.set(config);
        
        // Load selected services
        const selected = await GetSelectedServices();
        selectedServices.set(selected);
    } catch (err) {
        // No services configured yet
        servicesConfig.set(null);
        selectedServices.set([]);
    }
}

// Initialize event listeners for backend updates
export function initializeEventListeners() {
    // Listen for incident updates from backend polling
    EventsOn('incidents-updated', async (type: string) => {
        isPolling = true;
        
        if (type === 'both' || type === 'open') {
            // Store current state before updating
            const currentOpen = get(openIncidents);
            const currentOpenMap = new Map(currentOpen.map(i => [i.incident_id, i]));
            
            // Load new data
            await loadOpenIncidents();
            const newOpen = get(openIncidents);
            
            // Check for status changes
            for (const oldIncident of currentOpen) {
                const newIncident = newOpen.find(i => i.incident_id === oldIncident.incident_id);
                
                // If incident is no longer in open list or changed to resolved status
                if (!newIncident || newIncident.status === 'resolved') {
                    // Force reload resolved to show it there immediately
                    await loadResolvedIncidents();
                    break;
                }
                
                // Check for status changes within open (triggered -> acknowledged)
                if (newIncident && oldIncident.status !== newIncident.status) {
                    console.log(`Incident ${newIncident.incident_id} changed from ${oldIncident.status} to ${newIncident.status}`);
                }
            }
        }
        
        if (type === 'both' || type === 'resolved') {
            await loadResolvedIncidents();
        }
        
        isPolling = false;
    });
    
    // Listen for API key configuration
    EventsOn('api-key-configured', async () => {
        await loadOpenIncidents();
        await loadResolvedIncidents();
    });
    
    // Listen for services configuration updates
    EventsOn('services-config-updated', async () => {
        await loadServicesConfig();
        await loadOpenIncidents();
        await loadResolvedIncidents();
    });
}

// Helper function to format time
export function formatTime(date: Date | string): string {
    const d = typeof date === 'string' ? new Date(date) : date;
    const now = new Date();
    const diff = now.getTime() - d.getTime();
    
    const minutes = Math.floor(diff / 60000);
    const hours = Math.floor(diff / 3600000);
    const days = Math.floor(diff / 86400000);
    
    if (minutes < 1) return 'just now';
    if (minutes < 60) return `${minutes}m ago`;
    if (hours < 24) return `${hours}h ago`;
    if (days < 7) return `${days}d ago`;
    
    return d.toLocaleDateString();
}

// Helper function to get urgency level
export function getUrgency(incident: IncidentData): string {
    if (!incident.urgency) return 'low';
    return incident.urgency.toLowerCase();
}