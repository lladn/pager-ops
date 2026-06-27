<script lang="ts">
    import { createEventDispatcher, onDestroy } from 'svelte';
    import type { store } from '../../wailsjs/go/models';
    import { SetIncidentCustomFieldValue, GetIncidentCustomFieldValues } from '../../wailsjs/go/main/App';

    type CustomField = store.CustomField;

    export let field: CustomField;
    export let incidentId: string;

    const dispatch = createEventDispatcher<{ saved: CustomField }>();

    // Selections normalized to arrays internally (length 0/1 for single-value fields).
    let selected: string[] = [];
    let saved: string[] = [];
    let openDropdown = false;
    let saving = false;
    let error: string | null = null;
    let lastKey = '';
    let dropdownEl: HTMLElement;

    // Live status read back from GET /incidents/{id}/custom_fields/values.
    let currentValue: any = undefined;
    let statusLoading = false;
    let statusError: string | null = null;

    $: isMulti = field?.field_type === 'multi_value' || field?.field_type === 'multi_value_fixed';
    $: options = field?.options || [];

    // (Re)initialize state whenever the field or incident changes.
    $: key = `${incidentId}:${field?.id}`;
    $: if (field && key !== lastKey) {
        lastKey = key;
        saved = toArray(field.value);
        selected = [...saved];
        openDropdown = false;
        error = null;
        loadStatus();
    }

    // Reactive — depends on selected & saved, so the Save button updates on every pick.
    $: dirty = !arraysEqual(selected, saved);
    $: dropdownText = selected.length === 0
        ? 'Select...'
        : (isMulti ? `${selected.length} selected` : selected[0]);

    // Human-readable current status value.
    $: currentDisplay = formatValue(currentValue);
    $: isUnset = currentDisplay === 'Not set';

    function formatValue(value: any): string {
        if (value === null || value === undefined || value === '') return 'Not set';
        if (Array.isArray(value)) return value.length > 0 ? value.map(v => String(v)).join(', ') : 'Not set';
        return String(value);
    }

    // Fetch the current value for THIS field from the values endpoint.
    async function loadStatus() {
        if (!incidentId || !field?.id) return;

        const fieldId = field.id;
        statusLoading = true;
        statusError = null;

        try {
            const values = await GetIncidentCustomFieldValues(incidentId);
            // Ignore a stale response if the active field/incident changed mid-flight.
            if (`${incidentId}:${fieldId}` !== key) return;
            const match = (values || []).find(v => v.id === fieldId);
            currentValue = match ? match.value : null;
        } catch (err) {
            if (`${incidentId}:${fieldId}` !== key) return;
            statusError = err?.toString() || 'Failed to load current value';
            currentValue = undefined;
        } finally {
            if (`${incidentId}:${fieldId}` === key) {
                statusLoading = false;
            }
        }
    }

    function toArray(value: any): string[] {
        if (value === null || value === undefined) return [];
        if (Array.isArray(value)) return value.map(v => String(v));
        return [String(value)];
    }

    function arraysEqual(a: string[], b: string[]): boolean {
        if (a.length !== b.length) return false;
        const sa = [...a].sort();
        const sb = [...b].sort();
        return sa.every((v, i) => v === sb[i]);
    }

    function toggleDropdown() {
        openDropdown = !openDropdown;
    }

    function closeDropdown() {
        openDropdown = false;
    }

    // Close the dropdown when clicking anywhere outside it.
    function handleOutsideClick(event: MouseEvent) {
        if (dropdownEl && !dropdownEl.contains(event.target as Node)) {
            openDropdown = false;
        }
    }

    $: if (openDropdown) {
        document.addEventListener('click', handleOutsideClick);
    } else {
        document.removeEventListener('click', handleOutsideClick);
    }

    onDestroy(() => {
        document.removeEventListener('click', handleOutsideClick);
    });

    function handleSelect(value: string) {
        if (isMulti) {
            if (selected.includes(value)) {
                selected = selected.filter(v => v !== value);
            } else {
                selected = [...selected, value];
            }
        } else {
            selected = [value];
            closeDropdown();
        }
    }

    function removeValue(value: string) {
        selected = selected.filter(v => v !== value);
    }

    async function handleSave() {
        if (saving || !dirty) return;

        // Single-value fields send a string (empty clears); multi sends an array.
        const value: any = isMulti ? selected : (selected[0] ?? '');

        saving = true;
        error = null;

        try {
            await SetIncidentCustomFieldValue(incidentId, field.id, value);
            saved = [...selected];
            // Notify parent so its cached field value stays in sync.
            dispatch('saved', { ...field, value: isMulti ? selected : (selected[0] ?? null) } as CustomField);
            // Re-read the value from the server to confirm the persisted status.
            loadStatus();
        } catch (err) {
            error = err?.toString() || 'Failed to save custom field';
        } finally {
            saving = false;
        }
    }
</script>

