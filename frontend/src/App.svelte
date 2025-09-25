<script lang="ts">
    import { onMount } from 'svelte';
    import { 
        activeTab, 
        openCount, 
        resolvedCount,
        initializeEventListeners,
        loadOpenIncidents,
        loadResolvedIncidents,
        loadServicesConfig
    } from './stores/incidents';
    import { initializeNotificationListeners, loadNotificationConfig, loadAvailableSounds } from './stores/notifications';
    
    import ToolBar from './components/ToolBar.svelte';
    import Header from './components/Header.svelte';
    import IncidentPanel from './components/IncidentPanel.svelte';
    import Settings from './components/Settings.svelte';
    
    let searchQuery = '';
    let sortBy: 'time' | 'service' | 'alerts' = 'time';
    
    onMount(async () => {
        initializeEventListeners();
        initializeNotificationListeners();
        
        await Promise.all([
            loadOpenIncidents(),
            loadResolvedIncidents(),
            loadServicesConfig(),
            loadNotificationConfig(),
            loadAvailableSounds()
        ]);
    });
    
    function switchTab(tab: 'open' | 'resolved') {
        activeTab.set(tab);
        if (tab === 'open') {
            loadOpenIncidents();
        } else {
            loadResolvedIncidents();
        }
    }
    
    function handleSearch(event: CustomEvent) {
        searchQuery = event.detail;
    }
    
    function handleSort(event: CustomEvent) {
        sortBy = event.detail;
    }
</script>

<div class="app">
    <ToolBar />
    
    <Header bind:searchQuery bind:sortBy on:search={handleSearch} on:sort={handleSort} />
    
    <div class="tabs-container">
        <button 
            class="tab-button" 
            class:active={$activeTab === 'open'}
            on:click={() => switchTab('open')}
        >
            <span class="tab-label">Open</span>
            <span class="tab-count">{$openCount}</span>
        </button>
        <button 
            class="tab-button" 
            class:active={$activeTab === 'resolved'}
            on:click={() => switchTab('resolved')}
        >
            <span class="tab-label">Resolved</span>
            <span class="tab-count">{$resolvedCount}</span>
        </button>
    </div>
    
    <main class="main-content">
        <IncidentPanel type="open" {searchQuery} {sortBy} />
        <IncidentPanel type="resolved" {searchQuery} {sortBy} />
    </main>
    
    <Settings />
</div>

<style>
    :global(body) {
        margin: 0;
        padding: 0;
        font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
            'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
            sans-serif;
        -webkit-font-smoothing: antialiased;
        -moz-osx-font-smoothing: grayscale;
        background: #f9fafb;
    }
    
    :global(*) {
        box-sizing: border-box;
    }
    
    .app {
        width: 100%;
        height: 100vh;
        display: flex;
        flex-direction: column;
        background: #f9fafb;
    }
    
    .tabs-container {
        background: white;
        border-bottom: 1px solid #e5e7eb;
        display: flex;
        padding: 0 20px;
    }
    
    .tab-button {
        padding: 12px 16px;
        background: transparent;
        border: none;
        border-bottom: 2px solid transparent;
        font-size: 14px;
        font-weight: 500;
        color: #6b7280;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        transition: all 0.2s;
    }
    
    .tab-button:hover {
        color: #374151;
    }
    
    .tab-button.active {
        color: #2563eb;
        border-bottom-color: #2563eb;
    }
    
    .tab-label {
        font-weight: 500;
    }
    
    .tab-count {
        padding: 2px 8px;
        background: #e5e7eb;
        border-radius: 10px;
        font-size: 12px;
        font-weight: 600;
        min-width: 20px;
        text-align: center;
    }
    
    .tab-button.active .tab-count {
        background: #dbeafe;
        color: #2563eb;
    }
    
    .main-content {
        flex: 1;
        overflow: hidden;
        position: relative;
    }
</style>