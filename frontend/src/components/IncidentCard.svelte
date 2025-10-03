<script lang="ts">
    import { database } from '../../wailsjs/go/models';
    import { formatTime, selectedIncident, userAcknowledgedIncidents, markIncidentAcknowledged } from '../stores/incidents';
    import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
    import { AcknowledgeIncident } from '../../wailsjs/go/main/App';
    import { getServiceColor } from '../lib/serviceColors';
    
    type IncidentData = database.IncidentData;
    
    export let incident: IncidentData;
    
    $: statusColor = getStatusColor(incident.status);
    $: statusLabel = getStatusLabel(incident.status);
    $: serviceColor = getServiceColor(incident.service_summary || 'Unknown Service');
    $: isSelected = $selectedIncident?.incident_id === incident.incident_id;
    
    // Acknowledgment button visibility logic
    $: userAck = $userAcknowledgedIncidents.get(incident.incident_id);
    $: showAckButton = 
        incident.status !== 'resolved' && 
        (
            !userAck || // Never acknowledged
            new Date(incident.updated_at) > new Date(userAck.updated_at) // Incident updated after user ack'd
        );
    
    // Feedback states for action buttons
    let copyFeedback = '';
    let openFeedback = '';
    
    // Loading state for acknowledge button
    let acknowledging = false;
    
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
            openFeedback = 'Opening...';
            BrowserOpenURL(incident.html_url);
            setTimeout(() => {
                openFeedback = '';
            }, 1500);
        }
    }
    
    function copyIncidentLink(event: MouseEvent) {
        event.stopPropagation();
        if (incident.html_url) {
            const linkText = `[${incident.title}](${incident.html_url})`;
            navigator.clipboard.writeText(linkText).then(() => {
                copyFeedback = 'Copied!';
                setTimeout(() => {
                    copyFeedback = '';
                }, 1500);
            }).catch(err => {
                console.error('Failed to copy: ', err);
                copyFeedback = 'Failed';
                setTimeout(() => {
                    copyFeedback = '';
                }, 1500);
            });
        }
    }
    
    async function handleAcknowledge(event: MouseEvent) {
        event.stopPropagation();
        
        if (acknowledging) return;
        
        acknowledging = true;
        
        try {
            // Call backend to acknowledge incident
            await AcknowledgeIncident(incident.incident_id);
            
            // Mark as acknowledged locally for instant UI feedback
            markIncidentAcknowledged(incident.incident_id, incident.updated_at);
            
            console.log(`Incident ${incident.incident_id} acknowledged successfully`);
            
        } catch (err) {
            console.error('Failed to acknowledge incident:', err);
            alert(`Failed to acknowledge incident: ${err}`);
        } finally {
            acknowledging = false;
        }
    }
    
    function handleCardClick() {
        selectedIncident.set(incident);
    }
