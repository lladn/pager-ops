<script lang="ts">
    import type { database } from '../../wailsjs/go/models';
    
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
</script>

<div class="alerts-container">
    <div class="alert-item">
        <div class="alert-header">
            <span class="alert-icon">⚠</span>
            <div class="alert-meta">
                <span class="alert-source">Connection timeout exceeded</span>
                <span class="alert-timestamp">Database Monitor • {formatDate(incident.created_at)}</span>
            </div>
        </div>
        <div class="alert-links">
            <span class="links-label">Links:</span>
            <a href={incident.html_url} target="_blank" rel="noopener noreferrer">dashboard</a>
        </div>
    </div>
    
    {#if incident.alert_count > 1}
        <div class="alert-item">
            <div class="alert-header">
                <span class="alert-icon">⚠</span>
                <div class="alert-meta">
                    <span class="alert-source">Query response time &gt; 5s</span>
                    <span class="alert-timestamp">Performance Monitor • {formatDate(incident.updated_at)}</span>
                </div>
            </div>
            <div class="alert-links">
                <span class="links-label">Links:</span>
                <a href={incident.html_url} target="_blank" rel="noopener noreferrer">dashboard</a>
            </div>
        </div>
    {/if}
    
    <div class="alerts-footer">
        <p class="alerts-note">
            Alert data will be populated from API calls (to be implemented separately)
        </p>
    </div>
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
        font-size: 16px;
        line-height: 1.4;
    }
    
    .alert-meta {
        flex: 1;
        display: flex;
        flex-direction: column;
        gap: 2px;
    }
    
    .alert-source {
        font-size: 14px;
        font-weight: 500;
        color: #111827;
        line-height: 1.4;
    }
    
    .alert-timestamp {
        font-size: 12px;
        color: #6b7280;
    }
    
    .alert-links {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        margin-left: 26px;
    }
    
    .links-label {
        color: #6b7280;
        font-weight: 500;
    }
    
    .alert-links a {
        color: #3b82f6;
        text-decoration: none;
    }
    
    .alert-links a:hover {
        text-decoration: underline;
    }
    
    .alerts-footer {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid #e5e7eb;
    }
    
    .alerts-note {
        font-size: 12px;
        color: #9ca3af;
        text-align: center;
        margin: 0;
        font-style: italic;
    }
</style>