<script lang="ts">
    import { sidebarData, sidebarLoading, sidebarError, loadIncidentSidebarData } from '../stores/incidents';
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
    
    async function retry() {
        if (incident?.incident_id) {
            await loadIncidentSidebarData(incident.incident_id);
        }
    }
</script>

<div class="notes-container">
    {#if $sidebarLoading}
        <!-- Loading skeletons -->
        <div class="skeleton-container">
            <div class="skeleton-line"></div>
            <div class="skeleton-line"></div>
            <div class="skeleton-line short"></div>
        </div>
        <div class="skeleton-container">
            <div class="skeleton-line"></div>
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
    {:else if $sidebarData?.notes && $sidebarData.notes.length > 0}
        <div class="notes-header">
            <h6>Notes ({$sidebarData.notes.length})</h6>
        </div>
        <div class="notes-list">
            {#each $sidebarData.notes as note}
                <div class="note-item">
                    <div class="note-header">
                        <span class="note-author">{note.user_name || 'Unknown User'}</span>
                        <span class="note-time">{formatDate(note.created_at)}</span>
                    </div>
                    <div class="note-content">
                        {note.content}
                    </div>
                </div>
            {/each}
        </div>
    {:else}
        <!-- Empty state -->
        <div class="notes-empty">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <line x1="12" y1="18" x2="12" y2="12"></line>
                <line x1="9" y1="15" x2="15" y2="15"></line>
            </svg>
            <p>No notes found for this incident</p>
        </div>
    {/if}
</div>

<style>
    .notes-container {
        padding: 16px;
    }
    
    .notes-header {
        margin-bottom: 16px;
    }
    
    .notes-header h6 {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
        color: #111827;
    }
    
    .notes-list {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    .note-item {
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        padding: 12px;
    }
    
    .note-header {
        display: flex;
        justify-content: space-between;
        margin-bottom: 8px;
        gap: 8px;
        flex-wrap: wrap;
    }
    
    .note-author {
        font-weight: 500;
        color: #111827;
        font-size: 13px;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .note-time {
        color: #6b7280;
        font-size: 12px;
        flex-shrink: 0;
    }
    
    .note-content {
        color: #374151;
        font-size: 14px;
        line-height: 1.5;
        white-space: pre-wrap;
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
    
    .skeleton-container {
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        padding: 12px;
        margin-bottom: 12px;
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
    
    .notes-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        padding: 32px 16px;
        color: #9ca3af;
    }
    
    .notes-empty svg {
        margin-bottom: 12px;
        opacity: 0.5;
    }
    
    .notes-empty p {
        margin: 0;
        font-size: 14px;
        color: #6b7280;
        font-weight: 500;
    }
</style>