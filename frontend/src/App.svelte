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
    
    onMount(async () => {
        // Initialize all event listeners
        initializeEventListeners();
        initializeNotificationListeners();
        
        // Load initial data
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
  </script>
  
  <div class="app">
    <ToolBar />
    
    <Header bind:searchQuery on:search={handleSearch} />
    
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
        <IncidentPanel type="open" {searchQuery} />
        <IncidentPanel type="resolved" {searchQuery} />
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
        display: flex;
        flex-direction: column;
        height: 100vh;
        background: #f9fafb;
    }
    
    .tabs-container {
        background: white;
        display: flex;
        padding: 0 20px;
        border-bottom: 1px solid #e5e7eb;
    }
    
    .tab-button {
        background: transparent;
        border: none;
        padding: 14px 20px;
        cursor: pointer;
        display: flex;
        align-items: center;
        gap: 8px;
        color: #6b7280;
        font-size: 14px;
        font-weight: 500;
        border-bottom: 3px solid transparent;
        transition: all 0.2s;
        margin-right: 8px;
    }
    
    .tab-button:hover {
        color: #374151;
    }
    
    .tab-button.active {
        color: #111827;
        border-bottom-color: #111827;
    }
    
    .tab-label {
        font-weight: 500;
    }
    
    .tab-count {
        background: #f3f4f6;
        color: #374151;
        padding: 2px 8px;
        border-radius: 10px;
        font-size: 12px;
        font-weight: 600;
        min-width: 20px;
        text-align: center;
    }
    
    .tab-button.active .tab-count {
        background: #111827;
        color: white;
    }
    
    .main-content {
        flex: 1;
        overflow: hidden;
        position: relative;
        background: white;
    }
  </style>