<script lang="ts">
    import { settingsOpen, settingsTab, servicesConfig, loadServicesConfig, loadOpenIncidents, loadResolvedIncidents } from '../stores/incidents';
    import { notificationConfig, availableSounds, loadNotificationConfig, loadAvailableSounds } from '../stores/notifications';
    import { 
        ConfigureAPIKey, GetAPIKey, UploadServicesConfig, RemoveServicesConfig, 
        GetFilterByUser, SetFilterByUser, SetNotificationEnabled, SetNotificationSound,
        TestNotificationSound, SnoozeNotificationSound, UnsnoozeNotificationSound,
        IsNotificationSnoozed
    } from '../../wailsjs/go/main/App';
    import { store } from '../../wailsjs/go/models';
    
    let apiKey = '';
    let newServiceId = '';
    let newServiceName = '';
    let errorMessage = '';
    let successMessage = '';
    let filterByUser = true;
    let notificationSnoozed = false;
    
    // Load data when settings open
    $: if ($settingsOpen) {
        if ($settingsTab === 'general') {
            loadApiKey();
            loadFilterState();
            loadNotificationConfig();
            loadAvailableSounds();
            checkSnoozeStatus();
        }
    }
    
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
            errorMessage = 'API key is required';
            return;
        }
        
        try {
            await ConfigureAPIKey(apiKey);
            successMessage = 'API key saved successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to save API key';
        }
    }
    
    async function toggleAssignedFilter() {
        try {
            const newState = !filterByUser;
            await SetFilterByUser(newState);
            filterByUser = newState;
            await loadOpenIncidents();
            await loadResolvedIncidents();
        } catch (err) {
            errorMessage = 'Failed to toggle assignment filter';
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
    
    async function changeNotificationSound(event: Event) {
        const select = event.target as HTMLSelectElement;
        const sound = select.value;
        
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
    
    // Services tab functions
    async function addService() {
        errorMessage = '';
        successMessage = '';
        
        if (!newServiceId.trim() || !newServiceName.trim()) {
            errorMessage = 'Service ID and Name are required';
            return;
        }
        
        try {
            const config = $servicesConfig || { services: [] };
            const serviceIds = newServiceId.split(',').map(id => id.trim()).filter(id => id);
            
            for (const serviceId of serviceIds) {
                const exists = config.services.some((s: store.ServiceConfig) => {
                    if (typeof s.id === 'string') {
                        return s.id === serviceId;
                    } else if (Array.isArray(s.id)) {
                        return s.id.includes(serviceId);
                    }
                    return false;
                });
                
                if (exists) {
                    errorMessage = `Service with ID ${serviceId} already exists`;
                    return;
                }
            }
            
            const newService: store.ServiceConfig = new store.ServiceConfig({
                id: serviceIds.length === 1 ? serviceIds[0] : serviceIds,
                name: newServiceName
            });
            
            config.services.push(newService);
            await UploadServicesConfig(JSON.stringify(config));
            await loadServicesConfig();
            
            newServiceId = '';
            newServiceName = '';
            successMessage = 'Service added successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to add service';
        }
    }
    
    async function removeService(serviceToRemove: store.ServiceConfig) {
        try {
            const config = $servicesConfig;
            if (!config) return;
            
            config.services = config.services.filter((s: store.ServiceConfig) => {
                const serviceId = typeof s.id === 'string' ? s.id : JSON.stringify(s.id);
                const removeId = typeof serviceToRemove.id === 'string' ? 
                    serviceToRemove.id : JSON.stringify(serviceToRemove.id);
                return serviceId !== removeId;
            });
            
            await UploadServicesConfig(JSON.stringify(config));
            await loadServicesConfig();
            successMessage = 'Service removed successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to remove service';
        }
    }
    
    async function removeAllServices() {
        try {
            await RemoveServicesConfig();
            await loadServicesConfig();
            successMessage = 'All services removed successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to remove services';
        }
    }
    
    async function handleFileUpload(event: Event) {
        const input = event.target as HTMLInputElement;
        const file = input.files?.[0];
        
        if (!file) return;
        
        const reader = new FileReader();
        reader.onload = async (e) => {
            try {
                const content = e.target?.result as string;
                await UploadServicesConfig(content);
                await loadServicesConfig();
                successMessage = 'Services configuration uploaded successfully';
                setTimeout(() => successMessage = '', 3000);
            } catch (err) {
                errorMessage = err?.toString() || 'Failed to upload configuration';
            }
        };
        
        reader.readAsText(file);
        input.value = '';
    }
    
    function getServiceIdDisplay(id: string | string[] | undefined): string {
        if (!id) return '';
        if (typeof id === 'string') return id;
        if (Array.isArray(id)) return id.join(', ');
        return '';
    }
</script>

{#if $settingsOpen}
    <!-- svelte-ignore a11y-click-events-have-key-events -->
    <!-- svelte-ignore a11y-no-static-element-interactions -->
    <div class="settings-overlay" on:click={() => settingsOpen.set(false)}></div>
    <div class="settings-panel">
        <div class="settings-header">
            <h2>Settings</h2>
            <button class="close-button" on:click={() => settingsOpen.set(false)}>
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                    <line x1="18" y1="6" x2="6" y2="18"></line>
                    <line x1="6" y1="6" x2="18" y2="18"></line>
                </svg>
            </button>
        </div>
        
        <div class="tabs">
            <button 
                class="tab" 
                class:active={$settingsTab === 'general'}
                on:click={() => settingsTab.set('general')}
            >
                General
            </button>
            <button 
                class="tab" 
                class:active={$settingsTab === 'services'}
                on:click={() => settingsTab.set('services')}
            >
                Service Management
            </button>
        </div>
        
        {#if errorMessage}
            <div class="alert alert-error">{errorMessage}</div>
        {/if}
        
        {#if successMessage}
            <div class="alert alert-success">{successMessage}</div>
        {/if}
        
        {#if $settingsTab === 'general'}
            <div class="tab-content">
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
                            <div class="sound-selector">
                                <label for="notification-sound">Notification Sound</label>
                                <div class="sound-controls">
                                    <select 
                                        id="notification-sound"
                                        value={$notificationConfig.sound}
                                        on:change={changeNotificationSound}
                                        class="sound-dropdown"
                                    >
                                        {#each $availableSounds as sound}
                                            <option value={sound}>
                                                {sound === 'default' ? 'Default (Say Service Name)' : sound}
                                            </option>
                                        {/each}
                                    </select>
                                    <button class="btn-test" on:click={testSound}>
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
                                        Temporarily mute sound playback (desktop notifications will still appear)
                                    {/if}
                                </p>
                            </div>
                        </div>
                    {/if}
                </div>
            </div>
        {:else if $settingsTab === 'services'}
            <!-- ... existing services tab content ... -->
            <div class="tab-content">
                <div class="form-group">
                    <!-- svelte-ignore a11y-label-has-associated-control -->
                    <label>Upload Services Configuration</label>
                    <input 
                        type="file" 
                        accept=".json"
                        on:change={handleFileUpload}
                        class="file-input"
                    />
                    <p class="help-text">Upload a JSON file with your services configuration</p>
                </div>
                
                {#if $servicesConfig && $servicesConfig.services.length > 0}
                    <div class="services-list">
                        <h3>Configured Services</h3>
                        {#each $servicesConfig.services as service}
                            <div class="service-item">
                                <div class="service-info">
                                    <strong>{service.name}</strong>
                                    <span class="service-id">{getServiceIdDisplay(service.id)}</span>
                                </div>
                                <button class="btn-remove" on:click={() => removeService(service)}>
                                    Remove
                                </button>
                            </div>
                        {/each}
                    </div>
                {/if}
                
                <div class="form-group">
                    <h3>Add Service Manually</h3>
                    <input 
                        type="text" 
                        bind:value={newServiceId}
                        placeholder="Service ID (comma-separated for multiple)"
                    />
                    <input 
                        type="text" 
                        bind:value={newServiceName}
                        placeholder="Service Name"
                    />
                    <button class="btn btn-primary" on:click={addService}>
                        Add Service
                    </button>
                </div>
            </div>
        {/if}
    </div>
{/if}

<style>
    /* Keep all your existing styles exactly as they are */
    .settings-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        z-index: 999;
    }
    
    .settings-panel {
        position: fixed;
        top: 50%;
        left: 50%;
        transform: translate(-50%, -50%);
        background: white;
        border-radius: 12px;
        width: 90%;
        max-width: 500px;
        max-height: 80vh;
        overflow-y: auto;
        z-index: 1000;
        box-shadow: 0 20px 25px -5px rgb(0 0 0 / 0.1);
    }
    
    .settings-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 20px;
        border-bottom: 1px solid #e5e7eb;
    }
    
    .settings-header h2 {
        margin: 0;
        font-size: 20px;
        color: #111827;
    }
    
    .close-button {
        position: absolute;
        top: 20px;
        right: 20px;
        width: 20px;  
        height: 20px; 
        border: none;
        background: #f3f4f6;
        border-radius: 8px;
        cursor: pointer;
        display: flex;
        align-items: center;
        justify-content: center;
        transition: all 0.2s;
        color: #6b7280;
    }
    
    .close-button:hover {
        background: #e5e7eb;
        color: #111827;
    }
    
    .tabs {
        display: flex;
        border-bottom: 1px solid #e5e7eb;
    }
    
    .tab {
        flex: 1;
        padding: 12px;
        background: transparent;
        border: none;
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        color: #6b7280;
        border-bottom: 2px solid transparent;
        transition: all 0.2s;
    }
    
    .tab:hover {
        color: #374151;
    }
    
    .tab.active {
        color: #3b82f6;
        border-bottom-color: #3b82f6;
    }
    
    .tab-content {
        padding: 20px;
    }
    
    .form-group {
        margin-bottom: 20px;
    }
    
    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-size: 14px;
        font-weight: 500;
        color: #374151;
    }
    
    .form-group input[type="text"],
    .form-group input[type="password"] {
        width: 100%;
        padding: 8px 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        margin-bottom: 12px;
    }
    
    .form-group input:focus {
        outline: none;
        border-color: #3b82f6;
        box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
    }
    
    
    .btn {
        padding: 10px 20px;
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
    
    .btn-remove {
        padding: 4px 12px;
        background: #ef4444;
        color: white;
        border: none;
        border-radius: 4px;
        font-size: 12px;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-remove:hover {
        background: #dc2626;
    }
    
    .alert {
        padding: 12px;
        border-radius: 6px;
        margin: 16px 20px;
        font-size: 14px;
    }
    
    .alert-error {
        background: #fee2e2;
        color: #991b1b;
        border: 1px solid #fecaca;
    }
    
    .alert-success {
        background: #dcfce7;
        color: #166534;
        border: 1px solid #bbf7d0;
    }
    
    .services-list {
        margin-bottom: 24px;
    }
    
    .services-list h3 {
        font-size: 16px;
        font-weight: 600;
        color: #111827;
        margin: 0 0 12px 0;
    }
    
    .service-item {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px;
        background: #f9fafb;
        border-radius: 6px;
        margin-bottom: 8px;
    }
    
    .service-info {
        display: flex;
        flex-direction: column;
        gap: 4px;
    }
    
    .service-info strong {
        font-size: 14px;
        color: #111827;
    }
    
    .service-id {
        font-size: 12px;
        color: #6b7280;
        font-family: monospace;
    }
    
    .file-input {
        width: 100%;
        padding: 8px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        margin-bottom: 8px;
    }
    
    .help-text {
        font-size: 12px;
        color: #6b7280;
        margin: 4px 0 0 0;
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
        margin-top: 20px;
        padding-top: 20px;
        border-top: 1px solid #e5e7eb;
    }
    
    .sound-selector {
        margin-bottom: 20px;
    }
    
    .sound-selector label {
        display: block;
        font-size: 13px;
        font-weight: 500;
        color: #374151;
        margin-bottom: 8px;
    }
    
    .sound-controls {
        display: flex;
        gap: 8px;
    }
    
    .sound-dropdown {
        flex: 1;
        padding: 8px 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        background: white;
        cursor: pointer;
    }
    
    .btn-test {
        padding: 8px 16px;
        background: #f3f4f6;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-test:hover {
        background: #e5e7eb;
    }
    
    .snooze-controls {
        margin-top: 16px;
    }
    
    .btn-snooze {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 10px 16px;
        background: white;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
        width: 100%;
    }
    
    .btn-snooze:hover {
        background: #f9fafb;
        border-color: #9ca3af;
    }
    
    .btn-snooze.snoozed {
        background: #fef3c7;
        border-color: #fbbf24;
        color: #92400e;
    }
    
    .snooze-description {
        font-size: 12px;
        color: #6b7280;
        margin: 8px 0 0 0;
    }
</style>