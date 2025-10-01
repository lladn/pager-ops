<script lang="ts">
    import { database } from '../../wailsjs/go/models';
    import { formatTime, selectedIncident } from '../stores/incidents';
    import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
    import { getServiceColor } from '../lib/serviceColors';
    
    type IncidentData = database.IncidentData;
    
    export let incident: IncidentData;
    
    $: statusColor = getStatusColor(incident.status);
    $: statusLabel = getStatusLabel(incident.status);
    $: serviceColor = getServiceColor(incident.service_summary || 'Unknown Service');
    
    function getStatusColor(status: string): string {
        switch (status) {
            case 'triggered':
                return '#ef4444'; // red
            case 'acknowledged':
                return '#f59e0b'; // amber / orange
            case 'resolved':
                return '#10b981'; // green
            default:
                return '#6b7280'; // gray
        }
    }
    
    function getStatusLabel(status: string): string {
        switch (status) {
            case 'triggered':
                return 'Triggered';
            case 'acknowledged':
                return 'Acknowledged';
            case 'resolved':
                return 'Resolved';
            default:
                return 'Unknown';
        }
    }
    
    function openIncident(event: MouseEvent) {
        event.stopPropagation();
        if (incident.html_url) {
            BrowserOpenURL(incident.html_url);
        }
    }
    
    function copyIncidentLink(event: MouseEvent) {
        event.stopPropagation();
        if (incident.html_url) {
            const linkText = `[${incident.title}](${incident.html_url})`;
            navigator.clipboard.writeText(linkText).then(() => {
                console.log('Link copied to clipboard');
            }).catch(err => {
                console.error('Failed to copy: ', err);
            });
        }
    }
    
    function handleCardClick() {
        // Only update selection, DO NOT open panel
        // Panel is controlled only by toolbar button
        selectedIncident.set(incident);
    }
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="incident-card" on:click={handleCardClick}>
    <div class="action-buttons">
        <button class="action-button" on:click={openIncident} title="Open incident">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                <polyline points="15 3 21 3 21 9"></polyline>
                <line x1="10" y1="14" x2="21" y2="3"></line>
            </svg>
        </button>
        <button class="action-button" on:click={copyIncidentLink} title="Copy link">
            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
                <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
            </svg>
        </button>
    </div>
    
    <div class="incident-header">
        <h3 class="incident-title" title={incident.title}>{incident.title}</h3>
        <div class="incident-badges">
            <span class="status-badge outlined" style="color: {statusColor}; border-color: {statusColor}">
                {#if incident.status === 'triggered'}
                    <span class="status-icon">⚠</span>
                {:else if incident.status === 'acknowledged'}
                    <span class="status-icon">⏱</span>
                {:else if incident.status === 'resolved'}
                    <span class="status-icon">✓</span>
                {/if}
                {statusLabel}
            </span>
        </div>
    </div>
    
    <div class="incident-details">
        <span class="service-name" style="color: {serviceColor}">{incident.service_summary || 'Unknown Service'}</span>
        <span class="separator">•</span>
        <span class="incident-time">{formatTime(incident.created_at)}</span>
        {#if incident.alert_count > 0}
            <span class="separator">•</span>
            <span class="alert-count">{incident.alert_count} alert{incident.alert_count !== 1 ? 's' : ''}</span>
        {/if}
        {#if incident.incident_number}
            <span class="separator">•</span>
            <span class="incident-number">#{incident.incident_number}</span>
        {/if}
    </div>
</div>

<style>
    .incident-card {
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        padding: 16px;
        margin-bottom: 12px;
        transition: all 0.2s ease;
        position: relative;
        cursor: pointer;
    }
    
    .incident-card:hover {
        box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
        transform: translateY(-1px);
    }
    
    .action-buttons {
        position: absolute;
        top: 12px;
        right: 12px;
        display: flex;
        gap: 8px;
        z-index: 1;
    }
    
    .action-button {
        padding: 6px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        cursor: pointer;
        color: #6b7280;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
    }
    
    .action-button:hover {
        background: #f3f4f6;
        color: #111827;
        border-color: #d1d5db;
    }
    
    .incident-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 12px;
        gap: 12px;
        padding-right: 80px;
    }
    
    .incident-title {
        font-size: 15px;
        font-weight: 600;
        color: #111827;
        margin: 0;
        flex: 1;
        line-height: 1.4;
        overflow: hidden;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
    }
    
    .incident-badges {
        display: flex;
        gap: 8px;
        flex-shrink: 0;
    }
    
    .status-badge {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        font-size: 13px;
        font-weight: 600;
        white-space: nowrap;
        background: none;
    }

    /* outlined variant */
    .status-badge.outlined {
        background: white;            
        border: 1px solid transparent; 
        padding: 2px 8px;
        border-radius: 6px;
    }
    
    .status-icon {
        font-size: 11px;
    }
    
    .incident-details {
        display: flex;
        align-items: center;
        color: #6b7280;
        font-size: 13px;
        flex-wrap: wrap;
        gap: 0;
    }
    
    .service-name {
        font-weight: 500;
    }
    
    .separator {
        margin: 0 8px;
    }
    
    .alert-count {
        font-weight: 500;
        color: #dc2626;
    }
    
    .incident-number {
        font-weight: 500;
        color: #4b5563;
    }
</style>