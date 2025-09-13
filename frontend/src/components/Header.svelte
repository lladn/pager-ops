<script lang="ts">
    import ServiceFilter from './ServiceFilter.svelte';
    import { createEventDispatcher } from 'svelte';
    
    const dispatch = createEventDispatcher();
    
    export let searchQuery = '';
    
    function handleSearch(event: Event) {
        searchQuery = (event.target as HTMLInputElement).value;
        dispatch('search', searchQuery);
    }
</script>

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
            <ServiceFilter />
        </div>
    </div>
</div>

<style>
    .header {
        background: white;
        padding: 16px 20px;
        border-bottom: 1px solid #e5e7eb;
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
    }
</style>