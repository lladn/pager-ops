<script lang="ts">
    import { sidebarData, sidebarLoading, sidebarError, loadIncidentSidebarData } from '../stores/incidents';
    import { GetServiceConfigByServiceID } from '../../wailsjs/go/main/App';
    import type { database, store } from '../../wailsjs/go/models';
    import { onMount } from 'svelte';
    
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
    
    // Draft storage
    let draftStore: Record<string, any> = {};
    let lastIncidentId = '';
    
    // Dropdown state for custom dropdowns
    let openDropdown: string | null = null;
    
    // Load service configuration when incident changes
    $: if (incident?.service_id && incident.service_id !== lastIncidentId) {
        loadServiceConfig();
        lastIncidentId = incident.service_id;
    }
    
    async function loadServiceConfig() {
        if (!incident?.service_id) return;
        
        loadingConfig = true;
        try {
            serviceConfig = await GetServiceConfigByServiceID(incident.service_id);
            serviceTypes = serviceConfig?.types || null;
            
            // Load draft if exists
            const draftKey = `${incident.incident_id}_${incident.service_id}`;
            if (draftStore[draftKey]) {
                const draft = draftStore[draftKey];
                questionResponses = draft.questionResponses || {};
                tagSelections = draft.tagSelections || {};
                freeformNote = draft.freeformNote || '';
            } else {
                // Initialize empty state
                questionResponses = {};
                tagSelections = {};
                freeformNote = '';
                
                // Initialize tag selections with empty arrays
                if (serviceTypes?.tags) {
                    serviceTypes.tags.forEach(tag => {
                        tagSelections[tag.name] = [];
                    });
                }
            }
        } catch (err) {
            console.error('Failed to load service config:', err);
            serviceTypes = null;
        } finally {
            loadingConfig = false;
        }
    }
    
    function saveDraft() {
        if (!incident?.incident_id || !incident?.service_id) return;
        
        const draftKey = `${incident.incident_id}_${incident.service_id}`;
        draftStore[draftKey] = {
            questionResponses: { ...questionResponses },
            tagSelections: { ...tagSelections },
            freeformNote
        };
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
        
        if (incident?.incident_id && incident?.service_id) {
            const draftKey = `${incident.incident_id}_${incident.service_id}`;
            delete draftStore[draftKey];
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
                                    <svg 
                                        class="chevron" 
                                        class:rotated={openDropdown === tagConfig.name}
                                        width="16" 
                                        height="16" 
                                        viewBox="0 0 20 20" 
                                        fill="currentColor"
                                    >
                                        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                                    </svg>
                                </button>
                                
                                {#if openDropdown === tagConfig.name}
                                    <!-- svelte-ignore a11y-click-events-have-key-events -->
                                    <!-- svelte-ignore a11y-no-static-element-interactions -->
                                    <div class="dropdown-backdrop" on:click={closeDropdown}></div>
                                    <div class="dropdown-menu">
                                        {#if tagConfig.single}
                                            {#each tagConfig.single as option}
                                                <button 
                                                    class="dropdown-item"
                                                    class:selected={tagSelections[tagConfig.name]?.includes(option)}
                                                    on:click={() => handleTagSelection(tagConfig.name, option, true)}
                                                >
                                                    {option}
                                                </button>
                                            {/each}
                                        {:else if tagConfig.multiple}
                                            {#each tagConfig.multiple as option}
                                                <button 
                                                    class="dropdown-item"
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
    
    .section-title {
        margin: 0 0 16px 0;
        font-size: 14px;
        font-weight: 600;
        color: #374151;
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
        background: #f3f4f6;
        border-radius: 6px;
        color: #6b7280;
        text-align: center;
        font-size: 14px;
    }
    
    /* Questions Section */
    .questions-section {
        margin-bottom: 16px;
    }
    
    .question-group {
        margin-bottom: 16px;
    }
    
    .question-label {
        display: block;
        margin-bottom: 6px;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
    }
    
    .question-textarea {
        width: 100%;
        padding: 10px 12px;
        background: #f9fafb;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
        font-family: inherit;
        color: #374151;
        resize: vertical;
        box-sizing: border-box;
        transition: all 0.2s ease;
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
        margin-bottom: 16px;
    }
    
    .tag-label {
        display: block;
        margin-bottom: 6px;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
    }
    
    /* Custom Dropdown Styles (matching ServiceFilter) */
    .tag-dropdown {
        position: relative;
    }
    
    .dropdown-button {
        display: flex;
        align-items: center;
        justify-content: space-between;
        gap: 8px;
        width: 100%;
        padding: 10px 12px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        transition: all 0.2s;
        box-sizing: border-box;
    }
    
    .dropdown-button:hover {
        background: #f9fafb;
        border-color: #d1d5db;
    }
    
    .dropdown-text {
        flex: 1;
        text-align: left;
    }
    
    .chevron {
        transition: transform 0.2s;
        color: #6b7280;
        flex-shrink: 0;
    }
    
    .chevron.rotated {
        transform: rotate(180deg);
    }
    
    .dropdown-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 9;
    }
    
    .dropdown-menu {
        position: absolute;
        top: calc(100% + 4px);
        left: 0;
        right: 0;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1);
        max-height: 300px;
        overflow-y: auto;
        z-index: 10;
    }
    
    .dropdown-item {
        display: flex;
        align-items: center;
        gap: 12px;
        width: 100%;
        padding: 12px 16px;
        background: none;
        border: none;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        text-align: left;
        transition: background 0.2s;
    }
    
    .dropdown-item:hover {
        background: #f9fafb;
    }
    
    .dropdown-item.selected {
        background: #eff6ff;
        color: #3b82f6;
        font-weight: 500;
    }
    
    .checkbox {
        width: 16px;
        height: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: #10b981;
        flex-shrink: 0;
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
        font-size: 13px;
        font-weight: 500;
    }
    
    .chip-remove {
        background: none;
        border: none;
        color: #3b82f6;
        font-size: 18px;
        line-height: 1;
        cursor: pointer;
        padding: 0;
        width: 16px;
        height: 16px;
        display: flex;
        align-items: center;
        justify-content: center;
        border-radius: 50%;
        transition: background 0.2s ease;
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
        padding: 12px 14px;
        background: #f9fafb;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
        font-family: inherit;
        color: #374151;
        resize: vertical;
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
        background: #eff6ff;
        color: #3b82f6;
        border-radius: 3px;
        font-size: 12px;
        font-weight: 500;
    }
    
    .note-content {
        color: #4b5563;
        font-size: 14px;
        line-height: 1.6;
        white-space: pre-wrap;
        word-wrap: break-word;
    }
    
    /* Loading and Error States */
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