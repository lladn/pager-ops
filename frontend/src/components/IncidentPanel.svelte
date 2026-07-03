<script lang="ts">
    import { 
        openIncidents, 
        resolvedIncidents, 
        activeTab, 
        loading,
        serviceFilterLoading,
        activeServiceFilter,
        showServicePills } from '../stores/incidents';
    import IncidentCard from './IncidentCard.svelte';
    import { getServiceColor } from '../lib/serviceColors';
    import type { database } from '../../wailsjs/go/models';
    
    export let type: 'open' | 'resolved';
    export let searchQuery: string = '';
    export let sortBy: 'time' | 'service' | 'alerts' = 'time';
    
    type IncidentData = database.IncidentData;
    
    $: incidents = type === 'open' ? $openIncidents : $resolvedIncidents;
    $: filteredIncidents = filterIncidents(incidents, searchQuery);
    $: sortedIncidents = sortIncidents(filteredIncidents, sortBy);
    $: isActive = $activeTab === type;

    // Derive the unique services present in the current incident list
    $: serviceNames = getUniqueServices(incidents);

    // Final list after applying the service tab filter
    $: displayedIncidents = $activeServiceFilter === 'all'
        ? sortedIncidents
        : sortedIncidents.filter(i => (i.service_summary || 'Unknown Service') === $activeServiceFilter);

    function getUniqueServices(list: IncidentData[]): string[] {
        const seen = new Set<string>();
        list.forEach(i => seen.add(i.service_summary || 'Unknown Service'));
        return Array.from(seen).sort((a, b) => a.localeCompare(b));
    }

    function setServiceFilter(name: string) {
        activeServiceFilter.set(name);
    }

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

    // Count incidents per service for the badge
    function countForService(name: string): number {
        return sortedIncidents.filter(i => (i.service_summary || 'Unknown Service') === name).length;
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

        <!-- Service pill tabs — only visible when 2+ services are present and setting is enabled -->
        {#if $showServicePills && serviceNames.length >= 2}
            <div class="service-tabs">
                <button
                    class="service-pill"
                    class:active={$activeServiceFilter === 'all'}
                    on:click={() => setServiceFilter('all')}
                >
                    All
                    <span class="pill-count">{sortedIncidents.length}</span>
                </button>
                {#each serviceNames as name}
                    <button
                        class="service-pill"
                        class:active={$activeServiceFilter === name}
                        style="--pill-color: {getServiceColor(name)}"
                        on:click={() => setServiceFilter(name)}
                    >
                        <span class="pill-dot" style="background: {getServiceColor(name)}"></span>
                        <span class="pill-label">{name}</span>
                        <span class="pill-count">{countForService(name)}</span>
                    </button>
                {/each}
            </div>
        {/if}
        
        {#if $loading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading incidents...</p>
            </div>
        {:else if displayedIncidents.length === 0}
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
                        Found {displayedIncidents.length} {type} incident{displayedIncidents.length !== 1 ? 's' : ''} matching "{searchQuery}"
                    </div>
                {/if}
                {#each displayedIncidents as incident (incident.incident_id)}
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
        display: flex;
        flex-direction: column;
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
        background: var(--bg-overlay);
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        gap: 12px;
    }
    
    .loading-overlay p {
        color: var(--text-tertiary);
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
        color: var(--text-tertiary);
        font-size: 14px;
        margin: 0;
    }
    
    .spinner {
        width: 32px;
        height: 32px;
        border: 3px solid var(--border);
        border-top-color: var(--accent);
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
        color: var(--text-muted);
        gap: 16px;
        padding: 32px;
    }
    
    .empty-state h3 {
        margin: 0;
        color: var(--text-tertiary);
        font-size: 18px;
        font-weight: 600;
    }
    
    .empty-state p {
        margin: 0;
        color: var(--text-muted);
        font-size: 14px;
    }
    
    .incidents-list {
        flex: 1;
        overflow-y: auto;
        padding: 16px;
    }
    
    .search-results-header {
        padding: 8px 12px;
        background: var(--bg-tertiary);
        border-radius: 6px;
        margin-bottom: 12px;
        font-size: 13px;
        color: var(--text-tertiary);
    }

    /* ── Service pill tabs ─────────────────────────────────── */
    .service-tabs {
        display: flex;
        align-items: center;
        justify-content: center;
        flex-wrap: wrap;
        gap: 6px;
        padding: 8px 16px 6px;
        border-bottom: 1px solid var(--border);
        flex-shrink: 0;
    }

    .service-pill {
        display: inline-flex;
        align-items: center;
        gap: 5px;
        padding: 4px 10px;
        border-radius: 20px;
        border: 1px solid var(--border);
        background: var(--bg-secondary);
        color: var(--text-tertiary);
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        white-space: nowrap;
        max-width: 160px;
        flex-shrink: 0;
        transition: all 0.15s ease;
    }

    .pill-label {
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
    }

    .service-pill:hover {
        background: var(--bg-tertiary);
        border-color: var(--border-strong);
        color: var(--text-secondary);
    }

    .service-pill.active {
        background: var(--accent-soft);
        border-color: var(--accent-border);
        color: var(--accent);
    }

    .service-pill.active[style] {
        background: color-mix(in srgb, var(--pill-color) 15%, transparent);
        border-color: color-mix(in srgb, var(--pill-color) 40%, transparent);
        color: var(--pill-color);
    }

    .pill-dot {
        width: 7px;
        height: 7px;
        border-radius: 50%;
        flex-shrink: 0;
    }

    .pill-count {
        background: var(--bg-tertiary);
        color: var(--text-muted);
        padding: 1px 6px;
        border-radius: 10px;
        font-size: 11px;
        font-weight: 600;
        min-width: 18px;
        text-align: center;
    }

    .service-pill.active .pill-count {
        background: var(--accent-soft-strong);
        color: var(--accent);
    }

</style>