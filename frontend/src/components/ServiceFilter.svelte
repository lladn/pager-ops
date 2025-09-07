<script lang="ts">
    import { servicesConfig, selectedServices, loadOpenIncidents, loadResolvedIncidents } from '../stores/incidents';
    import { SetSelectedServices } from '../../wailsjs/go/main/App';
    import { store } from '../../wailsjs/go/models';
    
    let isOpen = false;
    let filterText = 'All Services';
    
    // Local state for immediate UI updates
    let localSelectedServices: string[] = [];
    
    // Sync local state with store
    $: localSelectedServices = [...$selectedServices];
    
    $: if ($servicesConfig) {
        updateFilterText(localSelectedServices);
    }
    
    function updateFilterText(selected: string[]) {
        if (!$servicesConfig) {
            filterText = 'All Services';
            return;
        }
        
        const allServiceIds = getAllServiceIds();
        
        if (selected.length === 0 || selected.length === allServiceIds.length) {
            filterText = 'All Services';
        } else {
            // Count how many service groups are selected
            let selectedGroups = 0;
            for (const service of $servicesConfig.services) {
                if (isServiceGroupSelected(service, selected)) {
                    selectedGroups++;
                }
            }
            
            if (selectedGroups === 1) {
                // Find the selected service group
                const selectedService = $servicesConfig.services.find(s => isServiceGroupSelected(s, selected));
                filterText = selectedService?.name || 'Unknown Service';
            } else {
                filterText = `${selectedGroups} Services`;
            }
        }
    }
    
    function toggleDropdown() {
        isOpen = !isOpen;
    }
    
    function closeDropdown() {
        isOpen = false;
    }
    
    function getAllServiceIds(): string[] {
        if (!$servicesConfig) return [];
        
        const allIds: string[] = [];
        for (const service of $servicesConfig.services) {
            if (typeof service.id === 'string') {
                allIds.push(service.id);
            } else if (Array.isArray(service.id)) {
                allIds.push(...service.id);
            }
        }
        return allIds;
    }
    
    async function selectAllServices() {
        if (!$servicesConfig) return;
        
        const allServiceIds = getAllServiceIds();
        
        // Update local state immediately for UI responsiveness
        localSelectedServices = [...allServiceIds];
        
        // Update store and backend
        selectedServices.set(allServiceIds);
        await SetSelectedServices(allServiceIds);
        await loadOpenIncidents();
        await loadResolvedIncidents();
    }
    
    async function toggleServiceGroup(service: store.ServiceConfig) {
        const serviceIds = typeof service.id === 'string' ? [service.id] : 
                          Array.isArray(service.id) ? service.id : [String(service.id)];
        
        const isCurrentlySelected = isServiceGroupSelected(service, localSelectedServices);
        
        // Update local state immediately for UI responsiveness
        if (isCurrentlySelected) {
            // Remove all IDs of this service group
            localSelectedServices = localSelectedServices.filter(id => !serviceIds.includes(id));
        } else {
            // Add all IDs of this service group
            localSelectedServices = [...new Set([...localSelectedServices, ...serviceIds])];
        }
        
        // Update store and backend
        selectedServices.set(localSelectedServices);
        await SetSelectedServices(localSelectedServices);
        
        await loadOpenIncidents();
        await loadResolvedIncidents();
    }
    
    function isServiceGroupSelected(service: store.ServiceConfig, selected?: string[]): boolean {
        const checkSelected = selected || localSelectedServices;
        const serviceIds = typeof service.id === 'string' ? [service.id] : 
                          Array.isArray(service.id) ? service.id : [String(service.id)];
        
        // Check if all IDs in this service group are selected
        return serviceIds.every(id => checkSelected.includes(id));
    }
    
    function isServiceGroupPartiallySelected(service: store.ServiceConfig, selected?: string[]): boolean {
        const checkSelected = selected || localSelectedServices;
        const serviceIds = typeof service.id === 'string' ? [service.id] : 
                          Array.isArray(service.id) ? service.id : [String(service.id)];
        
        // Check if some (but not all) IDs in this service group are selected
        const selectedCount = serviceIds.filter(id => checkSelected.includes(id)).length;
        return selectedCount > 0 && selectedCount < serviceIds.length;
    }
    
    // Close dropdown when clicking outside
    function handleClickOutside(event: MouseEvent) {
        const target = event.target as HTMLElement;
        if (!target.closest('.service-filter')) {
            closeDropdown();
        }
    }
