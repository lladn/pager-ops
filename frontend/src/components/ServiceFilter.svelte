<script lang="ts">
    import { 
        servicesConfig, 
        selectedServices, 
        loadOpenIncidents, 
        loadResolvedIncidents,
        assignedFilterEnabled } from '../stores/incidents';
    import { SetSelectedServices, GetFilterByUser, SetFilterByUser } from '../../wailsjs/go/main/App';
    import { store } from '../../wailsjs/go/models';
    import { onMount } from 'svelte';
    
    let isOpen = false;
    let filterText = 'All Services';
    
    // Local state for immediate UI updates
    let localSelectedServices: string[] = [];
    
    // Sync local state with store
    $: localSelectedServices = [...$selectedServices];
    
    $: if ($servicesConfig) {
        updateFilterText(localSelectedServices, $assignedFilterEnabled);
    }
    
    onMount(async () => {
        try {
            const isAssigned = await GetFilterByUser();
            assignedFilterEnabled.set(isAssigned);
            updateFilterText(localSelectedServices, isAssigned);
        } catch (err) {
            console.error('Failed to get filter mode:', err);
        }
    });
    
    function updateFilterText(selected: string[], assignedMode: boolean) {
        if (!$servicesConfig) {
            filterText = assignedMode ? 'Assigned' : 'All Services';
            return;
        }
        
        const allServiceIds = getAllServiceIds();
        const hasServiceFilter = selected.length > 0 && selected.length < allServiceIds.length;
        
        if (assignedMode && hasServiceFilter) {
            // Both filters active
            let selectedGroups = 0;
            for (const service of $servicesConfig.services) {
                if (isServiceGroupSelected(service, selected)) {
                    selectedGroups++;
                }
            }
            
            if (selectedGroups === 1) {
                const selectedService = $servicesConfig.services.find(s => isServiceGroupSelected(s, selected));
                filterText = `Assigned + ${selectedService?.name || 'Service'}`;
            } else {
                filterText = `Assigned + ${selectedGroups} Services`;
            }
        } else if (assignedMode) {
            // Only assigned mode
            filterText = 'Assigned';
        } else if (hasServiceFilter) {
            // Only service filter
            let selectedGroups = 0;
            for (const service of $servicesConfig.services) {
                if (isServiceGroupSelected(service, selected)) {
                    selectedGroups++;
                }
            }
            
            if (selectedGroups === 1) {
                const selectedService = $servicesConfig.services.find(s => isServiceGroupSelected(s, selected));
                filterText = selectedService?.name || 'Unknown Service';
            } else {
                filterText = `${selectedGroups} Services`;
            }
        } else {
            // No filters
            filterText = 'All Services';
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
    
    async function toggleAssignedMode() {
        const newMode = !$assignedFilterEnabled;
        assignedFilterEnabled.set(newMode);
        
        try {
            await SetFilterByUser(newMode);
            updateFilterText(localSelectedServices, newMode);
            await loadOpenIncidents();
            await loadResolvedIncidents();
        } catch (err) {
            console.error('Failed to toggle assigned mode:', err);
            // Revert on error
            assignedFilterEnabled.set(!newMode);
            updateFilterText(localSelectedServices, !newMode);
        }
    }
    
    
    async function toggleServiceGroup(service: store.ServiceConfig) {
        const serviceIds = typeof service.id === 'string' ? [service.id] : 
                          Array.isArray(service.id) ? service.id : [];
        
        const allSelected = serviceIds.every(id => localSelectedServices.includes(id));
        
        let newSelection: string[];
        if (allSelected) {
            // Remove all IDs from this service group
            newSelection = localSelectedServices.filter(id => !serviceIds.includes(id));
        } else {
            // Add all IDs from this service group
            const toAdd = serviceIds.filter(id => !localSelectedServices.includes(id));
            newSelection = [...localSelectedServices, ...toAdd];
        }
        
        // Update local state immediately
        localSelectedServices = newSelection;
        
        // Update store and backend
        selectedServices.set(newSelection);
        await SetSelectedServices(newSelection);
        await loadOpenIncidents();
        await loadResolvedIncidents();
    }
    
    function isServiceGroupSelected(service: store.ServiceConfig, selected: string[]): boolean {
        const serviceIds = typeof service.id === 'string' ? [service.id] : 
                          Array.isArray(service.id) ? service.id : [];
        return serviceIds.every(id => selected.includes(id));
    }
    
    function isServiceGroupPartiallySelected(service: store.ServiceConfig, selected: string[]): boolean {
        const serviceIds = typeof service.id === 'string' ? [service.id] : 
                          Array.isArray(service.id) ? service.id : [];
        
        if (serviceIds.length === 0) return false;
        const selectedCount = serviceIds.filter(id => selected.includes(id)).length;
        return selectedCount > 0 && selectedCount < serviceIds.length;
    }
</script>

<div class="service-filter">
    <button class="filter-button" on:click={toggleDropdown}>
        <span class="filter-text">{filterText}</span>
        <svg class="chevron" class:rotated={isOpen} width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
            <path fill-rule="evenodd" d="M5.293 7.293a1 1 0 011.414 0L10 10.586l3.293-3.293a1 1 0 111.414 1.414l-4 4a1 1 0 01-1.414 0l-4-4a1 1 0 010-1.414z" clip-rule="evenodd" />
        </svg>
    </button>
    
    {#if isOpen}
        <!-- svelte-ignore a11y-click-events-have-key-events -->
        <!-- svelte-ignore a11y-no-static-element-interactions -->
        <div class="dropdown-backdrop" on:click={closeDropdown}></div>
        <div class="dropdown-menu">
            <!-- Assigned to Me - Added at top -->
            <button class="dropdown-item" on:click={toggleAssignedMode}>
                <span class="checkbox">
                    {#if $assignedFilterEnabled}
                        <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                            <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                        </svg>
                    {/if}
                </span>
                <span>Assigned</span>
            </button>
            
            <div class="dropdown-divider"></div>
            
            {#if $servicesConfig && $servicesConfig.services.length > 0}
                {#each $servicesConfig.services as service}
                    <button class="dropdown-item" on:click={() => toggleServiceGroup(service)}>
                        <span class="checkbox">
                            {#if isServiceGroupSelected(service, localSelectedServices)}
                                <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                                    <path fill-rule="evenodd" d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z" clip-rule="evenodd" />
                                </svg>
                            {:else if isServiceGroupPartiallySelected(service, localSelectedServices)}
                                <span class="partial-check">−</span>
                            {/if}
                        </span>
                        <span>{service.name}</span>
                    </button>
                {/each}
            {:else}
                <div class="no-services">
                    No services configured
                </div>
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
        padding: 8px 12px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        transition: all 0.2s;
        min-width: 160px;
    }
    
    .filter-button:hover {
        background: #f9fafb;
        border-color: #d1d5db;
    }
    
    .filter-text {
        flex: 1;
        text-align: left;
    }
    
    .chevron {
        transition: transform 0.2s;
        flex-shrink: 0;
    }
    
    .chevron.rotated {
        transform: rotate(180deg);
    }
    
    .dropdown-backdrop {
        position: fixed;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        z-index: 999;
    }
    
    .dropdown-menu {
        position: absolute;
        top: calc(100% + 4px);
        left: 0;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06);
        min-width: 200px;
        max-height: 400px;
        overflow-y: auto;
        z-index: 1000;
    }
    
    .dropdown-item {
        display: flex;
        align-items: center;
        gap: 8px;
        width: 100%;
        padding: 10px 12px;
        border: none;
        background: none;
        cursor: pointer;
        font-size: 14px;
        color: #374151;
        text-align: left;
        transition: background-color 0.15s;
    }
    
    .dropdown-item:hover {
        background: #f3f4f6;
    }
    
    .checkbox {
        width: 16px;
        height: 16px;
        border: 2px solid #d1d5db;
        border-radius: 4px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
    }
    
    .dropdown-item:hover .checkbox {
        border-color: #9ca3af;
    }
    
    .partial-check {
        color: #6b7280;
        font-size: 18px;
        line-height: 1;
    }
    
    .dropdown-divider {
        height: 1px;
        background: #e5e7eb;
        margin: 4px 0;
    }
    
    .no-services {
        padding: 20px;
        text-align: center;
        color: #9ca3af;
        font-size: 14px;
    }
</style>