<script lang="ts">
    import { onMount } from 'svelte';
    import { 
        activeTab, 
        openCount, 
        resolvedCount,
        initializeEventListeners,
        loadOpenIncidents,
        loadResolvedIncidents,
        loadServicesConfig,
        activeServiceFilter
    } from './stores/incidents';
    import { initializeNotificationListeners, loadNotificationConfig, loadAvailableSounds } from './stores/notifications';
    import { theme, setTheme } from './stores/theme';
    import { setColorTheme, COLOR_THEME_OPTIONS, type ColorThemeName } from './stores/colorTheme';
    import { GetTheme, GetColorTheme } from '../wailsjs/go/main/App';
    
    import ToolBar from './components/ToolBar.svelte';
    import Header from './components/Header.svelte';
    import IncidentPanel from './components/IncidentPanel.svelte';
    import Settings from './components/Settings.svelte';
    import Panel from './components/Panel.svelte';
    
    let searchQuery = '';
    let sortBy: 'time' | 'service' | 'alerts' = 'time';
    
    onMount(async () => {
        initializeEventListeners();
        initializeNotificationListeners();
        
        try {
            const savedTheme = await GetTheme();
            if (savedTheme === 'light' || savedTheme === 'dark') {
                setTheme(savedTheme);
            }
        } catch (err) {
            console.error('Failed to load theme preference:', err);
        }

        try {
            const savedColorTheme = await GetColorTheme();
            if (COLOR_THEME_OPTIONS.some((option) => option.value === savedColorTheme)) {
                setColorTheme(savedColorTheme as ColorThemeName);
            }
        } catch (err) {
            console.error('Failed to load color theme preference:', err);
        }

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
        activeServiceFilter.set('all'); // reset service pill filter on tab switch
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
        activeServiceFilter.set('all'); // reset service tab filter when sort changes
    }
</script>

<div class="app">
    <ToolBar />
    
    <div class="app-body">
        <div class="main-container">
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
        </div>
        
        <Panel />
    </div>
    
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
    }

    :global(*) {
        box-sizing: border-box;
    }

    .app {
        width: 100vw;
        height: 100vh;
        display: flex;
        flex-direction: column;
        background: var(--bg-secondary);
        color: var(--text-primary);
        overflow: hidden;
    }

    .app-body {
        flex: 1;
        display: flex;
        overflow: hidden;
    }

    .main-container {
        flex: 1;
        display: flex;
        flex-direction: column;
        overflow: hidden;
        min-width: 0;
    }

    .tabs-container {
        display: flex;
        background: var(--bg-primary);
        border-bottom: 1px solid var(--border);
        padding: 0 16px;
    }

    .tab-button {
        display: flex;
        align-items: center;
        gap: 8px;
        background: none;
        border: none;
        padding: 12px 20px;
        cursor: pointer;
        font-size: 14px;
        font-weight: 500;
        color: var(--text-tertiary);
        border-bottom: 2px solid transparent;
        transition: all 0.2s;
        position: relative;
    }

    .tab-button:hover {
        color: var(--text-secondary);
        background: var(--bg-hover);
    }

    .tab-button.active {
        color: var(--accent);
        border-bottom-color: var(--accent);
    }

    .tab-count {
        background: var(--bg-tertiary);
        color: var(--text-tertiary);
        padding: 2px 8px;
        border-radius: 10px;
        font-size: 12px;
        font-weight: 600;
        min-width: 24px;
        text-align: center;
    }

    .tab-button.active .tab-count {
        background: var(--accent-soft-strong);
        color: var(--accent);
    }

    .main-content {
        flex: 1;
        overflow: hidden;
        position: relative;
    }
</style>