</script>

<svelte:window on:click={handleClickOutside} />

<div class="service-filter">
    <button class="filter-button" on:click={toggleDropdown}>
        <span class="filter-text">{filterText}</span>
        <svg class="filter-arrow" class:rotate={isOpen} width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
        </svg>
    </button>
    
    {#if isOpen}
        <div class="dropdown-menu">
            <button class="dropdown-item" on:click={selectAllServices}>
                <div class="checkbox">
                    {#if localSelectedServices.length === 0 || localSelectedServices.length === getAllServiceIds().length}
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                            <rect x="0.5" y="0.5" width="15" height="15" rx="2" fill="#4F46E5" stroke="#4F46E5"/>
                            <path d="M4 8L6.5 10.5L12 5" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                        </svg>
                    {:else}
                        <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                            <rect x="0.5" y="0.5" width="15" height="15" rx="2" stroke="#D1D5DB"/>
                        </svg>
                    {/if}
                </div>
                All Services
            </button>
            
            {#if $servicesConfig}
                {#each $servicesConfig.services as service}
                    <button class="dropdown-item" on:click={() => toggleServiceGroup(service)}>
                        <div class="checkbox">
                            {#if isServiceGroupSelected(service, localSelectedServices)}
                                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                                    <rect x="0.5" y="0.5" width="15" height="15" rx="2" fill="#4F46E5" stroke="#4F46E5"/>
                                    <path d="M4 8L6.5 10.5L12 5" stroke="white" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"/>
                                </svg>
                            {:else if isServiceGroupPartiallySelected(service, localSelectedServices)}
                                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                                    <rect x="0.5" y="0.5" width="15" height="15" rx="2" fill="#4F46E5" stroke="#4F46E5"/>
                                    <rect x="4" y="7" width="8" height="2" fill="white"/>
                                </svg>
                            {:else}
                                <svg width="16" height="16" viewBox="0 0 16 16" fill="none">
                                    <rect x="0.5" y="0.5" width="15" height="15" rx="2" stroke="#D1D5DB"/>
                                </svg>
                            {/if}
                        </div>
                        {service.name}
                        {#if Array.isArray(service.id)}
                            <span class="service-count">({service.id.length})</span>
                        {/if}
                    </button>
                {/each}
            {:else}
                <div class="dropdown-empty">No services configured</div>
            {/if}
        </div>
    {/if}
</div>

<style>
    .service-filter {
        position: relative;
    }
    
    .filter-button {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 16px;
        background: #f3f4f6;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        transition: all 0.2s ease;
    }
    
    .filter-button:hover {
        background: #e5e7eb;
    }
    
    .filter-text {
        font-weight: 500;
    }
    
    .filter-arrow {
        transition: transform 0.2s ease;
    }
    
    .filter-arrow.rotate {
        transform: rotate(180deg);
    }
    
    .dropdown-menu {
        position: absolute;
        top: calc(100% + 4px);
        right: 0;
        min-width: 200px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        box-shadow: 0 10px 15px -3px rgb(0 0 0 / 0.1);
        z-index: 50;
        padding: 4px;
    }
    
    .dropdown-item {
        display: flex;
        align-items: center;
        gap: 8px;
        width: 100%;
        padding: 8px 12px;
        background: transparent;
        border: none;
        border-radius: 4px;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        text-align: left;
        transition: background 0.2s ease;
    }
    
    .dropdown-item:hover {
        background: #f3f4f6;
    }
    
    .checkbox {
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    
    .service-count {
        margin-left: auto;
        font-size: 12px;
        color: #6b7280;
    }
    
    .dropdown-empty {
        padding: 12px;
        text-align: center;
        color: #6b7280;
        font-size: 14px;
    }
</style>