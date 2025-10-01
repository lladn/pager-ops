<script lang="ts">
    import { sidebarData, sidebarLoading, sidebarError, loadIncidentSidebarData } from '../stores/incidents';
    import type { database } from '../../wailsjs/go/models';
    
    type IncidentData = database.IncidentData;
    
    export let incident: IncidentData;
    
    // Draft storage in local component state (no backend)
    let noteDraft = '';
    let draftStore: Record<string, string> = {};
    let lastIncidentId = '';
    
    // Watch for incident changes and load/save drafts
    $: {
        if (incident?.incident_id && incident.incident_id !== lastIncidentId) {
            // Save current draft before switching
            if (lastIncidentId && noteDraft.trim()) {
                draftStore[lastIncidentId] = noteDraft;
            }
            
            // Load draft for new incident
            noteDraft = draftStore[incident.incident_id] || '';
            lastIncidentId = incident.incident_id;
        }
    }
    
    // Save draft when typing
    function saveDraft() {
        if (incident?.incident_id) {
            draftStore[incident.incident_id] = noteDraft;
        }
    }
    
    // Handle add note button (placeholder - no backend)
    function handleAddNote() {
        if (!noteDraft.trim()) return;
        
        // TODO: When backend is ready, call API here
        alert('Note functionality will be connected to backend in future update.');
        
        // Clear draft after "adding"
        noteDraft = '';
        if (incident?.incident_id) {
            delete draftStore[incident.incident_id];
        }
    }
    
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
    <!-- Add Notes Section -->
    <div class="add-notes-section">
        <p class="section-title">Add Notes</p>
        <textarea 
            class="note-textarea" 
            placeholder="Enter your note..."
            bind:value={noteDraft}
            on:input={saveDraft}
        />
        <button 
            class="add-note-button" 
            class:active={noteDraft.trim()}
            on:click={handleAddNote}
            disabled={!noteDraft.trim()}
        >
            Add Note
        </button>
    </div>

    <!-- Existing Notes Display -->
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
            <p>No notes added yet</p>
        </div>
    {/if}
</div>

<style>
    .notes-container {
        padding: 16px;
    }
    
    .add-notes-section {
        margin-bottom: 24px;
    }
    
    .section-title {
        margin: 0 0 12px 0;
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
    }
    
    .note-textarea {
        width: 100%;
        min-height: 100px;
        padding: 14px;
        background: #f3f4f6;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        font-size: 15px;
        font-family: inherit;
        color: #374151;
        resize: vertical;
        margin-bottom: 12px;
        box-sizing: border-box;
        transition: all 0.2s ease;
    }
    
    .note-textarea::placeholder {
        color: #9ca3af;
    }
    
    .note-textarea:focus {
        outline: none;
        background: #ffffff;
        border-color: #d1d5db;
    }
    
    .add-note-button {
    padding: 4px 9px; 
    background: transparent;
    color: #d1d5db;  
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: color 0.2s ease;  
    }
    
    .add-note-button:disabled {
        cursor: not-allowed;
    }
    
    .add-note-button.active {
    color: #374151;  
    cursor: pointer;
    }
    
    .add-note-button.active:hover {
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
        padding: 14px;
    }
    
    .note-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
        gap: 12px;
    }
    
    .note-author {
        font-weight: 500;
        color: #111827;
        font-size: 14px;
    }
    
    .note-time {
        color: #9ca3af;
        font-size: 12px;
        flex-shrink: 0;
    }
    
    .note-content {
        color: #4b5563;
        font-size: 14px;
        line-height: 1.6;
        white-space: pre-wrap;
        word-wrap: break-word;
    }
    
    .skeleton-container {
        background: #f9fafb;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        padding: 14px;
        margin-bottom: 12px;
    }
    
    .skeleton-line {
        height: 14px;
        background: linear-gradient(90deg, #f3f4f6 25%, #e5e7eb 50%, #f3f4f6 75%);
        background-size: 200% 100%;
        animation: loading 1.5s infinite;
        border-radius: 4px;
        margin-bottom: 8px;
    }
    
    .skeleton-line.short {
        width: 60%;
    }
    
    .skeleton-line:last-child {
        margin-bottom: 0;
    }
    
    @keyframes loading {
        0% { background-position: 200% 0; }
        100% { background-position: -200% 0; }
    }
    
    .error-banner {
        background: #fef2f2;
        border: 1px solid #fecaca;
        border-radius: 8px;
        padding: 14px;
        display: flex;
        align-items: center;
        gap: 10px;
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
    }
    
    .retry-button {
        padding: 6px 14px;
        background: #dc2626;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        flex-shrink: 0;
        transition: background 0.2s ease;
    }
    
    .retry-button:hover {
        background: #b91c1c;
    }
    
    .notes-empty {
        border: 2px dashed #e5e7eb;
        border-radius: 8px;
        padding: 48px 20px;
        text-align: center;
    }
    
    .notes-empty p {
        margin: 0;
        font-size: 15px;
        color: #9ca3af;
        font-weight: 400;
    }
</style>