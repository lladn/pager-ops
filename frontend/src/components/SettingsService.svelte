<script lang="ts">
    import { servicesConfig, loadServicesConfig, loadOpenIncidents } from '../stores/incidents';
    import { UploadServicesConfig, RemoveServicesConfig, ToggleServiceDisabled } from '../../wailsjs/go/main/App';
    import { store } from '../../wailsjs/go/models';
    import { onMount } from 'svelte';
    
    export let errorMessage: string = '';
    export let successMessage: string = '';
    
    let newServiceId = '';
    let newServiceName = '';
    
    onMount(async () => {
        await loadServicesConfig();
    });
    
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
                name: newServiceName.trim(),
                disabled: false
            });
            
            config.services.push(newService);
            
            const updatedConfig = JSON.stringify(config, null, 2);
            await UploadServicesConfig(updatedConfig);
            await loadServicesConfig();
            
            newServiceId = '';
            newServiceName = '';
            successMessage = 'Service added successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to add service';
        }
    }
    
    async function removeService(service: store.ServiceConfig) {
        try {
            const config = $servicesConfig;
            if (!config) return;
            
            config.services = config.services.filter(s => s.id !== service.id);
            
            if (config.services.length === 0) {
                await RemoveServicesConfig();
            } else {
                const updatedConfig = JSON.stringify(config, null, 2);
                await UploadServicesConfig(updatedConfig);
            }
            
            await loadServicesConfig();
            successMessage = 'Service removed successfully';
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = 'Failed to remove service';
        }
    }
    
    async function toggleServiceDisabled(service: store.ServiceConfig) {
        try {
            await ToggleServiceDisabled(service.id);
            await loadServicesConfig();
            await loadOpenIncidents();
            
            const action = service.disabled ? 'enabled' : 'disabled';
            successMessage = `Service ${service.name} ${action} for open incidents`;
            setTimeout(() => successMessage = '', 3000);
        } catch (err) {
            errorMessage = err?.toString() || 'Failed to toggle service state';
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

<div class="service-settings">
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
                <div class="service-item" class:disabled={service.disabled}>
                    <div class="service-info">
                        <strong>{service.name}</strong>
                        <span class="service-id">{getServiceIdDisplay(service.id)}</span>
                        {#if service.disabled}
                            <span class="disabled-badge">Disabled for open incidents</span>
                        {/if}
                    </div>
                    <div class="service-actions">
                        <button 
                            class="btn-toggle" 
                            class:btn-enable={service.disabled}
                            on:click={() => toggleServiceDisabled(service)}
                            title={service.disabled ? 'Enable for open incidents' : 'Disable for open incidents'}
                        >
                            {service.disabled ? 'Enable' : 'Disable'}
                        </button>
                        <button class="btn-remove" on:click={() => removeService(service)}>
                            Remove
                        </button>
                    </div>
                </div>
            {/each}
        </div>
    {/if}
    
    <div class="form-group add-service-section">
        <h3>Add Service Manually</h3>
        <div class="add-service-inputs">
            <input 
                type="text" 
                bind:value={newServiceId}
                placeholder="Service ID (comma-separated for multiple)"
                class="service-input"
            />
            <input 
                type="text" 
                bind:value={newServiceName}
                placeholder="Service Name"
                class="service-input"
            />
            <button class="btn btn-primary" on:click={addService}>
                Add Service
            </button>
        </div>
    </div>
</div>

<style>
    .service-settings {
        padding: 20px;
    }
    
    .form-group {
        margin-bottom: 24px;
    }
    
    .form-group label {
        display: block;
        margin-bottom: 8px;
        font-size: 14px;
        font-weight: 500;
        color: #374151;
    }
    
    .add-service-section h3 {
        font-size: 14px;
        font-weight: 600;
        color: #111827;
        margin: 0 0 12px 0;
    }
    
    .add-service-inputs {
        display: flex;
        gap: 8px;
        flex-direction: column;
    }
    
    .service-input {
        padding: 10px 12px;
        border: 1px solid #d1d5db;
        border-radius: 6px;
        font-size: 14px;
    }
    
    .services-list {
        background: #f9fafb;
        border-radius: 8px;
        padding: 16px;
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
        background: white;
        border-radius: 6px;
        margin-bottom: 8px;
        border: 1px solid #e5e7eb;
        transition: opacity 0.2s;
    }
    
    .service-item.disabled {
        opacity: 0.7;
        background: #f3f4f6;
    }
    
    .service-item:last-child {
        margin-bottom: 0;
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
    
    .disabled-badge {
        font-size: 11px;
        color: #991b1b;
        background: #fee2e2;
        padding: 2px 6px;
        border-radius: 3px;
        display: inline-block;
        margin-top: 4px;
    }
    
    .service-actions {
        display: flex;
        gap: 8px;
        align-items: center;
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
    
    .btn-remove {
        padding: 6px 12px;
        background: #fee2e2;
        color: #991b1b;
        border: 1px solid #fca5a5;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-remove:hover {
        background: #fecaca;
        border-color: #f87171;
    }
    
    .btn-toggle {
        padding: 6px 12px;
        background: #fef3c7;
        color: #92400e;
        border: 1px solid #fde68a;
        border-radius: 4px;
        font-size: 12px;
        font-weight: 500;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .btn-toggle:hover {
        background: #fde68a;
        border-color: #fbbf24;
    }
    
    .btn-toggle.btn-enable {
        background: #dcfce7;
        color: #166534;
        border-color: #bbf7d0;
    }
    
    .btn-toggle.btn-enable:hover {
        background: #bbf7d0;
        border-color: #86efac;
    }
</style>