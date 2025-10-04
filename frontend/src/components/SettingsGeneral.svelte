<script lang="ts">
    import { 
        notificationConfig, 
        availableSounds, 
        loadNotificationConfig, 
        loadAvailableSounds } from '../stores/notifications';
    import { assignedFilterEnabled } from '../stores/incidents';
    import { 
        ConfigureAPIKey, GetAPIKey, 
        GetFilterByUser, SetFilterByUser, 
        SetNotificationEnabled, SetNotificationSound, TestNotificationSound, 
        SnoozeNotificationSound, UnsnoozeNotificationSound, IsNotificationSnoozed, 
        SetBrowserRedirect, GetBrowserRedirect
    } from '../../wailsjs/go/main/App';
    import { onMount } from 'svelte';
    
    export let errorMessage = '';
    export let successMessage = '';
    
    let apiKey = '';
    let notificationSnoozed = false;
    let showSoundDropdown = false;
    let browserRedirect = false;
    
    onMount(async () => {
        try {
            apiKey = await GetAPIKey();
        } catch (err) {
            console.error('Failed to get API key:', err);
        }
        
        try {
            const isAssigned = await GetFilterByUser();
            assignedFilterEnabled.set(isAssigned);
        } catch (err) {
            console.error('Failed to get filter setting:', err);
        }
        
        await loadNotificationConfig();
        await loadAvailableSounds();
        
        try {
            notificationSnoozed = await IsNotificationSnoozed();
        } catch (err) {
            console.error('Failed to get snooze status:', err);
        }
        
        try {
            browserRedirect = await GetBrowserRedirect();
        } catch (err) {
            console.error('Failed to get browser redirect setting:', err);
        }
    });
    
    async function saveApiKey() {
        errorMessage = '';
        successMessage = '';
        
        if (!apiKey) {
            errorMessage = 'Please enter an API key';
            return;
        }
        
        try {
            await ConfigureAPIKey(apiKey);
            successMessage = 'API key saved successfully';
            setTimeout(() => {
                successMessage = '';
            }, 3000);
        } catch (err) {
            errorMessage = 'Failed to save API key: ' + err;
        }
    }
    
    async function toggleFilterByUser() {
        const newValue = !$assignedFilterEnabled;
        assignedFilterEnabled.set(newValue);
        
        try {
            await SetFilterByUser(newValue);
        } catch (err) {
            console.error('Failed to toggle filter by user:', err);
            // Revert on error
            assignedFilterEnabled.set(!newValue);
        }
    }
    
    async function toggleNotifications() {
        const newState = !$notificationConfig.enabled;
        await SetNotificationEnabled(newState);
        await loadNotificationConfig();
    }
    
    async function toggleBrowserRedirect() {
        browserRedirect = !browserRedirect;
        try {
            await SetBrowserRedirect(browserRedirect);
        } catch (err) {
            console.error('Failed to update browser redirect setting:', err);
            browserRedirect = !browserRedirect; // Revert on error
        }
    }
    
    async function selectSound(sound: string) {
        showSoundDropdown = false;
        await SetNotificationSound(sound);
        await loadNotificationConfig();
    }
    
    async function testSound() {
        try {
            await TestNotificationSound();
        } catch (err) {
            console.error('Failed to test sound:', err);
        }
    }
    
    async function snoozeSound() {
        await SnoozeNotificationSound(15); // 15 minutes default
        notificationSnoozed = true;
    }
    
    async function unsnoozeSound() {
        await UnsnoozeNotificationSound();
        notificationSnoozed = false;
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
            class:active={$assignedFilterEnabled}
            on:click={toggleFilterByUser}
        >
                <span class="toggle-slider"></span>
            </button>
        </div>
    </div>
    
    <!-- Notifications Section -->
    <div class="settings-section">
        <div class="toggle-setting">
            <div>
                <h3>Notifications</h3>
                <p class="setting-description">Show desktop notifications for triggered incidents</p>
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
                <!-- Sound Selector -->
                <div class="sound-setting">
                    <div class="sound-selector-container">
                        <button class="sound-selector" on:click={toggleSoundDropdown}>
                            <span>{selectedSound}</span>
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <polyline points="6 9 12 15 18 9"></polyline>
                            </svg>
                        </button>
                        
                        {#if showSoundDropdown}
                            <div class="sound-dropdown">
                                {#each $availableSounds as sound}
                                    <button 
                                        class="sound-option"
                                        class:selected={selectedSound === sound}
                                        on:click={() => selectSound(sound)}
                                    >
                                        {sound}
                                    </button>
                                {/each}
                            </div>
                        {/if}
                    </div>
                    
                    <button class="btn btn-secondary" on:click={testSound}>
                        Test Sound
                    </button>
                </div>
                
                <!-- Browser Redirect Toggle - Now inside notification settings -->
                <div class="toggle-setting browser-redirect-setting">
                    <div>
                        <h4>Auto-Open in Browser</h4>
                        <p class="setting-description">Automatically open triggered incidents in your browser</p>
                    </div>
                    <button 
                        class="toggle-button"
                        class:active={browserRedirect}
                        on:click={toggleBrowserRedirect}
                    >
                        <span class="toggle-slider"></span>
                    </button>
                </div>
                
                <!-- Snooze Button -->
                <div class="snooze-setting">
                    <button 
                        class="btn btn-secondary snooze-button"
                        on:click={notificationSnoozed ? unsnoozeSound : snoozeSound}
                    >
                        {#if notificationSnoozed}
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <circle cx="12" cy="12" r="10"></circle>
                                <polyline points="12 6 12 12 16 14"></polyline>
                            </svg>
                            Unsnooze
                        {:else}
                            <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                                <path d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path>
                            </svg>
                            Snooze 15 min
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
    
    .settings-section h4 {
        font-size: 13px;
        font-weight: 600;
        color: #374151;
        margin: 0 0 4px 0;
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
    
    .browser-redirect-setting {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid #f3f4f6;
    }
    
    .sound-setting {
        display: flex;
        gap: 8px;
        align-items: center;
    }
    
    .sound-selector-container {
        position: relative;
        flex: 1;
    }
    
    .sound-selector {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        background: white;
        cursor: pointer;
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 14px;
        color: #374151;
    }
    
    .sound-selector:hover {
        border-color: #9ca3af;
    }
    
    .sound-dropdown {
        position: absolute;
        top: calc(100% + 4px);
        left: 0;
        right: 0;
        background: white;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        box-shadow: 0 4px 6px -1px rgb(0 0 0 / 0.1);
        z-index: 100;
        max-height: 200px;
        overflow-y: auto;
    }
    
    .sound-option {
        width: 100%;
        padding: 8px 12px;
        background: transparent;
        border: none;
        text-align: left;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        transition: background 0.1s;
    }
    
    .sound-option:hover {
        background: #f9fafb;
    }
    
    .sound-option.selected {
        background: #eff6ff;
        color: #2563eb;
        font-weight: 500;
    }
    
    .snooze-setting {
        margin-top: 12px;
    }
    
    .snooze-button {
        display: flex;
        align-items: center;
        gap: 6px;
    }
    
    .snooze-description {
        font-size: 12px;
        color: #6b7280;
        margin: 4px 0 0 0;
    }
    
    .btn {
        padding: 8px 16px;
        border: none;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-primary {
        background: #2563eb;
        color: white;
    }
    
    .btn-primary:hover {
        background: #1d4ed8;
    }
    
    .btn-secondary {
        background: #f3f4f6;
        color: #374151;
    }
    
    .btn-secondary:hover {
        background: #e5e7eb;
    }
</style>