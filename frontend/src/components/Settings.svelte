<script lang="ts">
    import { settingsOpen, settingsTab, servicesConfig, loadServicesConfig } from '../stores/incidents';
    import { ConfigureAPIKey, GetAPIKey, UploadServicesConfig, RemoveServicesConfig } from '../../wailsjs/go/main/App';
    
    let apiKey = '';
    let newServiceId = '';
    let newServiceName = '';
    let errorMessage = '';
    let successMessage = '';
    
    // Load API key when settings open
    $: if ($settingsOpen) {
        loadApiKey();
    }
    
    async function loadApiKey() {
        try {
            const key = await GetAPIKey();
            apiKey = key || '';
        } catch (err) {
            apiKey = '';
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
    
    async function addService() {
        errorMessage = '';
        successMessage = '';
        
        if (!newServiceId.trim() || !newServiceName.trim()) {
            errorMessage = 'Service ID and Name are required';
            return;
        }
        
        try {
            const config = $servicesConfig || { services: [] };
            const newService = { id: newServiceId, name: newServiceName };
            
            // Check if service already exists
            const exists = config.services.some(s => {
                const id = typeof s.id === 'string' ? s.id : String(s.id);
                return id === newServiceId;
            });
            
            if (exists) {
                errorMessage = 'Service with this ID already exists';
                return;
            }
            
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
    
    async function removeService(serviceId: string) {
        try {
            if (!$servicesConfig) return;
            
            const config = {
                services: $servicesConfig.services.filter(s => {
                    const id = typeof s.id === 'string' ? s.id : String(s.id);
                    return id !== serviceId;
                })
            };
            
            if (config.services.length === 0) {
                await RemoveServicesConfig();
            } else {
                await UploadServicesConfig(JSON.stringify(config));
            }
            
            await loadServicesConfig();
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to remove service';
        }
    }
    
    async function handleFileUpload(event: Event) {
        const target = event.target as HTMLInputElement;
        const file = target.files?.[0];
        
        if (!file) return;
        
        errorMessage = '';
        successMessage = '';
        
        try {
            const content = await file.text();
            await UploadServicesConfig(content);
            await loadServicesConfig();
            successMessage = 'Services configuration uploaded successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to upload configuration';
        }
        
        target.value = '';
    }
    
    function closeSettings() {
        settingsOpen.set(false);
        errorMessage = '';
        successMessage = '';
    }
</script>

{#if $settingsOpen}
    <div class="modal-overlay" on:click={closeSettings}>
        <div class="modal-content" on:click|stopPropagation>
            <div class="modal-header">
                <h2 class="modal-title">Settings</h2>
                <button class="close-button" on:click={closeSettings}>
                    <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                        <path fill-rule="evenodd" d="M4.293 4.293a1 1 0 011.414 0L10 8.586l4.293-4.293a1 1 0 111.414 1.414L11.414 10l4.293 4.293a1 1 0 01-1.414 1.414L10 11.414l-4.293 4.293a1 1 0 01-1.414-1.414L8.586 10 4.293 5.707a1 1 0 010-1.414z" clip-rule="evenodd" />
                    </svg>
                </button>
            </div>
            
            <p class="modal-subtitle">Configure your PagerDuty settings</p>
            
            <div class="tabs">
                <button 
                    class="tab" 
                    class:active={$settingsTab === 'api'}
                    on:click={() => settingsTab.set('api')}
                >
                    API Configuration
                </button>
                <button 
                    class="tab" 
                    class:active={$settingsTab === 'services'}
                    on:click={() => settingsTab.set('services')}
                >
                    Service Management
                    {#if $servicesConfig}
                        <span class="preview-badge">Preview</span>
                    {/if}
                </button>
            </div>
            
            {#if errorMessage}
                <div class="alert alert-error">{errorMessage}</div>
            {/if}
            
            {#if successMessage}
                <div class="alert alert-success">{successMessage}</div>
            {/if}
            
            <div class="tab-content">
                {#if $settingsTab === 'api'}
                    <div class="form-group">
                        <label for="api-key">API Key</label>
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
                {:else if $settingsTab === 'services'}
                    <div class="services-section">
                        <h3>Add New Service</h3>
                        <div class="service-form">
                            <input 
                                type="text" 
                                bind:value={newServiceId}
                                placeholder="e.g., PDBSVC001"
                                class="service-input"
                            />
                            <input 
                                type="text" 
                                bind:value={newServiceName}
                                placeholder="e.g., Database Service"
                                class="service-input"
                            />
                            <button class="btn btn-add" on:click={addService}>
                                + Add Service
                            </button>
                        </div>
                        
                        <div class="upload-section">
                            <label for="file-upload" class="upload-label">
                                Or upload a JSON configuration file
                            </label>
                            <input 
                                id="file-upload"
                                type="file" 
                                accept=".json"
                                on:change={handleFileUpload}
                                class="file-input"
                            />
                        </div>
                        
                        <h3>Existing Services</h3>
                        <div class="services-list">
                            {#if $servicesConfig && $servicesConfig.services.length > 0}
                                {#each $servicesConfig.services as service}
                                    {@const serviceId = typeof service.id === 'string' ? service.id : String(service.id)}
                                    <div class="service-item">
                                        <div class="service-info">
                                            <span class="service-name">{service.name}</span>
                                            <span class="service-badge">active</span>
                                        </div>
                                        <span class="service-id">ID: {serviceId}</span>
                                        <button 
                                            class="delete-button"
                                            on:click={() => removeService(serviceId)}
                                            title="Remove service"
                                        >
                                            <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                                                <path fill-rule="evenodd" d="M9 2a1 1 0 00-.894.553L7.382 4H4a1 1 0 000 2v10a2 2 0 002 2h8a2 2 0 002-2V6a1 1 0 100-2h-3.382l-.724-1.447A1 1 0 0011 2H9zM7 8a1 1 0 012 0v6a1 1 0 11-2 0V8zm5-1a1 1 0 00-1 1v6a1 1 0 102 0V8a1 1 0 00-1-1z" clip-rule="evenodd" />
                                            </svg>
                                        </button>
                                    </div>
                                {/each}
                            {:else}
                                <div class="empty-state">No services configured</div>
                            {/if}
                        </div>
                    </div>
                {/if}
            </div>
        </div>
    </div>
{/if}

<style>
    .modal-overlay {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 100;
    }
    
    .modal-content {
        background: white;
        border-radius: 12px;
        width: 90%;
        max-width: 600px;
        max-height: 80vh;
        overflow-y: auto;
        padding: 24px;
    }
    
    .modal-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 8px;
    }
    
    .modal-title {
        font-size: 24px;
        font-weight: 600;
        color: #111827;
        margin: 0;
    }
    
    .modal-subtitle {
        color: #6b7280;
        margin-bottom: 24px;
    }
    
    .close-button {
        background: transparent;
        border: none;
        cursor: pointer;
        padding: 4px;
        color: #6b7280;
        transition: color 0.2s;
    }
    
    .close-button:hover {
        color: #111827;
    }
    
    .tabs {
        display: flex;
        gap: 24px;
        border-bottom: 1px solid #e5e7eb;
        margin-bottom: 24px;
    }
    
    .tab {
        position: relative;
        background: transparent;
        border: none;
        padding: 8px 0;
        font-size: 16px;
        font-weight: 500;
        color: #6b7280;
        cursor: pointer;
        transition: color 0.2s;
        display: flex;
        align-items: center;
        gap: 8px;
    }
    
    .tab:hover {
        color: #374151;
    }
    
    .tab.active {
        color: #111827;
    }
    
    .tab.active::after {
        content: '';
        position: absolute;
        bottom: -1px;
        left: 0;
        right: 0;
        height: 2px;
        background: #3b82f6;
    }
    
    .preview-badge {
        background: #6b7280;
        color: white;
        padding: 2px 6px;
        border-radius: 4px;
        font-size: 11px;
        font-weight: 600;
        text-transform: uppercase;
    }
    
    .alert {
        padding: 12px;
        border-radius: 8px;
        margin-bottom: 16px;
        font-size: 14px;
    }
    
    .alert-error {
        background: #fee;
        color: #dc2626;
        border: 1px solid #fecaca;
    }
    
    .alert-success {
        background: #f0fdf4;
        color: #16a34a;
        border: 1px solid #bbf7d0;
    }
    
    .form-group {
        display: flex;
        flex-direction: column;
        gap: 12px;
    }
    
    label {
        font-size: 14px;
        font-weight: 500;
        color: #374151;
    }
    
    input[type="text"],
    input[type="password"] {
        padding: 8px 12px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
    }
    
    input[type="text"]:focus,
    input[type="password"]:focus {
        outline: none;
        border-color: #3b82f6;
        box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
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
    
    .btn-add {
        background: #6b7280;
        color: white;
        display: flex;
        align-items: center;
        gap: 4px;
    }
    
    .btn-add:hover {
        background: #4b5563;
    }
    
    .services-section h3 {
        font-size: 16px;
        font-weight: 600;
        color: #111827;
        margin: 24px 0 12px 0;
    }
    
    .services-section h3:first-child {
        margin-top: 0;
    }
    
    .service-form {
        display: grid;
        grid-template-columns: 1fr 1fr auto;
        gap: 8px;
        margin-bottom: 16px;
    }
    
    .service-input {
        padding: 8px 12px;
        border: 1px solid #e5e7eb;
        border-radius: 6px;
        font-size: 14px;
    }
    
    .upload-section {
        margin: 16px 0;
        padding: 16px 0;
        border-top: 1px solid #e5e7eb;
        border-bottom: 1px solid #e5e7eb;
    }
    
    .upload-label {
        display: block;
        margin-bottom: 8px;
    }
    
    .file-input {
        font-size: 14px;
    }
    
    .services-list {
        display: flex;
        flex-direction: column;
        gap: 8px;
    }
    
    .service-item {
        display: flex;
        align-items: center;
        justify-content: space-between;
        padding: 12px;
        background: #f9fafb;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
    }
    
    .service-info {
        display: flex;
        align-items: center;
        gap: 8px;
        flex: 1;
    }
    
    .service-name {
        font-weight: 500;
        color: #111827;
    }
    
    .service-badge {
        background: #111827;
        color: white;
        padding: 2px 8px;
        border-radius: 12px;
        font-size: 12px;
        font-weight: 500;
    }
    
    .service-id {
        color: #6b7280;
        font-size: 13px;
        margin-right: 12px;
    }
    
    .delete-button {
        background: transparent;
        border: none;
        cursor: pointer;
        padding: 4px;
        color: #ef4444;
        transition: opacity 0.2s;
    }
    
    .delete-button:hover {
        opacity: 0.7;
    }
    
    .empty-state {
        text-align: center;
        padding: 24px;
        color: #6b7280;
    }
</style>