<div class="field-container">
    <!-- svelte-ignore a11y-label-has-associated-control -->
    <label class="field-label">{field.display_name}</label>

    {#if options.length > 0}
        <div class="cf-dropdown" bind:this={dropdownEl}>
            <div class="field-row">
                <div class="custom-dropdown">
                    <button class="dropdown-button" on:click={toggleDropdown} type="button">
                        <span>{dropdownText}</span>
                        <svg
                            class="dropdown-arrow"
                            class:rotated={openDropdown}
                            width="14"
                            height="14"
                            viewBox="0 0 20 20"
                            fill="currentColor"
                        >
                            <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
                        </svg>
                    </button>
                </div>

                <button
                    class="save-button"
                    class:active={dirty && !saving}
                    class:loading={saving}
                    on:click={handleSave}
                    disabled={!dirty || saving}
                >
                    {#if saving}
                        <span class="spinner"></span>
                        Saving...
                    {:else}
                        Save
                    {/if}
                </button>
            </div>

            {#if openDropdown}
                <div class="dropdown-list">
                    {#each options as option}
                        <button
                            class="dropdown-item"
                            class:selected={selected.includes(option.value)}
                            on:click={() => handleSelect(option.value)}
                            type="button"
                        >
                            {#if isMulti}
                                <span class="checkbox" class:checked={selected.includes(option.value)}>
                                    {#if selected.includes(option.value)}✓{/if}
                                </span>
                            {/if}
                            {option.value}
                        </button>
                    {/each}
                </div>
            {/if}
        </div>

        {#if isMulti && selected.length > 0}
            <div class="selected-tags">
                {#each selected as value}
                    <span class="tag-chip">
                        {value}
                        <button class="tag-remove" on:click={() => removeValue(value)} type="button">×</button>
                    </span>
                {/each}
            </div>
        {/if}
    {:else}
        <!-- No fixed options to choose from; show the current value read-only. -->
        <div class="field-readonly">{selected.length > 0 ? selected.join(', ') : 'No options available'}</div>
    {/if}

    <!-- Current value status (read back from custom_fields/values) -->
    <div class="field-status">
        <div class="status-header">
            <span class="status-label">Current value</span>
            <button class="status-refresh" on:click={loadStatus} disabled={statusLoading} title="Refresh current value" type="button">
                <svg class="refresh-icon" class:spinning={statusLoading} width="13" height="13" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd" />
                </svg>
            </button>
        </div>
        {#if statusLoading}
            <span class="status-value muted">Loading…</span>
        {:else if statusError}
            <span class="status-value status-error">{statusError}</span>
        {:else}
            <span class="status-value" class:muted={isUnset}>{currentDisplay}</span>
        {/if}
    </div>

    {#if error}
        <div class="error-banner">
            <span class="error-icon">⚠️</span>
            <p>{error}</p>
        </div>
    {/if}
</div>

<style>
    .field-container {
        padding: 16px;
    }

    .field-label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
        margin-bottom: 6px;
    }

    .field-row {
        display: flex;
        align-items: center;
        gap: 8px;
    }

    /* Custom Dropdown (mirrors PanelNotes tag dropdowns) */
    .cf-dropdown {
        position: relative;
    }

    .custom-dropdown {
        flex: 1;
        min-width: 0;
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

    /* In-flow option list: pushes content down so it can never be clipped by the
       scrollable panel (the previous absolute menu could get cut off). */
    .dropdown-list {
        margin-top: 6px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
        max-height: 240px;
        overflow-y: auto;
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

    .save-button {
        padding: 8px 12px;
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
        white-space: nowrap;
        flex-shrink: 0;
    }

    .save-button.active {
        color: #ffffff;
        background: #3b82f6;
        border-color: #3b82f6;
        cursor: pointer;
    }

    .save-button.active:hover:not(:disabled) {
        background: #2563eb;
    }

    .save-button:disabled {
        opacity: 0.6;
        cursor: not-allowed;
    }

    .save-button.loading {
        background: #eff6ff;
        color: #3b82f6;
        border-color: #93c5fd;
    }

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

    .field-readonly {
        padding: 8px 10px;
        border: 1px solid #f3f4f6;
        border-radius: 6px;
        font-size: 13px;
        color: #6b7280;
        background: #f9fafb;
    }

    .field-status {
        margin-top: 14px;
        padding-top: 12px;
        border-top: 1px solid #f3f4f6;
    }

    .status-header {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-bottom: 6px;
    }

    .status-label {
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
        text-transform: uppercase;
        letter-spacing: 0.05em;
    }

    .status-refresh {
        display: inline-flex;
        align-items: center;
        justify-content: center;
        padding: 2px;
        background: none;
        border: none;
        color: #9ca3af;
        cursor: pointer;
        border-radius: 4px;
        transition: color 0.15s ease;
    }

    .status-refresh:hover:not(:disabled) {
        color: #3b82f6;
    }

    .status-refresh:disabled {
        cursor: not-allowed;
    }

    .refresh-icon.spinning {
        animation: spin 0.8s linear infinite;
    }

    .status-value {
        display: block;
        font-size: 14px;
        color: #111827;
        font-weight: 500;
        word-wrap: break-word;
        overflow-wrap: anywhere;
    }

    .status-value.muted {
        color: #9ca3af;
        font-weight: 400;
        font-style: italic;
    }

    .status-value.status-error {
        color: #991b1b;
        font-weight: 400;
        font-size: 13px;
    }

    .error-banner {
        margin-top: 12px;
        padding: 12px;
        background: #fef2f2;
        border: 1px solid #fecaca;
        border-radius: 8px;
        display: flex;
        align-items: center;
        gap: 8px;
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
        word-wrap: break-word;
        overflow-wrap: break-word;
    }
</style>
