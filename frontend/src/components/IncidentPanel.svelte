<script lang="ts">
    import { openIncidents, resolvedIncidents, activeTab, loading } from '../stores/incidents';
    import IncidentCard from './IncidentCard.svelte';
    import type { database } from '../../wailsjs/go/models';
    
    export let type: 'open' | 'resolved';
    export let searchQuery: string = '';
    
    type IncidentData = database.IncidentData;
    
    $: incidents = type === 'open' ? $openIncidents : $resolvedIncidents;
    $: filteredIncidents = filterIncidents(incidents, searchQuery);
    $: isActive = $activeTab === type;
    
    function filterIncidents(incidentsList: IncidentData[], query: string): IncidentData[] {
        if (!query || !query.trim()) return incidentsList;
        
        const lowerQuery = query.toLowerCase().trim();
        return incidentsList.filter((incident: IncidentData) => {
            // Search in title
            if (incident.title?.toLowerCase().includes(lowerQuery)) return true;
            
            // Search in service name
            if (incident.service_summary?.toLowerCase().includes(lowerQuery)) return true;
            
            // Search in incident ID
            if (incident.incident_id?.toLowerCase().includes(lowerQuery)) return true;
            
            // Search in incident number
            if (incident.incident_number?.toString().includes(lowerQuery)) return true;
            
            // Search in status
            if (incident.status?.toLowerCase().includes(lowerQuery)) return true;
            
            // Search in urgency
            if (incident.urgency?.toLowerCase().includes(lowerQuery)) return true;
            
            return false;
        });
    }
</script>

{#if isActive}
    <div class="incident-panel">
        {#if $loading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading incidents...</p>
            </div>
        {:else if filteredIncidents.length === 0}
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
                        Found {filteredIncidents.length} {type} incident{filteredIncidents.length !== 1 ? 's' : ''} matching "{searchQuery}"
                    </div>
                {/if}
                {#each filteredIncidents as incident (incident.incident_id)}
                    <IncidentCard {incident} showAssignee={type === 'open'} />
                {/each}
            </div>
        {/if}
    </div>
{/if}

<style>
    .incident-panel {
        height: 100%;
        overflow-y: auto;
        padding: 16px;
    }
    
    .loading-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 400px;
        color: #6b7280;
    }
    
    .spinner {
        width: 48px;
        height: 48px;
        border: 3px solid #e5e7eb;
        border-top-color: #3b82f6;
        border-radius: 50%;
        animation: spin 1s linear infinite;
        margin-bottom: 16px;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 400px;
        color: #9ca3af;
    }
    
    .empty-state svg {
        margin-bottom: 16px;
    }
    
    .empty-state h3 {
        font-size: 18px;
        font-weight: 600;
        color: #6b7280;
        margin: 0 0 8px 0;
    }
    
    .empty-state p {
        color: #9ca3af;
        margin: 0;
        max-width: 300px;
    }
    
    .incidents-list {
        display: flex;
        flex-direction: column;
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