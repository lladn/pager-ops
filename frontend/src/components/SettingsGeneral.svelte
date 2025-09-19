<script lang="ts">
    import { notificationConfig, availableSounds, loadNotificationConfig, loadAvailableSounds } from '../stores/notifications';
    import { 
        ConfigureAPIKey, GetAPIKey, GetFilterByUser, SetFilterByUser, 
        SetNotificationEnabled, SetNotificationSound, TestNotificationSound, 
        SnoozeNotificationSound, UnsnoozeNotificationSound, IsNotificationSnoozed
    } from '../../wailsjs/go/main/App';
    import { onMount } from 'svelte';
    
    export let errorMessage: string = '';
    export let successMessage: string = '';
    
    let apiKey = '';
    let filterByUser = true;
    let notificationSnoozed = false;
    let selectedSound = 'default';
    let showSoundDropdown = false;
    
    onMount(async () => {
        await loadApiKey();
        await loadFilterState();
        await loadNotificationConfig();
        await loadAvailableSounds();
        await checkSnoozeStatus();
    });
    
    async function checkSnoozeStatus() {
        try {
            notificationSnoozed = await IsNotificationSnoozed();
        } catch (err) {
            notificationSnoozed = false;
        }
    }
    
    async function loadApiKey() {
        try {
            const key = await GetAPIKey();
            apiKey = key || '';
        } catch (err) {
            apiKey = '';
        }
    }
    
    async function loadFilterState() {
        try {
            const state = await GetFilterByUser();
            filterByUser = state;
        } catch (err) {
            filterByUser = true;
            try {
                await SetFilterByUser(true);
            } catch (e) {
                // Ignore error on setting default
            }
        }
    }
    
    async function saveApiKey() {
        errorMessage = '';
        successMessage = '';
        
        if (!apiKey.trim()) {
            errorMessage = 'API Key is required';
            return;
        }
        
        try {
            await ConfigureAPIKey(apiKey);
            successMessage = 'API Key saved successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = 'Failed to save API Key';
        }
    }
    
    async function toggleAssignedFilter() {
        try {
            filterByUser = !filterByUser;
            await SetFilterByUser(filterByUser);
            successMessage = `Filter set to show ${filterByUser ? 'assigned incidents only' : 'all incidents'}`;
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = 'Failed to update filter setting';
        }
    }
    
    async function toggleNotifications() {
        try {
            const newState = !$notificationConfig.enabled;
            await SetNotificationEnabled(newState);
            await loadNotificationConfig();
            successMessage = `Notifications ${newState ? 'enabled' : 'disabled'}`;
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = 'Failed to toggle notifications';
        }
    }
    
    async function selectSound(sound: string) {
        selectedSound = sound;
        showSoundDropdown = false;
        
        try {
            await SetNotificationSound(sound);
            await loadNotificationConfig();
            successMessage = 'Notification sound updated';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = 'Failed to update notification sound';
        }
    }
    
    async function testSound() {
        try {
            await TestNotificationSound();
        } catch (err) {
            errorMessage = 'Failed to test sound';
        }
    }
    
    async function toggleSnooze() {
        try {
            if (notificationSnoozed) {
                await UnsnoozeNotificationSound();
                notificationSnoozed = false;
                successMessage = 'Sound notifications resumed';
            } else {
                await SnoozeNotificationSound(30);
                notificationSnoozed = true;
                successMessage = 'Sound snoozed for 30 minutes';
            }
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = 'Failed to toggle snooze';
        }
    }
    
    function toggleSoundDropdown() {
        showSoundDropdown = !showSoundDropdown;
    }
    
    function handleOutsideClick(event: MouseEvent) {
        const target = event.target as HTMLElement;
        if (!target.closest('.sound-selector-container')) {
            showSoundDropdown = false;
        }
    }
    
    $: if (showSoundDropdown) {
        document.addEventListener('click', handleOutsideClick);
    } else {
        document.removeEventListener('click', handleOutsideClick);
    }
    
    $: selectedSound = $notificationConfig.sound || 'default';
</script>

