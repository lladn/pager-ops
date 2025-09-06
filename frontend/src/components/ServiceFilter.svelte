<script lang="ts">
    import { servicesConfig, selectedServices, loadOpenIncidents, loadResolvedIncidents } from '../stores/incidents';
    import { SetSelectedServices } from '../../wailsjs/go/main/App';
    
    let isOpen = false;
    let filterText = 'All Services';
    
    $: if ($servicesConfig) {
        updateFilterText($selectedServices);
    }
    
    function updateFilterText(selected: string[]) {
        if (!$servicesConfig) {
            filterText = 'All Services';
            return;
        }
        
        if (selected.length === 0 || selected.length === $servicesConfig.services.length) {
            filterText = 'All Services';
        } else if (selected.length === 1) {
            const service = $servicesConfig.services.find(s => {
                const serviceId = typeof s.id === 'string' ? s.id : String(s.id);
                return serviceId === selected[0];
            });
            filterText = service?.name || 'Unknown Service';
        } else {
            filterText = `${selected.length} Services`;
        }
    }
    
    function toggleDropdown() {
        isOpen = !isOpen;
    }
    
    function closeDropdown() {
        isOpen = false;
    }
    
    async function selectAllServices() {
        if (!$servicesConfig) return;
        
        const allServiceIds = $servicesConfig.services.map(s => 
            typeof s.id === 'string' ? s.id : String(s.id)
        );
        
        selectedServices.set(allServiceIds);
        await SetSelectedServices(allServiceIds);
        await loadOpenIncidents();
        await loadResolvedIncidents();
        closeDropdown();
    }
    
    async function selectService(serviceId: string) {
        selectedServices.set([serviceId]);
        await SetSelectedServices([serviceId]);
        await loadOpenIncidents();
        await loadResolvedIncidents();
        closeDropdown();
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
                <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                    <path d="M5 3a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2V5a2 2 0 00-2-2H5zm0 2h10v10H5V5z"/>
                    {#if $selectedServices.length === 0 || ($servicesConfig && $selectedServices.length === $servicesConfig.services.length)}
                        <path d="M8 11l2 2 4-4" stroke="currentColor" stroke-width="2"/>
                    {/if}
                </svg>
                All Services
            </button>
            
            {#if $servicesConfig}
                {#each $servicesConfig.services as service}
                    {@const serviceId = typeof service.id === 'string' ? service.id : String(service.id)}
                    <button class="dropdown-item" on:click={() => selectService(serviceId)}>
                        <svg width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                            <path d="M5 3a2 2 0 00-2 2v10a2 2 0 002 2h10a2 2 0 002-2V5a2 2 0 00-2-2H5zm0 2h10v10H5V5z"/>
                            {#if $selectedServices.includes(serviceId)}
                                <path d="M8 11l2 2 4-4" stroke="currentColor" stroke-width="2"/>
                            {/if}
                        </svg>
                        {service.name}
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
    
    .dropdown-empty {
        padding: 12px;
        text-align: center;
        color: #6b7280;
        font-size: 14px;
    }
</style>