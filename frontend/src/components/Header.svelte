<script lang="ts">
    import ServiceFilter from './ServiceFilter.svelte';
    import { createEventDispatcher } from 'svelte';
    
    const dispatch = createEventDispatcher();
    
    export let searchQuery = '';
    export let sortBy: 'time' | 'service' | 'alerts' = 'time';
    
    let showSortMenu = false;
    
    function handleSearch(event: Event) {
        searchQuery = (event.target as HTMLInputElement).value;
        dispatch('search', searchQuery);
    }
    
    function toggleSortMenu() {
        showSortMenu = !showSortMenu;
    }
    
    function selectSort(option: 'time' | 'service' | 'alerts') {
        sortBy = option;
        showSortMenu = false;
        dispatch('sort', sortBy);
    }
    
    function handleClickOutside(event: MouseEvent) {
        const target = event.target as HTMLElement;
        if (!target.closest('.sort-dropdown')) {
            showSortMenu = false;
        }
    }
</script>

<svelte:window on:click={handleClickOutside} />

<div class="header">
    <div class="header-content">
        <div class="search-container">
            <svg class="search-icon" width="16" height="16" viewBox="0 0 20 20" fill="currentColor">
                <path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd" />
            </svg>
            <input 
                type="text" 
                class="search-input" 
                placeholder="Search incidents..."
                bind:value={searchQuery}
                on:input={handleSearch}
            />
        </div>
        <div class="filter-container">
            <div class="sort-dropdown">
                <button class="sort-button" on:click={toggleSortMenu}>
                    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <line x1="4" y1="6" x2="20" y2="6"></line>
                        <line x1="4" y1="12" x2="20" y2="12"></line>
                        <line x1="4" y1="18" x2="16" y2="18"></line>
                    </svg>
                    <svg class="chevron" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
                        <polyline points="6 9 12 15 18 9"></polyline>
                    </svg>
                </button>
                {#if showSortMenu}
                    <div class="sort-menu">
                        <div class="sort-header">Sort by</div>
                        <button 
                            class="sort-option" 
                            class:active={sortBy === 'time'}
                            on:click={() => selectSort('time')}
                        >
                            <span>time</span>
                            {#if sortBy === 'time'}
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                                </svg>
                            {/if}
                        </button>
                        <button 
                            class="sort-option" 
                            class:active={sortBy === 'service'}
                            on:click={() => selectSort('service')}
                        >
                            <span>service</span>
                            {#if sortBy === 'service'}
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                                </svg>
                            {/if}
                        </button>
                        <button 
                            class="sort-option" 
                            class:active={sortBy === 'alerts'}
                            on:click={() => selectSort('alerts')}
                        >
                            <span>alerts</span>
                            {#if sortBy === 'alerts'}
                                <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
                                    <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                                </svg>
                            {/if}
                        </button>
                    </div>
                {/if}
            </div>
            <ServiceFilter />
        </div>
    </div>
</div>

<style>
    .header {
        background: white;
        padding: 16px 20px;
        border-bottom: 1px solid #e5e7eb;
        /* Prevent accidental text highlight */
        -webkit-user-select: none;
        -ms-user-select: none;
        user-select: none;
    }
    
    .header-content {
        display: flex;
        gap: 12px;
        align-items: center;
    }
    
    .search-container {
        flex: 1;
        position: relative;
        display: flex;
        align-items: center;
    }
    
    .search-icon {
        position: absolute;
        left: 12px;
        color: #9ca3af;
        pointer-events: none;
    }
    
    .search-input {
        width: 100%;
        padding: 8px 12px 8px 36px;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        font-size: 14px;
        background: #f9fafb;
        transition: all 0.2s;
    }
    
    .search-input:focus {
        outline: none;
        border-color: #d1d5db;
        background: white;
        box-shadow: 0 1px 2px 0 rgba(0, 0, 0, 0.05);
    }
    
    .search-input::placeholder {
        color: #9ca3af;
    }
    
    .filter-container {
        flex-shrink: 0;
        display: flex;
        gap: 8px;
    }
    
    .sort-dropdown {
        position: relative;
    }
    
    .sort-button {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 8px 12px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        font-size: 14px;
        color: #374151;
        cursor: pointer;
        transition: all 0.2s;
    }
    
    .sort-button:hover {
        background: #f9fafb;
        border-color: #d1d5db;
    }
    
    .chevron {
        color: #6b7280;
    }
    
    .sort-menu {
        position: absolute;
        top: calc(100% + 4px);
        right: 0;
        min-width: 150px;
        background: white;
        border: 1px solid #e5e7eb;
        border-radius: 8px;
        box-shadow: 0 10px 15px -3px rgba(0, 0, 0, 0.1);
        z-index: 50;
        overflow: hidden;
    }
    
    .sort-header {
        padding: 8px 12px;
        font-size: 12px;
        font-weight: 600;
        color: #6b7280;
        text-transform: uppercase;
        border-bottom: 1px solid #e5e7eb;
        background: #f9fafb;
        cursor: default;
    }
    
    .sort-option {
        width: 100%;
        padding: 10px 12px;
        background: white;
        border: none;
        font-size: 14px;
        color: #374151;
        cursor: pointer;
        transition: all 0.2s;
        text-align: left;
        display: flex;
        justify-content: space-between;
        align-items: center;
    }
    
    .sort-option:hover {
        background: #f9fafb;
    }
    
    .sort-option.active {
        background: #eff6ff;
        color: #2563eb;
    }
    
    .sort-option svg {
        color: #2563eb;
    }
</style>