</script>

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
<div class="incident-card" class:selected={isSelected} on:click={handleCardClick}>
    <div class="action-buttons">
        <button class="action-button" on:click={openIncident} title="Open incident">
            {#if openFeedback}
                <span class="feedback-text">{openFeedback}</span>
            {:else}
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path>
                    <polyline points="15 3 21 3 21 9"></polyline>
                    <line x1="10" y1="14" x2="21" y2="3"></line>
                </svg>
            {/if}
        </button>
        <button class="action-button" on:click={copyIncidentLink} title="Copy link">
            {#if copyFeedback}
                <span class="feedback-text">{copyFeedback}</span>
            {:else}
                <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <path d="M10 13a5 5 0 0 0 7.54.54l3-3a5 5 0 0 0-7.07-7.07l-1.72 1.71"></path>
                    <path d="M14 11a5 5 0 0 0-7.54-.54l-3 3a5 5 0 0 0 7.07 7.07l1.71-1.71"></path>
                </svg>
            {/if}
        </button>
    </div>
    
    <div class="incident-header">
        <h3 class="incident-title" title={incident.title}>{incident.title}</h3>
    </div>
    
    <div class="incident-details">
        <div class="service-row">
            <span class="service-name" style="color: {serviceColor}">{incident.service_summary || 'Unknown Service'}</span>
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
        <div class="meta-row">
            <span class="incident-time">{formatTime(incident.created_at)}</span>
            {#if incident.alert_count > 0}
                <span class="separator">•</span>
                <span class="alert-count">{incident.alert_count} alert{incident.alert_count !== 1 ? 's' : ''}</span>
            {/if}
            {#if incident.incident_number}
                <span class="separator">•</span>
                <span class="incident-number">#{incident.incident_number}</span>
            {/if}
            
            {#if showAckButton}
                <span class="acknowledge-spacer"></span>
                <button 
                    class="acknowledge-button" 
                    class:loading={acknowledging}
                    on:click={handleAcknowledge}
                    disabled={acknowledging}
                    title="Acknowledge incident"
                >
                    {#if acknowledging}
                        <span class="spinner"></span>
                        Acknowledging...
                    {:else}
                        <span class="ack-icon">!</span>
                        Acknowledge
                    {/if}
                </button>
            {/if}
        </div>
    </div>
</div>

<style>
    .incident-card {
        background: white;
        border: 1px solid #e5e7eb;
        border-left: 3px solid transparent;
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
    
    .incident-card.selected {
        border-left-color: var(--service-color, #3b82f6);
        background: #f9fafb;
        box-shadow: 0 2px 4px -1px rgb(0 0 0 / 0.08);
    }
    
    .action-buttons {
        position: absolute;
        top: 12px;
        right: 12px;
        display: flex;
        gap: 4px;
        z-index: 1;
    }
    
    .action-button {
        padding: 4px;
        background: transparent;
        border: 1px solid transparent;
        border-radius: 4px;
        cursor: pointer;
        color: #9ca3af;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.15s ease;
        min-width: 24px;
        min-height: 24px;
    }
    
    .action-button:hover {
        background: #f3f4f6;
        border-color: #e5e7eb;
        color: #6b7280;
        transform: translateY(-1px);
    }
    
    .action-button:active {
        transform: translateY(0);
        background: #e5e7eb;
    }
    
    .feedback-text {
        font-size: 10px;
        font-weight: 600;
        color: #059669;
        white-space: nowrap;
        animation: fadeIn 0.2s ease;
    }
    
    @keyframes fadeIn {
        from {
            opacity: 0;
            transform: scale(0.9);
        }
        to {
            opacity: 1;
            transform: scale(1);
        }
    }
    
    .incident-header {
        margin-bottom: 10px;
        padding-right: 60px;
    }
    
    .incident-title {
        font-size: 15px;
        font-weight: 600;
        color: #111827;
        margin: 0;
        line-height: 1.4;
        overflow: hidden;
        text-overflow: ellipsis;
        display: -webkit-box;
        -webkit-line-clamp: 2;
        -webkit-box-orient: vertical;
    }
    
    .incident-details {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    
    .service-row {
        display: flex;
        align-items: center;
        gap: 10px;
    }
    
    .service-name {
        font-weight: 500;
        font-size: 13px;
    }
    
    .status-badge {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        font-size: 12px;
        font-weight: 600;
        white-space: nowrap;
        background: white;            
        border: 1px solid transparent; 
        padding: 2px 6px;
        border-radius: 4px;
    }
    
    .status-icon {
        font-size: 10px;
    }
    
    .meta-row {
        display: flex;
        align-items: center;
        color: #6b7280;
        font-size: 13px;
        flex-wrap: wrap;
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
    
    /* Acknowledge button styles - RED theme */
    .acknowledge-spacer {
        flex: 1;
        min-width: 8px;
    }
    
    .acknowledge-button {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 4px 10px;
        font-size: 12px;
        font-weight: 500;
        color: #dc2626;
        background: transparent;
        border: 1px solid #fecaca;
        border-radius: 4px;
        cursor: pointer;
        transition: all 0.2s ease;
        margin-left: auto;
    }
    
    .acknowledge-button:hover {
        background: #fef2f2;
        border-color: #dc2626;
        color: #b91c1c;
        transform: translateY(-1px);
        box-shadow: 0 2px 4px rgba(220, 38, 38, 0.1);
    }
    
    .acknowledge-button:active {
        transform: translateY(0);
        box-shadow: 0 1px 2px rgba(220, 38, 38, 0.1);
    }
    
    .ack-icon {
        font-size: 13px;
        font-weight: 700;
        line-height: 1;
    }
</style>