<script lang="ts">
    import { sidebarData, sidebarLoading, sidebarError, loadIncidentSidebarData } from '../stores/incidents';
    import { GetServiceConfigByServiceID } from '../../wailsjs/go/main/App';
    import type { database, store } from '../../wailsjs/go/models';
    import { onMount, onDestroy } from 'svelte';
    
    type IncidentData = database.IncidentData;
    type ServiceConfig = store.ServiceConfig;
    type ServiceTypes = store.ServiceTypes;
    
    export let incident: IncidentData;
    
    // Service-specific notekit configuration
    let serviceConfig: ServiceConfig | null = null;
    let serviceTypes: ServiceTypes | null = null;
    let loadingConfig = false;
    
    // Form state for notekit
    let questionResponses: Record<string, string> = {};
    let tagSelections: Record<string, string[]> = {};
    let freeformNote = '';
    
    // Draft persistence
    let lastIncidentId = '';
    let saveTimeout: number | null = null;
    let draftSaveStatus = ''; // 'saving', 'saved', or ''
    let draftTimestamp: number | null = null;
    
    // Dropdown state for custom dropdowns
    let openDropdown: string | null = null;
    
    // LocalStorage key prefix
    const DRAFT_PREFIX = 'pagerduty_draft_';
    
    // Load service configuration when incident changes
    $: if (incident?.incident_id && incident.incident_id !== lastIncidentId) {
        handleIncidentChange();
        lastIncidentId = incident.incident_id;
    }
    
    onDestroy(() => {
        if (saveTimeout) {
            clearTimeout(saveTimeout);
        }
    });
    
    async function handleIncidentChange() {
        // Save current draft before switching
        if (lastIncidentId) {
            saveDraftToLocalStorage(lastIncidentId);
        }
        
        // Load new incident's configuration
        await loadServiceConfig();
        
        // Restore draft for new incident
        if (incident?.incident_id) {
            restoreDraftFromLocalStorage(incident.incident_id);
        }
    }
    
    async function loadServiceConfig() {
        if (!incident?.service_id) return;
        
        loadingConfig = true;
        try {
            serviceConfig = await GetServiceConfigByServiceID(incident.service_id);
            serviceTypes = serviceConfig?.types || null;
            
            // Initialize tag selections with empty arrays
            if (serviceTypes?.tags) {
                serviceTypes.tags.forEach(tag => {
                    if (!tagSelections[tag.name]) {
                        tagSelections[tag.name] = [];
                    }
                });
            }
        } catch (err) {
            console.error('Failed to load service config:', err);
            serviceTypes = null;
        } finally {
            loadingConfig = false;
        }
    }
    
    function getDraftKey(incidentId: string): string {
        return `${DRAFT_PREFIX}${incidentId}`;
    }
    
    function saveDraftToLocalStorage(incidentId: string) {
        if (!incidentId) return;
        
        const draftKey = getDraftKey(incidentId);
        const draftData = {
            questionResponses: { ...questionResponses },
            tagSelections: { ...tagSelections },
            freeformNote,
            timestamp: Date.now(),
            serviceId: incident?.service_id
        };
        
        // Only save if there's actual content
        if (hasContent()) {
            try {
                localStorage.setItem(draftKey, JSON.stringify(draftData));
                draftTimestamp = Date.now();
                draftSaveStatus = 'saved';
                
                // Clear status after 2 seconds
                setTimeout(() => {
                    draftSaveStatus = '';
                }, 2000);
            } catch (error) {
                console.error('Failed to save draft to localStorage:', error);
                draftSaveStatus = '';
            }
        } else {
            // Clear draft if no content
            clearDraft(incidentId);
        }
    }
    
    function restoreDraftFromLocalStorage(incidentId: string) {
        if (!incidentId) return;
        
        const draftKey = getDraftKey(incidentId);
        
        try {
            const draftJson = localStorage.getItem(draftKey);
            
            if (draftJson) {
                const draft = JSON.parse(draftJson);
                
                // Restore form state
                questionResponses = draft.questionResponses || {};
                tagSelections = draft.tagSelections || {};
                freeformNote = draft.freeformNote || '';
                draftTimestamp = draft.timestamp || null;
                
                // Show indicator that draft was restored
                if (hasContent()) {
                    draftSaveStatus = 'saved';
                    setTimeout(() => {
                        draftSaveStatus = '';
                    }, 2000);
                }
            } else {
                // No draft found, initialize empty
                questionResponses = {};
                tagSelections = {};
                freeformNote = '';
                draftTimestamp = null;
                
                // Initialize tag selections
                if (serviceTypes?.tags) {
                    serviceTypes.tags.forEach(tag => {
                        tagSelections[tag.name] = [];
                    });
                }
            }
        } catch (error) {
            console.error('Failed to restore draft from localStorage:', error);
            // Initialize empty on error
            questionResponses = {};
            tagSelections = {};
            freeformNote = '';
            draftTimestamp = null;
        }
    }
    
    function saveDraft() {
        if (!incident?.incident_id) return;
        
        // Show saving status
        draftSaveStatus = 'saving';
        
        // Debounce save to avoid hammering localStorage
        if (saveTimeout) {
            clearTimeout(saveTimeout);
        }
        
        saveTimeout = window.setTimeout(() => {
            saveDraftToLocalStorage(incident.incident_id);
        }, 500);
    }
    
    function clearDraft(incidentId: string) {
        if (!incidentId) return;
        
        const draftKey = getDraftKey(incidentId);
        try {
            localStorage.removeItem(draftKey);
            draftTimestamp = null;
            draftSaveStatus = '';
        } catch (error) {
            console.error('Failed to clear draft from localStorage:', error);
        }
    }
    
    function toggleDropdown(tagName: string) {
        openDropdown = openDropdown === tagName ? null : tagName;
    }
    
    function closeDropdown() {
        openDropdown = null;
    }
    
    function handleTagSelection(tagName: string, value: string, isSingle: boolean) {
        if (isSingle) {
            // Single selection - replace with new value
            tagSelections[tagName] = [value];
        } else {
            // Multiple selection - toggle value
            if (!tagSelections[tagName]) {
                tagSelections[tagName] = [];
            }
            
            const index = tagSelections[tagName].indexOf(value);
            if (index > -1) {
                tagSelections[tagName] = tagSelections[tagName].filter(v => v !== value);
            } else {
                tagSelections[tagName] = [...tagSelections[tagName], value];
            }
        }
        
        // Close dropdown for single selection
        if (isSingle) {
            closeDropdown();
        }
        
        saveDraft();
    }
    
    function removeTag(tagName: string, value: string) {
        if (tagSelections[tagName]) {
            tagSelections[tagName] = tagSelections[tagName].filter(v => v !== value);
            saveDraft();
        }
    }
    
    function getDropdownText(tagName: string, isSingle: boolean): string {
        const selected = tagSelections[tagName] || [];
        
        if (selected.length === 0) {
            return 'Select';
        }
        
        if (isSingle) {
            return selected[0];
        }
        
        return `${selected.length} selected`;
    }
    
    function handleAddNote() {
        // TODO: When backend API is ready, send structured note data
        // For now, just show alert with placeholder
        alert('Note functionality will be connected to backend in future update.\n\nData to be sent:\n' + 
              JSON.stringify({
                  serviceId: incident?.service_id,
                  responses: questionResponses,
                  tags: tagSelections,
                  freeformContent: freeformNote
              }, null, 2));
        
        // Clear form
        questionResponses = {};
        tagSelections = {};
        freeformNote = '';
        
        // Clear draft from localStorage
        if (incident?.incident_id) {
            clearDraft(incident.incident_id);
        }
        
        // Re-initialize tag selections
        if (serviceTypes?.tags) {
            serviceTypes.tags.forEach(tag => {
                tagSelections[tag.name] = [];
            });
        }
    }
    
    function hasContent(): boolean {
        const hasQuestions = Object.values(questionResponses).some(v => v.trim());
        const hasTags = Object.values(tagSelections).some(v => v.length > 0);
        const hasFreeform = freeformNote.trim().length > 0;
        return hasQuestions || hasTags || hasFreeform;
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
    
    function getTimeSince(timestamp: number): string {
        const seconds = Math.floor((Date.now() - timestamp) / 1000);
        
        if (seconds < 60) return 'just now';
        const minutes = Math.floor(seconds / 60);
        if (minutes < 60) return `${minutes}m ago`;
        const hours = Math.floor(minutes / 60);
        if (hours < 24) return `${hours}h ago`;
        const days = Math.floor(hours / 24);
        return `${days}d ago`;
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
        <div class="section-header">
            <p class="section-title">Add Notes</p>
            {#if draftSaveStatus === 'saving'}
                <span class="draft-status saving">Saving...</span>
            {:else if draftSaveStatus === 'saved' && draftTimestamp}
                <span class="draft-status saved">Draft saved {getTimeSince(draftTimestamp)}</span>
            {/if}
        </div>
        
        {#if loadingConfig}
            <div class="config-loading">Loading service configuration...</div>
        {:else if serviceTypes}
            <!-- Dynamic Questions -->
            {#if serviceTypes.questions && serviceTypes.questions.length > 0}
                <div class="questions-section">
                    <p class="subsection-title">Questions</p>
                    {#each serviceTypes.questions as question, index}
                        <div class="question-group">
                            <!-- svelte-ignore a11y-label-has-associated-control -->
                            <label class="question-label">{question}</label>
                            <textarea 
                                class="question-textarea" 
                                placeholder="Enter your response..."
                                bind:value={questionResponses[question]}
                                on:input={saveDraft}
                                rows="3"
                            />
                        </div>
                    {/each}
                </div>
            {/if}
            
            <!-- Dynamic Tags -->
            {#if serviceTypes.tags && serviceTypes.tags.length > 0}
                <div class="tags-section">
                    <p class="subsection-title">Tags</p>
                    {#each serviceTypes.tags as tagConfig}
                        <div class="tag-group">
                            <!-- svelte-ignore a11y-label-has-associated-control -->
                            <label class="tag-label">{tagConfig.name}</label>
                            
                            <!-- Custom Dropdown -->
                            <div class="tag-dropdown">
                                <button 
                                    class="dropdown-button"
                                    on:click={() => toggleDropdown(tagConfig.name)}
                                >
                                    <span class="dropdown-text">{getDropdownText(tagConfig.name, !!tagConfig.single)}</span>
                                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor" class="dropdown-icon">
                                        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                                    </svg>
                                </button>
                                
                                {#if openDropdown === tagConfig.name}
                                    <!-- svelte-ignore a11y-no-static-element-interactions -->
                                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                                    <div class="dropdown-overlay" on:click={closeDropdown}></div>
                                    <div class="dropdown-menu">
                                        {#if tagConfig.multiple && tagConfig.multiple.length > 0}
                                            {#each tagConfig.multiple as option}
                                                <button 
                                                    class="dropdown-option"
                                                    class:selected={tagSelections[tagConfig.name]?.includes(option)}
                                                    on:click={() => handleTagSelection(tagConfig.name, option, false)}
                                                >
                                                    <span class="checkbox">
                                                        {#if tagSelections[tagConfig.name]?.includes(option)}
                                                            <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                                                                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                                                            </svg>
                                                        {/if}
                                                    </span>
                                                    <span>{option}</span>
                                                </button>
                                            {/each}
                                        {:else if tagConfig.single && tagConfig.single.length > 0}
                                            {#each tagConfig.single as option}
                                                <button 
                                                    class="dropdown-option"
                                                    class:selected={tagSelections[tagConfig.name]?.includes(option)}
                                                    on:click={() => handleTagSelection(tagConfig.name, option, true)}
                                                >
                                                    <span class="checkbox">
                                                        {#if tagSelections[tagConfig.name]?.includes(option)}
                                                            <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                                                                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                                                            </svg>
                                                        {/if}
                                                    </span>
                                                    <span>{option}</span>
                                                </button>
                                            {/each}
                                        {/if}
                                    </div>
                                {/if}
                            </div>
                            
                            <!-- Selected tags as chips (for both single and multiple) -->
                            {#if tagSelections[tagConfig.name] && tagSelections[tagConfig.name].length > 0}
                                <div class="tag-chips">
                                    {#each tagSelections[tagConfig.name] as selectedValue}
                                        <div class="tag-chip">
                                            <span>{selectedValue}</span>
                                            <button 
                                                class="chip-remove"
                                                on:click={() => removeTag(tagConfig.name, selectedValue)}
                                                title="Remove"
                                            >×</button>
                                        </div>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                    {/each}
                </div>
            {/if}
            
            <!-- Freeform Note -->
            <div class="freeform-section">
                <!-- svelte-ignore a11y-label-has-associated-control -->
                <label class="question-label">Add Notes</label>
                <textarea 
                    class="note-textarea" 
                    placeholder="Enter your note..."
                    bind:value={freeformNote}
                    on:input={saveDraft}
                    rows="4"
                />
            </div>
        {:else}
            <!-- Fallback: no service config, show simple textarea -->
            <textarea 
                class="note-textarea" 
                placeholder="Enter your note..."
                bind:value={freeformNote}
                on:input={saveDraft}
                rows="6"
            />
        {/if}
        
        <button 
            class="add-note-button" 
            class:active={hasContent()}
            on:click={handleAddNote}
            disabled={!hasContent()}
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
                    
                    <!-- Display structured responses if available -->
                    {#if note.responses && note.responses.length > 0}
                        <div class="note-responses">
                            {#each note.responses as response}
                                <div class="response-item">
                                    <div class="response-question">{response.question}</div>
                                    <div class="response-answer">{response.answer}</div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                    
                    <!-- Display tags if available -->
                    {#if note.tags && note.tags.length > 0}
                        <div class="note-tags-display">
                            {#each note.tags as tag}
                                <div class="tag-display-group">
                                    <span class="tag-display-name">{tag.tag_name}:</span>
                                    <div class="tag-display-values">
                                        {#each tag.selected_values as value}
                                            <span class="tag-display-chip">{value}</span>
                                        {/each}
                                    </div>
                                </div>
                            {/each}
                        </div>
                    {/if}
                    
                    <!-- Display freeform content -->
                    {#if note.freeform_content || note.content}
                        <div class="note-content">
                            {note.freeform_content || note.content}
                        </div>
                    {/if}
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
    
    .section-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 16px;
    }
    
    .section-title {
        margin: 0;
        font-size: 14px;
        font-weight: 600;
        color: #374151;
    }
    
    .draft-status {
        font-size: 11px;
        font-weight: 500;
        padding: 2px 8px;
        border-radius: 4px;
        transition: all 0.2s ease;
    }
    
    .draft-status.saving {
        color: #3b82f6;
        background: #eff6ff;
    }
    
    .draft-status.saved {
        color: #059669;
        background: #d1fae5;
    }
    
    .subsection-title {
        margin: 16px 0 8px 0;
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
        text-transform: uppercase;
        letter-spacing: 0.5px;
    }
    
    .config-loading {
        padding: 12px;
        text-align: center;
        color: #6b7280;
        font-size: 13px;
    }
    
    /* Questions Section */
    .questions-section {
        margin-bottom: 16px;
    }
    
    .question-group {
        margin-bottom: 12px;
    }
    
    .question-label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
        margin-bottom: 6px;
    }
    
    .question-textarea {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
        color: #1f2937;
        background: #f9fafb;
        resize: vertical;
        font-family: inherit;
        transition: all 0.2s ease;
    }
    
    .question-textarea::placeholder {
        color: #9ca3af;
    }
    
    .question-textarea:focus {
        outline: none;
        background: #ffffff;
        border-color: #d1d5db;
    }
    
    /* Tags Section */
    .tags-section {
        margin-bottom: 16px;
    }
    
    .tag-group {
        margin-bottom: 12px;
    }
    
    .tag-label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
        margin-bottom: 6px;
    }
    
    .tag-dropdown {
        position: relative;
    }
    
    .dropdown-button {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        background: #f9fafb;
        display: flex;
        justify-content: space-between;
        align-items: center;
        cursor: pointer;
        transition: all 0.2s ease;
        font-size: 14px;
        color: #374151;
    }
    
    .dropdown-button:hover {
        background: #ffffff;
        border-color: #d1d5db;
    }
    
    .dropdown-text {
        flex: 1;
        text-align: left;
        color: #6b7280;
    }
    
    .dropdown-icon {
        flex-shrink: 0;
        color: #9ca3af;
        transition: transform 0.2s ease;
    }
    
    .dropdown-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 10;
    }
    
    .dropdown-menu {
        position: absolute;
        top: calc(100% + 4px);
        left: 0;
        right: 0;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
        z-index: 20;
        max-height: 200px;
        overflow-y: auto;
    }
    
    .dropdown-option {
        width: 100%;
        padding: 8px 12px;
        border: none;
        background: white;
        display: flex;
        align-items: center;
        gap: 8px;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        transition: background 0.15s ease;
    }
    
    .dropdown-option:hover {
        background: #f3f4f6;
    }
    
    .dropdown-option.selected {
        background: #eff6ff;
        color: #3b82f6;
    }
    
    .checkbox {
        width: 16px;
        height: 16px;
        border: 1px solid #d1d5db;
        border-radius: 3px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        background: white;
    }
    
    .dropdown-option.selected .checkbox {
        background: #3b82f6;
        border-color: #3b82f6;
        color: white;
    }
    
    .tag-chips {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin-top: 8px;
    }
    
    .tag-chip {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 4px 8px;
        background: #eff6ff;
        color: #3b82f6;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 500;
    }
    
    .chip-remove {
        padding: 0;
        width: 16px;
        height: 16px;
        border: none;
        background: transparent;
        color: #3b82f6;
        cursor: pointer;
        font-size: 18px;
        line-height: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 2px;
        transition: background 0.15s ease;
    }
    
    .chip-remove:hover {
        background: rgba(59, 130, 246, 0.1);
    }
    
    /* Freeform Section */
    .freeform-section {
        margin-bottom: 16px;
    }
    
    .note-textarea {
        width: 100%;
        padding: 10px 12px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
        color: #1f2937;
        background: #f9fafb;
        resize: vertical;
        font-family: inherit;
        min-height: 80px;
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
        padding: 8px 16px;
        background: transparent;
        color: #d1d5db;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: not-allowed;
        transition: all 0.2s ease;
    }
    
    .add-note-button.active {
        color: #ffffff;
        background: #3b82f6;
        border-color: #3b82f6;
        cursor: pointer;
    }
    
    .add-note-button.active:hover {
        background: #2563eb;
    }
    
    /* Notes Display */
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
        margin-bottom: 12px;
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
    
    .note-responses {
        margin-bottom: 12px;
    }
    
    .response-item {
        margin-bottom: 12px;
        padding-bottom: 12px;
        border-bottom: 1px solid #f3f4f6;
    }
    
    .response-item:last-child {
        margin-bottom: 0;
        padding-bottom: 0;
        border-bottom: none;
    }
    
    .response-question {
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
        margin-bottom: 4px;
    }
    
    .response-answer {
        font-size: 14px;
        color: #374151;
        line-height: 1.5;
        white-space: pre-wrap;
    }
    
    .note-tags-display {
        margin-bottom: 12px;
        padding: 10px;
        background: #f9fafb;
        border-radius: 6px;
    }
    
    .tag-display-group {
        margin-bottom: 8px;
    }
    
    .tag-display-group:last-child {
        margin-bottom: 0;
    }
    
    .tag-display-name {
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
        display: block;
        margin-bottom: 4px;
    }
    
    .tag-display-values {
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
    }
    
    .tag-display-chip {
        display: inline-block;
        padding: 2px 8px;
        background: #e5e7eb;
        color: #374151;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 500;
    }
    
    .note-content {
        font-size: 14px;
        color: #1f2937;
        line-height: 1.6;
        white-space: pre-wrap;
    }
    
    .notes-empty {
        padding: 32px 16px;
        text-align: center;
        color: #9ca3af;
        font-size: 14px;
    }
    
    /* Loading Skeletons */
    .skeleton-container {
        padding: 14px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        margin-bottom: 12px;
    }
    
    .skeleton-line {
        height: 12px;
        background: linear-gradient(90deg, #f3f4f6 25%, #e5e7eb 50%, #f3f4f6 75%);
        background-size: 200% 100%;
        animation: shimmer 1.5s infinite;
        border-radius: 4px;
        margin-bottom: 8px;
    }
    
    .skeleton-line:last-child {
        margin-bottom: 0;
    }
    
    .skeleton-line.short {
        width: 60%;
    }
    
    @keyframes shimmer {
        0% {
            background-position: 200% 0;
        }
        100% {
            background-position: -200% 0;
        }
    }
    
    /* Error Banner */
    .error-banner {
        padding: 16px;
        background: #fef2f2;
        border: 1px solid #fecaca;
        border-radius: 8px;
        display: flex;
        align-items: center;
        gap: 12px;
        margin-bottom: 16px;
    }
    
    .error-icon {
        font-size: 20px;
        flex-shrink: 0;
    }
    
    .error-banner p {
        flex: 1;
        margin: 0;
        color: #991b1b;
        font-size: 13px;
    }
    
    .retry-button {
        padding: 6px 12px;
        background: #dc2626;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        flex-shrink: 0;
        transition: background 0.2s ease;
    }
    
    .retry-button:hover {
        background: #b91c1c;
    }
</style>