<script lang="ts">
    import { panelOpen, panelWidth, activeTab, openIncidents, resolvedIncidents, selectedIncident } from '../stores/incidents';
    import type { database, store } from '../../wailsjs/go/models';
    import PanelAlerts from './PanelAlerts.svelte';
    import PanelNotes from './PanelNotes.svelte';
    import PanelCustomField from './PanelCustomField.svelte';
    import { getServiceColor } from '../lib/serviceColors';
    import { ResolveIncident, GetIncidentCustomFields } from '../../wailsjs/go/main/App';

    type IncidentData = database.IncidentData;
    type CustomField = store.CustomField;

    const MIN_WIDTH = 280;
    const MAX_WIDTH = 600;

    let isResizing = false;
    let startX = 0;
    let startWidth = 0;
    // Tab can be 'alerts', 'notes', or a custom field id.
    let panelTab: string = 'alerts';
    let resolving = false;

    // Custom fields for the selected incident; each becomes its own tab.
    let customFields: CustomField[] = [];
    let fieldsIncidentId = '';
    let fieldsLoading = false;

    // Load custom fields whenever the selected incident changes.
    $: if ($selectedIncident?.incident_id && $selectedIncident.incident_id !== fieldsIncidentId) {
        fieldsIncidentId = $selectedIncident.incident_id;
        loadCustomFields(fieldsIncidentId);
    } else if (!$selectedIncident) {
        customFields = [];
        fieldsIncidentId = '';
        fieldsLoading = false;
    }

    // If the active field tab disappears (e.g. switching incidents), fall back to Alerts.
    $: if (panelTab !== 'alerts' && panelTab !== 'notes' && !customFields.some(f => f.id === panelTab)) {
        panelTab = 'alerts';
    }

    async function loadCustomFields(incidentId: string) {
        fieldsLoading = true;
        try {
            const result = await GetIncidentCustomFields(incidentId);
            if (fieldsIncidentId === incidentId) {
                customFields = result || [];
            }
        } catch (err) {
            console.error('Failed to load custom fields:', err);
            if (fieldsIncidentId === incidentId) {
                customFields = [];
            }
        } finally {
            if (fieldsIncidentId === incidentId) {
                fieldsLoading = false;
            }
        }
    }

    // ── Runbook Status gate ───────────────────────────────────────────────────
    // Find the Runbook Status field by display_name (case-insensitive).
    $: runbookField = customFields.find(
        f => f.display_name?.toLowerCase() === 'runbook status'
    );
    // Populated = field exists and has a non-empty saved value.
    $: runbookPopulated = (
        runbookField != null &&
        runbookField.value != null &&
        String(runbookField.value).trim() !== ''
    );
    // Block resolve while fields are loading OR runbook field exists but is unpopulated.
    $: resolveBlocked = fieldsLoading || (runbookField != null && !runbookPopulated);
    $: resolveBlockReason = fieldsLoading
        ? 'Loading field status…'
        : 'Set and save Runbook Status before resolving';

    // Replace a single field in place after it's saved, so its value/saved-state stays in sync.
    function handleFieldSaved(event: CustomEvent<CustomField>) {
        const updated = event.detail;
        customFields = customFields.map(f => (f.id === updated.id ? updated : f));
    }

    $: serviceColor = $selectedIncident ? getServiceColor($selectedIncident.service_summary || 'Unknown Service') : '#6b7280';
    
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
        event.preventDefault(); 
        isResizing = true;
        startX = event.clientX;
        startWidth = $panelWidth;
        
        document.body.style.cursor = 'ew-resize';
        document.body.style.userSelect = 'none';
        document.body.classList.add('resizing');
        
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
        document.body.classList.remove('resizing'); 
        
        document.removeEventListener('mousemove', handleResize);
        document.removeEventListener('mouseup', stopResize);
    }
    
    function closePanel() {
        panelOpen.set(false);
        selectedIncident.set(null);
    }
    
    async function handleResolve(event: MouseEvent) {
        event.stopPropagation();

        if (resolving || !$selectedIncident || resolveBlocked) return;

        resolving = true;

        try {
            await ResolveIncident($selectedIncident.incident_id);
            console.log(`Incident ${$selectedIncident.incident_id} resolved successfully`);
        } catch (err) {
            console.error('Failed to resolve incident:', err);
            alert(`Failed to resolve incident: ${err}`);
        } finally {
            resolving = false;
        }
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
                    <div class="service-row">
                        <span class="service-name" style="color: {serviceColor}">
                            {$selectedIncident.service_summary}
                        </span>
                        
                        {#if $activeTab === 'open' && $selectedIncident.status !== 'resolved'}
                            <button 
                                class="resolve-button" 
                                class:loading={resolving}
                                class:blocked={resolveBlocked}
                                on:click={handleResolve}
                                disabled={resolving || resolveBlocked}
                                title={resolveBlocked ? resolveBlockReason : 'Resolve incident'}
                            >
                                {#if resolving}
                                    <span class="spinner"></span>
                                    Resolving...
                                {:else}
                                    ✓ Resolve
                                {/if}
                            </button>
                        {/if}
                    </div>
                </div>
                {#if $activeTab === 'open' && $selectedIncident.status !== 'resolved' && resolveBlocked && !fieldsLoading}
                    <div class="runbook-warning">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
                            <path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/>
                            <line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/>
                        </svg>
                        Set and save <strong>Runbook Status</strong> before resolving
                    </div>
                {/if}
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
                {#each customFields as field (field.id)}
                    <button
                        class="panel-tab"
                        class:active={panelTab === field.id}
                        on:click={() => panelTab = field.id}
                        title={field.display_name}
                    >
                        {field.display_name}
                    </button>
                {/each}
            </div>

            <!-- Tab Content -->
            <div class="panel-content">
                {#if panelTab === 'alerts'}
                    <PanelAlerts incident={$selectedIncident} />
                {:else if panelTab === 'notes'}
                    <PanelNotes incident={$selectedIncident} />
                {:else}
                    {#each customFields as field (field.id)}
                        {#if panelTab === field.id}
                            <PanelCustomField
                                {field}
                                incidentId={$selectedIncident.incident_id}
                                on:saved={handleFieldSaved}
                            />
                        {/if}
                    {/each}
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
        background: var(--bg-primary);
        border-left: 1px solid var(--border);
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
        -webkit-user-select: none;
        -moz-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .resize-handle:hover {
        background: var(--accent-soft);
    }
    
    .resize-handle:active {
        background: var(--accent-soft-strong);
    }
    
    .panel-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 14px;
        border-bottom: 1px solid var(--border);
        background: var(--bg-secondary);
        flex-shrink: 0;
        -webkit-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .panel-header h5 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
        color: var(--text-primary);
    }
    
    .close-button {
        background: transparent;
        border: none;
        padding: 4px;
        border-radius: 4px;
        cursor: pointer;
        color: var(--text-tertiary);
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
    }
    
    .close-button:hover {
        background: var(--bg-active);
        color: var(--text-primary);
    }
    
    .incident-info {
        padding: 16px;
        border-bottom: 1px solid var(--border);
        background: var(--bg-secondary);
        flex-shrink: 0;
    }
    
    .incident-title-section h4 {
        margin: 0 0 8px 0;
        font-size: 15px;
        font-weight: 600;
        color: var(--text-primary);
        line-height: 1.4;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .service-row {
        display: flex;
        align-items: center;
        gap: 12px;
        justify-content: space-between;
    }
    
    .service-name {
        font-size: 13px;
        font-weight: 500;
        display: block;
        flex: 1;
    }
    
    .resolve-button {
        padding: 4px 10px;
        background: var(--success);
        color: var(--text-on-accent);
        border: none;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        display: flex;
        align-items: center;
        gap: 4px;
        white-space: nowrap;
    }
    
    .resolve-button:hover:not(:disabled) {
        background: var(--success-hover);
    }
    
    .resolve-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .resolve-button.blocked {
        background: var(--bg-tertiary);
        color: var(--text-muted);
        border: 1px solid var(--border);
        cursor: not-allowed;
        opacity: 1;
    }
    
    .resolve-button.loading {
        background: var(--text-tertiary);
    }

    .runbook-warning {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-top: 8px;
        padding: 6px 10px;
        background: var(--warning-soft, rgba(234, 179, 8, 0.1));
        border: 1px solid var(--warning-border, rgba(234, 179, 8, 0.3));
        border-radius: 6px;
        font-size: 12px;
        color: var(--warning-text, #b45309);
    }

    .runbook-warning svg {
        flex-shrink: 0;
        stroke: currentColor;
    }

    .runbook-warning strong {
        font-weight: 600;
    }
    
    .spinner {
        width: 12px;
        height: 12px;
        border: 2px solid #fff;
        border-top-color: transparent;
        border-radius: 50%;
        animation: spin 0.6s linear infinite;
    }
    
    @keyframes spin {
        to { transform: rotate(360deg); }
    }
    
    .panel-tabs {
        display: flex;
        border-bottom: 1px solid var(--border);
        background: var(--bg-primary);
        flex-shrink: 0;
        overflow-x: auto;
        -webkit-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .panel-tab {
        flex: 1;
        padding: 12px 16px;
        background: transparent;
        border: none;
        border-bottom: 2px solid transparent;
        font-size: 14px;
        font-weight: 500;
        color: var(--text-tertiary);
        cursor: pointer;
        transition: all 0.2s;
        white-space: nowrap;
        -webkit-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .panel-tab:hover {
        background: var(--bg-secondary);
        color: var(--text-secondary);
    }
    
    .panel-tab.active {
        color: var(--accent);
        border-bottom-color: var(--accent);
    }
    
    .panel-content {
        flex: 1;
        overflow-y: auto;
        min-height: 0;
    }
    
    .panel-empty-state {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 48px 24px;
        color: var(--text-muted);
        text-align: center;
    }
    
    .panel-empty-state svg {
        margin-bottom: 16px;
        opacity: 0.5;
    }
    
    .panel-empty-state p {
        margin: 0;
        font-size: 14px;
        color: var(--text-tertiary);
    }
</style>