<div class="general-settings">
    <!-- API Key Section -->
    <div class="settings-section">
        <h3>API Key</h3>
        <p class="setting-description">Enter your PagerDuty API key</p>
        <div class="api-key-controls">
            <input 
                type="password" 
                bind:value={apiKey}
                placeholder="Enter your PagerDuty API key"
                class="settings-input"
            />
            <button class="btn btn-primary" on:click={saveApiKey}>
                Save API Key
            </button>
        </div>
    </div>
    
    <!-- Show Assigned Incidents Only Toggle -->
    <div class="settings-section">
        <div class="toggle-setting">
            <div>
                <h3>Show Assigned Incidents Only</h3>
                <p class="setting-description">Display only incidents assigned to you</p>
            </div>
            <button 
                class="toggle-button"
                class:active={filterByUser}
                on:click={toggleAssignedFilter}
            >
                <span class="toggle-slider"></span>
            </button>
        </div>
    </div>
    
    <!-- Notifications Section -->
    <div class="settings-section">
        <div class="toggle-setting">
            <div>
                <h3>Enable Notifications</h3>
                <p class="setting-description">Get sound notifications for new incidents</p>
            </div>
            <button 
                class="toggle-button"
                class:active={$notificationConfig.enabled}
                on:click={toggleNotifications}
            >
                <span class="toggle-slider"></span>
            </button>
        </div>
        
        {#if $notificationConfig.enabled}
            <div class="notification-settings">
                <!-- Custom Sound Selector with Dropdown UI -->
                <div class="sound-selector-container">
                    <label for="notification-sound">Notification Sound</label>
                    <div class="sound-controls">
                        <div class="custom-dropdown">
                            <button 
                                class="dropdown-toggle"
                                on:click|stopPropagation={toggleSoundDropdown}
                                aria-expanded={showSoundDropdown}
                            >
                                <span class="dropdown-value">
                                    {selectedSound === 'default' ? 'Default' : selectedSound}
                                </span>
                                <svg class="dropdown-arrow" width="12" height="12" viewBox="0 0 12 12" fill="none" stroke="currentColor" stroke-width="1.5">
                                    <path d="M2 4L6 8L10 4" />
                                </svg>
                            </button>
                            
                            {#if showSoundDropdown}
                                <div class="dropdown-menu">
                                    {#each $availableSounds as sound}
                                        <button 
                                            class="dropdown-item"
                                            class:selected={selectedSound === sound}
                                            on:click={() => selectSound(sound)}
                                        >
                                            {#if selectedSound === sound}
                                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                                    <polyline points="20 6 9 17 4 12"></polyline>
                                                </svg>
                                            {/if}
                                            <span class="sound-name">
                                                {sound === 'default' ? 'Default' : sound}
                                            </span>
                                        </button>
                                    {/each}
                                </div>
                            {/if}
                        </div>
                        
                        <button class="btn-test" on:click={testSound}>
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
                                <path d="M15.54 8.46a5 5 0 0 1 0 7.07"></path>
                                <path d="M19.07 4.93a10 10 0 0 1 0 14.14"></path>
                            </svg>
                            Test
                        </button>
                    </div>
                </div>
                
                <div class="snooze-controls">
                    <button 
                        class="btn-snooze"
                        class:snoozed={notificationSnoozed}
                        on:click={toggleSnooze}
                    >
                        {#if notificationSnoozed}
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
                                <line x1="22" y1="9" x2="22" y2="15"></line>
                            </svg>
                            Resume Sound
                        {:else}
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polygon points="11 5 6 9 2 9 2 15 6 15 11 19 11 5"></polygon>
                                <line x1="23" y1="9" x2="17" y2="15"></line>
                                <line x1="17" y1="9" x2="23" y2="15"></line>
                            </svg>
                            Snooze Sound (30 min)
                        {/if}
                    </button>
                    <p class="snooze-description">
                        {#if notificationSnoozed}
                            Sound notifications are temporarily muted
                        {:else}
                            Temporarily mute sound playback
                        {/if}
                    </p>
                </div>
            </div>
        {/if}
    </div>
</div>

<style>
    .general-settings {
        padding: 0;
    }
    
    .settings-section {
        padding: 20px;
        border-bottom: 1px solid #e5e7eb;
    }
    
    .settings-section:last-child {
        border-bottom: none;
    }
    
    .settings-section h3 {
        font-size: 14px;
        font-weight: 600;
        color: #111827;
        margin: 0 0 8px 0;
    }

    .api-key-controls {
        display: flex;
        gap: 12px;
        align-items: flex-start;
        flex-direction: column;
    }
    
    .settings-input {
        width: 100%;
        padding: 10px 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
    }
    
    .toggle-setting {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .setting-description {
        font-size: 12px;
        color: #6b7280;
        margin: 4px 0 0 0;
    }
    
    .toggle-button {
        position: relative;
        width: 48px;
        height: 24px;
        background: #d1d5db;
        border: none;
        border-radius: 12px;
        cursor: pointer;
        transition: background 0.2s;
    }
    
    .toggle-button.active {
        background: #3b82f6;
    }
    
    .toggle-slider {
        position: absolute;
        top: 2px;
        left: 2px;
        width: 20px;
        height: 20px;
        background: white;
        border-radius: 50%;
        transition: transform 0.2s;
    }
    
    .toggle-button.active .toggle-slider {
        transform: translateX(24px);
    }
    
    .notification-settings {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid #f3f4f6;
    }
    
    .sound-selector-container {
        margin-bottom: 16px;
    }
    
    .sound-selector-container label {
        display: block;
        margin-bottom: 8px;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
    }
    
    .sound-controls {
        display: flex;
        gap: 8px;
        align-items: center;
    }
    
    .custom-dropdown {
        position: relative;
        flex: 1;
    }
    
    .dropdown-toggle {
        width: 100%;
        padding: 8px 12px;
        background: white;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        text-align: left;
        cursor: pointer;
        display: flex;
        justify-content: space-between;
        align-items: center;
        transition: all 0.2s;
    }
    
    .dropdown-toggle:hover {
        border-color: #9ca3af;
    }
    
    .dropdown-toggle:focus {
        outline: none;
        border-color: #3b82f6;
        box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
    }
    
    .dropdown-value {
        color: #111827;
        font-weight: 500;
    }
    
    .dropdown-arrow {
        transition: transform 0.2s;
        color: #6b7280;
    }
    
    .dropdown-toggle[aria-expanded="true"] .dropdown-arrow {
        transform: rotate(180deg);
    }
    
    .dropdown-menu {
        position: absolute;
        top: calc(100% + 4px);
        left: 0;
        right: 0;
        background: white;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1);
        z-index: 10;
        max-height: 200px;
        overflow-y: auto;
    }
    
    .dropdown-item {
        width: 100%;
        padding: 8px 12px;
        background: none;
        border: none;
        text-align: left;
        cursor: pointer;
        transition: background 0.15s;
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        color: #374151;
    }
    
    .dropdown-item:hover {
        background: #f3f4f6;
    }
    
    .dropdown-item.selected {
        background: #eff6ff;
        color: #3b82f6;
        font-weight: 500;
    }
    
    .dropdown-item svg {
        width: 14px;
        height: 14px;
        flex-shrink: 0;
        color: #3b82f6;
    }
    
    .dropdown-item:not(.selected) svg {
        visibility: hidden;
        width: 14px;
    }
    
    .sound-name {
        flex: 1;
    }
    
    .btn-test {
        padding: 8px 16px;
        background: #f3f4f6;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        color: #374151;
        display: flex;
        align-items: center;
        gap: 6px;
    }
    
    .btn-test:hover {
        background: #e5e7eb;
        border-color: #9ca3af;
    }
    
    .btn-test svg {
        width: 16px;
        height: 16px;
    }
    
    .snooze-controls {
        margin-top: 12px;
    }
    
    .btn-snooze {
        padding: 8px 14px;
        background: #fef3c7;
        border: 1px solid #fcd34d;
        color: #78350f;
        border-radius: 6px;
        font-size: 13px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        display: inline-flex;
        align-items: center;
        gap: 6px;
    }
    
    .btn-snooze:hover {
        background: #fed7aa;
        border-color: #fb923c;
    }
    
    .btn-snooze.snoozed {
        background: #dcfce7;
        border: 1px solid #86efac;
        color: #14532d;
    }
    
    .btn-snooze.snoozed:hover {
        background: #bbf7d0;
        border-color: #4ade80;
    }
    
    .snooze-description {
        font-size: 11px;
        color: #6b7280;
        margin: 8px 0 0 0;
    }
    
    .btn {
        padding: 10px 16px;
        border: none;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-primary {
        background: #3b82f6;
        color: white;
    }
    
    .btn-primary:hover {
        background: #2563eb;
    }
</style>