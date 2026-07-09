<script lang="ts">
    import { 
        notificationConfig, 
        availableSounds, 
        loadNotificationConfig, 
        loadAvailableSounds } from '../stores/notifications';
    import { assignedFilterEnabled, staleThresholdMinutes, STALE_THRESHOLD_OPTIONS } from '../stores/incidents';
    import { theme, setTheme } from '../stores/theme';
    import { colorTheme, setColorTheme, ACCENT_COLOR_OPTIONS, FULL_THEME_OPTIONS, type ColorThemeName } from '../stores/colorTheme';
    import {
        ConfigureAPIKey, GetAPIKey,
        GetFilterByUser, SetFilterByUser,
        SetNotificationEnabled, SetNotificationSound, TestNotificationSound,
        SnoozeNotificationSound, UnsnoozeNotificationSound, IsNotificationSnoozed,
        SetBrowserRedirect, GetBrowserRedirect, SetTheme, SetColorTheme
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
    
    async function selectTheme(value: 'light' | 'dark') {
        setTheme(value);
        try {
            await SetTheme(value);
        } catch (err) {
            console.error('Failed to persist theme setting:', err);
        }
    }

    async function selectColorTheme(value: ColorThemeName) {
        setColorTheme(value);
        try {
            await SetColorTheme(value);
        } catch (err) {
            console.error('Failed to persist color theme setting:', err);
        }
    }
    
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

    function staleOptionLabel(minutes: number): string {
        return minutes >= 60 ? `${minutes / 60} hour${minutes >= 120 ? 's' : ''}` : `${minutes} minutes`;
    }

    function setStaleThreshold(event: Event) {
        const value = parseInt((event.currentTarget as HTMLSelectElement).value, 10);
        if (!isNaN(value)) {
            staleThresholdMinutes.set(value);
        }
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
    <!-- Appearance Section -->
    <div class="settings-section">
        <h3>Appearance</h3>
        <p class="setting-description">Choose how PagerOps looks</p>
        <div class="theme-toggle-group">
            <button
                class="theme-option"
                class:active={$theme === 'light'}
                on:click={() => selectTheme('light')}
                type="button"
            >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <circle cx="12" cy="12" r="5"></circle>
                    <line x1="12" y1="1" x2="12" y2="3"></line>
                    <line x1="12" y1="21" x2="12" y2="23"></line>
                    <line x1="4.22" y1="4.22" x2="5.64" y2="5.64"></line>
                    <line x1="18.36" y1="18.36" x2="19.78" y2="19.78"></line>
                    <line x1="1" y1="12" x2="3" y2="12"></line>
                    <line x1="21" y1="12" x2="23" y2="12"></line>
                    <line x1="4.22" y1="19.78" x2="5.64" y2="18.36"></line>
                    <line x1="18.36" y1="5.64" x2="19.78" y2="4.22"></line>
                </svg>
                Light
            </button>
            <button
                class="theme-option"
                class:active={$theme === 'dark'}
                on:click={() => selectTheme('dark')}
                type="button"
            >
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                    <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"></path>
                </svg>
                Dark
            </button>
        </div>

        <p class="setting-description color-theme-label">Accent color</p>
        <div class="color-theme-group">
            {#each ACCENT_COLOR_OPTIONS as option}
                <button
                    class="color-swatch"
                    class:active={$colorTheme === option.value}
                    style="--swatch-color: {option.swatch}"
                    on:click={() => selectColorTheme(option.value)}
                    type="button"
                    title={option.label}
                    aria-label={option.label}
                ></button>
            {/each}
        </div>
    </div>

    <!-- Full App Themes Section -->
    <div class="settings-section">
        <h3>Theme</h3>
        <p class="setting-description">Give the whole app a distinct look &mdash; each theme adapts to Light/Dark above</p>
        <div class="full-theme-grid">
            {#each FULL_THEME_OPTIONS as option}
                <button
                    class="full-theme-option"
                    class:active={$colorTheme === option.value}
                    on:click={() => selectColorTheme(option.value)}
                    type="button"
                >
                    <span class="full-theme-swatch" style="--swatch-color: {option.swatch}"></span>
                    <span class="full-theme-label">{option.label}</span>
                </button>
            {/each}
        </div>
    </div>

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

    <!-- Stale Incident Highlight -->
    <div class="settings-section">
        <h3>Stale Incident Highlight</h3>
        <p class="setting-description">Highlight an unresolved incident card in red once it has been open this long</p>
        <select class="settings-select" value={$staleThresholdMinutes} on:change={setStaleThreshold}>
            {#each STALE_THRESHOLD_OPTIONS as minutes}
                <option value={minutes}>{staleOptionLabel(minutes)}</option>
            {/each}
        </select>
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
    /* Prevent accidental text highlight */
        -webkit-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .settings-section {
        padding: 20px;
        border-bottom: 1px solid var(--border);
    }
    
    .settings-section:last-child {
        border-bottom: none;
    }
    
    .settings-section h3 {
        font-size: 14px;
        font-weight: 600;
        color: var(--text-primary);
        margin: 0 0 8px 0;
    }
    
    .settings-section h4 {
        font-size: 13px;
        font-weight: 600;
        color: var(--text-secondary);
        margin: 0 0 4px 0;
    }

    .theme-toggle-group {
        display: flex;
        gap: 8px;
    }

    .theme-option {
        flex: 1;
        display: flex;
        align-items: center;
        justify-content: center;
        gap: 8px;
        padding: 10px 12px;
        border: 1px solid var(--border-strong);
        border-radius: 8px;
        background: var(--bg-input);
        color: var(--text-secondary);
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }

    .theme-option:hover {
        border-color: var(--text-muted);
    }

    .theme-option.active {
        background: var(--accent-soft);
        border-color: var(--accent);
        color: var(--accent);
    }

    .color-theme-label {
        margin-top: 16px;
    }

    .color-theme-group {
        display: flex;
        gap: 10px;
    }

    .color-swatch {
        width: 28px;
        height: 28px;
        border-radius: 50%;
        border: 2px solid transparent;
        background: var(--swatch-color);
        cursor: pointer;
        padding: 0;
        box-shadow: inset 0 0 0 1px var(--border);
        transition: all 0.2s;
    }

    .color-swatch:hover {
        transform: scale(1.08);
    }

    .color-swatch.active {
        border-color: var(--accent);
        box-shadow: 0 0 0 2px var(--bg-primary), 0 0 0 4px var(--accent);
    }

    .full-theme-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 8px;
    }

    .full-theme-option {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 10px;
        border: 1px solid var(--border-strong);
        border-radius: 8px;
        background: var(--bg-input);
        cursor: pointer;
        transition: all 0.2s;
        overflow: hidden;
    }

    .full-theme-option:hover {
        border-color: var(--text-muted);
    }

    .full-theme-option.active {
        background: var(--accent-soft);
        border-color: var(--accent);
    }

    .full-theme-swatch {
        flex-shrink: 0;
        width: 20px;
        height: 20px;
        border-radius: 50%;
        background: var(--swatch-color);
        box-shadow: inset 0 0 0 1px rgba(0, 0, 0, 0.15);
    }

    .full-theme-label {
        font-size: 12.5px;
        font-weight: 500;
        color: var(--text-secondary);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
    }

    .full-theme-option.active .full-theme-label {
        color: var(--accent);
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
        border: 1px solid var(--border-strong);
        border-radius: 6px;
        font-size: 14px;
        background: var(--bg-input);
        color: var(--text-primary);
    }

    .settings-select {
        width: 100%;
        padding: 10px 12px;
        border: 1px solid var(--border-strong);
        border-radius: 6px;
        font-size: 14px;
        color: var(--text-secondary);
        background: var(--bg-input);
        cursor: pointer;
    }

    .settings-select:hover {
        border-color: var(--text-muted);
    }
    
    .toggle-setting {
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .setting-description {
        font-size: 12px;
        color: var(--text-tertiary);
        margin: 4px 0 0 0;
        margin-bottom: 7px;
    }
    
    .toggle-button {
        position: relative;
        width: 48px;
        height: 24px;
        background: var(--border-strong);
        border: none;
        border-radius: 12px;
        cursor: pointer;
        transition: background 0.2s;
    }
    
    .toggle-button.active {
        background: var(--accent);
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
        border-top: 1px solid var(--border-soft);
    }
    
    .browser-redirect-setting {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid var(--border-soft);
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
        border: 1px solid var(--border-strong);
        border-radius: 6px;
        background: var(--bg-input);
        cursor: pointer;
        display: flex;
        justify-content: space-between;
        align-items: center;
        font-size: 14px;
        color: var(--text-secondary);
    }
    
    .sound-selector:hover {
        border-color: var(--text-muted);
    }
    
    .sound-dropdown {
        position: absolute;
        top: calc(100% + 4px);
        left: 0;
        right: 0;
        background: var(--bg-elevated);
        border: 1px solid var(--border-strong);
        border-radius: 6px;
        box-shadow: var(--shadow-md);
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
        color: var(--text-secondary);
        transition: background 0.1s;
    }
    
    .sound-option:hover {
        background: var(--bg-hover);
    }
    
    .sound-option.selected {
        background: var(--accent-soft);
        color: var(--accent);
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
        color: var(--text-tertiary);
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
        background: var(--accent);
        color: var(--text-on-accent);
    }
    
    .btn-primary:hover {
        background: var(--accent-hover);
    }
    
    .btn-secondary {
        background: var(--bg-tertiary);
        color: var(--text-secondary);
    }
    
    .btn-secondary:hover {
        background: var(--bg-active);
    }
</style>