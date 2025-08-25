<script>
  import { onMount } from 'svelte';
  import Settings from './components/Settings.svelte';
  import IncidentsPanel from './components/IncidentsPanel.svelte';
  import { LogDebug } from '../wailsjs/runtime/runtime.js';
  
  let showSettings = false;
  
  onMount(() => {
    LogDebug("App mounted successfully");
    
    // Log any click events on the app
    document.addEventListener('click', (e) => {
      LogDebug(`Global click detected on: ${e.target.tagName}, class: ${e.target.className}`);
    });
  });
  
  function toggleSettings() {
    LogDebug(`Settings toggle clicked, current state: ${showSettings}`);
    console.log('Settings button clicked, current state:', showSettings);
    showSettings = !showSettings;
    LogDebug(`Settings new state: ${showSettings}`);
  }
</script>

<main style="--wails-draggable: no-drag;">
  <div class="app-header">
    <h1>PagerOps</h1>
    <button 
      class="settings-btn" 
      on:click={toggleSettings}
      on:mousedown|preventDefault|stopPropagation
      on:mouseup|preventDefault|stopPropagation
      type="button"
      style="--wails-draggable: no-drag;">
      <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
        <circle cx="12" cy="12" r="3"></circle>
        <path d="M12 1v6m0 6v6m4.22-13.22l4.24 4.24M1.54 1.54l4.24 4.24M20.46 20.46l-4.24-4.24M1.54 20.46l4.24-4.24M23 12h-6m-6 0H1"></path>
      </svg>
    </button>
  </div>
  
  <div class="app-content" style="--wails-draggable: no-drag;">
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

  :global(*) {
    box-sizing: border-box;
  }

  main {
    height: 100vh;
    display: flex;
    flex-direction: column;
    background: #1a1a1a;
    overflow: hidden;
  }

  .app-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 15px 20px;
    background: #111;
    border-bottom: 1px solid #333;
    position: relative;
    z-index: 10;
  }

  .app-header h1 {
    margin: 0;
    color: #fff;
    font-size: 20px;
    user-select: none;
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

  .settings-btn:active {
    transform: scale(0.95);
  }

  .app-content {
    flex: 1;
    overflow: hidden;
    position: relative;
  }
</style>