<script lang="ts">
    import { sidebarData, sidebarLoading, sidebarError, loadIncidentSidebarData } from '../stores/incidents';
    import { GetServiceConfigByServiceID, AddIncidentNote } from '../../wailsjs/go/main/App';
    import { store, type database } from '../../wailsjs/go/models';
    import { onMount, onDestroy } from 'svelte';
    import { BrowserOpenURL } from '../../wailsjs/runtime/runtime';
    
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
    
    // Dropdown state for custom dropdowns
    let openDropdown: string | null = null;
    
    // Loading state for add note button
    let addingNote = false;
    
    // LocalStorage key prefix
    const DRAFT_PREFIX = 'pagerduty_draft_';
    
    // REACTIVE: Make hasContent a reactive variable instead of a function
    $: hasContent = computeHasContent(questionResponses, tagSelections, freeformNote);
    
    function computeHasContent(
        questions: Record<string, string>, 
        tags: Record<string, string[]>, 
        freeform: string
    ): boolean {
        const hasQuestions = Object.values(questions).some(v => v && v.trim().length > 0);
        const hasTags = Object.values(tags).some(v => v && v.length > 0);
        const hasFreeform = freeform ? freeform.trim().length > 0 : false;
        return hasQuestions || hasTags || hasFreeform;
    }
    
    // NEW: Linkify URLs in text content with XSS prevention
    function linkifyText(text: string): string {
        if (!text) return '';
        
        // Escape HTML to prevent XSS
        const escapeHtml = (unsafe: string): string => {
            return unsafe
                .replace(/&/g, "&amp;")
                .replace(/</g, "&lt;")
                .replace(/>/g, "&gt;")
                .replace(/"/g, "&quot;")
                .replace(/'/g, "&#039;");
        };
        
        // Escape the text first
        const escaped = escapeHtml(text);
        
        // URL regex pattern - matches http:// and https:// URLs
        const urlRegex = /(https?:\/\/[^\s<>"{}|\\^`\[\]]+)/g;
        
        // Replace URLs with clickable links
        return escaped.replace(urlRegex, (url) => {
            // Additional safety check - only allow http and https
            if (!url.startsWith('http://') && !url.startsWith('https://')) {
                return url;
            }
            return `<a href="#" data-url="${url}" class="note-link">${url}</a>`;
        });
    }
    
    // NEW: Handle link clicks - open in default browser
    function handleLinkClick(event: MouseEvent) {
        const target = event.target as HTMLElement;
        
        // Check if clicked element is a link
        if (target.tagName === 'A' && target.classList.contains('note-link')) {
            event.preventDefault();
            const url = target.getAttribute('data-url');
            
            if (url) {
                try {
                    BrowserOpenURL(url);
                } catch (err) {
                    console.error('Failed to open URL in browser:', err);
                    alert(`Failed to open URL: ${err}`);
                }
            }
        }
    }
    
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
        
        // Initialize tag selections
        if (serviceTypes?.tags) {
            serviceTypes.tags.forEach(tag => {
                if (!tagSelections[tag.name]) {
                    tagSelections[tag.name] = [];
                    }
                });
            }
        } catch (err) {
            console.error('Failed to load service config:', err);
        } finally {
            loadingConfig = false;
        }
    }
    
    function getDraftKey(incidentId: string): string {
        return `${DRAFT_PREFIX}${incidentId}`;
    }
    
    function saveDraftToLocalStorage(incidentId: string) {
    const draftKey = getDraftKey(incidentId);
    const draftData = {
        questionResponses,
        tagSelections,
        freeformNote
    };
    
    try {
        localStorage.setItem(draftKey, JSON.stringify(draftData));
    } catch (err) {
        console.error('Failed to save draft:', err);
    }
}
    
    function restoreDraftFromLocalStorage(incidentId: string) {
        const draftKey = getDraftKey(incidentId);
        
        try {
            const stored = localStorage.getItem(draftKey);
            if (stored) {
                const draftData = JSON.parse(stored);
                questionResponses = draftData.questionResponses || {};
                tagSelections = draftData.tagSelections || {};
                freeformNote = draftData.freeformNote || '';
                
                // Show saved status if draft exists
                draftSaveStatus = 'saved';
            }
        } catch (err) {
            console.error('Failed to restore draft:', err);
        }
    }
    
        function saveDraft() {
        draftSaveStatus = 'saving';
        
        if (saveTimeout) {
            clearTimeout(saveTimeout);
        }
        
        saveTimeout = window.setTimeout(() => {
            if (incident?.incident_id) {
                saveDraftToLocalStorage(incident.incident_id);
                draftSaveStatus = 'saved';
            }
        }, 500);
    }
    
        function clearDraft(incidentId: string) {
        const draftKey = getDraftKey(incidentId);
        try {
            localStorage.removeItem(draftKey);
            draftSaveStatus = '';
        } catch (err) {
            console.error('Failed to clear draft:', err);
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
            return 'Select...';
        }
        
        if (isSingle) {
            return selected[0];
        }
        
        return `${selected.length} selected`;
    }
    
    async function handleAddNote() {
        if (!hasContent) {
            alert('Please fill in at least one field before adding a note.');
            return;
        }
        
        if (addingNote) return;
        
        addingNote = true;
        
        try {
            // Build structured note data using Wails models
            const responses = Object.entries(questionResponses)
                .filter(([_, answer]) => answer && answer.trim())
                .map(([question, answer]) => 
                    new store.NoteResponse({
                        question,
                        answer: answer.trim()
                    })
                );
            
            const tags = Object.entries(tagSelections)
                .filter(([_, values]) => values && values.length > 0)
                .map(([tagName, values]) => 
                    new store.NoteTag({
                        tag_name: tagName,
                        selected_values: values
                    })
                );
            
            // Use any type to bypass strict type checking for the noteData object
            const noteData: any = {
                responses,
                tags,
                freeform_content: freeformNote.trim()
            };
            
            // Call backend to add note
            await AddIncidentNote(incident.incident_id, noteData);
            
            // Clear form on success
            questionResponses = {};
            tagSelections = {};
            freeformNote = '';
            
            // Clear draft from localStorage
            if (incident?.incident_id) {
                clearDraft(incident.incident_id);
            }
            
            // Reinitialize tag selections
            if (serviceTypes?.tags) {
                serviceTypes.tags.forEach(tag => {
                    tagSelections[tag.name] = [];
                });
            }
            
            // The backend will emit "sidebar-data-updated" event which will trigger the sidebar to refetch
            // Give it a moment, then manually refetch to show the new note immediately
            setTimeout(async () => {
                if (incident?.incident_id) {
                    await loadIncidentSidebarData(incident.incident_id);
                }
            }, 500);
            
        } catch (err) {
            console.error('Failed to add note:', err);
            alert(`Failed to add note: ${err}`);
        } finally {
            addingNote = false;
        }
    }
    
    function formatDate(date: Date | string): string {
        const d = typeof date === 'string' ? new Date(date) : date;
        const now = new Date();
        const diff = now.getTime() - d.getTime();
        const seconds = Math.floor(diff / 1000);
        const minutes = Math.floor(seconds / 60);
        const hours = Math.floor(minutes / 60);
        const days = Math.floor(hours / 24);
        
        if (days > 7) {
            return d.toLocaleDateString();
        } else if (days > 0) {
            return `${days}d ago`;
        } else if (hours > 0) {
            return `${hours}h ago`;
        } else if (minutes > 0) {
            return `${minutes}m ago`;
        } else {
            return 'Just now';
        }
    }
    
    function retry() {
        if (incident?.incident_id) {
            loadIncidentSidebarData(incident.incident_id);
        }
    }
</script>

<div class="notes-container">
    <!-- Add Notes Section -->
    <div class="add-notes-section">
        <div class="section-header">
            <p class="section-title">Add Notes</p>
            {#if draftSaveStatus === 'saving'}
            <span class="draft-status saving">Saving draft...</span>
            {:else if draftSaveStatus === 'saved'}
                <span class="draft-status saved">Draft saved</span>
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
                    {#each serviceTypes.tags as tag}
                        {@const isSingle = !!(tag.single && tag.single.length > 0)}
                        {@const options = (isSingle ? tag.single : tag.multiple) || []}
                        {@const selected = tagSelections[tag.name] || []}
                        
                        <div class="tag-group">
                            <!-- svelte-ignore a11y-label-has-associated-control -->
                            <label class="tag-label">{tag.name}</label>
                            
                            <!-- Custom dropdown -->
                            <div class="custom-dropdown">
                                <button 
                                    class="dropdown-button"
                                    on:click={() => toggleDropdown(tag.name)}
                                    type="button"
                                >
                                    <span>{getDropdownText(tag.name, isSingle)}</span>
                                    <svg 
                                        class="dropdown-arrow" 
                                        class:rotated={openDropdown === tag.name} 
                                        width="14" 
                                        height="14" 
                                        viewBox="0 0 20 20" 
                                        fill="currentColor"
                                    >
                                        <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                                    </svg>
                                </button>
                                
                                {#if openDropdown === tag.name}
                                    <div class="dropdown-menu">
                                        {#each options as option}
                                            <button
                                                class="dropdown-item"
                                                class:selected={selected.includes(option)}
                                                on:click={() => handleTagSelection(tag.name, option, isSingle)}
                                                type="button"
                                            >
                                                {#if !isSingle}
                                                    <span class="checkbox" class:checked={selected.includes(option)}>
                                                        {#if selected.includes(option)}✓{/if}
                                                    </span>
                                                {/if}
                                                {option}
                                            </button>
                                        {/each}
                                    </div>
                                {/if}
                            </div>
                            
                            <!-- Selected tags chips -->
                            {#if selected.length > 0}
                                <div class="selected-tags">
                                    {#each selected as value}
                                        <span class="tag-chip">
                                            {value}
                                            <button 
                                                class="tag-remove"
                                                on:click={() => removeTag(tag.name, value)}
                                                type="button"
                                            >×</button>
                                        </span>
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
            class:active={hasContent && !addingNote}
            class:loading={addingNote}
            on:click={handleAddNote}
            disabled={!hasContent || addingNote}
        >
            {#if addingNote}
                <span class="spinner"></span>
                Adding Note...
            {:else}
                Add Note
            {/if}
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
                    
                    <!-- Display freeform content with linkified URLs -->
                    {#if note.freeform_content || note.content}
                        <!-- svelte-ignore a11y-click-events-have-key-events -->
                        <!-- svelte-ignore a11y-no-static-element-interactions -->
                        <div class="note-content" on:click={handleLinkClick}>
                            {@html linkifyText(note.freeform_content || note.content)}
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

<!-- svelte-ignore a11y-click-events-have-key-events -->
<!-- svelte-ignore a11y-no-static-element-interactions -->
{#if openDropdown}
    <div class="dropdown-overlay" on:click={closeDropdown}></div>
{/if}

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
        color: #303030;
    }
    
    .draft-status {
        font-size: 11px;
        padding: 2px 8px;
        border-radius: 4px;
    }
    
    .draft-status.saving {
        color: #3b82f6;
        background: #eff6ff;
    }
    
    .draft-status.saved {
        color: #10b981;
        background: #f0fdf4;
    }
    
    .config-loading {
        padding: 12px;
        text-align: center;
        color: #6b7280;
        font-size: 13px;
    }
    
    .questions-section, .tags-section, .freeform-section {
        margin-bottom: 16px;
    }
    
    .subsection-title {
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
        margin: 0 0 8px 0;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }
    
    .question-group, .tag-group {
        margin-bottom: 12px;
    }
    
    .question-label, .tag-label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
        margin-bottom: 6px;
    }
    
    .question-textarea {
        width: 100%;
        padding: 8px 10px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 13px;
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
    
    /* Custom Dropdown Styles */
    .custom-dropdown {
        position: relative;
    }
    
    .dropdown-button {
        width: 100%;
        padding: 8px 10px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 13px;
        background: #f9fafb;
        text-align: left;
        cursor: pointer;
        display: flex;
        justify-content: space-between;
        align-items: center;
        transition: all 0.2s ease;
    }
    
    .dropdown-button:hover {
        background: #ffffff;
        border-color: #d1d5db;
    }
    
    .dropdown-arrow {
    color: #9ca3af;
    flex-shrink: 0;
    transition: transform 0.2s ease;
    }

    .dropdown-arrow.rotated {
        transform: rotate(180deg);
    }
    
    .dropdown-menu {
        position: absolute;
        top: 100%;
        left: 0;
        right: 0;
        margin-top: 4px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1);
        max-height: 200px;
        overflow-y: auto;
        z-index: 100;
    }
    
    .dropdown-item {
        width: 100%;
        padding: 8px 12px;
        border: none;
        background: white;
        text-align: left;
        font-size: 13px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: background 0.15s ease;
    }
    
    .dropdown-item:hover {
        background: #f9fafb;
    }
    
    .dropdown-item.selected {
        background: #eff6ff;
        color: #2563eb;
    }
    
    .checkbox {
        width: 16px;
        height: 16px;
        border: 1px solid #d1d5db;
        border-radius: 3px;
        display: flex;
        align-items: center;
        justify-content: center;
        font-size: 10px;
        flex-shrink: 0;
    }
    
    .checkbox.checked {
        background: #3b82f6;
        border-color: #3b82f6;
        color: white;
    }
    
    .dropdown-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 99;
    }
    
    .selected-tags {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
        margin-top: 8px;
    }
    
    .tag-chip {
        display: inline-flex;
        align-items: center;
        gap: 4px;
        padding: 4px 8px;
        background: #eff6ff;
        color: #1e40af;
        font-size: 12px;
        font-weight: 500;
        border-radius: 4px;
    }
    
    .tag-remove {
        background: none;
        border: none;
        color: #1e40af;
        font-size: 16px;
        line-height: 1;
        cursor: pointer;
        padding: 0;
        margin-left: 2px;
    }
    
    .tag-remove:hover {
        color: #1e3a8a;
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
        margin-bottom: 4px; 
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
        padding: 4px 8px;
        background: transparent;
        color: #d1d5db;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 13px;
        font-weight: 500;
        cursor: not-allowed;
        transition: all 0.2s ease;
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }
    
    .add-note-button.active {
        color: #ffffff;
        background: #3b82f6;
        border-color: #3b82f6;
        cursor: pointer;
    }
    
    .add-note-button.active:hover:not(:disabled) {
        background: #2563eb;
    }
    
    .add-note-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }
    
    .add-note-button.loading {
        background: #eff6ff;
        color: #3b82f6;
        border-color: #93c5fd;
    }
    
    /* Spinner animation */
    .spinner {
        width: 14px;
        height: 14px;
        border: 2px solid #93c5fd;
        border-top: 2px solid #3b82f6;
        border-radius: 50%;
        animation: spin 0.8s linear infinite;
    }
    
    @keyframes spin {
        0% { transform: rotate(0deg); }
        100% { transform: rotate(360deg); }
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
    font-size: 12px;
    color: #374151;
    line-height: 1.4;
    white-space: pre-wrap;
    word-wrap: break-word;
    word-break: break-word;
    overflow-wrap: anywhere;
    max-width: 100%;
    overflow: hidden;
    }

    .note-content :global(a.note-link) {
        color: #2563eb;
        text-decoration: underline;
        cursor: pointer;
        word-break: break-all;
        transition: all 0.15s ease;
    }

    .note-content :global(a.note-link:hover) {
        color: #1d4ed8;
        background: #eff6ff;
        text-decoration: underline;
    }

    .note-content :global(a.note-link:active) {
        color: #1e40af;
    }
    
    .notes-empty {
        padding: 24px;
        text-align: center;
        color: #9ca3af;
        font-size: 14px;
    }
    
    /* Loading skeletons */
    .skeleton-container {
        padding: 14px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        margin-bottom: 12px;
    }
    
    .skeleton-line {
        height: 16px;
        background: linear-gradient(90deg, #f3f4f6 25%, #e5e7eb 50%, #f3f4f6 75%);
        background-size: 200% 100%;
        animation: skeleton-loading 1.5s ease-in-out infinite;
        border-radius: 4px;
        margin-bottom: 8px;
    }
    
    .skeleton-line.short {
        width: 60%;
        margin-bottom: 0;
    }
    
    @keyframes skeleton-loading {
        0% {
            background-position: 200% 0;
        }
        100% {
            background-position: -200% 0;
        }
    }
    
    /* Error banner */
    .error-banner {
        padding: 12px;
        background: #fef2f2;
        border: 1px solid #fecaca;
        border-radius: 8px;
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 12px;
    }
    
    .error-icon {
        font-size: 16px;
        flex-shrink: 0;
    }
    
    .error-banner p {
        flex: 1;
        margin: 0;
        color: #991b1b;
        font-size: 13px;
    }
    
    .retry-button {
        padding: 4px 12px;
        background: #dc2626;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        transition: background 0.2s ease;
    }
    
    .retry-button:hover {
        background: #b91c1c;
    }
</style>