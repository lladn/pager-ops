<script lang="ts">
    import { panelOpen, panelWidth, activeTab, openIncidents, resolvedIncidents, selectedIncident } from '../stores/incidents';
    import type { database } from '../../wailsjs/go/models';
    import PanelAlerts from './PanelAlerts.svelte';
    import PanelNotes from './PanelNotes.svelte';
    
    type IncidentData = database.IncidentData;
    
    const MIN_WIDTH = 280;
    const MAX_WIDTH = 600;
    
    let isResizing = false;
    let startX = 0;
    let startWidth = 0;
    let panelTab: 'alerts' | 'notes' = 'alerts';
    
    
    // Handle tab switching behavior
    $: if ($activeTab) {
        // When switching tabs, check if selected incident exists in new tab
        if ($selectedIncident) {
            const incidents = $activeTab === 'open' ? $openIncidents : $resolvedIncidents;
            const incidentExists = incidents.some(i => i.incident_id === $selectedIncident?.incident_id);
            
            if (!incidentExists) {
                // Clear selection if incident doesn't exist in new tab
                selectedIncident.set(null);
            }
        }
    }
    
    function startResize(event: MouseEvent) {
        isResizing = true;
        startX = event.clientX;
        startWidth = $panelWidth;
        
        document.body.style.cursor = 'ew-resize';
        document.body.style.userSelect = 'none';
        
        document.addEventListener('mousemove', handleResize);
        document.addEventListener('mouseup', stopResize);
    }
    
    function handleResize(event: MouseEvent) {
        if (!isResizing) return;
        
        const delta = startX - event.clientX;
        const newWidth = Math.max(MIN_WIDTH, Math.min(MAX_WIDTH, startWidth + delta));
        
        panelWidth.set(newWidth);
    }
    
    function stopResize() {
        isResizing = false;
        document.body.style.cursor = '';
        document.body.style.userSelect = '';
        
        document.removeEventListener('mousemove', handleResize);
        document.removeEventListener('mouseup', stopResize);
    }
    
    function closePanel() {
        panelOpen.set(false);
        selectedIncident.set(null);
    }
</script>

{#if $panelOpen}
    <div class="panel-container" style="width: {$panelWidth}px;">
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="resize-handle" on:mousedown={startResize}></div>
        
        <div class="panel-header">
            {#if $selectedIncident}
                <h5>Incident Details</h5>
            {:else}
                <h5>Select an incident to view details</h5>
            {/if}
            <button class="close-button" on:click={closePanel} title="Close Panel">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
            </button>
        </div>
        
        {#if $selectedIncident}
            <!-- Show incident title and service -->
            <div class="incident-info">
                <div class="incident-title-section">
                    <h4>{$selectedIncident.title}</h4>
                    <span class="service-badge">{$selectedIncident.service_summary}</span>
                </div>
            </div>
            
            <!-- Tab Navigation -->
            <div class="panel-tabs">
                <button 
                    class="panel-tab" 
                    class:active={panelTab === 'alerts'}
                    on:click={() => panelTab = 'alerts'}
                >
                    Alerts
                </button>
                <button 
                    class="panel-tab" 
                    class:active={panelTab === 'notes'}
                    on:click={() => panelTab = 'notes'}
                >
                    Notes
                </button>
            </div>
            
            <!-- Tab Content -->
            <div class="panel-content">
                {#if panelTab === 'alerts'}
                    <PanelAlerts incident={$selectedIncident} />
                {:else if panelTab === 'notes'}
                    <PanelNotes incident={$selectedIncident} />
                {/if}
            </div>
        {:else}
            <!-- Empty State -->
            <div class="panel-empty-state">
                <svg width="64" height="64" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M9 2v6h6V2"></path>
                    <path d="M16 8V2H8v6"></path>
                    <rect x="4" y="8" width="16" height="14" rx="2"></rect>
                </svg>
                <p>Select an incident to view details</p>
            </div>
        {/if}
    </div>
{/if}

<style>
    .panel-container {
        height: 100%;
        background: white;
        border-left: 1px solid #e0e0e0;
        display: flex;
        flex-direction: column;
        flex-shrink: 0;
        position: relative;
        min-width: 280px;
        max-width: 600px;
    }
    
    .resize-handle {
        position: absolute;
        left: 0;
        top: 0;
        bottom: 0;
        width: 4px;
        cursor: ew-resize;
        background: transparent;
        z-index: 10;
        transition: background 0.2s;
    }
    
    .resize-handle:hover {
        background: rgba(59, 130, 246, 0.3);
    }
    
    .resize-handle:active {
        background: rgba(59, 130, 246, 0.5);
    }
    
    .panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 14px;
        border-bottom: 1px solid #e0e0e0;
        background: #fafafa;
    }
    
    .panel-header h5 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
        color: #2c2c2c;
    }
    
    .close-button {
        background: transparent;
        border: none;
        padding: 4px;
        border-radius: 4px;
        cursor: pointer;
        color: #666;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
    }
    
    .close-button:hover {
        background: rgba(0, 0, 0, 0.06);
        color: #333;
    }
    
    .incident-info {
        padding: 16px;
        border-bottom: 1px solid #e0e0e0;
        background: #f9fafb;
    }
    
    .incident-title-section h4 {
        margin: 0 0 8px 0;
        font-size: 15px;
        font-weight: 600;
        color: #111827;
        line-height: 1.4;
    }
    
    .service-badge {
        display: inline-block;
        padding: 4px 10px;
        background: #3b82f6;
        color: white;
        border-radius: 12px;
        font-size: 12px;
        font-weight: 500;
    }
    
    .panel-tabs {
        display: flex;
        border-bottom: 1px solid #e0e0e0;
        background: #f9fafb;
    }
    
    .panel-tab {
        flex: 1;
        padding: 12px 16px;
        background: transparent;
        border: none;
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        color: #6b7280;
        border-bottom: 2px solid transparent;
        transition: all 0.2s;
    }
    
    .panel-tab:hover {
        color: #374151;
        background: rgba(255, 255, 255, 0.5);
    }
    
    .panel-tab.active {
        color: #3b82f6;
        background: white;
        border-bottom-color: #3b82f6;
    }
    
    .panel-content {
        flex: 1;
        overflow-y: auto;
    }
    
    .panel-empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 100%;
        padding: 32px;
        color: #9ca3af;
    }
    
    .panel-empty-state svg {
        margin-bottom: 16px;
        opacity: 0.5;
    }
    
    .panel-empty-state p {
        margin: 0;
        font-size: 14px;
        text-align: center;
        color: #6b7280;
    }
    
    /* Custom scrollbar */
    .panel-content::-webkit-scrollbar {
        width: 8px;
    }
    
    .panel-content::-webkit-scrollbar-track {
        background: #f3f4f6;
    }
    
    .panel-content::-webkit-scrollbar-thumb {
        background: #d1d5db;
        border-radius: 4px;
    }
    
    .panel-content::-webkit-scrollbar-thumb:hover {
        background: #9ca3af;
    }
</style>