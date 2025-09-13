<script lang="ts">
    import { settingsOpen, settingsTab, servicesConfig, loadServicesConfig, loadOpenIncidents, loadResolvedIncidents } from '../stores/incidents';
    import { ConfigureAPIKey, GetAPIKey, UploadServicesConfig, RemoveServicesConfig, GetFilterByUser, SetFilterByUser } from '../../wailsjs/go/main/App';
    import { store } from '../../wailsjs/go/models';
    
    let apiKey = '';
    let newServiceId = '';
    let newServiceName = '';
    let errorMessage = '';
    let successMessage = '';
    let filterByUser = true; // Default to ON
    
    // Load API key and filter state when settings open
    $: if ($settingsOpen) {
        if ($settingsTab === 'api') {
            loadApiKey();
            loadFilterState();
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
            // If error or not set, default to true (ON)
            filterByUser = true;
            // Set it in backend as well
            await SetFilterByUser(true);
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
            // Reload incidents with new filter
            await loadOpenIncidents();
            await loadResolvedIncidents();
        } catch (err) {
            errorMessage = 'Failed to toggle assignment filter';
        }
    }
    
    async function addService() {
        errorMessage = '';
        successMessage = '';
        
        if (!newServiceId.trim() || !newServiceName.trim()) {
            errorMessage = 'Service ID and Name are required';
            return;
        }
        
        try {
            const config = $servicesConfig || { services: [] };
            
            // Parse multiple service IDs separated by commas
            const serviceIds = newServiceId.split(',').map(id => id.trim()).filter(id => id);
            
            // Check if any of the service IDs already exist
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
            
            // Create a single service entry with multiple IDs if needed
            const newService: store.ServiceConfig = new store.ServiceConfig({
                id: serviceIds.length === 1 ? serviceIds[0] : serviceIds,
                name: newServiceName
            });
            
            config.services.push(newService);
            
            // Upload the updated configuration
            await UploadServicesConfig(JSON.stringify(config));
            await loadServicesConfig();
            
            // Reset form
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
                const removeId = typeof serviceToRemove.id === 'string' ? serviceToRemove.id : JSON.stringify(serviceToRemove.id);
                return serviceId !== removeId;
            });
            
            if (config.services.length === 0) {
                await RemoveServicesConfig();
            } else {
                await UploadServicesConfig(JSON.stringify(config));
            }
            
            await loadServicesConfig();
            successMessage = 'Service removed successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to remove service';
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
        input.value = ''; // Reset input
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
    <div class="settings-overlay" on:click={() => settingsOpen.set(false)}></div>
    <div class="settings-panel">
        <div class="settings-header">
            <h2>Settings</h2>
            <button class="close-button" on:click={() => settingsOpen.set(false)}>
                <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                    <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
                </svg>
            </button>
        </div>
        
        <div class="tabs">
            <button 
                class="tab" 
                class:active={$settingsTab === 'api'}
                on:click={() => settingsTab.set('api')}
            >
                API Key
            </button>
            <button 
                class="tab" 
                class:active={$settingsTab === 'services'}
                on:click={() => settingsTab.set('services')}
            >
                Services
            </button>
        </div>
        
        {#if errorMessage}
            <div class="alert alert-error">{errorMessage}</div>
        {/if}
        
        {#if successMessage}
            <div class="alert alert-success">{successMessage}</div>
        {/if}
        
        {#if $settingsTab === 'api'}
            <div class="tab-content">
                <div class="form-group">
                    <label for="api-key">PagerDuty API Key</label>
                    <input 
                        id="api-key"
                        type="password" 
                        bind:value={apiKey}
                        placeholder="Enter your PagerDuty API key"
                    />
                    <button class="btn btn-primary" on:click={saveApiKey}>
                        Save API Key
                    </button>
                </div>
                
                <!-- Assignment Filter Button -->
                <div class="form-group" style="margin-top: 20px;">
                    <button 
                        class="assigned-button"
                        class:active={filterByUser}
                        on:click={toggleAssignedFilter}
                    >
                        {#if filterByUser}
                            <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor" style="margin-right: 6px;">
                                <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                            </svg>
                        {/if}
                        Assigned
                    </button>
                    <p style="margin-top: 8px; font-size: 12px; color: #6b7280;">
                        Show only incidents assigned to you
                    </p>
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
        background: transparent;
        border: none;
        padding: 4px;
        cursor: pointer;
        color: #6b7280;
        border-radius: 6px;
        transition: all 0.2s;
    }
    
    .close-button:hover {
        background: #f3f4f6;
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
    
    /* Assigned Button Styles */
    .assigned-button {
        display: inline-flex;
        align-items: center;
        padding: 8px 16px;
        background: white;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
        font-weight: 500;
        color: #6b7280;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .assigned-button:hover {
        background: #f9fafb;
        border-color: #9ca3af;
    }
    
    .assigned-button.active {
        background: #3b82f6;
        border-color: #3b82f6;
        color: white;
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
</style>