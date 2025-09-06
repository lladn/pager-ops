<script lang="ts">
  import { onMount } from 'svelte';
  import { 
      activeTab, 
      openCount, 
      resolvedCount, 
      settingsOpen,
      initializeEventListeners,
      loadOpenIncidents,
      loadResolvedIncidents,
      loadServicesConfig
  } from './stores/incidents';
  
  import IncidentPanel from './components/IncidentPanel.svelte';
  import ServiceFilter from './components/ServiceFilter.svelte';
  import Settings from './components/Settings.svelte';
  
  onMount(async () => {
      // Initialize event listeners for backend updates
      initializeEventListeners();
      
      // Load initial data
      await loadServicesConfig();
      await loadOpenIncidents();
      await loadResolvedIncidents();
  });
  
  function switchTab(tab: 'open' | 'resolved') {
      activeTab.set(tab);
      if (tab === 'open') {
          loadOpenIncidents();
      } else {
          loadResolvedIncidents();
      }
  }
  
  function openSettings() {
      settingsOpen.set(true);
  }
</script>

<div class="app">
  <header class="app-header">
      <div class="header-content">
          <div class="header-left">
              <h3 class="app-title">PagerOps</h3>
              <p class="app-subtitle">Monitor and manage your incidents</p>
          </div>
          <div class="header-right">
              <ServiceFilter />
              <button class="settings-button" on:click={openSettings} title="Settings">
                  <svg width="20" height="20" viewBox="0 0 20 20" fill="currentColor">
                      <path fill-rule="evenodd" d="M11.49 3.17c-.38-1.56-2.6-1.56-2.98 0a1.532 1.532 0 01-2.286.948c-1.372-.836-2.942.734-2.106 2.106.54.886.061 2.042-.947 2.287-1.561.379-1.561 2.6 0 2.978a1.532 1.532 0 01.947 2.287c-.836 1.372.734 2.942 2.106 2.106a1.532 1.532 0 012.287.947c.379 1.561 2.6 1.561 2.978 0a1.533 1.533 0 012.287-.947c1.372.836 2.942-.734 2.106-2.106a1.533 1.533 0 01.947-2.287c1.561-.379 1.561-2.6 0-2.978a1.532 1.532 0 01-.947-2.287c.836-1.372-.734-2.942-2.106-2.106a1.532 1.532 0 01-2.287-.947zM10 13a3 3 0 100-6 3 3 0 000 6z" clip-rule="evenodd" />
                  </svg>
              </button>
          </div>
      </div>
  </header>
  
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
      <IncidentPanel type="open" />
      <IncidentPanel type="resolved" />
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
  
  .app-header {
      background: white;
      border-bottom: 1px solid #e5e7eb;
      padding: 16px 24px;
  }
  
  .header-content {
      display: flex;
      justify-content: space-between;
      align-items: center;
  }
  
  .header-left {
      flex: 1;
  }
  
  .app-title {
      font-size: 20px;
      font-weight: 700;
      color: #111827;
      margin: 0 0 4px 0;
  }
  
  .app-subtitle {
      font-size: 14px;
      color: #6b7280;
      margin: 0;
  }
  
  .header-right {
      display: flex;
      align-items: center;
      gap: 12px;
  }
  
  .settings-button {
      background: transparent;
      border: 1px solid #e5e7eb;
      border-radius: 8px;
      padding: 8px;
      cursor: pointer;
      color: #6b7280;
      transition: all 0.2s ease;
      display: flex;
      align-items: center;
      justify-content: center;
  }
  
  .settings-button:hover {
      background: #f3f4f6;
      color: #111827;
  }
  
  .tabs-container {
      display: flex;
      background: white;
      padding: 0 24px;
      border-bottom: 1px solid #e5e7eb;
  }
  
  .tab-button {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 16px 24px;
      background: transparent;
      border: none;
      border-bottom: 2px solid transparent;
      cursor: pointer;
      font-size: 15px;
      font-weight: 500;
      color: #6b7280;
      transition: all 0.2s ease;
      margin-bottom: -1px;
  }
  
  .tab-button:hover {
      color: #374151;
  }
  
  .tab-button.active {
      color: #111827;
      background: #f3f4f6;
      border-bottom-color: #111827;
      border-radius: 8px 8px 0 0;
  }
  
  .tab-label {
      font-weight: 600;
  }
  
  .tab-count {
      background: #e5e7eb;
      color: #374151;
      padding: 2px 8px;
      border-radius: 12px;
      font-size: 13px;
      font-weight: 600;
      min-width: 24px;
      text-align: center;
  }
  
  .tab-button.active .tab-count {
      background: #111827;
      color: white;
  }
  
  .main-content {
      flex: 1;
      overflow: hidden;
      background: white;
  }
</style>