<script>
  import Settings from './components/Settings.svelte';
  import IncidentsPanel from './components/IncidentsPanel.svelte';
  
  let showSettings = false;
  
  function toggleSettings() {
    showSettings = !showSettings;
  }
</script>

<main>
  <div class="app-header">
    <h1>PagerOps</h1>
    <button class="settings-btn" on:click={toggleSettings}>
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"></circle>
        <path d="M12 1v6m0 6v6m4.22-13.22l4.24 4.24M1.54 1.54l4.24 4.24M20.46 20.46l-4.24-4.24M1.54 20.46l4.24-4.24M23 12h-6m-6 0H1"></path>
      </svg>
    </button>
  </div>
  
  <div class="app-content">
    <IncidentsPanel />
  </div>
  
  <Settings bind:isOpen={showSettings} />
</main>

<style>
  :global(body) {
    margin: 0;
    padding: 0;
    font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, 'Open Sans', 'Helvetica Neue', sans-serif;
  }

  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: #1a1a1a;
  }

  .app-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    background: #111;
    border-bottom: 1px solid #333;
  }

  .app-header h1 {
    margin: 0;
    color: #fff;
    font-size: 20px;
  }

  .settings-btn {
    background: none;
    border: none;
    color: #999;
    cursor: pointer;
    padding: 8px;
    border-radius: 4px;
    transition: all 0.3s;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .settings-btn:hover {
    background: #333;
    color: #fff;
  }

  .app-content {
    flex: 1;
    overflow: hidden;
  }
</style>