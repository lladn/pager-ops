<script lang="ts">
    import { sidebarData, sidebarLoading, sidebarError, loadIncidentSidebarData } from '../stores/incidents';
    import type { database } from '../../wailsjs/go/models';
    import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
    
    type IncidentData = database.IncidentData;
    
    export let incident: IncidentData;
    
    function formatDate(date: Date | string): string {
        const d = typeof date === 'string' ? new Date(date) : date;
        return d.toLocaleString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: '2-digit',
            minute: '2-digit'
        });
    }
    
    async function retry() {
        if (incident?.incident_id) {
            await loadIncidentSidebarData(incident.incident_id);
        }
    }
    
    function openLink(url: string) {
        if (url) {
            BrowserOpenURL(url);
        }
    }
</script>

<div class="alerts-container">
    {#if $sidebarLoading}
        <!-- Loading skeletons -->
        <div class="skeleton-container">
            <div class="skeleton-box"></div>
            <div class="skeleton-line"></div>
            <div class="skeleton-line short"></div>
        </div>
        <div class="skeleton-container">
            <div class="skeleton-box"></div>
            <div class="skeleton-line"></div>
            <div class="skeleton-line short"></div>
        </div>
    {:else if $sidebarError}
        <!-- Error state -->
        <div class="error-banner">
            <span class="error-icon">⚠️</span>
            <p>{$sidebarError}</p>
            <button class="retry-button" on:click={retry}>
                Retry
            </button>
        </div>
    {:else if $sidebarData?.alerts && $sidebarData.alerts.length > 0}
        <!-- Display real alerts -->
        {#each $sidebarData.alerts as alert}
            <div class="alert-item">
                <div class="alert-header">
                    <span class="alert-icon">⚠</span>
                    <div class="alert-meta">
                        <span class="alert-source">{alert.summary}</span>
                        <span class="alert-timestamp">
                            {alert.service_name || 'Unknown Service'} • {formatDate(alert.created_at)}
                        </span>
                    </div>
                </div>
                {#if alert.status}
                    <div class="alert-status">
                        Status: <span class="status-{alert.status.toLowerCase()}">{alert.status}</span>
                    </div>
                {/if}
                {#if alert.links && alert.links.length > 0}
                    <div class="alert-links">
                        <span class="links-label">Links:</span>
                        {#each alert.links as link}
                            <button 
                                class="link-button" 
                                on:click={() => openLink(link.href)}
                                title={link.href}
                            >
                                {link.text || 'View'}
                            </button>
                        {/each}
                    </div>
                {/if}
            </div>
        {/each}
    {:else}
        <!-- Empty state -->
        <div class="alerts-empty">
            <p>No alerts found for this incident</p>
        </div>
    {/if}
</div>

<style>
    .alerts-container {
        padding: 16px;
    }
    
    .alert-item {
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        padding: 12px;
        margin-bottom: 12px;
    }
    
    .alert-item:last-child {
        margin-bottom: 0;
    }
    
    .alert-header {
        display: flex;
        gap: 10px;
        margin-bottom: 8px;
    }
    
    .alert-icon {
        font-size: 18px;
        color: #f59e0b;
        flex-shrink: 0;
    }
    
    .alert-meta {
        flex: 1;
        min-width: 0;
    }
    
    .alert-source {
        display: block;
        font-weight: 500;
        color: #111827;
        font-size: 14px;
        margin-bottom: 2px;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .alert-timestamp {
        display: block;
        color: #6b7280;
        font-size: 12px;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .alert-status {
        font-size: 13px;
        color: #374151;
        margin-bottom: 8px;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .status-triggered {
        color: #dc2626;
        font-weight: 500;
    }
    
    .status-acknowledged {
        color: #f59e0b;
        font-weight: 500;
    }
    
    .status-resolved {
        color: #10b981;
        font-weight: 500;
    }
    
    .alert-links {
        margin-top: 8px;
        font-size: 13px;
        word-wrap: break-word;
        overflow-wrap: break-word;
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 8px;
    }
    
    .links-label {
        color: #6b7280;
    }
    
    .link-button {
        background: none;
        border: none;
        padding: 0;
        color: #3b82f6;
        text-decoration: none;
        cursor: pointer;
        font-size: 13px;
        word-break: break-all;
        transition: color 0.2s;
    }
    
    .link-button:hover {
        text-decoration: underline;
        color: #2563eb;
    }
    
    .link-button:active {
        color: #1d4ed8;
    }
    
    .skeleton-container {
        margin-bottom: 12px;
        padding: 12px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
    }
    
    .skeleton-box {
        width: 100px;
        height: 20px;
        background: linear-gradient(90deg, #f3f4f6 25%, #e5e7eb 50%, #f3f4f6 75%);
        background-size: 200% 100%;
        animation: loading 1.5s infinite;
        border-radius: 4px;
        margin-bottom: 8px;
    }
    
    .skeleton-line {
        height: 16px;
        background: linear-gradient(90deg, #f3f4f6 25%, #e5e7eb 50%, #f3f4f6 75%);
        background-size: 200% 100%;
        animation: loading 1.5s infinite;
        border-radius: 4px;
        margin-bottom: 4px;
    }
    
    .skeleton-line.short {
        width: 60%;
    }
    
    @keyframes loading {
        0% { background-position: 200% 0; }
        100% { background-position: -200% 0; }
    }
    
    .error-banner {
        background: #fef2f2;
        border: 1px solid #fecaca;
        border-radius: 8px;
        padding: 12px;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .error-icon {
        font-size: 20px;
        flex-shrink: 0;
    }
    
    .error-banner p {
        flex: 1;
        margin: 0;
        color: #991b1b;
        font-size: 14px;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .retry-button {
        padding: 4px 12px;
        background: #dc2626;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 12px;
        cursor: pointer;
        flex-shrink: 0;
    }
    
    .retry-button:hover {
        background: #b91c1c;
    }
    
    .alerts-empty {
        text-align: center;
        color: #6b7280;
        padding: 24px;
    }
</style>