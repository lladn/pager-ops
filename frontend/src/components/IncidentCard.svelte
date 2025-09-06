<script lang="ts">
    import type { IncidentData } from '../../wailsjs/go/models';
    import { formatTime, getUrgency } from '../stores/incidents';
    import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
    
    export let incident: IncidentData;
    export let showAssignee = false;
    
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
        <h3 class="incident-title">{incident.title}</h3>
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
    </div>
    
    {#if showAssignee}
        <div class="incident-assignee">
            <span class="assignee-label">Assigned to:</span>
            <span class="assignee-name">John Doe</span>
        </div>
    {/if}
    
    {#if incident.status === 'resolved'}
        <div class="incident-resolver">
            <span class="resolver-label">Resolved by:</span>
            <span class="resolver-name">Jane Smith</span>
        </div>
    {/if}
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
    }
    
    .incident-title {
        font-size: 16px;
        font-weight: 600;
        color: #111827;
        margin: 0;
        flex: 1;
        margin-right: 12px;
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
        font-size: 13px;
        font-weight: 500;
    }
    
    .status-icon {
        font-size: 12px;
    }
    
    .bg-red-500 { background-color: #ef4444; }
    .bg-yellow-500 { background-color: #f59e0b; }
    .bg-green-500 { background-color: #10b981; }
    .bg-gray-500 { background-color: #6b7280; }
    
    .urgency-badge {
        padding: 4px 10px;
        border-radius: 16px;
        font-size: 13px;
        font-weight: 500;
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
        font-size: 14px;
    }
    
    .service-name {
        font-weight: 500;
    }
    
    .separator {
        margin: 0 8px;
    }
    
    .incident-assignee,
    .incident-resolver {
        margin-top: 8px;
        color: #6b7280;
        font-size: 14px;
    }
    
    .assignee-label,
    .resolver-label {
        margin-right: 4px;
    }
    
    .assignee-name,
    .resolver-name {
        font-weight: 500;
        color: #374151;
    }
</style>