<script lang="ts">
    import type { database } from '../../wailsjs/go/models';
    
    type IncidentData = database.IncidentData;
    
    export let incident: IncidentData;
    
    let noteText = '';
    
    function addNote() {
        if (noteText.trim()) {
            // API call will be implemented separately
            console.log('Add note:', noteText);
            noteText = '';
        }
    }
</script>

<div class="notes-container">
    <div class="add-note-section">
        <h6>Add Notes</h6>
        <textarea 
            bind:value={noteText}
            placeholder="Enter your note..."
            rows="4"
        ></textarea>
        <button 
            class="add-note-button"
            on:click={addNote}
            disabled={!noteText.trim()}
        >
            Add Note
        </button>
    </div>
    
    <div class="notes-list">
        <div class="notes-empty">
            <svg width="48" height="48" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"></path>
                <polyline points="14 2 14 8 20 8"></polyline>
                <line x1="12" y1="18" x2="12" y2="12"></line>
                <line x1="9" y1="15" x2="15" y2="15"></line>
            </svg>
            <p>No notes added yet</p>
            <span class="notes-hint">Note data will be populated from API calls (to be implemented separately)</span>
        </div>
    </div>
</div>

<style>
    .notes-container {
        padding: 16px;
    }
    
    .add-note-section {
        margin-bottom: 20px;
    }
    
    .add-note-section h6 {
        margin: 0 0 10px 0;
        font-size: 14px;
        font-weight: 600;
        color: #111827;
    }
    
    textarea {
        width: 100%;
        padding: 10px 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        font-family: inherit;
        resize: vertical;
        margin-bottom: 10px;
    }
    
    textarea:focus {
        outline: none;
        border-color: #3b82f6;
        box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
    }
    
    .add-note-button {
        width: 100%;
        padding: 10px 16px;
        background: #6b7280;
        color: white;
        border: none;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: background 0.2s;
    }
    
    .add-note-button:hover:not(:disabled) {
        background: #4b5563;
    }
    
    .add-note-button:disabled {
        opacity: 0.5;
        cursor: not-allowed;
    }
    
    .notes-list {
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid #e5e7eb;
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
        margin: 0 0 8px 0;
        font-size: 14px;
        color: #6b7280;
        font-weight: 500;
    }
    
    .notes-hint {
        font-size: 12px;
        color: #9ca3af;
        text-align: center;
        font-style: italic;
        max-width: 250px;
    }
</style>