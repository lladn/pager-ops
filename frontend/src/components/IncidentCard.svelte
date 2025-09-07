<!-- frontend/src/components/IncidentCard.svelte -->
<script lang="ts">
    import { database } from '../../wailsjs/go/models';
    import { formatTime, getUrgency } from '../stores/incidents';
    import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
    
    type IncidentData = database.IncidentData;
    
    export let incident: IncidentData;
    export let showAssignee: boolean = false;
    
    $: urgency = getUrgency(incident);
    $: statusColor = getStatusColor(incident.status);
    $: statusLabel = getStatusLabel(incident.status);
    
    function getStatusColor(status: string): string {
        switch (status) {
            case 'triggered': return 'bg-red-500';
            case 'acknowledged': return 'bg-yellow-500';
            case 'resolved': return 'bg-green-500';
            default: return 'bg-gray-500';
        }
    }
    
    function getStatusLabel(status: string): string {
        switch (status) {
            case 'triggered': return 'Triggered';
            case 'acknowledged': return 'Acknowledged';
            case 'resolved': return 'Resolved';
            default: return status;
        }
    }
    
    function openIncident() {
        if (incident.html_url) {
            BrowserOpenURL(incident.html_url);
        }
    }
</script>

<div class="incident-card" on:click={openIncident} role="button" tabindex="0">
    <div class="incident-header">
        <h3 class="incident-title" title={incident.title}>{incident.title}</h3>
        <div class="incident-badges">
            <span class="status-badge {statusColor}">
                {#if incident.status === 'triggered'}
                    <span class="status-icon">⚠</span>
                {:else if incident.status === 'acknowledged'}
                    <span class="status-icon">⏱</span>
                {:else if incident.status === 'resolved'}
                    <span class="status-icon">✓</span>
                {/if}
                {statusLabel}
            </span>
            <span class="urgency-badge urgency-{urgency}">{urgency}</span>
        </div>
    </div>
    
    <div class="incident-details">
        <span class="service-name">{incident.service_summary || 'Unknown Service'}</span>
        <span class="separator">•</span>
        <span class="incident-time">{formatTime(incident.created_at)}</span>
        {#if incident.alert_count > 0}
            <span class="separator">•</span>
            <span class="alert-count">{incident.alert_count} alert{incident.alert_count !== 1 ? 's' : ''}</span>
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
        cursor: pointer;
        transition: all 0.2s ease;
        position: relative;
    }
    
    .incident-card:hover {
        box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
        transform: translateY(-1px);
    }
    
    .incident-header {
        display: flex;
        justify-content: space-between;
        align-items: flex-start;
        margin-bottom: 12px;
        gap: 12px;
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
        gap: 4px;
        padding: 4px 10px;
        border-radius: 16px;
        color: white;
        font-size: 12px;
        font-weight: 500;
        white-space: nowrap;
    }
    
    .status-icon {
        font-size: 11px;
    }
    
    .bg-red-500 { background-color: #ef4444; }
    .bg-yellow-500 { background-color: #f59e0b; }
    .bg-green-500 { background-color: #10b981; }
    .bg-gray-500 { background-color: #6b7280; }
    
    .urgency-badge {
        padding: 4px 10px;
        border-radius: 16px;
        font-size: 12px;
        font-weight: 500;
        text-transform: uppercase;
        white-space: nowrap;
    }
    
    .urgency-high {
        background-color: #dc2626;
        color: white;
    }
    
    .urgency-medium {
        background-color: #f59e0b;
        color: white;
    }
    
    .urgency-low {
        background-color: #6b7280;
        color: white;
    }
    
    .incident-details {
        display: flex;
        align-items: center;
        color: #6b7280;
        font-size: 13px;
        flex-wrap: wrap;
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
</style>