<script lang="ts">
    import { openIncidents, resolvedIncidents, activeTab, loading } from '../stores/incidents';
    import IncidentCard from './IncidentCard.svelte';
    
    export let type: 'open' | 'resolved';
    
    $: incidents = type === 'open' ? $openIncidents : $resolvedIncidents;
    $: isActive = $activeTab === type;
</script>

{#if isActive}
    <div class="incident-panel">
        {#if $loading}
            <div class="loading-state">
                <div class="spinner"></div>
                <p>Loading incidents...</p>
            </div>
        {:else if incidents.length === 0}
            <div class="empty-state">
                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <circle cx="12" cy="12" r="10"></circle>
                    <path d="M8 14s1.5 2 4 2 4-2 4-2"></path>
                    <line x1="9" y1="9" x2="9.01" y2="9"></line>
                    <line x1="15" y1="9" x2="15.01" y2="9"></line>
                </svg>
                <h3>No {type} incidents</h3>
                <p>
                    {#if type === 'open'}
                        All systems operational. Great job!
                    {:else}
                        No resolved incidents in the past week.
                    {/if}
                </p>
            </div>
        {:else}
            <div class="incidents-list">
                {#each incidents as incident (incident.incident_id)}
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
    }
    
    .incidents-list {
        display: flex;
        flex-direction: column;
    }
</style>