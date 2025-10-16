<script lang="ts">
    import { 
        openIncidents, 
        resolvedIncidents, 
        activeTab, 
        loading,
        serviceFilterLoading } from '../stores/incidents';
    import IncidentCard from './IncidentCard.svelte';
    import type { database } from '../../wailsjs/go/models';
    
    export let type: 'open' | 'resolved';
    export let searchQuery: string = '';
    export let sortBy: 'time' | 'service' | 'alerts' = 'time';
    
    type IncidentData = database.IncidentData;
    
    $: incidents = type === 'open' ? $openIncidents : $resolvedIncidents;
    $: filteredIncidents = filterIncidents(incidents, searchQuery);
    $: sortedIncidents = sortIncidents(filteredIncidents, sortBy);
    $: isActive = $activeTab === type;
    
    function filterIncidents(incidentsList: IncidentData[], query: string): IncidentData[] {
        if (!query || !query.trim()) return incidentsList;
        
        const lowerQuery = query.toLowerCase().trim();
        return incidentsList.filter((incident: IncidentData) => {
            if (incident.title?.toLowerCase().includes(lowerQuery)) return true;
            if (incident.service_summary?.toLowerCase().includes(lowerQuery)) return true;
            if (incident.incident_id?.toLowerCase().includes(lowerQuery)) return true;
            if (incident.incident_number?.toString().includes(lowerQuery)) return true;
            if (incident.status?.toLowerCase().includes(lowerQuery)) return true;
            return false;
        });
    }
    
    function sortIncidents(incidentsList: IncidentData[], sortOption: string): IncidentData[] {
        const sorted = [...incidentsList];
        
        switch (sortOption) {
            case 'service':
                return sorted.sort((a, b) => {
                    const serviceA = a.service_summary || '';
                    const serviceB = b.service_summary || '';
                    return serviceA.localeCompare(serviceB);
                });
            case 'alerts':
                return sorted.sort((a, b) => {
                    const alertsA = a.alert_count || 0;
                    const alertsB = b.alert_count || 0;
                    return alertsB - alertsA;
                });
            case 'time':
            default:
                return sorted.sort((a, b) => {
                    const timeA = new Date(a.created_at).getTime();
                    const timeB = new Date(b.created_at).getTime();
                    return timeB - timeA;
                });
        }
    }
</script>

{#if isActive}
    <div class="incident-panel">
        {#if $serviceFilterLoading}
            <div class="service-filter-loading">
                <div class="loading-overlay">
                    <div class="spinner"></div>
                    <p>Updating incidents...</p>
                </div>
            </div>
        {/if}
        
        {#if $loading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading incidents...</p>
            </div>
        {:else if sortedIncidents.length === 0}
            <div class="empty-state">
                {#if searchQuery}
                    <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="11" cy="11" r="8"></circle>
                        <path d="m21 21-4.35-4.35"></path>
                    </svg>
                    <h3>No matching incidents</h3>
                    <p style="text-align: center;">
                        No {type} incidents match your search for "{searchQuery}"
                    </p>
                {:else}
                    <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <circle cx="12" cy="12" r="10"></circle>
                        <path d="M8 14s1.5 2 4 2 4-2 4-2"></path>
                        <line x1="9" y1="9" x2="9.01" y2="9"></line>
                        <line x1="15" y1="9" x2="15.01" y2="9"></line>
                    </svg>
                    <h3>No {type} incidents</h3>
                    <p style="text-align: center;">
                        {#if type === 'open'}
                            No News is a Good News
                        {:else}
                            No resolved incidents in the past week.
                        {/if}
                    </p>
                {/if}
            </div>
        {:else}
            <div class="incidents-list">
                {#if searchQuery}
                    <div class="search-results-header">
                        Found {sortedIncidents.length} {type} incident{sortedIncidents.length !== 1 ? 's' : ''} matching "{searchQuery}"
                    </div>
                {/if}
                {#each sortedIncidents as incident (incident.incident_id)}
                    <IncidentCard {incident} />
                {/each}
            </div>
        {/if}
    </div>
{/if}

<style>
    .incident-panel {
        position: relative;
        width: 100%;
        height: 100%;
        overflow: hidden;
    }
    
    .service-filter-loading {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 100;
        pointer-events: all;
    }
    
    .loading-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(255, 255, 255, 0.95);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 12px;
    }
    
    .loading-overlay p {
        color: #6b7280;
        font-size: 14px;
        margin: 0;
    }
    
    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        gap: 12px;
    }
    
    .loading-state p {
        color: #6b7280;
        font-size: 14px;
        margin: 0;
    }
    
    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid #e5e7eb;
        border-top-color: #3b82f6;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        color: #9ca3af;
        gap: 16px;
        padding: 32px;
    }
    
    .empty-state h3 {
        margin: 0;
        color: #6b7280;
        font-size: 18px;
        font-weight: 600;
    }
    
    .empty-state p {
        margin: 0;
        color: #9ca3af;
        font-size: 14px;
    }
    
    .incidents-list {
        height: 100%;
        overflow-y: auto;
        padding: 16px;
    }
    
    .search-results-header {
        padding: 8px 12px;
        background: #f3f4f6;
        border-radius: 6px;
        margin-bottom: 12px;
        font-size: 13px;
        color: #6b7280;
    }
</style>